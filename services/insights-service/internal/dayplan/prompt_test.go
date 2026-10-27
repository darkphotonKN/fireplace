package dayplan

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
)

func TestBuildPrompt_UserMessageIsSnippetsThenPlanDataThenRequest(t *testing.T) {
	tf := newTreeFixture()
	date := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)

	p := buildPrompt(promptInput{
		Tree:     tree(tf),
		Profile:  PlanProfile{Type: "learning", Focus: "Get through\n the graph chapter"},
		Date:     date,
		Snippets: []guided.Snippet{{Origin: guided.OriginPlaybook, Text: "Break a topic into problems"}},
	})

	order := []string{
		"WORKED EXAMPLES", "- Break a topic into problems",
		"PLAN TYPE: learning\n",
		"PLAN FOCUS: Get through the graph chapter\n",
		"PLAN TREE (one JSON object per line):\n",
		`"text":"Graphs"`,
		"DUE REVISIONS: none\n",
		"ALREADY TICKED TODAY: none\n",
		"REQUEST: Today is 2026-10-25. Plan today: at most 6 entries in at most 3 groups.",
	}
	at := 0
	for _, want := range order {
		i := strings.Index(p.User[at:], want)
		if i < 0 {
			t.Fatalf("user message lacks %q after offset %d:\n%s", want, at, p.User)
		}
		at += i + len(want)
	}
	if strings.Contains(p.System, "learning") || strings.Contains(p.System, "2026") {
		t.Fatal("plan data leaked into the static rules")
	}
}

func TestBuildPrompt_StaticRulesAreByteIdenticalAcrossCalls(t *testing.T) {
	tf := newTreeFixture()
	a := buildPrompt(promptInput{Tree: tree(tf), Profile: PlanProfile{Type: "learning", Focus: "A"}, Date: time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)})
	b := buildPrompt(promptInput{Tree: guided.PlanTree{}, Profile: PlanProfile{Type: "project"}, Date: time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC),
		Snippets: []guided.Snippet{{Origin: guided.OriginPersonal, Text: "x"}}})

	if a.System != b.System || a.System != systemRules {
		t.Fatal("system rules differ between calls")
	}
	for _, rule := range []string{"at most 6 entries", "at most 3 groups", "topic affinity", "1 to 40 characters",
		"overdue", "nearer due dates", "minutes", "between 5 and 240", `role is "top-level"`, "never a step's id"} {
		if !strings.Contains(systemRules, rule) {
			t.Fatalf("system rules lack %q", rule)
		}
	}
}

func TestBuildPrompt_NoSnippetsOmitsTheBlockAndAnEmptyFocusSaysNone(t *testing.T) {
	p := buildPrompt(promptInput{Tree: guided.PlanTree{}, Profile: PlanProfile{Type: "project"}, Date: time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)})

	if !strings.HasPrefix(p.User, "PLAN TYPE: project\nPLAN FOCUS: none\n") {
		t.Fatalf("user message = %q, want it to open with the plan data", p.User)
	}
}

func TestBuildPrompt_TreeLinesSayWhetherAnIdIsTopLevelOrAStep(t *testing.T) {
	tf := newTreeFixture()
	p := buildPrompt(promptInput{Tree: tree(tf), Date: time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)})

	start := strings.Index(p.User, "PLAN TREE")
	end := strings.Index(p.User, "DUE REVISIONS")
	lines := strings.Split(strings.TrimSpace(p.User[start:end]), "\n")[1:]
	roles := map[string]string{}
	for _, l := range lines {
		var line treeLine
		if err := json.Unmarshal([]byte(l), &line); err != nil {
			t.Fatalf("line %q: %v", l, err)
		}
		roles[line.ID] = line.Role
		if (line.Role == roleStep) != (line.ParentID != "") {
			t.Fatalf("line %q: role must match having a parent", l)
		}
	}
	if len(roles) != 4 || roles[tf.graphs.String()] != "top-level" || roles[tf.bfs.String()] != "step" ||
		roles[tf.dfs.String()] != "step" || roles[tf.dp.String()] != "top-level" {
		t.Fatalf("roles = %v, want the four tree items labelled", roles)
	}
	if !strings.Contains(p.User, `"due":"2026-10-26"`) {
		t.Fatal("due date missing from the tree")
	}
}

func TestSchema_HasNoRevisionFieldAndBoundsTheOutput(t *testing.T) {
	s := string(schema)
	if strings.Contains(s, "revision") {
		t.Fatal("the model's schema must carry no revision field (R10)")
	}
	for _, want := range []string{`"maxItems": 3`, `"maxItems": 6`, `"minItems": 1`, `["string", "null"]`} {
		if !strings.Contains(s, want) {
			t.Fatalf("schema lacks %s", want)
		}
	}
	if strings.Contains(strings.ToLower(s), "milestone") {
		t.Fatal("schema names say item, never milestone")
	}
	var v any
	if err := json.Unmarshal(schema, &v); err != nil {
		t.Fatalf("schema is not JSON: %v", err)
	}
}

func TestBuildPrompt_DueRevisionsAreTopPriorityCandidatesLabelledRevisionN(t *testing.T) {
	tf := newTreeFixture()
	p := buildPrompt(promptInput{
		Tree: tree(tf), Profile: PlanProfile{Type: "learning"}, Date: time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC),
		Due: []dueRevision{{ItemID: tf.dfs, Rung: 2, DueOn: oct(23)}, {ItemID: tf.bfs, Rung: 1, DueOn: oct(25)}},
	})

	want := "DUE REVISIONS (top priority; place each as an existing_item for its item):\n" +
		`{"item_id":"` + tf.dfs.String() + `","revision":"revision 2","text":"Read the DFS chapter"}` + "\n" +
		`{"item_id":"` + tf.bfs.String() + `","revision":"revision 1","text":"Solve 3 BFS problems"}` + "\n" +
		"ALREADY TICKED TODAY: none\n"
	if !strings.Contains(p.User, want) {
		t.Fatalf("user message lacks the due revisions block %q:\n%s", want, p.User)
	}
	if strings.Contains(p.System, "revision 2") {
		t.Fatal("due revisions leaked into the static rules")
	}
}
