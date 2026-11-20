package dayplan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
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

// goldenIDs are fixed ids, so a prompt or a hash can be compared byte for
// byte with one captured before a change.
var (
	goldenGraphs = uuid.MustParse("11111111-1111-4111-8111-111111111111")
	goldenBFS    = uuid.MustParse("22222222-2222-4222-8222-222222222222")
	goldenDFS    = uuid.MustParse("33333333-3333-4333-8333-333333333333")
	goldenDP     = uuid.MustParse("44444444-4444-4444-8444-444444444444")
)

// goldenSystemRulesSHA256 is systemRules' digest before FS-02JBZ §Change:
// Hours a day; the change must not edit the static rules (R97).
const goldenSystemRulesSHA256 = "88a00eff77e46e17a3a6ce0133572cd0b70145b33eb2a065c499e33729c767d2"

// goldenInput is a prompt input with every block filled: a snippet, a tree
// with a due date, a due revision and an entry ticked today with minutes.
func goldenInput() promptInput {
	due := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	return promptInput{
		Tree: guided.NewPlanTree([]guided.PlanItem{
			{ID: goldenGraphs, Description: "Graphs", Type: "task", Scope: "longterm", DueDate: &due},
			{ID: goldenBFS, ParentID: &goldenGraphs, Description: "Solve 3 BFS problems", Type: "task", Scope: "longterm"},
			{ID: goldenDFS, ParentID: &goldenGraphs, Description: "Read the DFS chapter", Type: "task", Scope: "longterm"},
			{ID: goldenDP, Description: "Dynamic programming", Type: "task", Scope: "longterm"},
		}),
		Profile:  PlanProfile{Type: "learning", Focus: "Get through the graph chapter"},
		Date:     time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC),
		Snippets: []guided.Snippet{{Origin: guided.OriginPlaybook, Text: "Break a topic into problems"}},
		Ticked: []Group{{Label: "Graphs", Entries: []Entry{
			{ID: uuid.MustParse("55555555-5555-4555-8555-555555555555"), Kind: EntryExistingItem, ItemID: &goldenBFS,
				Text: "Solve 3 BFS problems", Minutes: ptr(60), Ticked: true},
		}}},
		Due: []dueRevision{{ItemID: goldenDFS, Rung: 1, DueOn: time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)}},
	}
}

func readGolden(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// requestOf is the user message's REQUEST line onward.
func requestOf(t *testing.T, user string) string {
	t.Helper()
	i := strings.Index(user, "REQUEST: ")
	if i < 0 {
		t.Fatalf("user message has no request:\n%s", user)
	}
	return user[i:]
}

// The goldens were captured from buildPrompt before FS-02JBZ §Change: Hours a
// day: with no daily time the prompt is exactly what it was (R96).
func TestBuildPrompt_NoDailyMinutes_IsByteIdenticalToBeforeTheChange(t *testing.T) {
	in := goldenInput()
	ticked := buildPrompt(in)
	in.Ticked = nil
	plain := buildPrompt(in)

	if ticked.User != readGolden(t, "day_plan_prompt_ticked.golden") {
		t.Fatalf("user message with a ticked entry changed:\n%s", ticked.User)
	}
	if plain.User != readGolden(t, "day_plan_prompt.golden") {
		t.Fatalf("user message changed:\n%s", plain.User)
	}
	sum := sha256.Sum256([]byte(plain.System))
	if hex.EncodeToString(sum[:]) != goldenSystemRulesSHA256 || ticked.System != plain.System {
		t.Fatal("system rules changed")
	}
}

func TestBuildPrompt_DailyMinutes_StatesTheTimeOnceInTheRequestAndLeavesTheRulesAlone(t *testing.T) {
	in := goldenInput()
	in.Ticked = nil
	before := buildPrompt(in)
	in.DailyMinutes = ptr(150)

	p := buildPrompt(in)

	if p.System != before.System {
		t.Fatal("a daily time must never change the system rules (R97)")
	}
	head := before.User[:strings.Index(before.User, "REQUEST: ")]
	if !strings.HasPrefix(p.User, head) {
		t.Fatalf("a daily time changed the plan data:\n%s", p.User)
	}
	req := requestOf(t, p.User)
	if strings.Count(p.User, "150") != 1 || !strings.Contains(req, "150 minutes a day") {
		t.Fatalf("request = %q, want the daily time stated once", req)
	}
	for _, want := range []string{"Plan today: at most 6 entries in at most 3 groups.", "sum to at most", "Fewer entries is fine."} {
		if !strings.Contains(req, want) {
			t.Fatalf("request = %q, want %q", req, want)
		}
	}
}

func TestBuildPrompt_DailyMinutesWithTickedEntries_ListsTheirMinutesAndStatesTheTimeLeft(t *testing.T) {
	in := goldenInput()
	in.DailyMinutes = ptr(150)

	p := buildPrompt(in)

	line := `{"group":"Graphs","item_id":"` + goldenBFS.String() + `","text":"Solve 3 BFS problems","minutes":60}` + "\n"
	if !strings.Contains(p.User, "ALREADY TICKED TODAY (fixed and done; never repeat them):\n"+line) {
		t.Fatalf("ticked block lacks the entry's minutes:\n%s", p.User)
	}
	req := requestOf(t, p.User)
	for _, want := range []string{"at most 5 more entries", "150 minutes a day", "already take 60", "the 90 minutes left", "Fewer entries is fine."} {
		if !strings.Contains(req, want) {
			t.Fatalf("request = %q, want %q", req, want)
		}
	}
}

func TestBuildPrompt_TickedEntriesUseTheWholeDailyTime_SaysAddingNothingIsFine(t *testing.T) {
	for name, minutes := range map[string]int{"exactly": 60, "over": 90} {
		t.Run(name, func(t *testing.T) {
			in := goldenInput()
			in.Ticked[0].Entries[0].Minutes = ptr(minutes)
			in.DailyMinutes = ptr(60)

			req := requestOf(t, buildPrompt(in).User)

			for _, want := range []string{"60 minutes a day", "no time is left", "adding nothing is fine"} {
				if !strings.Contains(req, want) {
					t.Fatalf("request = %q, want %q", req, want)
				}
			}
			if strings.Contains(req, "minutes left") {
				t.Fatalf("request = %q, want no time left stated, not a negative one", req)
			}
		})
	}
}

func TestBuildPrompt_TickedEntryWithoutMinutes_CountsNothingAgainstTheTime(t *testing.T) {
	in := goldenInput()
	in.Ticked[0].Entries[0].Minutes = nil
	in.DailyMinutes = ptr(120)

	p := buildPrompt(in)

	if strings.Contains(p.User, `"minutes"`) {
		t.Fatalf("a ticked entry with no minutes must list none:\n%s", p.User)
	}
	req := requestOf(t, p.User)
	if !strings.Contains(req, "120 minutes a day") || strings.Contains(req, "already take") ||
		!strings.Contains(req, "added entries' minutes sum to at most that") {
		t.Fatalf("request = %q, want the whole daily time for the added entries", req)
	}
}
