package guided

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func ptr[T any](v T) *T { return &v }

func TestNewPlanTree_KeepsOnlyLongTermNonArchivedTasks(t *testing.T) {
	parent := uuid.New()
	due := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	items := []PlanItem{
		{ID: parent, Description: "Ship the API", Type: "task", Scope: "longterm", DueDate: &due},
		{ID: uuid.New(), ParentID: &parent, Description: "Write handler", Type: "task", Scope: "longterm", Done: true},
		{ID: uuid.New(), Description: "a note", Type: "note", Scope: "longterm"},
		{ID: uuid.New(), Description: "daily task", Type: "task", Scope: "daily"},
		{ID: uuid.New(), Description: "archived task", Type: "task", Scope: "longterm", Archived: true},
	}

	tree := NewPlanTree(items)

	if len(tree.Items) != 2 {
		t.Fatalf("tree has %d items, want 2: %+v", len(tree.Items), tree.Items)
	}
	top := tree.Items[0]
	if top.ID != parent || top.ParentID != nil || top.Done || top.DueDate == nil || !top.DueDate.Equal(due) {
		t.Fatalf("top-level item = %+v, want id/no parent/open/due carried", top)
	}
	child := tree.Items[1]
	if child.ParentID == nil || *child.ParentID != parent || !child.Done {
		t.Fatalf("child = %+v, want parent id and done carried", child)
	}
}

func TestPlanTree_ParentItemsAreOpenTopLevelTasks(t *testing.T) {
	open, closed := uuid.New(), uuid.New()
	tree := NewPlanTree([]PlanItem{
		{ID: open, Description: "open", Type: "task", Scope: "longterm"},
		{ID: closed, Description: "closed", Type: "task", Scope: "longterm", Done: true},
		{ID: uuid.New(), ParentID: &open, Description: "step", Type: "task", Scope: "longterm"},
	})

	parents := tree.ParentItems()
	if len(parents) != 1 || parents[0].ID != open {
		t.Fatalf("parent items = %+v, want only the open top-level task", parents)
	}
}
