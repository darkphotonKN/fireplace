package dayplan

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
)

// tickedBFS draws Graphs (BFS, Graphs) and Dynamic programming (DP), ticks
// the BFS entry and returns it as stored.
func tickedBFS(t *testing.T, f *fixture, tf treeFixture) Entry {
	t.Helper()
	p := drawn(t, f,
		genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30), existing(tf.graphs, "x", 20)}},
		genGroup{Label: "Dynamic programming", Entries: []genEntry{existing(tf.dp, "x", 30)}},
	)
	e, err := f.tick(entryFor(t, p, tf.bfs, "").ID, nil)
	if err != nil {
		t.Fatalf("TickEntry: %v", err)
	}
	return e
}

func entryCount(p DayPlan) int {
	n := 0
	for _, g := range p.Groups {
		n += len(g.Entries)
	}
	return n
}

func TestPlanMyDay_AITickedEntryKeepsItsPlaceGroupTextAndTickAndTheModelFillsTheRest(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	bfs := tickedBFS(t, f, tf)
	f.gen.answer(modelOutput(
		genGroup{Label: "Extra", Entries: []genEntry{newStep(tf.dp, "Write the memo table by hand", 20)}},
		genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "again", 30), existing(tf.graphs, "x", 20)}},
		genGroup{Label: "Dynamic programming", Entries: []genEntry{
			existing(tf.dp, "x", 30),
			newStep(tf.dp, "Solve the coin change problem", 30),
			newStep(tf.dp, "Solve the knapsack problem", 30),
			newStep(tf.dp, "Solve the edit distance problem", 30),
		}},
	))

	p, err := f.planMyDay()

	if err != nil {
		t.Fatalf("PlanMyDay: %v", err)
	}
	if f.gen.calls() != 2 {
		t.Fatalf("generator calls = %d: a tick changes the hash, so the redraw must reach the model", f.gen.calls())
	}
	if len(p.Groups) != 3 || entryCount(p) != MaxEntries {
		t.Fatalf("groups = %+v, want 3 groups and 6 entries", p.Groups)
	}
	first := p.Groups[0]
	if first.Label != "Graphs" || first.Entries[0].ID != bfs.ID || !first.Entries[0].Ticked ||
		first.Entries[0].Text != bfs.Text || first.Entries[0].Reason != bfs.Reason || *first.Entries[0].Minutes != *bfs.Minutes {
		t.Fatalf("first group = %+v, want the ticked BFS entry first, as it was", first)
	}
	if len(first.Entries) != 2 || *first.Entries[1].ItemID != tf.graphs || first.Entries[1].Ticked {
		t.Fatalf("Graphs group = %+v, want the model's Graphs entry joined by exact label after the pinned one", first.Entries)
	}
	if p.Groups[1].Label != "Extra" || p.Groups[2].Label != "Dynamic programming" || len(p.Groups[2].Entries) != 3 {
		t.Fatalf("groups = %+v, want Extra then Dynamic programming cut at the sixth entry", p.Groups)
	}
	stored, _ := f.store.stored(f.key())
	if e := entryFor(t, stored, tf.bfs, ""); e.ID != bfs.ID || !e.Ticked {
		t.Fatalf("stored = %+v, want the pinned entry stored with its id and tick", e)
	}
}

func TestPlanMyDay_ModelCannotOpenAFourthGroupBesideATickedGroup(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	tickedBFS(t, f, tf)
	f.gen.answer(modelOutput(
		genGroup{Label: "A", Entries: []genEntry{existing(tf.dp, "x", 30)}},
		genGroup{Label: "B", Entries: []genEntry{newStep(tf.dp, "Solve the coin change problem", 30)}},
		genGroup{Label: "C", Entries: []genEntry{existing(tf.graphs, "x", 30)}},
	))

	p, err := f.planMyDay()

	if err != nil {
		t.Fatal(err)
	}
	if len(p.Groups) != MaxGroups || p.Groups[0].Label != "Graphs" || p.Groups[1].Label != "A" || p.Groups[2].Label != "B" {
		t.Fatalf("groups = %+v, want Graphs (ticked), A, B and C dropped", p.Groups)
	}
}

