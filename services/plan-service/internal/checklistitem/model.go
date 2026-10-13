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
	Intention   *string    `db:"intention" json:"intention,omitempty"`
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

	// MaxIntentionChars bounds an item's intention, counted in characters
	// (runes), not bytes. The column's CHECK holds the same limit.
	MaxIntentionChars = 280
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
//
// Intention is three-state the same way (FS-043WP R38):
//
//	SetIntention=false        → leave column alone
//	SetIntention=true, I=nil  → clear column
//	SetIntention=true, I=&s   → trim s; an empty result clears, over 280 characters is refused
//
// PlanID is the plan the caller addressed: the write touches the row only when
// it belongs to that plan (FS-12ZMF R11).
type UpdateItemInput struct {
	ID           uuid.UUID
	PlanID       uuid.UUID
	Description  *string
	Done         *bool
	Scope        *string
	Type         *string
	Archived     *bool
	SetParent    bool
	ParentID     *uuid.UUID
	SetIntention bool
	Intention    *string
}

// UpdateDatesInput allows setting / leaving start_date and due_date. The
// SetX bool distinguishes "leave alone" from "set to this value (possibly nil)".
// PlanID scopes the row the same way UpdateItemInput's does.
type UpdateDatesInput struct {
	ID        uuid.UUID
	PlanID    uuid.UUID
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

// SiblingSetInput identifies ONE sibling set: the top-level items of a
// (plan, scope) when ParentID is nil, or the children of that parent when it is
// not. It is the unit order is written for — see FS-0009 R2.1.
type SiblingSetInput struct {
	PlanID   uuid.UUID
	Scope    string
	ParentID *uuid.UUID
}

// ReorderInput carries the complete, final order of one sibling set. IDs must be
// exactly a permutation of the set's members; a partial order is refused rather
// than merged (FS-0009 R2.1).
type ReorderInput struct {
	PlanID   uuid.UUID
	Scope    string
	ParentID *uuid.UUID
	IDs      []uuid.UUID
}

func (in ReorderInput) siblingSet() SiblingSetInput {
	return SiblingSetInput{PlanID: in.PlanID, Scope: in.Scope, ParentID: in.ParentID}
}
