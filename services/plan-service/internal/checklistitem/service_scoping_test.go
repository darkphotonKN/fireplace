package checklistitem

import (
	"context"
	"testing"
	"time"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
)

// scopedRepo holds rows the way the table does, each in its own plan, and
// answers every item-by-id call only for a row whose plan matches, like the
// real WHERE id AND plan_id. It records whether any write reached it, so a test
// can prove a refused request changed nothing.
type scopedRepo struct {
	Repository

	rows   map[uuid.UUID]*Item
	writes int
}

func newScopedRepo(items ...*Item) *scopedRepo {
	r := &scopedRepo{rows: map[uuid.UUID]*Item{}}
	for _, it := range items {
		r.rows[it.ID] = it
	}
	return r
}

func (r *scopedRepo) inPlan(planID, id uuid.UUID) (*Item, error) {
	it, ok := r.rows[id]
	if !ok || it.PlanID != planID {
		return nil, commonconstants.ErrNotFound
	}
	return it, nil
}

func (r *scopedRepo) GetInPlan(ctx context.Context, planID, id uuid.UUID) (*Item, error) {
	it, err := r.inPlan(planID, id)
	if err != nil {
		return nil, err
	}
	copied := *it
	return &copied, nil
}

func (r *scopedRepo) GetByID(ctx context.Context, id uuid.UUID) (*Item, error) {
	if it, ok := r.rows[id]; ok {
		copied := *it
		return &copied, nil
	}
	return nil, commonconstants.ErrNotFound
}

func (r *scopedRepo) HasChildren(ctx context.Context, id uuid.UUID) (bool, error) {
	for _, it := range r.rows {
		if it.ParentID != nil && *it.ParentID == id {
			return true, nil
		}
	}
	return false, nil
}

func (r *scopedRepo) Update(ctx context.Context, in UpdateItemInput) error {
	it, err := r.inPlan(in.PlanID, in.ID)
	if err != nil {
		return err
	}
	r.writes++
	if in.Description != nil {
		it.Description = *in.Description
	}
	if in.Done != nil {
		it.Done = *in.Done
	}
	if in.Archived != nil {
		it.Archived = *in.Archived
	}
	if in.Type != nil {
		it.Type = *in.Type
	}
	if in.SetIntention {
		it.Intention = in.Intention
	}
	return nil
}

func (r *scopedRepo) UpdateDates(ctx context.Context, planID, id uuid.UUID, start, due *time.Time) error {
	it, err := r.inPlan(planID, id)
	if err != nil {
		return err
	}
	r.writes++
	it.StartDate, it.DueDate = start, due
	return nil
}

func (r *scopedRepo) Delete(ctx context.Context, planID, id uuid.UUID) error {
	if _, err := r.inPlan(planID, id); err != nil {
		return err
	}
	r.writes++
	delete(r.rows, id)
	return nil
}

// twoPlans seeds one item in plan A and one in plan B.
func twoPlans() (planA, planB uuid.UUID, itemA, itemB *Item, repo *scopedRepo) {
	planA, planB = uuid.New(), uuid.New()
	itemA = &Item{ID: uuid.New(), PlanID: planA, Description: "a", Type: TypeTask, Scope: ScopeLongterm}
	itemB = &Item{ID: uuid.New(), PlanID: planB, Description: "b", Type: TypeTask, Scope: ScopeLongterm}
	repo = newScopedRepo(itemA, itemB)
	return
}

// R11: an item that exists in another plan is answered exactly like an item
// that does not exist, and nothing is written.
func TestUpdate_ItemFromAnotherPlan_IsNotFoundAndUnchanged(t *testing.T) {
	planA, _, _, itemB, repo := twoPlans()
	svc := newTestService(repo)

	desc, done := "hijacked", true
	cases := []struct {
		name string
		in   UpdateItemInput
	}{
		{"description and done", UpdateItemInput{ID: itemB.ID, PlanID: planA, Description: &desc, Done: &done}},
		{"to a note", UpdateItemInput{ID: itemB.ID, PlanID: planA, Type: strPtr(TypeNote)}},
		{"re-parent", UpdateItemInput{ID: itemB.ID, PlanID: planA, SetParent: true, ParentID: ptrUUID(uuid.New())}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Update(context.Background(), tc.in)
			if !isNotFound(err) {
				t.Fatalf("err = %v, want ErrNotFound", err)
			}
			if repo.writes != 0 {
				t.Errorf("a write reached the repository for another plan's item")
			}
			if repo.rows[itemB.ID].Description != "b" || repo.rows[itemB.ID].Done {
				t.Errorf("plan B's item changed: %+v", repo.rows[itemB.ID])
			}
		})
	}
}