func TestPlanMyDay_PromptListsTickedEntriesAsFixedAndAsksForTheRemainingSlots(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	bfs := tickedBFS(t, f, tf)
	f.gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.graphs, "x", 30)}}))

	if _, err := f.planMyDay(); err != nil {
		t.Fatal(err)
	}

	user := f.gen.requests[1].User
	for _, want := range []string{
		"ALREADY TICKED TODAY (fixed and done",
		`"group":"Graphs"`, `"item_id":"` + tf.bfs.String() + `"`, `"text":"` + bfs.Text + `"`,
		"at most 5 more entries",
	} {
		if !strings.Contains(user, want) {
			t.Fatalf("prompt lacks %q:\n%s", want, user)
		}
	}
	if strings.Contains(user, "ALREADY TICKED TODAY: none") {
		t.Fatal("prompt says nothing is ticked")
	}
	if f.gen.requests[1].System != systemRules {
		t.Fatal("ticks leaked into the static rules")
	}
}

func TestPlanMyDay_RulesRedrawKeepsTheTickedEntry(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	bfs := tickedBFS(t, f, tf)
	f.gen.fail(errors.New("provider down"))

	p, err := f.planMyDay()

	if err != nil {
		t.Fatal(err)
	}
	if p.Source != SourceRules {
		t.Fatalf("source = %s, want rules", p.Source)
	}
	first := p.Groups[0]
	if first.Label != "Graphs" || first.Entries[0].ID != bfs.ID || !first.Entries[0].Ticked || first.Entries[0].Reason != bfs.Reason {
		t.Fatalf("first group = %+v, want the ticked entry kept in place", first)
	}
	seen := 0
	for _, g := range p.Groups {
		for _, e := range g.Entries {
			if *e.ItemID == tf.bfs {
				seen++
			}
		}
	}
	if seen != 1 || len(p.Groups) > MaxGroups || entryCount(p) > MaxEntries {
		t.Fatalf("groups = %+v, want BFS once within the bounds", p.Groups)
	}
}

func TestPlanMyDay_NoCandidatesStillKeepsTheTickedEntries(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	bfs := tickedBFS(t, f, tf)
	items := tf.items()
	for i := range items {
		items[i].Done = true // done in the plan view; the day tick is independent (R28)
	}
	f.plans.setItems(items)

	p, err := f.planMyDay()

	if err != nil {
		t.Fatal(err)
	}
	if f.gen.calls() != 1 {
		t.Fatalf("generator calls = %d, want no call for a plan with no candidates", f.gen.calls())
	}
	if len(p.Groups) != 1 || len(p.Groups[0].Entries) != 1 || p.Groups[0].Entries[0].ID != bfs.ID || !p.Groups[0].Entries[0].Ticked {
		t.Fatalf("groups = %+v, want only the ticked entry", p.Groups)
	}
}

func TestPlanMyDay_SixTickedEntriesLeaveNothingToDraw(t *testing.T) {
	var items []guided.PlanItem
	var gen []genEntry
	for i := range MaxEntries {
		id := uuid.New()
		items = append(items, guided.PlanItem{ID: id, Description: "Topic " + string(rune('A'+i)), Type: "task", Scope: "longterm"})
		gen = append(gen, existing(id, "x", 30))
	}
	items = append(items, guided.PlanItem{ID: uuid.New(), Description: "One more", Type: "task", Scope: "longterm"})
	f := newFixture(items)
	p := drawn(t, f, genGroup{Label: "All", Entries: gen})
	for _, e := range p.Groups[0].Entries {
		if _, err := f.tick(e.ID, nil); err != nil {
			t.Fatal(err)
		}
	}

	again, err := f.planMyDay()

	if err != nil {
		t.Fatal(err)
	}
	if f.gen.calls() != 1 || f.counter.spentFor(f.key()) != 0 {
		t.Fatalf("calls %d spent %d: with every slot ticked there is nothing to ask the model", f.gen.calls(), f.counter.spentFor(f.key()))
	}
	if entryCount(again) != MaxEntries {
		t.Fatalf("groups = %+v, want the six ticked entries", again.Groups)
	}
	for _, e := range again.Groups[0].Entries {
		if !e.Ticked {
			t.Fatalf("entry %+v unticked", e)
		}
	}
}

