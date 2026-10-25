package dayplan

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
)

// planDate is the plan date of every ranker test.
var planDate = time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)

func oct(day int) time.Time { return time.Date(2026, 10, day, 0, 0, 0, 0, time.UTC) }

// rulesTree builds a ranker input from items given in stored order; updated
// holds each item's last update.
type rulesTree struct {
	items   []guided.TreeItem
	updated map[uuid.UUID]time.Time
	names   map[uuid.UUID]string
}

func newRulesTree() *rulesTree {
	return &rulesTree{updated: map[uuid.UUID]time.Time{}, names: map[uuid.UUID]string{}}
}

// add appends an item and returns its id. parent is uuid.Nil for top-level.
func (rt *rulesTree) add(parent uuid.UUID, text string, done bool, due *time.Time, updated time.Time) uuid.UUID {
	id := uuid.New()
	it := guided.TreeItem{ID: id, Description: text, Done: done, DueDate: due}
	if parent != uuid.Nil {
		p := parent
		it.ParentID = &p
	}
	rt.items = append(rt.items, it)
	rt.updated[id] = updated
	rt.names[id] = text
	return id
}

func (rt *rulesTree) input() rulesInput {
	return rulesInput{Tree: guided.PlanTree{Items: rt.items}, UpdatedAt: rt.updated, Date: planDate}
}

// render writes groups as "Label: text (reason) | text (reason); Label: ..."
// so a whole draw compares in one string.
func render(groups []Group) string {
	var parts []string
	for _, g := range groups {
		var entries []string
		for _, e := range g.Entries {
			s := fmt.Sprintf("%s (%s)", e.Text, e.Reason)
			if e.Revision != nil {
				s += fmt.Sprintf(" r%d", *e.Revision)
			}
			entries = append(entries, s)
		}
		parts = append(parts, g.Label+": "+strings.Join(entries, " | "))
	}
	return strings.Join(parts, "; ")
}

func TestDrawRules_RanksOverdueThenDueSoonestThenNextStepUnderTopLevelGroups(t *testing.T) {
	rt := newRulesTree()
	graphs := rt.add(uuid.Nil, "Graphs", false, nil, oct(20))
	rt.add(graphs, "Solve 3 BFS problems", false, ptr(oct(28)), oct(20))
	rt.add(graphs, "Read the DFS chapter", false, ptr(oct(20)), oct(20))
	rt.add(graphs, "Dijkstra by hand", false, nil, oct(20))
	dp := rt.add(uuid.Nil, "Dynamic programming", false, ptr(oct(22)), oct(24))
	rt.add(dp, "Memoisation notes", true, nil, oct(24))
	rt.add(dp, "Knapsack problem", false, nil, oct(24))
	rt.add(uuid.Nil, "Strings", false, nil, oct(23))
	long := rt.add(uuid.Nil, "Operating systems: scheduling, memory and file systems", false, nil, oct(22))
	rt.add(long, "Round-robin exercise", false, ptr(oct(27)), oct(22))

	got := drawRules(rt.input())

	want := "Graphs: Read the DFS chapter (Overdue since Oct 20) | Solve 3 BFS problems (Due Oct 28); " +
		"Dynamic programming: Dynamic programming (Overdue since Oct 22) | Knapsack problem (Next step); " +
		"Operating systems: scheduling, memory an: Round-robin exercise (Due Oct 27)"
	if render(got) != want {
		t.Fatalf("draw\n got %s\nwant %s", render(got), want)
	}
	// Strings would open a fourth group, so it is skipped though a slot is free.
	for _, g := range got {
		for _, e := range g.Entries {
			if e.Kind != EntryExistingItem || e.ItemID == nil || e.Minutes != nil || e.Revision != nil {
				t.Fatalf("entry %+v: want an existing_item with no minutes and no revision", e)
			}
		}
	}
}

func TestDrawRules_EntriesCarryTheirItemAndParentFromTheTree(t *testing.T) {
	rt := newRulesTree()
	top := rt.add(uuid.Nil, "Graphs", false, nil, oct(20))
	step := rt.add(top, "BFS", false, nil, oct(20))

	got := drawRules(rt.input())

	if len(got) != 1 || len(got[0].Entries) != 1 {
		t.Fatalf("draw %s, want one entry", render(got))
	}
	e := got[0].Entries[0]
	if *e.ItemID != step || e.ParentItemID == nil || *e.ParentItemID != top || e.Text != "BFS" {
		t.Fatalf("entry %+v, want the step under its parent", e)
	}
}

