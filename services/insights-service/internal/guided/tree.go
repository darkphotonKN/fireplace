// Package guided is the shared machinery of FS-DF4ZB that the day plan
// (FS-02JBZ A2) generates with: the plan tree and its plan-state hash, the
// per-day key and its refresh counter, retrieval and the prompt's snippet
// blocks, the duplicate key of suggested text, and the usage ledger. The daily
// brief that introduced them is retired (FS-02JBZ R92); its tables stay.
//
// A parent item is an open, top-level, long-term task. The client calls it a
// milestone; nothing in this package does (ADR-0014, FS-DF4ZB R6). Prompt
// prose may.
package guided

import (
	"time"

	"github.com/google/uuid"
)

// Item type and scope values as plan-service reports them.
const (
	itemTypeTask  = "task"
	scopeLongterm = "longterm"
)

// PlanItem is one checklist item as plan-service reports it: the raw input a
// plan tree is built from.
type PlanItem struct {
	ID          uuid.UUID
	ParentID    *uuid.UUID
	Description string
	Type        string
	Scope       string
	Done        bool
	Archived    bool
	DueDate     *time.Time
	// UpdatedAt is the item's last update. The plan tree leaves it out (the
	// model never sees it); the day plan's rules draw reads it (FS-02JBZ R37).
	UpdatedAt time.Time
}

// TreeItem is one node of the plan tree: only what the model sees.
type TreeItem struct {
	ID          uuid.UUID
	ParentID    *uuid.UUID
	Description string
	Done        bool
	DueDate     *time.Time
}

// PlanTree is the plan context of guided suggestions (R1): every non-archived,
// long-term task, in plan-service's order. Notes, daily-scope and archived items
// are not in it.
type PlanTree struct {
	Items []TreeItem
}

// NewPlanTree builds the plan tree from plan-service's items.
func NewPlanTree(items []PlanItem) PlanTree {
	out := make([]TreeItem, 0, len(items))
	for _, it := range items {
		if it.Archived || it.Type != itemTypeTask || it.Scope != scopeLongterm {
			continue
		}
		out = append(out, TreeItem{
			ID:          it.ID,
			ParentID:    it.ParentID,
			Description: it.Description,
			Done:        it.Done,
			DueDate:     it.DueDate,
		})
	}
	return PlanTree{Items: out}
}

// ParentItems returns the open, top-level tasks: the only valid anchors.
func (t PlanTree) ParentItems() []TreeItem {
	var out []TreeItem
	for _, it := range t.Items {
		if it.ParentID == nil && !it.Done {
			out = append(out, it)
		}
	}
	return out
}
