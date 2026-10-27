package dayplan

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// The repository is proven against the insights test database (ADR-0010 §5),
// as guided's is: TEST_DB_* (not DB_*, which the Makefile exports from .env
// for the dev database), the app role by default. Skipped under -short, when
// the database is unreachable, or when day_plans has not been migrated there.

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func openTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("needs the insights test database; run without -short")
	}
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		env("TEST_DB_HOST", "localhost"), env("TEST_DB_PORT", "6500"),
		env("TEST_DB_USER", "fireplace_insights_app"), env("TEST_DB_PASSWORD", "app"),
		env("TEST_DB_NAME", "fireplace_insights_test"),
	)
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("insights test database unreachable (%v)", err)
	}
	t.Cleanup(func() { db.Close() })

	var exists bool
	if err := db.Get(&exists, `SELECT to_regclass('public.day_plans') IS NOT NULL`); err != nil || !exists {
		t.Skip("day_plans not migrated in the test database")
	}
	return db
}

func testKey(t *testing.T, db *sqlx.DB) guided.DayKey {
	t.Helper()
	key := guided.DayKey{PlanID: uuid.New(), UserID: uuid.New(), Date: time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)}
	t.Cleanup(func() {
		queries := []string{
			`DELETE FROM day_plans WHERE plan_id = $1`, // cascades to its entries
			`DELETE FROM day_ticks WHERE plan_id = $1`,
			`DELETE FROM refresh_counters WHERE plan_id = $1`,
			`DELETE FROM generations WHERE plan_id = $1`,
		}
		// A learning plan's ticks write the revision ladder once it is migrated.
		var ladder bool
		if err := db.Get(&ladder, `SELECT to_regclass('public.revisions') IS NOT NULL`); err == nil && ladder {
			queries = append(queries, `DELETE FROM revisions WHERE plan_id = $1`, `DELETE FROM revision_switches WHERE plan_id = $1`)
		}
		for _, q := range queries {
			if _, err := db.Exec(q, key.PlanID); err != nil {
				t.Errorf("cleanup %s: %v", q, err)
			}
		}
	})
	return key
}

func testPlan(key guided.DayKey, label string) DayPlan {
	item, parent, newParent := uuid.New(), uuid.New(), uuid.New()
	return DayPlan{
		Key: key, Planned: true, Source: SourceAI,
		GeneratedAt:   time.Date(2026, 10, 25, 9, 30, 0, 0, time.UTC),
		PlanStateHash: "hash-" + label, DrawID: uuid.New(), FreeDrawUsed: true,
		Groups: []Group{
			{Label: label, Entries: []Entry{
				{ID: uuid.New(), Kind: EntryExistingItem, ItemID: &item, ParentItemID: &parent, Text: "Solve 3 BFS problems", Reason: "due tomorrow", Minutes: ptr(45)},
				{ID: uuid.New(), Kind: EntryExistingItem, ItemID: &parent, Text: "Graphs", Reason: "overdue"},
			}},
			{Label: "Dynamic programming", Entries: []Entry{
				{ID: uuid.New(), Kind: EntryNewStep, ParentItemID: &newParent, Text: "Do the knapsack problem", Reason: "start it", Minutes: ptr(30), Revision: ptr(2)},
			}},
		},
	}
}

func assertSamePlan(t *testing.T, got, want DayPlan) {
	t.Helper()
	if got.Key != want.Key || !got.Planned || got.Source != want.Source || !got.GeneratedAt.Equal(want.GeneratedAt) ||
		got.PlanStateHash != want.PlanStateHash || got.DrawID != want.DrawID || got.FreeDrawUsed != want.FreeDrawUsed {
		t.Fatalf("plan = %+v\nwant %+v", got, want)
	}
	if len(got.Groups) != len(want.Groups) {
		t.Fatalf("groups = %+v, want %+v", got.Groups, want.Groups)
	}
	for gi := range want.Groups {
		g, w := got.Groups[gi], want.Groups[gi]
		if g.Label != w.Label || len(g.Entries) != len(w.Entries) {
			t.Fatalf("group %d = %+v, want %+v", gi, g, w)
		}
		for ei := range w.Entries {
			if e, x := g.Entries[ei], w.Entries[ei]; !sameEntry(e, x) {
				t.Fatalf("entry %d.%d = %+v, want %+v", gi, ei, e, x)
			}
		}
	}
}

func sameEntry(a, b Entry) bool {
	return a.ID == b.ID && a.Kind == b.Kind && eq(a.ItemID, b.ItemID) && eq(a.ParentItemID, b.ParentItemID) &&
		a.Text == b.Text && a.Reason == b.Reason && eq(a.Minutes, b.Minutes) && eq(a.Revision, b.Revision)
}