func TestDrawRules_TiesKeepPlanOrder(t *testing.T) {
	rt := newRulesTree()
	a := rt.add(uuid.Nil, "A", false, nil, oct(24))
	rt.add(a, "a2 overdue", false, ptr(oct(21)), oct(24))
	rt.add(a, "a1 overdue", false, ptr(oct(21)), oct(24))
	rt.add(a, "a3 due", false, ptr(oct(26)), oct(24))
	rt.add(a, "a4 due", false, ptr(oct(26)), oct(24))
	rt.add(uuid.Nil, "B", false, nil, oct(23))
	rt.add(uuid.Nil, "C", false, nil, oct(23))

	got := drawRules(rt.input())

	want := "A: a2 overdue (Overdue since Oct 21) | a1 overdue (Overdue since Oct 21) | a3 due (Due Oct 26) | a4 due (Due Oct 26); " +
		"B: B (Next step); C: C (Next step)"
	if render(got) != want {
		t.Fatalf("draw\n got %s\nwant %s", render(got), want)
	}
}

func TestDrawRules_DueDatesCompareByDayNotInstant(t *testing.T) {
	rt := newRulesTree()
	top := rt.add(uuid.Nil, "A", false, nil, oct(24))
	rt.add(top, "due today late", false, ptr(planDate.Add(20*time.Hour)), oct(24))
	rt.add(top, "due today early", false, ptr(planDate.Add(time.Hour)), oct(24))
	rt.add(top, "due yesterday late", false, ptr(oct(24).Add(23*time.Hour)), oct(24))

	got := drawRules(rt.input())

	want := "A: due yesterday late (Overdue since Oct 24) | due today late (Due Oct 25) | due today early (Due Oct 25)"
	if render(got) != want {
		t.Fatalf("draw\n got %s\nwant %s", render(got), want)
	}
}

func TestDrawRules_NextStepIsTheFirstOpenChildOfEachTopLevelItemMostRecentlyUpdatedFirst(t *testing.T) {
	rt := newRulesTree()
	old := rt.add(uuid.Nil, "Old", false, nil, oct(1))
	rt.add(old, "old done", true, nil, oct(1))
	rt.add(old, "old open", false, nil, oct(1))
	rt.add(old, "old open later", false, nil, oct(1))
	fresh := rt.add(uuid.Nil, "Fresh", false, nil, oct(24))
	rt.add(fresh, "fresh done", true, nil, oct(24))
	finished := rt.add(uuid.Nil, "Finished", true, nil, oct(25))
	rt.add(finished, "finished done", true, nil, oct(25))
	doneWithOpenStep := rt.add(uuid.Nil, "Done parent", true, nil, oct(23))
	rt.add(doneWithOpenStep, "left over step", false, nil, oct(23))

	got := drawRules(rt.input())

	want := "Fresh: Fresh (Next step); Done parent: left over step (Next step); Old: old open (Next step)"
	if render(got) != want {
		t.Fatalf("draw\n got %s\nwant %s", render(got), want)
	}
}

func TestDrawRules_GroupsByTopLevelAncestorThroughDeeperNesting(t *testing.T) {
	rt := newRulesTree()
	top := rt.add(uuid.Nil, "Graphs", false, nil, oct(20))
	mid := rt.add(top, "Traversals", false, nil, oct(20))
	rt.add(mid, "BFS on a grid", false, ptr(oct(21)), oct(20))

	got := drawRules(rt.input())

	want := "Graphs: BFS on a grid (Overdue since Oct 21) | Traversals (Next step)"
	if render(got) != want {
		t.Fatalf("draw\n got %s\nwant %s", render(got), want)
	}
}

func TestDrawRules_StopsAtSixEntries(t *testing.T) {
	rt := newRulesTree()
	top := rt.add(uuid.Nil, "A", false, nil, oct(24))
	for i := range 8 {
		rt.add(top, fmt.Sprintf("step %d", i), false, ptr(oct(26+i)), oct(24))
	}

	got := drawRules(rt.input())

	if len(got) != 1 || len(got[0].Entries) != MaxEntries || got[0].Entries[5].Text != "step 5" {
		t.Fatalf("draw %s, want the first six by due date", render(got))
	}
}