func TestUpdate_ItemInItsOwnPlan_Updates(t *testing.T) {
	_, planB, _, itemB, repo := twoPlans()
	svc := newTestService(repo)

	desc := "renamed"
	got, err := svc.Update(context.Background(), UpdateItemInput{ID: itemB.ID, PlanID: planB, Description: &desc})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Description != "renamed" || got.PlanID != planB {
		t.Errorf("got %+v, want description renamed in plan B", got)
	}
}

func TestGetInPlan_ItemFromAnotherPlan_IsNotFound(t *testing.T) {
	planA, planB, _, itemB, repo := twoPlans()
	svc := newTestService(repo)

	if _, err := svc.GetInPlan(context.Background(), planA, itemB.ID); !isNotFound(err) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	got, err := svc.GetInPlan(context.Background(), planB, itemB.ID)
	if err != nil || got.ID != itemB.ID {
		t.Fatalf("own plan: got %v, %v; want item B", got, err)
	}
}

func TestUpdateDates_ItemFromAnotherPlan_IsNotFoundAndUnchanged(t *testing.T) {
	planA, planB, _, itemB, repo := twoPlans()
	svc := newTestService(repo)
	day := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)

	for _, in := range []UpdateDatesInput{
		{ID: itemB.ID, PlanID: planA, SetStart: true, StartDate: &day},
		{ID: itemB.ID, PlanID: planA}, // the read-only path, nothing to set
	} {
		if _, err := svc.UpdateDates(context.Background(), in); !isNotFound(err) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	}
	if repo.writes != 0 || repo.rows[itemB.ID].StartDate != nil {
		t.Errorf("plan B's item changed: %+v", repo.rows[itemB.ID])
	}

	got, err := svc.UpdateDates(context.Background(), UpdateDatesInput{ID: itemB.ID, PlanID: planB, SetStart: true, StartDate: &day})
	if err != nil || got.StartDate == nil || !got.StartDate.Equal(day) {
		t.Fatalf("own plan: got %v, %v; want start date set", got, err)
	}
}

func TestArchive_ItemFromAnotherPlan_IsNotFoundAndUnchanged(t *testing.T) {
	planA, planB, _, itemB, repo := twoPlans()
	svc := newTestService(repo)

	if _, err := svc.Archive(context.Background(), planA, itemB.ID, true); !isNotFound(err) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if repo.writes != 0 || repo.rows[itemB.ID].Archived {
		t.Errorf("plan B's item changed: %+v", repo.rows[itemB.ID])
	}

	got, err := svc.Archive(context.Background(), planB, itemB.ID, true)
	if err != nil || !got.Archived {
		t.Fatalf("own plan: got %v, %v; want archived", got, err)
	}
}

func TestDelete_ItemFromAnotherPlan_IsNotFoundAndKept(t *testing.T) {
	planA, planB, _, itemB, repo := twoPlans()
	svc := newTestService(repo)

	if err := svc.Delete(context.Background(), planA, itemB.ID); !isNotFound(err) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if _, ok := repo.rows[itemB.ID]; !ok {
		t.Fatal("plan B's item was deleted through plan A")
	}

	if err := svc.Delete(context.Background(), planB, itemB.ID); err != nil {
		t.Fatalf("own plan: %v", err)
	}
	if _, ok := repo.rows[itemB.ID]; ok {
		t.Error("own plan: item still there")
	}
}

func strPtr(s string) *string         { return &s }
func ptrUUID(id uuid.UUID) *uuid.UUID { return &id }
