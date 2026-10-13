package plan

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// The relation query and the share cleanup are SQL, so they are proven against
// the local Postgres (docker compose up postgres). Skipped under -short, like
// the checklistitem integration tests.

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func openTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("needs the plan database; run without -short")
	}
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		env("DB_HOST", "localhost"), env("DB_PORT", "5500"),
		env("DB_USER", "fireplace_plans_app"), env("DB_PASSWORD", "app"),
		env("DB_NAME", "fireplace_plans"),
	)
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("plan database unreachable (%v)", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// seedPlan makes a plan owned by owner and removes it afterwards; its shares
// go with it (ON DELETE CASCADE).
func seedPlan(t *testing.T, db *sqlx.DB, owner uuid.UUID) uuid.UUID {
	t.Helper()
	planID := uuid.New()
	_, err := db.Exec(
		`INSERT INTO plans (id, user_id, name, plan_type, focus, description) VALUES ($1, $2, $3, $4, $5, '')`,
		planID, owner, "FS-12ZMF access fixture", "project", "access fixture",
	)
	if err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM plans WHERE id = $1`, planID); err != nil {
			t.Errorf("cleanup plan %s: %v", planID, err)
		}
	})
	return planID
}

func seedShare(t *testing.T, db *sqlx.DB, planID, userID uuid.UUID, role string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO plan_shares (plan_id, user_id, role) VALUES ($1, $2, $3)`, planID, userID, role); err != nil {
		t.Fatalf("seed share: %v", err)
	}
}

func shareCount(t *testing.T, db *sqlx.DB, userID uuid.UUID) int {
	t.Helper()
	var n int
	if err := db.Get(&n, `SELECT COUNT(*) FROM plan_shares WHERE user_id = $1`, userID); err != nil {
		t.Fatalf("count shares: %v", err)
	}
	return n
}

func TestIntegrationGetAccess_AnswersTheRelation(t *testing.T) {
	db := openTestDB(t)
	owner, member, viewer, stranger := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	planID := seedPlan(t, db, owner)
	seedShare(t, db, planID, member, "edit")
	seedShare(t, db, planID, viewer, "view") // role is ignored in v1 (R3)
	// A share on another plan must not leak into this one.
	seedShare(t, db, seedPlan(t, db, uuid.New()), stranger, "edit")

	svc := NewService(NewRepository(db), nil, nil)
	cases := []struct {
		name      string
		planID    uuid.UUID
		user      uuid.UUID
		want      Relation
		wantOwner uuid.UUID
	}{
		{"owner", planID, owner, RelationOwner, owner},
		{"member", planID, member, RelationMember, owner},
		{"view-role member", planID, viewer, RelationMember, owner},
		{"stranger", planID, stranger, RelationNone, owner},
		{"missing plan", uuid.New(), owner, RelationNone, uuid.Nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := svc.GetAccess(context.Background(), tc.planID, tc.user)
			if err != nil {
				t.Fatalf("GetAccess: %v", err)
			}
			if got.Relation != tc.want || got.OwnerID != tc.wantOwner {
				t.Errorf("got (%s, %s), want (%s, %s)", got.Relation, got.OwnerID, tc.want, tc.wantOwner)
			}
		})
	}
}

// R25 against the real tables: after user.deleted no share row names the user,
// their own plans are gone, other people's plans stay, and redelivery succeeds.
func TestIntegrationCascadeDeleteForUser_ClearsShares(t *testing.T) {
	db := openTestDB(t)
	gone, other := uuid.New(), uuid.New()
	othersPlan := seedPlan(t, db, other)
	seedShare(t, db, othersPlan, gone, "edit")
	seedShare(t, db, seedPlan(t, db, other), gone, "view")
	ownPlan := seedPlan(t, db, gone)
	seedShare(t, db, ownPlan, other, "edit")

	svc := NewService(NewRepository(db), nil, nil)
	for delivery := 1; delivery <= 2; delivery++ {
		if err := svc.CascadeDeleteForUser(context.Background(), gone); err != nil {
			t.Fatalf("delivery %d: %v", delivery, err)
		}
	}

	if n := shareCount(t, db, gone); n != 0 {
		t.Errorf("%d share rows still name the deleted user", n)
	}
	if n := shareCount(t, db, other); n != 0 {
		t.Errorf("the share on the deleted user's own plan survived (%d)", n)
	}
	var plans int
	if err := db.Get(&plans, `SELECT COUNT(*) FROM plans WHERE id = ANY($1::uuid[])`,
		fmt.Sprintf("{%s,%s}", ownPlan, othersPlan)); err != nil {
		t.Fatalf("count plans: %v", err)
	}
	if plans != 1 {
		t.Errorf("%d of the two plans remain, want only the other user's", plans)
	}
}

