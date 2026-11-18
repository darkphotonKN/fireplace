package notecomment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/services/plan-service/internal/checklistitem"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// FS-65B3N R1-R3, R5, R7, R11, R20-R21 against the real tables. Needs the
// plan database (as checklistitem's integration tests do) and is skipped
// under -short or when it is unreachable.

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
	var exists bool
	if err := db.Get(&exists, `SELECT to_regclass('public.note_comments') IS NOT NULL`); err != nil || !exists {
		t.Skip("note_comments not migrated in the plan database")
	}
	return db
}

// seedPlan makes a plan and removes it afterwards; items and their comments
// go with it by cascade.
func seedPlan(t *testing.T, db *sqlx.DB) uuid.UUID {
	t.Helper()
	planID := uuid.New()
	if _, err := db.Exec(
		`INSERT INTO plans (id, user_id, name, plan_type, focus) VALUES ($1, $2, $3, $4, $5)`,
		planID, uuid.New(), "FS-65B3N fixture", "project", "comments fixture",
	); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM plans WHERE id = $1`, planID); err != nil {
			t.Errorf("cleanup plan %s: %v", planID, err)
		}
	})
	return planID
}

var updatedLongAgo = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

func seedItem(t *testing.T, db *sqlx.DB, planID uuid.UUID, itemType string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if _, err := db.Exec(`
		INSERT INTO checklist_items (id, plan_id, description, done, sequence, scope, type, created_at, updated_at)
		VALUES ($1, $2, 'the body', false, 1, 'longterm', $3, $4, $4)`,
		id, planID, itemType, updatedLongAgo,
	); err != nil {
		t.Fatalf("seed %s: %v", itemType, err)
	}
	return id
}

func countComments(t *testing.T, db *sqlx.DB, itemID uuid.UUID) int {
	t.Helper()
	var n int
	if err := db.Get(&n, `SELECT count(*) FROM note_comments WHERE item_id = $1`, itemID); err != nil {
		t.Fatalf("count comments: %v", err)
	}
	return n
}

func TestIntegrationCreate_OnANote_StoresItByTheAuthor(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)
	noteID := seedItem(t, db, planID, "note")
	author := uuid.New()

	c, err := NewRepository(db).Create(context.Background(), CreateInput{PlanID: planID, ItemID: noteID, AuthorID: author, Body: "hello"})

	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.ID == uuid.Nil || c.ItemID != noteID || c.AuthorID != author || c.Body != "hello" || c.CreatedAt.IsZero() || c.EditedAt != nil {
		t.Fatalf("comment = %+v", c)
	}
}

// R7: adding a comment leaves the item's description and updated_at alone.
func TestIntegrationCreate_LeavesTheItemUntouched(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)
	noteID := seedItem(t, db, planID, "note")

	if _, err := NewRepository(db).Create(context.Background(), CreateInput{PlanID: planID, ItemID: noteID, AuthorID: uuid.New(), Body: "x"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	var row struct {
		Description string    `db:"description"`
		UpdatedAt   time.Time `db:"updated_at"`
	}
	if err := db.Get(&row, `SELECT description, updated_at FROM checklist_items WHERE id = $1`, noteID); err != nil {
		t.Fatal(err)
	}
	if row.Description != "the body" || !row.UpdatedAt.Equal(updatedLongAgo) {
		t.Fatalf("item = %+v, want it unchanged", row)
	}
}

// R2, R3: a task is ErrNotANote, another plan's note ErrNotFound; nothing is
// stored for either, on create or on list.
func TestIntegrationRefusals(t *testing.T) {
	db := openTestDB(t)
	planID, otherPlan := seedPlan(t, db), seedPlan(t, db)
	taskID := seedItem(t, db, planID, "task")
	foreignNote := seedItem(t, db, otherPlan, "note")
	repo := NewRepository(db)
	ctx := context.Background()

	cases := []struct {
		name   string
		itemID uuid.UUID
		want   error
	}{
		{"task", taskID, ErrNotANote},
		{"other plan", foreignNote, commonconstants.ErrNotFound},
		{"missing", uuid.New(), commonconstants.ErrNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := repo.Create(ctx, CreateInput{PlanID: planID, ItemID: tc.itemID, AuthorID: uuid.New(), Body: "x"})
			if !errors.Is(err, tc.want) {
				t.Fatalf("create err = %v, want %v", err, tc.want)
			}
			if _, err := repo.List(ctx, planID, tc.itemID); !errors.Is(err, tc.want) {
				t.Fatalf("list err = %v, want %v", err, tc.want)
			}
			if n := countComments(t, db, tc.itemID); n != 0 {
				t.Fatalf("%d comments stored", n)
			}
		})
	}
}

// R11: oldest first, ties broken by id; none is empty.
func TestIntegrationList_OldestFirstTiesByID(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)
	noteID := seedItem(t, db, planID, "note")
	repo := NewRepository(db)

	if got, err := repo.List(context.Background(), planID, noteID); err != nil || len(got) != 0 {
		t.Fatalf("empty note: got %v, err %v", got, err)
	}

	at := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	lowID, highID := uuid.MustParse("00000000-0000-0000-0000-000000000001"), uuid.MustParse("ffffffff-0000-0000-0000-000000000001")
	insert := func(id uuid.UUID, created time.Time) {
		t.Helper()
		if _, err := db.Exec(`INSERT INTO note_comments (id, item_id, author_id, body, created_at) VALUES ($1, $2, $3, 'b', $4)`,
			id, noteID, uuid.New(), created); err != nil {
			t.Fatal(err)
		}
	}
	first := uuid.New()
	insert(highID, at.Add(time.Hour))
	insert(lowID, at.Add(time.Hour))
	insert(first, at)

	got, err := repo.List(context.Background(), planID, noteID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 || got[0].ID != first || got[1].ID != lowID || got[2].ID != highID {
		t.Fatalf("order = %v, want first, then the tie by id", ids(got))
	}
}

func ids(cs []*Comment) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.ID)
	}
	return out
}

// R20, R21: deleting the note deletes its comments, archiving leaves them,
// and a duplicate starts with none.
func TestIntegrationLifecycle(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)
	noteID := seedItem(t, db, planID, "note")
	repo := NewRepository(db)
	items := checklistitem.NewRepository(db)
	ctx := context.Background()
	if _, err := repo.Create(ctx, CreateInput{PlanID: planID, ItemID: noteID, AuthorID: uuid.New(), Body: "x"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	copies, err := items.Duplicate(ctx, planID, noteID)
	if err != nil || len(copies) != 1 {
		t.Fatalf("Duplicate: %v (%d copies)", err, len(copies))
	}
	if n := countComments(t, db, copies[0].ID); n != 0 {
		t.Fatalf("the copy has %d comments, want none", n)
	}

	if _, err := db.Exec(`UPDATE checklist_items SET archived = true WHERE id = $1`, noteID); err != nil {
		t.Fatal(err)
	}
	if n := countComments(t, db, noteID); n != 1 {
		t.Fatalf("after archive: %d comments, want 1", n)
	}

	if err := items.Delete(ctx, planID, noteID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if n := countComments(t, db, noteID); n != 0 {
		t.Fatalf("after delete: %d comments, want 0", n)
	}
}
