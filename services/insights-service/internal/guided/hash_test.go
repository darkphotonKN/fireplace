package guided

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPlanTreeStateHash_OrderOfRowsDoesNotMatterButEveryFieldDoes(t *testing.T) {
	parent, step, other := uuid.New(), uuid.New(), uuid.New()
	due := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	base := func() []TreeItem {
		return []TreeItem{
			{ID: parent, Description: "Ship the API", DueDate: ptr(due)},
			{ID: step, ParentID: ptr(parent), Description: "Write handler"},
			{ID: other, Description: "Write docs", Done: true},
		}
	}
	want := PlanTree{Items: base()}.StateHash()
	if want == "" {
		t.Fatal("empty hash")
	}

	reordered := base()
	reordered[0], reordered[2] = reordered[2], reordered[0]
	if got := (PlanTree{Items: reordered}).StateHash(); got != want {
		t.Fatalf("reordered rows hash %s, want %s", got, want)
	}

	changes := map[string]func([]TreeItem) []TreeItem{
		"id":              func(it []TreeItem) []TreeItem { it[2].ID = uuid.New(); return it },
		"parent id set":   func(it []TreeItem) []TreeItem { it[2].ParentID = ptr(parent); return it },
		"parent id moved": func(it []TreeItem) []TreeItem { it[1].ParentID = ptr(other); return it },
		"parent id unset": func(it []TreeItem) []TreeItem { it[1].ParentID = nil; return it },
		"description":     func(it []TreeItem) []TreeItem { it[1].Description = "Write the handler"; return it },
		"done":            func(it []TreeItem) []TreeItem { it[1].Done = true; return it },
		"due set":         func(it []TreeItem) []TreeItem { it[1].DueDate = ptr(due); return it },
		"due moved":       func(it []TreeItem) []TreeItem { it[0].DueDate = ptr(due.AddDate(0, 0, 1)); return it },
		"due unset":       func(it []TreeItem) []TreeItem { it[0].DueDate = nil; return it },
		"row added":       func(it []TreeItem) []TreeItem { return append(it, TreeItem{ID: uuid.New(), Description: "new"}) },
		"row removed":     func(it []TreeItem) []TreeItem { return it[:2] },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			if got := (PlanTree{Items: change(base())}).StateHash(); got == want {
				t.Fatalf("changing %s left the hash unchanged", name)
			}
		})
	}
}

func TestPlanTreeStateHash_EmptyTreeIsStable(t *testing.T) {
	if (PlanTree{}).StateHash() != (PlanTree{Items: []TreeItem{}}).StateHash() {
		t.Fatal("nil and empty trees hash differently")
	}
}