func eq[T comparable](a, b *T) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func TestRepositoryIntegration_SaveIntoAnEmptyKeyThenFindRoundTrips(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	key := testKey(t, db)
	p := testPlan(key, "Graphs")

	stored, won, err := repo.Save(context.Background(), p, DrawRef{})
	if err != nil || !won {
		t.Fatalf("Save: won %v err %v", won, err)
	}
	assertSamePlan(t, stored, p)

	found, ok, err := repo.Find(context.Background(), key)
	if err != nil || !ok {
		t.Fatalf("Find: ok %v err %v", ok, err)
	}
	assertSamePlan(t, found, p)

	if _, ok, err := repo.Find(context.Background(), guided.DayKey{PlanID: key.PlanID, UserID: uuid.New(), Date: key.Date}); err != nil || ok {
		t.Fatalf("another user's key: ok %v err %v, want nothing stored", ok, err)
	}
}

func TestRepositoryIntegration_EmptyPlannedPlanRoundTrips(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	key := testKey(t, db)
	p := DayPlan{Key: key, Planned: true, Source: SourceAI, GeneratedAt: time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC), PlanStateHash: "h", DrawID: uuid.New()}

	if _, won, err := repo.Save(context.Background(), p, DrawRef{}); err != nil || !won {
		t.Fatalf("Save: won %v err %v", won, err)
	}
	found, ok, err := repo.Find(context.Background(), key)
	if err != nil || !ok || !found.Planned || len(found.Groups) != 0 || found.FreeDrawUsed {
		t.Fatalf("found %+v ok %v err %v, want a planned plan with no groups and the free draw unused", found, ok, err)
	}
}

func TestRepositoryIntegration_SaveOverTheDrawItStartedFromReplacesTheWholePlan(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	key := testKey(t, db)
	first := testPlan(key, "Graphs")
	if _, _, err := repo.Save(context.Background(), first, DrawRef{}); err != nil {
		t.Fatal(err)
	}
	second := testPlan(key, "Trees")
	second.Source, second.FreeDrawUsed = SourceRules, false

	stored, won, err := repo.Save(context.Background(), second, first.Ref())

	if err != nil || !won {
		t.Fatalf("Save: won %v err %v", won, err)
	}
	assertSamePlan(t, stored, second)
	found, _, _ := repo.Find(context.Background(), key)
	assertSamePlan(t, found, second)

	// The replaced draw's entries stay stored under it: entry ids remain valid
	// suggestion ids, and a reader holding the old draw still reads it whole.
	var kept int
	if err := db.Get(&kept, `SELECT count(*) FROM day_plan_entries WHERE draw_id = $1`, first.DrawID); err != nil || kept != 3 {
		t.Fatalf("entries kept for the replaced draw = %d (err %v), want 3", kept, err)
	}
}

func TestRepositoryIntegration_SaveOverAStaleDrawLosesAndReturnsTheStoredPlan(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	key := testKey(t, db)
	first := testPlan(key, "Graphs")
	if _, _, err := repo.Save(context.Background(), first, DrawRef{}); err != nil {
		t.Fatal(err)
	}

	for name, prev := range map[string]DrawRef{
		"nothing stored assumed": {},
		"another draw assumed":   {DrawID: uuid.New()},
		"another tick version":   {DrawID: first.DrawID, TickVersion: 7},
	} {
		t.Run(name, func(t *testing.T) {
			stored, won, err := repo.Save(context.Background(), testPlan(key, "Trees"), prev)
			if err != nil || won {
				t.Fatalf("Save: won %v err %v, want a lost save", won, err)
			}
			assertSamePlan(t, stored, first)
		})
	}
}

func TestRepositoryIntegration_ConcurrentSavesFromOneDrawStoreExactlyOne(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	key := testKey(t, db)
	base := testPlan(key, "Graphs")
	if _, _, err := repo.Save(context.Background(), base, DrawRef{}); err != nil {
		t.Fatal(err)
	}

	const n = 4
	results := make([]DayPlan, n)
	wins := make([]bool, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], wins[i], errs[i] = repo.Save(context.Background(), testPlan(key, fmt.Sprintf("Draw %d", i)), base.Ref())
		}()
	}
	wg.Wait()

	winners := 0
	for i := range n {
		if errs[i] != nil {
			t.Fatalf("save %d: %v", i, errs[i])
		}
		if wins[i] {
			winners++
		}
	}
	found, _, _ := repo.Find(context.Background(), key)
	if winners != 1 {
		t.Fatalf("winners = %d, want exactly 1", winners)
	}
	for i := range n {
		if results[i].DrawID != found.DrawID {
			t.Fatalf("save %d returned draw %s, want the stored %s", i, results[i].DrawID, found.DrawID)
		}
	}
}

func TestRepositoryIntegration_ConcurrentFirstSavesStoreExactlyOne(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	key := testKey(t, db)

	const n = 4
	results := make([]DayPlan, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], _, errs[i] = repo.Save(context.Background(), testPlan(key, fmt.Sprintf("Draw %d", i)), DrawRef{})
		}()
	}
	wg.Wait()

	found, _, _ := repo.Find(context.Background(), key)
	for i := range n {
		if errs[i] != nil || results[i].DrawID != found.DrawID {
			t.Fatalf("save %d: draw %s err %v, want the stored %s", i, results[i].DrawID, errs[i], found.DrawID)
		}
	}
}
