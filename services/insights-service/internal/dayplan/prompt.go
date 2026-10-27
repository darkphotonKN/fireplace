package dayplan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
)

// Tree roles as the prompt labels them. Each line says outright whether its
// id is a top-level item or a step, so the model never has to infer it from
// parent_id: live output once put a step's id where a top-level id belongs.
const (
	roleTopLevel = "top-level"
	roleStep     = "step"
)

// systemRules is the day plan generator's system message: byte-identical
// across calls (DF4ZB R2), so a stable prefix can hit OpenAI's prompt cache.
// Prompt prose may say "milestone" (DF4ZB R6); no type or field here does.
const systemRules = `You plan one user's day on one plan in Fireplace, a productivity app.

The plan is a tree. A line with role "top-level" is a milestone. A line with role "step" is a step under the item whose id is its parent_id. Done items are context only.

Pick at most 6 entries for today and put them in at most 3 groups. Each entry is one of:
- "existing_item": an open item (top-level or step) that already exists. Set item_id to its id, parent_item_id to null, and text to its text.
- "new_step": one new step for an open top-level item. Set parent_item_id to the id of a line whose role is "top-level" and whose done is false, never a step's id. Set item_id to null and text to the step: a concrete action that starts with a verb, finishable in one sitting, 4 to 20 words, and not a repeat of a step that item already has.

Rules:
- Copy ids exactly from the plan tree; never invent one.
- Group entries by topic affinity. Give each group a short label, 1 to 40 characters, naming the topic.
- Put overdue items and nearer due dates first, both in the order of groups and within a group. Items under the plan's focus come next.
- Give every entry a rough estimate of the minutes it takes today, between 5 and 240.
- Use each item at most once, and each new step's text at most once.
- reason is one short line, under 20 words, saying why this entry matters today.
- Due revisions, when listed, come first: place each as an existing_item for its item. Already ticked entries are fixed and done: never repeat them, and fill only the remaining entries.
- Worked examples, when given, show how other goals were broken down. Use them for ideas only; never copy their wording as the user's own history.
- Refer to the user's past work only when a block of things this user finished on other plans is given, and only to what it says. Without that block, never claim or imply what the user did before ("last time you...", "like you did before").`

// Prompt is one generation's input: the static system rules and the user
// message (snippets, plan data, request, in that order).
type Prompt struct {
	System string
	User   string
}

// promptInput is everything the user message is built from.
type promptInput struct {
	Tree     guided.PlanTree
	Profile  PlanProfile
	Date     time.Time
	Snippets []guided.Snippet
	// Ticked are the groups of entries ticked today (R12), fixed and done.
	Ticked []Group
	// Due are the revisions due on the plan date (R44), oldest due first,
	// each of an item in Tree.
	Due []dueRevision
}

// buildPrompt assembles the prompt in DF4ZB R2's order: system rules, then
// snippets (omitted entirely when there are none), then the plan data (type,
// focus, tree, due revisions, ticked entries), then the request (R7).
//
// Due revisions are listed as top-priority candidates labelled "revision N"
// (R7, D6): the server decided they are due, the model only places them
// (ADR-0019 clause 5). Ticked entries are listed as fixed and done with their
// group labels, and the request asks only for the slots left (R12); with none
// ticked, the request is the plain one.
func buildPrompt(in promptInput) Prompt {
	var b strings.Builder
	guided.WriteSnippets(&b, in.Snippets)
	b.WriteString("PLAN TYPE: ")
	b.WriteString(orNone(in.Profile.Type))
	b.WriteString("\nPLAN FOCUS: ")
	b.WriteString(orNone(in.Profile.Focus))
	b.WriteString("\nPLAN TREE (one JSON object per line):\n")
	b.WriteString(serializeTree(in.Tree))
	writeDue(&b, in.Tree, in.Due)
	ticked := writeTicked(&b, in.Ticked)
	b.WriteString("REQUEST: Today is ")
	b.WriteString(in.Date.Format(guided.DateLayout))
	if ticked == 0 {
		b.WriteString(". Plan today: at most 6 entries in at most 3 groups.")
	} else {
		fmt.Fprintf(&b, ". Plan today: at most %d more entries, in at most 3 groups counting the ticked entries' groups. "+
			"To add to a ticked entry's group, use its label exactly.", MaxEntries-ticked)
	}
	return Prompt{System: systemRules, User: b.String()}
}

// dueLine is one due revision as the model sees it: the item and its label.
type dueLine struct {
	ItemID   string `json:"item_id"`
	Revision string `json:"revision"`
	Text     string `json:"text"`
}

// writeDue writes the DUE REVISIONS block, one JSON object per due revision,
// or says none.
func writeDue(b *strings.Builder, tree guided.PlanTree, due []dueRevision) {
	if len(due) == 0 {
		b.WriteString("DUE REVISIONS: none\n")
		return
	}
	text := make(map[string]string, len(tree.Items))
	for _, it := range tree.Items {
		text[it.ID.String()] = it.Description
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, r := range due {
		id := r.ItemID.String()
		// Encoding a struct of strings cannot fail.
		_ = enc.Encode(dueLine{ItemID: id, Revision: revisionLabel(r.Rung), Text: text[id]})
	}
	b.WriteString("DUE REVISIONS (top priority; place each as an existing_item for its item):\n")
	b.Write(buf.Bytes())
}

// tickedLine is one entry ticked today as the model sees it.
type tickedLine struct {
	Group  string `json:"group"`
	ItemID string `json:"item_id"`
	Text   string `json:"text"`
}

// writeTicked writes the ALREADY TICKED TODAY block, one JSON object per
// entry, or says none, and returns how many entries it listed.
func writeTicked(b *strings.Builder, groups []Group) int {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	n := 0
	for _, g := range groups {
		for _, e := range g.Entries {
			line := tickedLine{Group: g.Label, Text: e.Text}
			if e.ItemID != nil {
				line.ItemID = e.ItemID.String()
			}
			// Encoding a struct of strings cannot fail.
			_ = enc.Encode(line)
			n++
		}
	}
	if n == 0 {
		b.WriteString("ALREADY TICKED TODAY: none\n")
		return 0
	}
	b.WriteString("ALREADY TICKED TODAY (fixed and done; never repeat them):\n")
	b.Write(buf.Bytes())
	return n
}

// orNone collapses s onto one line, or says none.
func orNone(s string) string {
	if s = strings.Join(strings.Fields(s), " "); s == "" {
		return "none"
	}
	return s
}

// treeLine is one plan-tree item as the day plan's model sees it. Field order
// is the serialization order.
type treeLine struct {
	ID       string `json:"id"`
	Role     string `json:"role"`
	ParentID string `json:"parent_id,omitempty"`
	Done     bool   `json:"done"`
	Due      string `json:"due,omitempty"`
	Text     string `json:"text"`
}

// serializeTree writes the tree one JSON object per line, each labelled with
// its role. It carries the same fields the plan-state hash covers.
func serializeTree(tree guided.PlanTree) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, it := range tree.Items {
		line := treeLine{ID: it.ID.String(), Role: roleTopLevel, Done: it.Done, Text: it.Description}
		if it.ParentID != nil {
			line.Role = roleStep
			line.ParentID = it.ParentID.String()
		}
		if it.DueDate != nil {
			line.Due = it.DueDate.UTC().Format(guided.DateLayout)
		}
		// Encoding a struct of strings and a bool cannot fail.
		_ = enc.Encode(line)
	}
	return buf.String()
}
