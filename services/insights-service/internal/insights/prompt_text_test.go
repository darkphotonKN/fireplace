package insights

import (
	"strings"
	"testing"
	"unicode/utf8"

	planpb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
)

func strPtr(s string) *string { return &s }

// FS-PGG2N R45: a note contributes its summary, else its body's first 280
// characters; a task its full description.
func TestFormatChecklist_NotesUseSummaryOrTheBodysStart(t *testing.T) {
	longTask := strings.Repeat("t", 400)
	body := strings.Repeat("é", 400)
	pc := &PlanContext{ChecklistItems: []ChecklistItem{
		{Description: longTask, Scope: "longterm", Type: "task"},
		{Description: "a long body " + body, Scope: "longterm", Type: "note", Summary: strPtr("gist")},
		{Description: body, Scope: "longterm", Type: "note"},
	}}

	lines := strings.Split(strings.TrimRight(formatChecklist(pc), "\n"), "\n")

	if len(lines) != 3 {
		t.Fatalf("%d lines, want 3:\n%s", len(lines), formatChecklist(pc))
	}
	if lines[0] != "A longterm task: "+longTask {
		t.Errorf("task line = %q, want it unchanged", lines[0])
	}
	if !strings.HasSuffix(lines[1], ": gist") || strings.Contains(lines[1], "task") || strings.Contains(lines[1], "body") {
		t.Errorf("note with summary = %q, want its summary on a note line", lines[1])
	}
	if !strings.HasSuffix(lines[2], ": "+strings.Repeat("é", NotePromptCap)) || strings.Contains(lines[2], "task") {
		t.Errorf("note without summary = %q, want its first %d characters", lines[2], NotePromptCap)
	}
}

// R45: a note's text is one line of the one-line-per-item prompt, however
// its body or summary breaks: runs of whitespace become one space.
func TestFormatChecklist_AMultiLineNoteIsOneLine(t *testing.T) {
	pc := &PlanContext{ChecklistItems: []ChecklistItem{
		{Description: "first para\n\n  second\tpara\r\n", Scope: "longterm", Type: "note"},
		{Description: "body", Scope: "longterm", Type: "note", Summary: strPtr("the\ngist ")},
		{Description: "a task", Scope: "longterm", Type: "task"},
	}}

	lines := strings.Split(strings.TrimRight(formatChecklist(pc), "\n"), "\n")

	want := []string{
		"A longterm note: first para second para",
		"A longterm note: the gist",
		"A longterm task: a task",
	}
	if len(lines) != len(want) {
		t.Fatalf("%d lines, want %d:\n%s", len(lines), len(want), formatChecklist(pc))
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, lines[i], want[i])
		}
	}
}

// R45: however long the note, it adds at most 280 characters of its text to
// the suggestion and video prompts.
func TestPrompts_ALongNoteAddsAtMost280Characters(t *testing.T) {
	without := &PlanContext{Focus: "Learn Go"}
	with := &PlanContext{Focus: "Learn Go", ChecklistItems: []ChecklistItem{
		{Description: strings.Repeat("ж", 5000), Scope: "longterm", Type: "note"},
	}}
	for name, build := range map[string]func(*PlanContext) string{
		"suggestion": func(pc *PlanContext) string { return buildChecklistPrompt(pc, "") },
		"videos":     buildVideoSearchPrompt,
	} {
		grew := utf8.RuneCountInString(build(with)) - utf8.RuneCountInString(build(without))
		if grew > NotePromptCap+40 { // the note's text plus its short line label
			t.Errorf("%s prompt grew by %d characters for a 5000-character note", name, grew)
		}
		if strings.Count(build(with), "ж") > NotePromptCap {
			t.Errorf("%s prompt carries more than %d characters of the note", name, NotePromptCap)
		}
	}
}

// GetPlanContext's mapping carries type and summary beside the description.
func TestToChecklistItem_CarriesTypeAndSummary(t *testing.T) {
	gist := "gist"
	got := toChecklistItem(&planpb.ChecklistItem{Description: "body", Scope: "longterm", Type: "note", Summary: &gist})
	if got.Description != "body" || got.Scope != "longterm" || got.Type != "note" || got.Summary == nil || *got.Summary != "gist" {
		t.Fatalf("item = %+v", got)
	}
	if got := toChecklistItem(&planpb.ChecklistItem{Description: "t", Type: "task"}); got.Summary != nil {
		t.Fatalf("task summary = %v, want nil", *got.Summary)
	}
}