func TestDrawRules_DueRevisionsComeFirstInARevisionsGroupOldestDueFirst(t *testing.T) {
	rt := newRulesTree()
	top := rt.add(uuid.Nil, "Graphs", false, nil, oct(20))
	bfs := rt.add(top, "BFS", true, nil, oct(20))
	dfs := rt.add(top, "DFS", false, ptr(oct(21)), oct(20))
	rt.add(uuid.Nil, "Strings", false, nil, oct(19))
	in := rt.input()
	in.DueRevisions = []dueRevision{
		{ItemID: bfs, Rung: 1, DueOn: oct(25)},
		{ItemID: dfs, Rung: 2, DueOn: oct(23)},
		{ItemID: uuid.New(), Rung: 1, DueOn: oct(22)}, // item gone from the tree
	}

	got := drawRules(in)

	want := "Revisions: DFS (Revision 2 due) r2 | BFS (Revision 1 due) r1; Strings: Strings (Next step)"
	if render(got) != want {
		t.Fatalf("draw\n got %s\nwant %s", render(got), want)
	}
}

func TestDrawRules_PinnedEntriesKeepTheirPlaceAndCountTowardTheBounds(t *testing.T) {
	rt := newRulesTree()
	a := rt.add(uuid.Nil, "A", false, nil, oct(24))
	pinnedItem := rt.add(a, "a ticked", false, ptr(oct(20)), oct(24))
	rt.add(a, "a overdue", false, ptr(oct(21)), oct(24))
	rt.add(uuid.Nil, "B", false, nil, oct(23))
	rt.add(uuid.Nil, "C", false, nil, oct(22))
	pinnedID := uuid.New()
	in := rt.input()
	in.Pinned = []Group{
		{Label: "Warm-up", Entries: []Entry{{ID: pinnedID, Kind: EntryNewStep, Text: "Stretch", Reason: "start easy"}}},
		{Label: "A", Entries: []Entry{{ID: uuid.New(), Kind: EntryExistingItem, ItemID: &pinnedItem, Text: "a ticked", Reason: "Overdue since Oct 20"}}},
	}

	got := drawRules(in)

	want := "Warm-up: Stretch (start easy); A: a ticked (Overdue since Oct 20) | a overdue (Overdue since Oct 21); B: B (Next step)"
	if render(got) != want {
		t.Fatalf("draw\n got %s\nwant %s", render(got), want)
	}
	if got[0].Entries[0].ID != pinnedID {
		t.Fatal("a pinned entry lost its id")
	}
	if len(in.Pinned[1].Entries) != 1 {
		t.Fatal("the draw changed the caller's pinned groups")
	}
}

func TestDrawRules_LabelsAreCutToFortyCharacters(t *testing.T) {
	rt := newRulesTree()
	rt.add(uuid.Nil, "  Ünïcödé "+strings.Repeat("é", 50), false, nil, oct(24))
	rt.add(uuid.Nil, "   ", false, nil, oct(23))

	got := drawRules(rt.input())

	if len(got) != 2 {
		t.Fatalf("draw %s, want two groups", render(got))
	}
	for _, g := range got {
		if n := len([]rune(g.Label)); n < 1 || n > MaxLabelLength || strings.TrimSpace(g.Label) != g.Label {
			t.Fatalf("label %q: want 1–%d characters, trimmed", g.Label, MaxLabelLength)
		}
	}
	if !strings.HasPrefix(got[0].Label, "Ünïcödé é") {
		t.Fatalf("label %q, want the description's start", got[0].Label)
	}
}

func TestDrawRules_BlankItemDescriptionGetsTheUntitledText(t *testing.T) {
	rt := newRulesTree()
	top := rt.add(uuid.Nil, "  ", false, nil, oct(20))
	rt.add(top, "", false, nil, oct(20))

	got := drawRules(rt.input())

	if len(got) != 1 || got[0].Label != untitledLabel || len(got[0].Entries) != 1 || got[0].Entries[0].Text != untitledLabel {
		t.Fatalf("draw %s, want one Untitled entry under the Untitled group", render(got))
	}
}
