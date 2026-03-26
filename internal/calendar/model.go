package calendar

import (
	"time"

	"github.com/google/uuid"
)

// CalendarEntry represents a single scheduled slot on the calendar.
type CalendarEntry struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	PlanID          uuid.UUID  `db:"plan_id" json:"planId"`
	ChecklistItemID *uuid.UUID `db:"checklist_item_id" json:"checklistItemId"`
	EntryType       string     `db:"entry_type" json:"entryType"`
	ScheduledDate   time.Time  `db:"scheduled_date" json:"scheduledDate"`
	Position        int        `db:"position" json:"position"`
	Pinned          bool       `db:"pinned" json:"pinned"`
	RecTitle        *string    `db:"rec_title" json:"recTitle,omitempty"`
	RecURL          *string    `db:"rec_url" json:"recUrl,omitempty"`
	RecDescription  *string    `db:"rec_description" json:"recDescription,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updatedAt"`

	// Joined fields from checklist_items (not stored in calendar_entries)
	Description *string `db:"description" json:"description,omitempty"`
	Done        *bool   `db:"done" json:"done,omitempty"`
}

// DaySlot represents all entries for a single day.
type DaySlot struct {
	Date    string          `json:"date"`
	Entries []CalendarEntry `json:"entries"`
}

// MonthResponse is the API response for a calendar month.
type MonthResponse struct {
	PlanID string    `json:"planId"`
	Month  string    `json:"month"`
	Days   []DaySlot `json:"days"`
}

// MoveEntryReq is the request body for moving an entry to a new date/position.
type MoveEntryReq struct {
	TargetDate     string `json:"targetDate" binding:"required"`
	TargetPosition int    `json:"targetPosition" binding:"required,min=1,max=8"`
}

// ReorderReq is the request body for changing an entry's position within the same day.
type ReorderReq struct {
	TargetPosition int `json:"targetPosition" binding:"required,min=1,max=8"`
}
