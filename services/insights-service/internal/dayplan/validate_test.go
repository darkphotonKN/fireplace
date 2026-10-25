package dayplan

import (
	"strings"
	"testing"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
)

func tree(tf treeFixture) guided.PlanTree { return guided.NewPlanTree(tf.items()) }

// entryKeys names each kept entry by item id or new-step text, per group, so a
// table row can say exactly what survives.
func entryKeys(groups []Group) [][]string {
	out := make([][]string, 0, len(groups))
	for _, g := range groups {
		var keys []string
		for _, e := range g.Entries {
			if e.ItemID != nil {
				keys = append(keys, e.ItemID.String())
			} else {
				keys = append(keys, e.Text)
			}
		}
		out = append(out, keys)
	}
	return out
}

func TestValidate_EachDropRule(t *testing.T) {
	tf := newTreeFixture()
	bad := "not-a-uuid"
	unknown := uuid.NewString()
	bfs, graphs, dp := tf.bfs.String(), tf.graphs.String(), tf.dp.String()

	tests := []struct {
		name   string
		groups []genGroup
		want   [][]string
	}{
		{"valid existing child, existing top-level and new step are kept",
			[]genGroup{{Label: "G", Entries: []genEntry{existing(tf.bfs, "x", 30), existing(tf.graphs, "x", 30), newStep(tf.dp, "Do the knapsack problem", 30)}}},
			[][]string{{bfs, graphs, "Do the knapsack problem"}}},
		{"existing item with a malformed id",
			[]genGroup{{Label: "G", Entries: []genEntry{{Kind: "existing_item", ItemID: &bad, Text: "x", Reason: "r", Minutes: 30}, existing(tf.bfs, "x", 30)}}},
			[][]string{{bfs}}},
		{"existing item not in the plan",
			[]genGroup{{Label: "G", Entries: []genEntry{{Kind: "existing_item", ItemID: &unknown, Text: "x", Reason: "r", Minutes: 30}, existing(tf.bfs, "x", 30)}}},
			[][]string{{bfs}}},
		{"existing item that is done, a note, or archived",
			[]genGroup{{Label: "G", Entries: []genEntry{existing(tf.dfs, "x", 30), existing(tf.note, "x", 30), existing(tf.archived, "x", 30), existing(tf.bfs, "x", 30)}}},
			[][]string{{bfs}}},
		{"existing item with no id",
			[]genGroup{{Label: "G", Entries: []genEntry{{Kind: "existing_item", Text: "x", Reason: "r", Minutes: 30}, existing(tf.bfs, "x", 30)}}},
			[][]string{{bfs}}},
		{"new step under a step, not a top-level item",
			[]genGroup{{Label: "G", Entries: []genEntry{newStep(tf.bfs, "Do the knapsack problem", 30), existing(tf.bfs, "x", 30)}}},
			[][]string{{bfs}}},
		{"new step under an unknown parent or with no parent",
			[]genGroup{{Label: "G", Entries: []genEntry{newStep(uuid.New(), "Do the knapsack problem", 30), {Kind: "new_step", Text: "Do X", Reason: "r", Minutes: 30}, existing(tf.bfs, "x", 30)}}},
			[][]string{{bfs}}},
		{"new step repeating an existing child, done or open, in any case and spacing",
			[]genGroup{{Label: "G", Entries: []genEntry{newStep(tf.graphs, "  read the DFS   chapter", 30), newStep(tf.graphs, "SOLVE 3 BFS PROBLEMS", 30), existing(tf.bfs, "x", 30)}}},
			[][]string{{bfs}}},
		{"new step with empty text",
			[]genGroup{{Label: "G", Entries: []genEntry{newStep(tf.dp, "   ", 30), existing(tf.bfs, "x", 30)}}},
			[][]string{{bfs}}},
		{"unknown kind",
			[]genGroup{{Label: "G", Entries: []genEntry{{Kind: "existing_step", ItemID: &bfs, Text: "x", Reason: "r", Minutes: 30}, existing(tf.graphs, "x", 30)}}},
			[][]string{{graphs}}},
		{"empty reason",
			[]genGroup{{Label: "G", Entries: []genEntry{func() genEntry { e := existing(tf.bfs, "x", 30); e.Reason = "  "; return e }(), existing(tf.graphs, "x", 30)}}},
			[][]string{{graphs}}},
		{"second entry for the same item, even in another group",
			[]genGroup{{Label: "A", Entries: []genEntry{existing(tf.bfs, "x", 30)}}, {Label: "B", Entries: []genEntry{existing(tf.bfs, "x", 30), existing(tf.graphs, "x", 30)}}},
			[][]string{{bfs}, {graphs}}},
		{"second new step with the same text, even under another parent",
			[]genGroup{{Label: "A", Entries: []genEntry{newStep(tf.dp, "Do the knapsack problem", 30), newStep(tf.graphs, "do the  KNAPSACK problem", 30)}}},
			[][]string{{"Do the knapsack problem"}}},
		{"fourth group",
			[]genGroup{
				{Label: "A", Entries: []genEntry{existing(tf.bfs, "x", 30)}},
				{Label: "B", Entries: []genEntry{existing(tf.graphs, "x", 30)}},
				{Label: "C", Entries: []genEntry{existing(tf.dp, "x", 30)}},
				{Label: "D", Entries: []genEntry{newStep(tf.dp, "Do the knapsack problem", 30)}},
			},
			[][]string{{bfs}, {graphs}, {dp}}},
		{"seventh entry",
			[]genGroup{
				{Label: "A", Entries: []genEntry{existing(tf.bfs, "x", 30), existing(tf.graphs, "x", 30), existing(tf.dp, "x", 30)}},
				{Label: "B", Entries: []genEntry{newStep(tf.dp, "Step one here now", 30), newStep(tf.dp, "Step two here now", 30), newStep(tf.dp, "Step three here now", 30), newStep(tf.dp, "Step four here now", 30)}},
			},
			[][]string{{bfs, graphs, dp}, {"Step one here now", "Step two here now", "Step three here now"}}},
		{"group left with no entries is dropped",
			[]genGroup{{Label: "A", Entries: []genEntry{existing(tf.dfs, "x", 30)}}, {Label: "B", Entries: []genEntry{existing(tf.bfs, "x", 30)}}},
			[][]string{{bfs}}},
		{"group with an empty or overlong label is dropped",
			[]genGroup{{Label: "  ", Entries: []genEntry{existing(tf.bfs, "x", 30)}}, {Label: strings.Repeat("x", 41), Entries: []genEntry{existing(tf.graphs, "x", 30)}}, {Label: "C", Entries: []genEntry{existing(tf.dp, "x", 30)}}},
			[][]string{{dp}}},
		{"only invalid entries leave an empty plan",
			[]genGroup{{Label: "A", Entries: []genEntry{existing(tf.dfs, "x", 30), existing(uuid.New(), "x", 30)}}},
			[][]string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := entryKeys(validate(tree(tf), nil, nil, toGenerated(tt.groups)))
			if len(got) != len(tt.want) {
				t.Fatalf("groups = %v, want %v", got, tt.want)
			}
			for i := range got {
				if strings.Join(got[i], "|") != strings.Join(tt.want[i], "|") {
					t.Fatalf("groups = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestValidate_OutOfRangeMinutesIsLeftOutAndTheEntryKept(t *testing.T) {
	tf := newTreeFixture()
	for _, m := range []int{0, 4, 241, -10} {
		groups := validate(tree(tf), nil, nil, toGenerated([]genGroup{{Label: "A", Entries: []genEntry{existing(tf.bfs, "x", m)}}}))
		if len(groups) != 1 || len(groups[0].Entries) != 1 || groups[0].Entries[0].Minutes != nil {
			t.Fatalf("minutes %d: groups = %+v, want the entry kept with no minutes", m, groups)
		}
	}
	for _, m := range []int{5, 240} {
		groups := validate(tree(tf), nil, nil, toGenerated([]genGroup{{Label: "A", Entries: []genEntry{existing(tf.bfs, "x", m)}}}))
		if got := groups[0].Entries[0].Minutes; got == nil || *got != m {
			t.Fatalf("minutes %d kept as %v", m, got)
		}
	}
}

func TestValidate_LabelsAreTrimmedAndANewStepsParentIsTheModelsTopLevelChoice(t *testing.T) {
	tf := newTreeFixture()
	groups := validate(tree(tf), nil, nil, toGenerated([]genGroup{{Label: "  Dynamic programming ", Entries: []genEntry{newStep(tf.dp, "  Do the knapsack problem ", 30)}}}))

	if len(groups) != 1 || groups[0].Label != "Dynamic programming" {
		t.Fatalf("groups = %+v, want the trimmed label", groups)
	}
	e := groups[0].Entries[0]
	if e.Text != "Do the knapsack problem" || *e.ParentItemID != tf.dp || e.ItemID != nil || e.Revision != nil {
		t.Fatalf("entry = %+v", e)
	}
}

// toGenerated turns test groups into the parsed model output.
func toGenerated(groups []genGroup) []generatedGroup {
	out, err := parseGenerated(modelOutput(groups...))
	if err != nil {
		panic(err)
	}
	return out
}

func TestValidate_ADoneItemIsACandidateOnlyWithARevisionDue(t *testing.T) {
	tf := newTreeFixture()
	generated := toGenerated([]genGroup{{Label: "A", Entries: []genEntry{existing(tf.dfs, "x", 20)}}})

	if got := validate(tree(tf), nil, nil, generated); len(got) != 0 {
		t.Fatalf("done item with nothing due = %+v, want dropped", got)
	}
	got := validate(tree(tf), nil, []dueRevision{{ItemID: tf.dfs, Rung: 1, DueOn: oct(25)}}, generated)
	if len(got) != 1 || *got[0].Entries[0].ItemID != tf.dfs {
		t.Fatalf("done item with a revision due = %+v, want kept (S7)", got)
	}
}

// An item with a blank description still makes a storable entry: its text
// falls back to the label a blank top-level item gets, never blank text
// (day_plan_entries.text must not be blank).
func TestValidate_BlankItemDescriptionGetsTheUntitledText(t *testing.T) {
	blank, child := uuid.New(), uuid.New()
	tr := guided.NewPlanTree([]guided.PlanItem{
		{ID: blank, Description: "   ", Type: "task", Scope: "longterm"},
		{ID: child, ParentID: &blank, Description: "", Type: "task", Scope: "longterm"},
	})

	got := validate(tr, nil, nil, toGenerated([]genGroup{{Label: "G", Entries: []genEntry{existing(blank, "x", 30), existing(child, "x", 30)}}}))

	if len(got) != 1 || len(got[0].Entries) != 2 {
		t.Fatalf("groups = %+v, want both entries kept", got)
	}
	for _, e := range got[0].Entries {
		if e.Text != untitledLabel {
			t.Fatalf("entry text = %q, want %q", e.Text, untitledLabel)
		}
	}
}
