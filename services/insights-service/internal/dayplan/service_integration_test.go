package dayplan

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/ai"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
)

// Plan my day's redraws are capped by the Postgres refresh counter: the key's
// first AI draw is free, the next three each spend one, and a fourth is
// refused with the stored plan untouched (R16, R17, R20). The counter is the
// one per (plan, user, date) the brief shared while it existed (A3).
func TestServiceIntegration_PlanMyDayIsCappedByThePostgresRefreshCounter(t *testing.T) {
	db := openTestDB(t)
	key := testKey(t, db)
	tf := newTreeFixture()
	plans := &fakePlans{profile: PlanProfile{Type: "learning", Focus: "Graphs"}, items: tf.items()}
	gen := &fakeGenerator{}
	counter := guided.NewRefreshCounters(db)
	svc := NewService(plans, gen, guided.NewSnippetSource(guided.NoRetrieval{}, noMemory{}), NewRepository(db), NewRepository(db), counter)
	svc.now = func() time.Time { return key.Date.Add(12 * time.Hour) }
	ctx := context.Background()
	date := key.Date.Format(guided.DateLayout)
	gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}))
	edit := func(n int) {
		items := tf.items()
		items[3].Description = fmt.Sprintf("Dynamic programming %d", n)
		plans.setItems(items)
	}

	p, err := svc.PlanMyDay(ctx, key.PlanID, key.UserID, date)
	if err != nil || p.RefreshesRemaining != guided.RefreshCap {
		t.Fatalf("free draw: remaining %d err %v, want %d", p.RefreshesRemaining, err, guided.RefreshCap)
	}

	for i := range guided.RefreshCap {
		edit(i)
		p, err = svc.PlanMyDay(ctx, key.PlanID, key.UserID, date)
		if want := guided.RefreshCap - 1 - i; err != nil || p.RefreshesRemaining != want {
			t.Fatalf("counted draw %d: remaining %d err %v, want %d", i+1, p.RefreshesRemaining, err, want)
		}
	}
	capped := p

	edit(10)
	if _, err := svc.PlanMyDay(ctx, key.PlanID, key.UserID, date); !errors.Is(err, guided.ErrRefreshLimitReached) {
		t.Fatalf("Plan my day past the cap: err %v, want ErrRefreshLimitReached", err)
	}
	if left, err := counter.Remaining(ctx, key); err != nil || left != 0 {
		t.Fatalf("counter remaining %d err %v, want 0", left, err)
	}
	stored, err := svc.GetDayPlan(ctx, key.PlanID, key.UserID, date)
	if err != nil || !stored.GeneratedAt.Equal(capped.GeneratedAt) || stored.RefreshesRemaining != 0 {
		t.Fatalf("stored plan after the refusal = %+v err %v, want the last counted draw (%s)", stored, err, capped.GeneratedAt)
	}
}

// Over budget, Plan my day stores a rules plan in Postgres, spending nothing
// from the refresh counter; once the budget is back the model is tried and,
// as the key's first successful AI draw, it is free (R16, R17, R36).
func TestServiceIntegration_RulesPlanIsStoredFreeAndTheModelIsTriedAfter(t *testing.T) {
	db := openTestDB(t)
	key := testKey(t, db)
	tf := newTreeFixture()
	plans := &fakePlans{profile: PlanProfile{Type: "learning", Focus: "Graphs"}, items: tf.items()}
	gen := &fakeGenerator{}
	svc := NewService(plans, gen, guided.NewSnippetSource(guided.NoRetrieval{}, noMemory{}), NewRepository(db), NewRepository(db), guided.NewRefreshCounters(db))
	ctx := context.Background()
	date := key.Date.Format(guided.DateLayout)
	svc.now = func() time.Time { return key.Date.Add(12 * time.Hour) }

	gen.fail(ai.ErrBudgetExhausted)
	rules, err := svc.PlanMyDay(ctx, key.PlanID, key.UserID, date)
	if err != nil || rules.Source != SourceRules || rules.RefreshesRemaining != 3 || rules.FreeDrawUsed {
		t.Fatalf("over budget: %+v err %v, want a free rules plan", rules, err)
	}
	read, err := svc.GetDayPlan(ctx, key.PlanID, key.UserID, date)
	if err != nil || read.Source != SourceRules || read.DrawID != rules.DrawID || render(read.Groups) != render(rules.Groups) {
		t.Fatalf("read back: %+v err %v, want the stored rules plan", read, err)
	}
	for _, g := range read.Groups {
		for _, e := range g.Entries {
			if e.Minutes != nil || e.Kind != EntryExistingItem {
				t.Fatalf("stored rules entry %+v, want an existing item with no minutes", e)
			}
		}
	}

	gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}))
	aiPlan, err := svc.PlanMyDay(ctx, key.PlanID, key.UserID, date)
	if err != nil || aiPlan.Source != SourceAI || !aiPlan.FreeDrawUsed || aiPlan.RefreshesRemaining != 3 {
		t.Fatalf("budget back: %+v err %v, want a free ai plan", aiPlan, err)
	}
}

