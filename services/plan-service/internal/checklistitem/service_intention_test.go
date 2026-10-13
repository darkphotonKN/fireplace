package checklistitem

import (
	"context"
	"strings"
	"testing"
)

// FS-043WP R38: an item's intention is free text set only through Update. The
// service trims it, stores an empty result as NULL, and refuses more than 280
// characters (runes, not bytes).

func updateIntention(t *testing.T, value *string) (*Item, *scopedRepo, error) {
	t.Helper()
	_, planB, _, itemB, repo := twoPlans()
	svc := newTestService(repo)
	got, err := svc.Update(context.Background(), UpdateItemInput{
		ID: itemB.ID, PlanID: planB, SetIntention: true, Intention: value,
	})
	return got, repo, err
}

func TestUpdate_Intention_IsTrimmed(t *testing.T) {
	got, _, err := updateIntention(t, strPtr("  ship it  "))
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Intention == nil || *got.Intention != "ship it" {
		t.Errorf("intention = %v, want \"ship it\"", got.Intention)
	}
}

func TestUpdate_Intention_EmptyOrWhitespaceClearsToNull(t *testing.T) {
	for _, value := range []string{"", "   ", "\t\n "} {
		t.Run(value, func(t *testing.T) {
			_, planB, _, itemB, repo := twoPlans()
			repo.rows[itemB.ID].Intention = strPtr("old reason")
			svc := newTestService(repo)

			got, err := svc.Update(context.Background(), UpdateItemInput{
				ID: itemB.ID, PlanID: planB, SetIntention: true, Intention: strPtr(value),
			})
			if err != nil {
				t.Fatalf("Update: %v", err)
			}
			if got.Intention != nil {
				t.Errorf("intention = %q, want NULL", *got.Intention)
			}
		})
	}
}

func TestUpdate_Intention_UnsetLeavesItAlone(t *testing.T) {
	_, planB, _, itemB, repo := twoPlans()
	repo.rows[itemB.ID].Intention = strPtr("kept")
	svc := newTestService(repo)

	got, err := svc.Update(context.Background(), UpdateItemInput{
		ID: itemB.ID, PlanID: planB, Description: strPtr("renamed"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Intention == nil || *got.Intention != "kept" {
		t.Errorf("intention = %v, want \"kept\"", got.Intention)
	}
	if got.Description != "renamed" {
		t.Errorf("description = %q, want renamed", got.Description)
	}
}

func TestUpdate_Intention_LengthIsCountedInCharacters(t *testing.T) {
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
			got, repo, err := updateIntention(t, strPtr(tc.value))
			if tc.wantErr {
				if !isInvalidInput(err) {
					t.Fatalf("err = %v, want ErrInvalidInput", err)
				}
				if repo.writes != 0 {
					t.Errorf("a refused intention still reached the repository")
				}
				return
			}
			if err != nil {
				t.Fatalf("Update: %v", err)
			}
			if got.Intention == nil || *got.Intention != strings.TrimSpace(tc.value) {
				t.Errorf("intention not stored as the trimmed value")
			}
		})
	}
}

// R11 still holds: an intention aimed at another plan's item is not found, not
// a length error that would confirm the item exists.
func TestUpdate_Intention_ItemFromAnotherPlan_IsNotFound(t *testing.T) {
	planA, _, _, itemB, repo := twoPlans()
	svc := newTestService(repo)

	_, err := svc.Update(context.Background(), UpdateItemInput{
		ID: itemB.ID, PlanID: planA, SetIntention: true, Intention: strPtr(strings.Repeat("a", 281)),
	})
	if !isNotFound(err) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if repo.rows[itemB.ID].Intention != nil {
		t.Errorf("plan B's item gained an intention")
	}
}
