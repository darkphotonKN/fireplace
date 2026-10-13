package checklistitem

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// FS-12ZMF R11 against the real table: an item-by-id write addressed under
// another plan touches nothing. A fake can prove the service passes the plan
// id along; only the database can prove the WHERE clause uses it.

type itemRow struct {
	Description string     `db:"description"`
	Archived    bool       `db:"archived"`
	StartDate   *time.Time `db:"start_date"`
}

func readRow(t *testing.T, db *sqlx.DB, id uuid.UUID) (itemRow, bool) {
	t.Helper()
	var rows []itemRow
	if err := db.Select(&rows, `SELECT description, archived, start_date FROM checklist_items WHERE id = $1`, id); err != nil {
		t.Fatalf("read item: %v", err)
	}
	if len(rows) == 0 {
		return itemRow{}, false
	}
	return rows[0], true
}

func TestIntegrationItemByID_AnotherPlansItem_IsNotFoundAndUnchanged(t *testing.T) {
	db := openTestDB(t)
	planA, planB := seedPlan(t, db), seedPlan(t, db)
	itemB := uuid.New()
	seedItem(t, db, planB, itemB, 1, time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC))
	before, _ := readRow(t, db, itemB)

	repo := NewRepository(db)
	svc := NewService(repo, nil)
	ctx := context.Background()
	desc := "written through plan A"
	archived := true
	day := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)

	calls := map[string]func() error{
		// The service reads the item in its plan before writing, so these two
		// go straight at the repository to prove the write is scoped as well.
		"repository update": func() error {
			return repo.Update(ctx, UpdateItemInput{ID: itemB, PlanID: planA, Description: &desc, Archived: &archived})
		},
		"repository dates": func() error { return repo.UpdateDates(ctx, planA, itemB, &day, &day) },
		"get":              func() error { _, err := svc.GetInPlan(ctx, planA, itemB); return err },
		"update": func() error {
			_, err := svc.Update(ctx, UpdateItemInput{ID: itemB, PlanID: planA, Description: &desc})
			return err
		},
		"dates": func() error {
			_, err := svc.UpdateDates(ctx, UpdateDatesInput{ID: itemB, PlanID: planA, SetStart: true, StartDate: &day})
			return err
		},
		"archive": func() error { _, err := svc.Archive(ctx, planA, itemB, true); return err },
		"delete":  func() error { return svc.Delete(ctx, planA, itemB) },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			if err := call(); !isNotFound(err) {
				t.Fatalf("err = %v, want ErrNotFound", err)
			}
			after, ok := readRow(t, db, itemB)
			if !ok {
				t.Fatal("plan B's item is gone")
			}
			if after.Description != before.Description || after.Archived || after.StartDate != nil {
				t.Errorf("plan B's item changed: %+v, was %+v", after, before)
			}
		})
	}
}

func TestIntegrationItemByID_OwnPlan_Works(t *testing.T) {
	db := openTestDB(t)
	planB := seedPlan(t, db)
	itemB := uuid.New()
	seedItem(t, db, planB, itemB, 1, time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC))

	svc := NewService(NewRepository(db), nil)
	ctx := context.Background()
	desc := "renamed"
	day := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)

	if _, err := svc.GetInPlan(ctx, planB, itemB); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got, err := svc.Update(ctx, UpdateItemInput{ID: itemB, PlanID: planB, Description: &desc}); err != nil || got.Description != desc {
		t.Fatalf("update: %v, %v", got, err)
	}
	if got, err := svc.UpdateDates(ctx, UpdateDatesInput{ID: itemB, PlanID: planB, SetStart: true, StartDate: &day}); err != nil || got.StartDate == nil {
		t.Fatalf("dates: %v, %v", got, err)
	}
	if got, err := svc.Archive(ctx, planB, itemB, true); err != nil || !got.Archived {
		t.Fatalf("archive: %v, %v", got, err)
	}
	if err := svc.Delete(ctx, planB, itemB); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := readRow(t, db, itemB); ok {
		t.Error("delete: item still there")
	}
}
