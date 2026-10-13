package checklistitem

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// FS-043WP R37/R38 against the real table: the column is nullable, the update
// is three-state (a COALESCE could never clear it), and every read returns it.

func TestIntegrationIntention_NewItemReadsNull(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)
	id := uuid.New()
	seedItem(t, db, planID, id, 1, time.Now())

	got, err := NewRepository(db).GetInPlan(context.Background(), planID, id)
	if err != nil {
		t.Fatalf("GetInPlan: %v", err)
	}
	if got.Intention != nil {
		t.Errorf("intention = %q, want NULL", *got.Intention)
	}
}

func TestIntegrationIntention_UpdateSetsLeavesAndClears(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)
	id := uuid.New()
	seedItem(t, db, planID, id, 1, time.Now())
	repo := NewRepository(db)
	ctx := context.Background()

	read := func() *Item {
		t.Helper()
		it, err := repo.GetInPlan(ctx, planID, id)
		if err != nil {
			t.Fatalf("GetInPlan: %v", err)
		}
		return it
	}

	reason := "ship it"
	if err := repo.Update(ctx, UpdateItemInput{ID: id, PlanID: planID, SetIntention: true, Intention: &reason}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if it := read(); it.Intention == nil || *it.Intention != reason {
		t.Fatalf("after set: intention = %v, want %q", it.Intention, reason)
	}

	desc := "renamed"
	if err := repo.Update(ctx, UpdateItemInput{ID: id, PlanID: planID, Description: &desc}); err != nil {
		t.Fatalf("leave: %v", err)
	}
	if it := read(); it.Intention == nil || *it.Intention != reason || it.Description != desc {
		t.Fatalf("after an update without intention: %+v, want intention kept", it)
	}

	if err := repo.Update(ctx, UpdateItemInput{ID: id, PlanID: planID, SetIntention: true}); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if it := read(); it.Intention != nil {
		t.Fatalf("after clear: intention = %q, want NULL", *it.Intention)
	}
}

func TestIntegrationIntention_EveryReadCarriesIt(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)
	live, archived := uuid.New(), uuid.New()
	soon := time.Now().UTC().Truncate(24 * time.Hour).Add(48 * time.Hour)
	seedDatedItem(t, db, planID, live, 1, time.Now(), soon)
	seedArchivedItem(t, db, planID, archived, 2, time.Now())
	if _, err := db.Exec(`UPDATE checklist_items SET intention = 'why' WHERE plan_id = $1`, planID); err != nil {
		t.Fatalf("set intention: %v", err)
	}
	repo := NewRepository(db)
	ctx := context.Background()
	week := UpcomingWeek

	reads := map[string]func() ([]*Item, error){
		"GetByID": func() ([]*Item, error) { it, err := repo.GetByID(ctx, live); return []*Item{it}, err },
		"GetInPlan": func() ([]*Item, error) {
			it, err := repo.GetInPlan(ctx, planID, live)
			return []*Item{it}, err
		},
		"ListByPlanID": func() ([]*Item, error) { return repo.ListByPlanID(ctx, ListItemsInput{PlanID: planID}) },
		"ListByPlanID upcoming": func() ([]*Item, error) {
			return repo.ListByPlanID(ctx, ListItemsInput{PlanID: planID, Upcoming: &week})
		},
		"ListArchivedByPlanID": func() ([]*Item, error) { return repo.ListArchivedByPlanID(ctx, planID, nil) },
		"ListSiblings": func() ([]*Item, error) {
			return repo.ListSiblings(ctx, SiblingSetInput{PlanID: planID, Scope: ScopeLongterm})
		},
		"Reorder": func() ([]*Item, error) { return repo.Reorder(ctx, []uuid.UUID{live}) },
		"ListInDateWindow": func() ([]*Item, error) {
			return repo.ListInDateWindow(ctx, planID, soon.Add(-time.Hour), soon.Add(time.Hour))
		},
	}
	for name, read := range reads {
		t.Run(name, func(t *testing.T) {
			items, err := read()
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if len(items) == 0 {
				t.Fatalf("%s returned no items", name)
			}
			for _, it := range items {
				if it.Intention == nil || *it.Intention != "why" {
					t.Errorf("%s: item %s intention = %v, want \"why\"", name, it.ID, it.Intention)
				}
			}
		})
	}
}

// The column's CHECK backs the service's rule: 280 multi-byte characters fit,
// 281 do not, whatever path writes them.
func TestIntegrationIntention_ColumnHoldsAtMost280Characters(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)
	id := uuid.New()
	seedItem(t, db, planID, id, 1, time.Now())

	if _, err := db.Exec(`UPDATE checklist_items SET intention = $2 WHERE id = $1`, id, strings.Repeat("é", 280)); err != nil {
		t.Errorf("280 characters refused: %v", err)
	}
	if _, err := db.Exec(`UPDATE checklist_items SET intention = $2 WHERE id = $1`, id, strings.Repeat("a", 281)); err == nil {
		t.Errorf("281 characters accepted, want the CHECK to refuse them")
	}
}
