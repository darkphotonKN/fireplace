package checklistitem

import (
	"context"
	"strings"
	"testing"
)

// FS-PGG2N R2, R3, R6: a note's summary is set only through Update. The service
// trims it, stores an empty result as NULL, refuses more than 280 characters
// (runes, not bytes), and refuses a non-empty summary on anything that is or
// becomes a task. Converting a note to a task clears it in the same write.

// noteInPlanB turns plan B's item into a note carrying the given summary.
func noteInPlanB(summary *string) (note *Item, repo *scopedRepo) {
	_, _, _, itemB, repo := twoPlans()
	itemB.Type = TypeNote
	itemB.Summary = summary
	return itemB, repo
}

func TestUpdate_Summary_IsTrimmed(t *testing.T) {
	note, repo := noteInPlanB(nil)
	got, err := newTestService(repo).Update(context.Background(), UpdateItemInput{
		ID: note.ID, PlanID: note.PlanID, SetSummary: true, Summary: strPtr("  gist  "),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Summary == nil || *got.Summary != "gist" {
		t.Errorf("summary = %v, want \"gist\"", got.Summary)
	}
}

func TestUpdate_Summary_EmptyOrWhitespaceClearsToNull(t *testing.T) {
	for _, value := range []string{"", "   ", "\t\n "} {
		t.Run(value, func(t *testing.T) {
			note, repo := noteInPlanB(strPtr("old gist"))
			got, err := newTestService(repo).Update(context.Background(), UpdateItemInput{
				ID: note.ID, PlanID: note.PlanID, SetSummary: true, Summary: strPtr(value),
			})
			if err != nil {
				t.Fatalf("Update: %v", err)
			}
			if got.Summary != nil {
				t.Errorf("summary = %q, want NULL", *got.Summary)
			}
		})
	}
}

func TestUpdate_Summary_UnsetLeavesItAlone(t *testing.T) {
	note, repo := noteInPlanB(strPtr("kept"))
	got, err := newTestService(repo).Update(context.Background(), UpdateItemInput{
		ID: note.ID, PlanID: note.PlanID, Description: strPtr("a longer body"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Summary == nil || *got.Summary != "kept" {
		t.Errorf("summary = %v, want \"kept\"", got.Summary)
	}
	if got.Description != "a longer body" {
		t.Errorf("description = %q, want the new body", got.Description)
	}
}

func TestUpdate_Summary_LengthIsCountedInCharacters(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"280 multi-byte characters", strings.Repeat("é", 280), false},
		{"280 characters inside whitespace", "  " + strings.Repeat("a", 280) + "  ", false},
		{"281 characters", strings.Repeat("a", 281), true},
		{"281 multi-byte characters", strings.Repeat("界", 281), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			note, repo := noteInPlanB(nil)
			got, err := newTestService(repo).Update(context.Background(), UpdateItemInput{
				ID: note.ID, PlanID: note.PlanID, SetSummary: true, Summary: strPtr(tc.value),
			})
			if tc.wantErr {
				if !isInvalidInput(err) {
					t.Fatalf("err = %v, want ErrInvalidInput", err)
				}
				if repo.writes != 0 {
					t.Errorf("a refused summary still reached the repository")
				}
				return
			}
			if err != nil {
				t.Fatalf("Update: %v", err)
			}
			if got.Summary == nil || *got.Summary != strings.TrimSpace(tc.value) {
				t.Errorf("summary not stored as the trimmed value")
			}
		})
	}
}

// R3: only notes carry a summary.
func TestUpdate_Summary_NonEmptyOnATask_IsInvalidInput(t *testing.T) {
	cases := []struct {
		name string
		in   func(note, task *Item) UpdateItemInput
	}{
		{"on a task", func(_, task *Item) UpdateItemInput {
			return UpdateItemInput{ID: task.ID, PlanID: task.PlanID, SetSummary: true, Summary: strPtr("gist")}
		}},
		{"with type task on a note", func(note, _ *Item) UpdateItemInput {
			return UpdateItemInput{ID: note.ID, PlanID: note.PlanID, Type: strPtr(TypeTask),
				SetSummary: true, Summary: strPtr("gist")}
		}},
		{"with type task on a task", func(_, task *Item) UpdateItemInput {
			return UpdateItemInput{ID: task.ID, PlanID: task.PlanID, Type: strPtr(TypeTask),
				SetSummary: true, Summary: strPtr("gist")}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, task, note, repo := twoPlans()
			note.Type = TypeNote
			_, err := newTestService(repo).Update(context.Background(), tc.in(note, task))
			if !isInvalidInput(err) {
				t.Fatalf("err = %v, want ErrInvalidInput", err)
			}
			if repo.writes != 0 {
				t.Errorf("a refused summary still reached the repository")
			}
		})
	}
}

func TestUpdate_Summary_EmptyOnATask_SucceedsAndChangesNothing(t *testing.T) {
	_, _, task, _, repo := twoPlans()
	got, err := newTestService(repo).Update(context.Background(), UpdateItemInput{
		ID: task.ID, PlanID: task.PlanID, SetSummary: true, Summary: strPtr("   "),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Summary != nil || got.Type != TypeTask {
		t.Errorf("got %+v, want an unchanged task with no summary", got)
	}
}

func TestUpdate_Summary_TaskBecomingANoteMayCarryOne(t *testing.T) {
	_, _, task, _, repo := twoPlans()
	got, err := newTestService(repo).Update(context.Background(), UpdateItemInput{
		ID: task.ID, PlanID: task.PlanID, Type: strPtr(TypeNote), SetSummary: true, Summary: strPtr("gist"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Type != TypeNote || got.Summary == nil || *got.Summary != "gist" {
		t.Errorf("got type %q summary %v, want a note with \"gist\"", got.Type, got.Summary)
	}
}

// R6: Make a task clears the summary in the same write.
func TestUpdate_Summary_NoteToTaskClearsIt(t *testing.T) {
	note, repo := noteInPlanB(strPtr("gist"))
	got, err := newTestService(repo).Update(context.Background(), UpdateItemInput{
		ID: note.ID, PlanID: note.PlanID, Type: strPtr(TypeTask),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Type != TypeTask || got.Summary != nil {
		t.Errorf("got type %q summary %v, want a task with no summary", got.Type, got.Summary)
	}
	if repo.writes != 1 {
		t.Errorf("writes = %d, want the type and the summary in one write", repo.writes)
	}
}