// R17 against the real table: a re-add writes no second row, keeps the original
// since, and the row is always role 'edit'.
func TestIntegrationAddMember_ReAddKeepsOneEditRowAndItsSince(t *testing.T) {
	db := openTestDB(t)
	owner, member := uuid.New(), uuid.New()
	planID := seedPlan(t, db, owner)
	svc := NewService(NewRepository(db), nil, nil)

	first, err := svc.AddMember(context.Background(), planID, owner, member)
	if err != nil {
		t.Fatalf("first add: %v", err)
	}
	again, err := svc.AddMember(context.Background(), planID, owner, member)
	if err != nil {
		t.Fatalf("second add: %v", err)
	}
	if !again.Since.Equal(first.Since) {
		t.Errorf("since = %s, want the original %s", again.Since, first.Since)
	}
	var roles []string
	if err := db.Select(&roles, `SELECT role FROM plan_shares WHERE plan_id = $1 AND user_id = $2`, planID, member); err != nil {
		t.Fatalf("read shares: %v", err)
	}
	if len(roles) != 1 || roles[0] != "edit" {
		t.Errorf("share rows = %v, want exactly one 'edit'", roles)
	}
}

// R15 against the real tables: members in add order, without another plan's
// shares or a stray row naming the owner.
func TestIntegrationListMembers_InAddOrderWithoutTheOwner(t *testing.T) {
	db := openTestDB(t)
	owner, first, second := uuid.New(), uuid.New(), uuid.New()
	planID := seedPlan(t, db, owner)
	if _, err := db.Exec(`INSERT INTO plan_shares (plan_id, user_id, created_at) VALUES
		($1, $2, NOW() - INTERVAL '2 hours'), ($1, $3, NOW() - INTERVAL '1 hour'), ($1, $4, NOW())`,
		planID, first, second, owner); err != nil {
		t.Fatalf("seed shares: %v", err)
	}
	seedShare(t, db, seedPlan(t, db, uuid.New()), uuid.New(), "edit")

	got, err := NewService(NewRepository(db), nil, nil).ListMembers(context.Background(), planID)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if got.OwnerID != owner || got.PlanCreatedAt.IsZero() {
		t.Errorf("owner/created = %s/%s", got.OwnerID, got.PlanCreatedAt)
	}
	if len(got.Members) != 2 || got.Members[0].UserID != first || got.Members[1].UserID != second {
		t.Fatalf("members = %+v, want [%s %s]", got.Members, first, second)
	}
}

// R19 against the real table: removal deletes the row; removing it again, or
// a stranger's id, is NotFound.
func TestIntegrationRemoveMember_DeletesOnceThenNotFound(t *testing.T) {
	db := openTestDB(t)
	owner, member := uuid.New(), uuid.New()
	planID := seedPlan(t, db, owner)
	seedShare(t, db, planID, member, "edit")
	svc := NewService(NewRepository(db), nil, nil)

	if err := svc.RemoveMember(context.Background(), planID, owner, member); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
	if n := shareCount(t, db, member); n != 0 {
		t.Errorf("%d share rows remain", n)
	}
	for name, target := range map[string]uuid.UUID{"again": member, "stranger": uuid.New()} {
		if err := svc.RemoveMember(context.Background(), planID, owner, target); !errors.Is(err, commonconstants.ErrNotFound) {
			t.Errorf("%s: err = %v, want ErrNotFound", name, err)
		}
	}
}

// R22 against the real tables: owned ∪ shared, each plan once even when a
// share row names the caller's own plan, newest first, nobody else's plans.
// The owned-only list is untouched.
func TestIntegrationListAccessible_OwnedAndSharedOnceNewestFirst(t *testing.T) {
	db := openTestDB(t)
	member, other := uuid.New(), uuid.New()
	owned := seedPlan(t, db, member)
	shared := seedPlan(t, db, other)
	seedShare(t, db, shared, member, "edit")
	seedShare(t, db, owned, member, "edit") // must not list the owned plan twice
	seedPlan(t, db, other)                  // not shared: never listed
	if _, err := db.Exec(`UPDATE plans SET created_at = now() - interval '1 day' WHERE id = $1`, owned); err != nil {
		t.Fatalf("age owned plan: %v", err)
	}
	svc := NewService(NewRepository(db), nil, nil)

	all, err := svc.ListAccessible(context.Background(), member)
	if err != nil {
		t.Fatalf("ListAccessible: %v", err)
	}
	if len(all) != 2 || all[0].ID != shared || all[1].ID != owned {
		ids := []uuid.UUID{}
		for _, p := range all {
			ids = append(ids, p.ID)
		}
		t.Errorf("got %v, want [shared %s, owned %s]", ids, shared, owned)
	}

	ownedOnly, err := svc.ListByUser(context.Background(), member)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(ownedOnly) != 1 || ownedOnly[0].ID != owned {
		t.Errorf("owned-only list has %d plans, want just the owned one", len(ownedOnly))
	}
}
