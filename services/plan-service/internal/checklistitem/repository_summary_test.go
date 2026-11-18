package checklistitem

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// FS-PGG2N R2, R3, R7, R9 against the real table: the column is nullable, the
// update is three-state, every read returns it, the CHECKs refuse a long
// summary and a summary on a task, and a duplicate copies it.

func seedNote(t *testing.T, db *sqlx.DB, planID, id uuid.UUID, seq int, summary *string) {
	t.Helper()
	seedItem(t, db, planID, id, seq, time.Now())
	if _, err := db.Exec(`UPDATE checklist_items SET type = 'note', summary = $2 WHERE id = $1`,
		id, summary); err != nil {
		t.Fatalf("make note %s: %v", id, err)
	}
}

func TestIntegrationSummary_NewItemReadsNull(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)
	id := uuid.New()
	seedItem(t, db, planID, id, 1, time.Now())

	got, err := NewRepository(db).GetInPlan(context.Background(), planID, id)
	if err != nil {
		t.Fatalf("GetInPlan: %v", err)
	}
	if got.Summary != nil {
		t.Errorf("summary = %q, want NULL", *got.Summary)
	}
}

func TestIntegrationSummary_UpdateSetsLeavesAndClears(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)
	id := uuid.New()
	seedNote(t, db, planID, id, 1, nil)
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

	gist := "the gist"
	if err := repo.Update(ctx, UpdateItemInput{ID: id, PlanID: planID, SetSummary: true, Summary: &gist}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if it := read(); it.Summary == nil || *it.Summary != gist {
		t.Fatalf("after set: summary = %v, want %q", it.Summary, gist)
	}

	body := "a longer body"
	if err := repo.Update(ctx, UpdateItemInput{ID: id, PlanID: planID, Description: &body}); err != nil {
		t.Fatalf("leave: %v", err)
	}
	if it := read(); it.Summary == nil || *it.Summary != gist || it.Description != body {
		t.Fatalf("after an update without summary: %+v, want summary kept", it)
	}

	if err := repo.Update(ctx, UpdateItemInput{ID: id, PlanID: planID, SetSummary: true}); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if it := read(); it.Summary != nil {
		t.Fatalf("after clear: summary = %q, want NULL", *it.Summary)
	}
}

// R6 in storage: the type and the cleared summary land in one UPDATE, which
// the note-only CHECK would refuse if they were two.
func TestIntegrationSummary_NoteToTaskWithClearInOneWrite(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)
	id := uuid.New()
	seedNote(t, db, planID, id, 1, strp("gist"))
	repo := NewRepository(db)

	task := TypeTask
	if err := repo.Update(context.Background(),
		UpdateItemInput{ID: id, PlanID: planID, Type: &task, SetSummary: true}); err != nil {
		t.Fatalf("make a task: %v", err)
	}
	got, err := repo.GetInPlan(context.Background(), planID, id)
	if err != nil {
		t.Fatalf("GetInPlan: %v", err)
	}
	if got.Type != TypeTask || got.Summary != nil {
		t.Errorf("got type %q summary %v, want a task with NULL summary", got.Type, got.Summary)
	}
}

func TestIntegrationSummary_EveryReadCarriesIt(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)
	live, archived := uuid.New(), uuid.New()
	soon := time.Now().UTC().Truncate(24 * time.Hour).Add(48 * time.Hour)
	seedDatedItem(t, db, planID, live, 1, time.Now(), soon)
	seedArchivedItem(t, db, planID, archived, 2, time.Now())
	if _, err := db.Exec(`UPDATE checklist_items SET type = 'note', summary = 'gist' WHERE plan_id = $1`, planID); err != nil {
		t.Fatalf("set summary: %v", err)
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
		"Duplicate": func() ([]*Item, error) { return repo.Duplicate(ctx, planID, live) },
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
				if it.Summary == nil || *it.Summary != "gist" {
					t.Errorf("%s: item %s summary = %v, want \"gist\"", name, it.ID, it.Summary)
				}
			}
		})
	}
}

// R7: a duplicated note carries its body and its summary, and so do the
// note children of a duplicated parent.
func TestIntegrationSummary_DuplicateCopiesIt(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)
	parent, kid := uuid.New(), uuid.New()
	seedItem(t, db, planID, parent, 1, time.Now())
	seedNote(t, db, planID, kid, 1, strp("kid gist"))
	if _, err := db.Exec(`UPDATE checklist_items SET parent_id = $2 WHERE id = $1`, kid, parent); err != nil {
		t.Fatalf("nest: %v", err)
	}

	got, err := NewRepository(db).Duplicate(context.Background(), planID, parent)
	if err != nil {
		t.Fatalf("Duplicate: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("returned %d items, want the copy and its child", len(got))
	}
	if got[0].Summary != nil {
		t.Errorf("task copy summary = %q, want NULL", *got[0].Summary)
	}
	if got[1].Summary == nil || *got[1].Summary != "kid gist" {
		t.Errorf("child copy summary = %v, want \"kid gist\"", got[1].Summary)
	}
}

// The CHECKs back the service's rules whatever path writes them.
func TestIntegrationSummary_ColumnChecks(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)
	note, task := uuid.New(), uuid.New()
	seedNote(t, db, planID, note, 1, nil)
	seedItem(t, db, planID, task, 2, time.Now())

	set := func(id uuid.UUID, v string) error {
		_, err := db.Exec(`UPDATE checklist_items SET summary = $2 WHERE id = $1`, id, v)
		return err
	}
	if err := set(note, strings.Repeat("é", 280)); err != nil {
		t.Errorf("280 characters refused: %v", err)
	}
	if err := set(note, strings.Repeat("a", 281)); err == nil {
		t.Errorf("281 characters accepted, want the length CHECK to refuse them")
	}
	if err := set(task, "gist"); err == nil {
		t.Errorf("a summary on a task accepted, want the note-only CHECK to refuse it")
	}
}