func TestStateHash_CoversTheItemsTickedToday(t *testing.T) {
	tf := newTreeFixture()
	date := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	profile := PlanProfile{Type: "learning"}

	none := stateHash(tree(tf), profile, date, nil, nil)
	one := stateHash(tree(tf), profile, date, []uuid.UUID{tf.bfs}, nil)
	two := stateHash(tree(tf), profile, date, []uuid.UUID{tf.bfs, tf.dp}, nil)
	twoReordered := stateHash(tree(tf), profile, date, []uuid.UUID{tf.dp, tf.bfs}, nil)

	if none == one || one == two {
		t.Fatal("changing the ticked items must change the hash")
	}
	if two != twoReordered {
		t.Fatal("the hash must not depend on the order ticks are read in")
	}
}

func TestPlanMyDay_UntickAfterARedrawMakesTheNextRedrawReachTheModel(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	bfs := tickedBFS(t, f, tf)
	f.gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.graphs, "x", 30)}}))
	if _, err := f.planMyDay(); err != nil {
		t.Fatal(err)
	}
	if _, err := f.planMyDay(); err != nil || f.gen.calls() != 2 {
		t.Fatalf("calls %d err %v: an unchanged plan after the redraw is gated", f.gen.calls(), err)
	}
	if err := f.svc.UntickEntry(t.Context(), bfs.ID, f.userID); err != nil {
		t.Fatal(err)
	}

	if _, err := f.planMyDay(); err != nil {
		t.Fatal(err)
	}
	if f.gen.calls() != 3 {
		t.Fatalf("calls = %d, want the untick to change the hash", f.gen.calls())
	}
}

func TestStateHash_CoversTheSetOfDueRevisions(t *testing.T) {
	tf := newTreeFixture()
	profile := PlanProfile{Type: "learning"}
	date := oct(25)
	a := dueRevision{ItemID: tf.bfs, Rung: 1, DueOn: oct(25)}
	b := dueRevision{ItemID: tf.dfs, Rung: 2, DueOn: oct(24)}

	none := stateHash(tree(tf), profile, date, nil, nil)
	one := stateHash(tree(tf), profile, date, nil, []dueRevision{a})
	both := stateHash(tree(tf), profile, date, nil, []dueRevision{a, b})
	reordered := stateHash(tree(tf), profile, date, nil, []dueRevision{b, a})
	nextRung := stateHash(tree(tf), profile, date, nil, []dueRevision{{ItemID: tf.bfs, Rung: 2, DueOn: oct(25)}})

	if none == one || one == both || one == nextRung {
		t.Fatal("hash must change with the set of due revisions")
	}
	if both != reordered {
		t.Fatal("hash must not depend on the order of due revisions")
	}
}

// A tick that lands while a counted redraw is generating makes the redraw
// lose: it gives its spend back and returns the stored plan, which shows the
// tick, rather than storing a draw that read the plan before the tick and so
// drops the ticked entry.
func TestPlanMyDay_ATickDuringARedrawWinsAndTheRedrawGivesItsSpendBack(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	first := drawn(t, f, genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30), existing(tf.graphs, "x", 20)}})
	bfs := entryFor(t, first, tf.bfs, "")
	editPlan(f, tf, 1)
	// The redraw would drop BFS: the model never saw it ticked.
	f.gen.answer(modelOutput(genGroup{Label: "Dynamic programming", Entries: []genEntry{existing(tf.dp, "x", 30)}}))
	f.gen.during = func() {
		f.gen.during = nil
		if _, err := f.tick(bfs.ID, nil); err != nil {
			t.Errorf("tick during the redraw: %v", err)
		}
	}

	p, err := f.planMyDay()

	if err != nil {
		t.Fatalf("PlanMyDay: %v", err)
	}
	if p.DrawID != first.DrawID {
		t.Fatalf("redraw returned draw %s, want the stored %s: the tick landed first", p.DrawID, first.DrawID)
	}
	if e := entryFor(t, p, tf.bfs, ""); !e.Ticked {
		t.Fatalf("returned BFS entry = %+v, want ticked", e)
	}
	if spent := f.counter.spentFor(f.key()); spent != 0 || p.RefreshesRemaining != guided.RefreshCap {
		t.Fatalf("spent %d remaining %d, want the lost redraw's spend given back", spent, p.RefreshesRemaining)
	}
}