// In Postgres: a tick survives an AI redraw and then a rules redraw with its
// entry id, group and tick, and untick clears it (R12, R29, R39).
func TestServiceIntegration_TickedEntrySurvivesAIAndRulesRedraws(t *testing.T) {
	db := openTestDB(t)
	var ticks bool
	if err := db.Get(&ticks, `SELECT to_regclass('public.day_ticks') IS NOT NULL`); err != nil || !ticks {
		t.Skip("day_ticks not migrated in the test database")
	}
	key := testKey(t, db)
	tf := newTreeFixture()
	plans := &fakePlans{profile: PlanProfile{Type: "learning", Focus: "Graphs"}, items: tf.items()}
	gen := &fakeGenerator{}
	svc := NewService(plans, gen, guided.NewSnippetSource(guided.NoRetrieval{}, noMemory{}), NewRepository(db), NewRepository(db), guided.NewRefreshCounters(db))
	ctx := context.Background()
	date := key.Date.Format(guided.DateLayout)
	svc.now = func() time.Time { return key.Date.Add(12 * time.Hour) }

	gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30), existing(tf.graphs, "x", 20)}}))
	first, err := svc.PlanMyDay(ctx, key.PlanID, key.UserID, date)
	if err != nil {
		t.Fatal(err)
	}
	bfs := first.Groups[0].Entries[0]
	if _, err := svc.TickEntry(ctx, bfs.ID, key.UserID, nil); err != nil {
		t.Fatalf("TickEntry: %v", err)
	}

	gen.answer(modelOutput(genGroup{Label: "Dynamic programming", Entries: []genEntry{existing(tf.dp, "x", 30)}}))
	aiPlan, err := svc.PlanMyDay(ctx, key.PlanID, key.UserID, date)
	if err != nil || aiPlan.Source != SourceAI || aiPlan.DrawID == first.DrawID {
		t.Fatalf("AI redraw: %+v err %v", aiPlan, err)
	}
	gen.fail(ai.ErrBudgetExhausted)
	plans.setItems(append(tf.items(), guided.PlanItem{ID: uuid.New(), Description: "New topic", Type: "task", Scope: "longterm"}))
	rules, err := svc.PlanMyDay(ctx, key.PlanID, key.UserID, date)
	if err != nil || rules.Source != SourceRules {
		t.Fatalf("rules redraw: %+v err %v", rules, err)
	}
	for name, p := range map[string]DayPlan{"ai": aiPlan, "rules": rules} {
		read := p
		if name == "rules" {
			if read, err = svc.GetDayPlan(ctx, key.PlanID, key.UserID, date); err != nil {
				t.Fatal(err)
			}
		}
		e := read.Groups[0].Entries[0]
		if read.Groups[0].Label != "Graphs" || e.ID != bfs.ID || !e.Ticked || *e.ItemID != tf.bfs {
			t.Fatalf("%s redraw first group = %+v, want the ticked BFS entry kept", name, read.Groups[0])
		}
	}

	if err := svc.UntickEntry(ctx, bfs.ID, key.UserID); err != nil {
		t.Fatalf("UntickEntry: %v", err)
	}
	read, err := svc.GetDayPlan(ctx, key.PlanID, key.UserID, date)
	if err != nil || read.Groups[0].Entries[0].Ticked {
		t.Fatalf("after untick: %+v err %v, want unticked", read.Groups[0], err)
	}
}

// In Postgres: a learning plan's first tick schedules revision 1 for the next
// day, the next day's plan stamps it, ticking it climbs the ladder, and the
// untick takes the climb back (R42, R43, R46, R10).
func TestServiceIntegration_TheRevisionLadderThroughTicks(t *testing.T) {
	db := openRevisionDB(t)
	key := testKey(t, db)
	tf := newTreeFixture()
	plans := &fakePlans{profile: PlanProfile{Type: "learning", Focus: "Graphs"}, items: tf.items()}
	gen := &fakeGenerator{}
	repo := NewRepository(db)
	svc := NewService(plans, gen, guided.NewSnippetSource(guided.NoRetrieval{}, noMemory{}), repo, repo, guided.NewRefreshCounters(db))
	ctx := context.Background()
	svc.now = func() time.Time { return key.Date.Add(12 * time.Hour) }
	today, tomorrow := key.Date.Format(guided.DateLayout), key.Date.AddDate(0, 0, 1).Format(guided.DateLayout)
	answer := modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}})

	gen.answer(answer)
	p, err := svc.PlanMyDay(ctx, key.PlanID, key.UserID, today)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.TickEntry(ctx, entryFor(t, p, tf.bfs, "").ID, key.UserID, nil); err != nil {
		t.Fatal(err)
	}

	p, err = svc.PlanMyDay(ctx, key.PlanID, key.UserID, tomorrow)
	if err != nil {
		t.Fatal(err)
	}
	rev := entryFor(t, p, tf.bfs, "")
	if rev.Revision == nil || *rev.Revision != 1 || rev.RevisionsEnabled == nil || !*rev.RevisionsEnabled {
		t.Fatalf("tomorrow's entry = %+v, want revision 1 with revisions enabled", rev)
	}
	if _, err := svc.TickEntry(ctx, rev.ID, key.UserID, nil); err != nil {
		t.Fatal(err)
	}
	next := key
	next.Date = key.Date.AddDate(0, 0, 4)
	if due, err := repo.DueRevisions(ctx, next); err != nil || len(due) != 1 || due[0].Rung != 2 {
		t.Fatalf("due 4 days on = %+v, %v; want revision 2 (tomorrow + 3)", due, err)
	}

	if err := svc.UntickEntry(ctx, rev.ID, key.UserID); err != nil {
		t.Fatal(err)
	}
	if due, err := repo.DueRevisions(ctx, next); err != nil || len(due) != 1 || due[0].Rung != 1 {
		t.Fatalf("due after the untick = %+v, %v; want revision 1 reopened", due, err)
	}
}
