package checklistitem

import (
	"time"

	"github.com/google/uuid"
)

// Item mirrors the checklist_items table.
type Item struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	PlanID      uuid.UUID  `db:"plan_id" json:"plan_id"`
	Description string     `db:"description" json:"description"`
	Done        bool       `db:"done" json:"done"`
	Sequence    int        `db:"sequence" json:"sequence"`
	Type        string     `db:"type" json:"type"`
	ParentID    *uuid.UUID `db:"parent_id" json:"parent_id,omitempty"`
	StartDate   *time.Time `db:"start_date" json:"start_date,omitempty"`
	DueDate     *time.Time `db:"due_date" json:"due_date,omitempty"`
	Scope       string     `db:"scope" json:"scope"`
	Archived    bool       `db:"archived" json:"archived"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

// String constants ported from monolith internal/constants/checklistitems.go.
const (
	ScopeLongterm = "longterm"
	ScopeDaily    = "daily"
	UpcomingWeek  = "week"
	UpcomingMonth = "month"
	TypeTask      = "task"
	TypeNote      = "note"
)

// CreateItemInput is the service-layer input for creating a checklist item.
type CreateItemInput struct {
	PlanID      uuid.UUID
	Description string
	Scope       *string
	Type        *string
	ParentID    *uuid.UUID
}

// UpdateItemInput captures the proto request fields. Parent is three-state:
//
//	SetParent=false           → leave column alone
//	SetParent=true, P=nil     → clear column (outdent)
//	SetParent=true, P=&id     → re-parent (indent)
type UpdateItemInput struct {
	ID          uuid.UUID
	Description *string
	Done        *bool
	Scope       *string
	Type        *string
	Archived    *bool
	SetParent   bool
	ParentID    *uuid.UUID
}

// UpdateDatesInput allows setting / leaving start_date and due_date. The
// SetX bool distinguishes "leave alone" from "set to this value (possibly nil)".
type UpdateDatesInput struct {
	ID        uuid.UUID
	SetStart  bool
	StartDate *time.Time
	SetDue    bool
	DueDate   *time.Time
}

// ListItemsInput collects the filter knobs ListItems exposes.
type ListItemsInput struct {
	PlanID   uuid.UUID
	Scope    *string
	Type     *string
	Upcoming *string // "week" | "month" — filters by start_date window
}
