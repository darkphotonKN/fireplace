package checklistitem

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Ordering is a property of the database, not of the Go code: an ORDER BY can
// only be proven by running it. These tests therefore need the local Postgres
// (docker compose up postgres — see .env.example for the connection this
// defaults to) and are skipped under -short, matching the Makefile's split
// between `test-unit` and `test-integration`.

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
		t.Skipf("plan database unreachable (%v).\n"+
			"Start it with `docker compose up -d postgres` from the repo root, or point\n"+
			"DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME at your own.", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// seedPlan makes a plan to hang items off and removes it afterwards. The
// checklist_items FK is ON DELETE CASCADE, so dropping the plan takes every
// item with it and the dev database is left as it was found.
func seedPlan(t *testing.T, db *sqlx.DB) uuid.UUID {
	t.Helper()
	planID := uuid.New()
	// `focus` is still NOT NULL in the schema even though CONTEXT.md retires
	// the term in favour of Plan Draft, so the fixture has to supply it.
	_, err := db.Exec(
		`INSERT INTO plans (id, user_id, name, plan_type, focus) VALUES ($1, $2, $3, $4, $5)`,
		planID, uuid.New(), "FS-0009 ordering fixture", "project", "ordering fixture",
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

func seedItem(t *testing.T, db *sqlx.DB, planID, id uuid.UUID, seq int, createdAt time.Time) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO checklist_items
			(id, plan_id, description, done, sequence, scope, type, created_at, updated_at)
		VALUES ($1, $2, $3, false, $4, 'longterm', 'task', $5, $5)`,
		id, planID, "item "+id.String()[:8], seq, createdAt,
	)
	if err != nil {
		t.Fatalf("seed item: %v", err)
	}
}

func seedArchivedItem(t *testing.T, db *sqlx.DB, planID, id uuid.UUID, seq int, createdAt time.Time) {
	t.Helper()
	seedItem(t, db, planID, id, seq, createdAt)
	if _, err := db.Exec(`UPDATE checklist_items SET archived = true WHERE id = $1`, id); err != nil {
		t.Fatalf("archive item: %v", err)
	}
}

func seedDatedItem(t *testing.T, db *sqlx.DB, planID, id uuid.UUID, seq int, createdAt, start time.Time) {
	t.Helper()
	seedItem(t, db, planID, id, seq, createdAt)
	_, err := db.Exec(
		`UPDATE checklist_items SET start_date = $2, due_date = $2 WHERE id = $1`, id, start)
	if err != nil {
		t.Fatalf("date item: %v", err)
	}
}

func listIDs(t *testing.T, db *sqlx.DB, planID uuid.UUID) []uuid.UUID {
	t.Helper()
	items, err := NewRepository(db).ListByPlanID(context.Background(), ListItemsInput{PlanID: planID})
	if err != nil {
		t.Fatalf("list by plan: %v", err)
	}
	ids := make([]uuid.UUID, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	return ids
}

func TestIntegrationListByPlanID_TiedSequences_OrdersByCreatedAt(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)

	base := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	oldest, middle, newest := uuid.New(), uuid.New(), uuid.New()
	// Inserted newest-first on purpose. With no tiebreak the rows come back in
	// whatever order the scan finds them, which is this one — so a test that
	// seeded in the expected order would pass without the fix and prove
	// nothing.
	seedItem(t, db, planID, newest, 7, base.Add(2*time.Hour))
	seedItem(t, db, planID, oldest, 7, base)
	seedItem(t, db, planID, middle, 7, base.Add(time.Hour))

	got := listIDs(t, db, planID)

	want := []uuid.UUID{oldest, middle, newest}
	if len(got) != len(want) {
		t.Fatalf("got %d items, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("position %d: got %s, want %s\nfull order: %v", i, got[i], want[i], got)
		}
	}
}

func TestIntegrationListByPlanID_TiedSequenceAndTime_OrdersByID(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)

	// created_at breaks nearly every tie, but two rows written in the same
	// transaction can share a timestamp to the microsecond. id is what makes
	// the order total.
	at := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	first, second := uuid.New(), uuid.New()
	if first.String() > second.String() {
		first, second = second, first
	}
	// Seeded high id first, so insertion order is the wrong answer.
	seedItem(t, db, planID, second, 7, at)
	seedItem(t, db, planID, first, 7, at)

	got := listIDs(t, db, planID)

	want := []uuid.UUID{first, second}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestIntegrationListArchivedByPlanID_TiedSequences_OrdersByCreatedAt(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)

	base := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	oldest, newest := uuid.New(), uuid.New()
	seedArchivedItem(t, db, planID, newest, 7, base.Add(time.Hour))
	seedArchivedItem(t, db, planID, oldest, 7, base)

	items, err := NewRepository(db).ListArchivedByPlanID(context.Background(), planID, nil)
	if err != nil {
		t.Fatalf("list archived: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("got %d archived items, want 2", len(items))
	}
	if items[0].ID != oldest || items[1].ID != newest {
		t.Fatalf("got [%s %s], want [%s %s]", items[0].ID, items[1].ID, oldest, newest)
	}
}

func TestIntegrationListInDateWindow_TiedDatesAndSequences_OrdersByCreatedAt(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)

	// The date keys come first here and stay first — the tiebreak only decides
	// between items the dates cannot separate.
	//
	// Five items rather than two, seeded in a scrambled order: this query is
	// not a plain table scan, so with two rows a wrong ORDER BY has an even
	// chance of looking right. Five makes an accidental pass a 1-in-120 event.
	day := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
	base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	want := make([]uuid.UUID, 5)
	for i := range want {
		want[i] = uuid.New()
	}
	for _, i := range []int{3, 0, 4, 1, 2} {
		seedDatedItem(t, db, planID, want[i], 7, base.Add(time.Duration(i)*time.Hour), day)
	}

	items, err := NewRepository(db).ListInDateWindow(
		context.Background(), planID, day.AddDate(0, 0, -1), day.AddDate(0, 0, 1),
	)
	if err != nil {
		t.Fatalf("list in date window: %v", err)
	}

	if len(items) != len(want) {
		t.Fatalf("got %d items in the window, want %d", len(items), len(want))
	}
	for i := range want {
		if items[i].ID != want[i] {
			got := make([]uuid.UUID, len(items))
			for j, it := range items {
				got[j] = it.ID
			}
			t.Fatalf("position %d: got %s, want %s\nfull order: %v\nwanted:     %v",
				i, items[i].ID, want[i], got, want)
		}
	}
}

func TestIntegrationListByPlanID_DistinctSequences_UnchangedBySequence(t *testing.T) {
	// A regression guard, green from the start: the tiebreak is only allowed to
	// decide between rows `sequence` cannot separate. If it ever leaked ahead of
	// sequence, this is what would catch it.
	db := openTestDB(t)
	planID := seedPlan(t, db)

	base := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	first, second, third := uuid.New(), uuid.New(), uuid.New()
	// Sequence ascending, created_at descending: the two keys disagree, and
	// sequence has to win.
	seedItem(t, db, planID, first, 1, base.Add(2*time.Hour))
	seedItem(t, db, planID, second, 2, base.Add(time.Hour))
	seedItem(t, db, planID, third, 3, base)

	got := listIDs(t, db, planID)

	want := []uuid.UUID{first, second, third}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("position %d: got %s, want %s\nfull order: %v", i, got[i], want[i], got)
		}
	}
}
