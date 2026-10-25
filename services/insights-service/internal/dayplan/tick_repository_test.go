package dayplan

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Day ticks against the insights test database, as the rest of the
// repository is proven. Skipped as openTestDB skips, and when day_ticks has not
// been migrated there.

func openTickDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db := openTestDB(t)
	var exists bool
	if err := db.Get(&exists, `SELECT to_regclass('public.day_ticks') IS NOT NULL`); err != nil || !exists {
		t.Skip("day_ticks not migrated in the test database")
	}
	return db
}

func TestRepositoryIntegration_TickIsReadOntoTheEntryOnceAndUntickRemovesIt(t *testing.T) {
	db := openTickDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	key := testKey(t, db)
	p := testPlan(key, "Graphs")
	if _, _, err := repo.Save(ctx, p, DrawRef{}); err != nil {
		t.Fatal(err)
	}
	e := p.Groups[0].Entries[0]
	tick := Tick{Key: key, ItemID: *e.ItemID, EntryID: e.ID, DrawID: p.DrawID}

	created, err := repo.Tick(ctx, tick)
	if err != nil || !created {
		t.Fatalf("Tick = %v, %v; want created", created, err)
	}
	if again, err := repo.Tick(ctx, tick); err != nil || again {
		t.Fatalf("re-Tick = %v, %v; want nothing created", again, err)
	}
	got, _, err := repo.Find(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Groups[0].Entries[0].Ticked || got.Groups[0].Entries[1].Ticked {
		t.Fatalf("entries = %+v, want only the first ticked", got.Groups[0].Entries)
	}

	// Ticks are per user: another member's tick on the same item and date is
	// not this user's (FS-12ZMF).
	other := key
	other.UserID = uuid.New()
	if removed, err := repo.Untick(ctx, other, *e.ItemID); err != nil || removed {
		t.Fatalf("other user's Untick = %v, %v; want nothing removed", removed, err)
	}

	if removed, err := repo.Untick(ctx, key, *e.ItemID); err != nil || !removed {
		t.Fatalf("Untick = %v, %v; want removed", removed, err)
	}
	if removed, err := repo.Untick(ctx, key, *e.ItemID); err != nil || removed {
		t.Fatalf("second Untick = %v, %v; want nothing removed", removed, err)
	}
	if got, _, _ := repo.Find(ctx, key); got.Groups[0].Entries[0].Ticked {
		t.Fatal("entry still ticked after untick")
	}
}

func TestRepositoryIntegration_APinnedEntryKeepsItsIDInTheNextDrawAndFindByEntryFollowsIt(t *testing.T) {
	db := openTickDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	key := testKey(t, db)
	first := testPlan(key, "Graphs")
	if _, _, err := repo.Save(ctx, first, DrawRef{}); err != nil {
		t.Fatal(err)
	}
	pinned, dropped := first.Groups[0].Entries[0], first.Groups[0].Entries[1]

	next := testPlan(key, "Graphs")
	next.Groups[0].Entries[0] = pinned
	if _, won, err := repo.Save(ctx, next, first.Ref()); err != nil || !won {
		t.Fatalf("Save with a pinned entry = %v, %v; want stored", won, err)
	}

	got, ok, err := repo.FindByEntry(ctx, pinned.ID, key.UserID)
	if err != nil || !ok || got.DrawID != next.DrawID {
		t.Fatalf("FindByEntry = draw %s ok %v err %v, want the new draw", got.DrawID, ok, err)
	}
	if _, ok, _ := repo.FindByEntry(ctx, dropped.ID, key.UserID); ok {
		t.Fatal("an entry only in a replaced draw was found")
	}
	if _, ok, _ := repo.FindByEntry(ctx, pinned.ID, uuid.New()); ok {
		t.Fatal("another user's entry was found")
	}
}

func TestRepositoryIntegration_BindTurnsANewStepIntoAnExistingItemOnceOnTheCurrentDraw(t *testing.T) {
	db := openTickDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	key := testKey(t, db)
	p := testPlan(key, "Graphs")
	if _, _, err := repo.Save(ctx, p, DrawRef{}); err != nil {
		t.Fatal(err)
	}
	var step Entry
	for _, g := range p.Groups {
		for _, e := range g.Entries {
			if e.Kind == EntryNewStep {
				step = e
			}
		}
	}
	if step.ID == uuid.Nil {
		t.Fatal("testPlan has no new step")
	}
	created := uuid.New()

	if _, err := repo.Tick(ctx, Tick{Key: key, ItemID: created, EntryID: step.ID, DrawID: uuid.New(), Bind: true}); !errors.Is(err, ErrEntryNotTickable) {
		t.Fatalf("bind on a draw the key does not serve = %v, want ErrEntryNotTickable", err)
	}
	if ok, err := repo.Tick(ctx, Tick{Key: key, ItemID: created, EntryID: step.ID, DrawID: p.DrawID, Bind: true}); err != nil || !ok {
		t.Fatalf("bind = %v, %v; want ticked", ok, err)
	}
	got, _, err := repo.Find(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	bound, _ := got.entry(step.ID)
	if bound.Kind != EntryExistingItem || bound.ItemID == nil || *bound.ItemID != created || !bound.Ticked ||
		bound.Text != step.Text || bound.Reason != step.Reason || *bound.ParentItemID != *step.ParentItemID {
		t.Fatalf("bound = %+v, want an existing item for the created item keeping text, reason and parent", bound)
	}
	if _, err := repo.Tick(ctx, Tick{Key: key, ItemID: uuid.New(), EntryID: step.ID, DrawID: p.DrawID, Bind: true}); !errors.Is(err, ErrEntryNotTickable) {
		t.Fatalf("second bind = %v, want ErrEntryNotTickable", err)
	}
}

// A tick or untick that commits while a redraw is in flight (the redraw read
// the stored plan before it) makes the redraw's save lose: it reads back the
// stored plan, which shows the tick, instead of replacing it with a draw that
// never saw the tick.
func TestRepositoryIntegration_ATickOrUntickMakesAnInFlightRedrawLose(t *testing.T) {
	db := openTickDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	key := testKey(t, db)
	p := testPlan(key, "Graphs")
	if _, _, err := repo.Save(ctx, p, DrawRef{}); err != nil {
		t.Fatal(err)
	}
	e := p.Groups[0].Entries[0]

	// The redraw reads the stored plan, then the tick lands.
	started, _, err := repo.Find(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Tick(ctx, Tick{Key: key, ItemID: *e.ItemID, EntryID: e.ID, DrawID: p.DrawID}); err != nil {
		t.Fatal(err)
	}
	stored, won, err := repo.Save(ctx, testPlan(key, "Trees"), started.Ref())
	if err != nil || won {
		t.Fatalf("redraw over a ticked plan = won %v err %v, want lost", won, err)
	}
	if stored.DrawID != p.DrawID || !stored.Groups[0].Entries[0].Ticked {
		t.Fatalf("lost redraw read draw %s entries %+v, want the stored draw with its tick", stored.DrawID, stored.Groups[0].Entries)
	}

	// The same for an untick.
	started, _, err = repo.Find(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if removed, err := repo.Untick(ctx, key, *e.ItemID); err != nil || !removed {
		t.Fatalf("Untick = %v, %v", removed, err)
	}
	if _, won, err := repo.Save(ctx, testPlan(key, "Trees"), started.Ref()); err != nil || won {
		t.Fatalf("redraw over an unticked plan = won %v err %v, want lost", won, err)
	}

	// A redraw that starts after them wins.
	started, _, err = repo.Find(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	next := testPlan(key, "Trees")
	if stored, won, err := repo.Save(ctx, next, started.Ref()); err != nil || !won || stored.DrawID != next.DrawID {
		t.Fatalf("redraw from the current state = won %v err %v, want stored", won, err)
	}
	// And the next one starts from what that save returned.
	if _, won, err := repo.Save(ctx, testPlan(key, "Heaps"), next.Ref()); err != nil || won {
		t.Fatalf("redraw from a ref the save did not return = won %v err %v", won, err)
	}
}

// twoSteps is a stored plan with two new steps under one parent.
func twoSteps(t *testing.T, repo *Repository, key guided.DayKey) (DayPlan, Entry, Entry) {
	t.Helper()
	parent := uuid.New()
	p := DayPlan{
		Key: key, Planned: true, Source: SourceAI, GeneratedAt: time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC),
		PlanStateHash: "h", DrawID: uuid.New(), FreeDrawUsed: true,
		Groups: []Group{{Label: "Graphs", Entries: []Entry{
			{ID: uuid.New(), Kind: EntryNewStep, ParentItemID: &parent, Text: "Solve the climbing stairs problem", Reason: "start it"},
			{ID: uuid.New(), Kind: EntryNewStep, ParentItemID: &parent, Text: "Solve the house robber problem", Reason: "start it"},
		}}},
	}
	if _, _, err := repo.Save(context.Background(), p, DrawRef{}); err != nil {
		t.Fatal(err)
	}
	return p, p.Groups[0].Entries[0], p.Groups[0].Entries[1]
}

// One item binds to at most one entry of a draw: a bind of a second entry to
// an item another entry already holds is not tickable, whatever the service's
// earlier read said.
func TestRepositoryIntegration_BindRefusesAnItemAnotherEntryOfTheDrawHolds(t *testing.T) {
	db := openTickDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	key := testKey(t, db)
	p, a, b := twoSteps(t, repo, key)
	item := uuid.New()

	if _, err := repo.Tick(ctx, Tick{Key: key, ItemID: item, EntryID: a.ID, DrawID: p.DrawID, Bind: true}); err != nil {
		t.Fatalf("first bind: %v", err)
	}
	if _, err := repo.Tick(ctx, Tick{Key: key, ItemID: item, EntryID: b.ID, DrawID: p.DrawID, Bind: true}); !errors.Is(err, ErrEntryNotTickable) {
		t.Fatalf("second entry's bind to the same item = %v, want ErrEntryNotTickable", err)
	}
	got, _, err := repo.Find(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if e, _ := got.entry(b.ID); e.Kind != EntryNewStep || e.ItemID != nil {
		t.Fatalf("second entry = %+v, want still an unbound new step", e)
	}
}

// Two concurrent binds of different entries to one item: exactly one binds,
// the other is not tickable.
func TestRepositoryIntegration_ConcurrentBindsOfOneItemBindOnce(t *testing.T) {
	db := openTickDB(t)
	repo := NewRepository(db)
	key := testKey(t, db)
	p, a, b := twoSteps(t, repo, key)
	item := uuid.New()

	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i, e := range []Entry{a, b} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = repo.Tick(context.Background(), Tick{Key: key, ItemID: item, EntryID: e.ID, DrawID: p.DrawID, Bind: true})
		}()
	}
	wg.Wait()

	ok, refused := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrEntryNotTickable):
			refused++
		default:
			t.Fatalf("bind: %v", err)
		}
	}
	if ok != 1 || refused != 1 {
		t.Fatalf("binds ok %d refused %d (%v), want exactly one of each", ok, refused, errs)
	}
	var bound int
	if err := db.Get(&bound, `SELECT count(*) FROM day_plan_entries WHERE draw_id = $1 AND item_id = $2`, p.DrawID, item); err != nil || bound != 1 {
		t.Fatalf("entries bound to the item = %d (%v), want 1", bound, err)
	}
}

// FindByEntry joins an entry to its plan by draw id on every tick: the draw id
// is unique across day_plans and indexed.
func TestRepositoryIntegration_DrawIDIsUniqueAcrossDayPlans(t *testing.T) {
	db := openTickDB(t)
	repo := NewRepository(db)
	key := testKey(t, db)
	p := testPlan(key, "Graphs")
	if _, _, err := repo.Save(context.Background(), p, DrawRef{}); err != nil {
		t.Fatal(err)
	}
	other := key
	other.Date = key.Date.AddDate(0, 0, 1)
	dup := testPlan(other, "Graphs")
	dup.DrawID = p.DrawID
	if _, _, err := repo.Save(context.Background(), dup, DrawRef{}); err == nil {
		t.Fatal("a second plan with the same draw id was stored")
	}
}
