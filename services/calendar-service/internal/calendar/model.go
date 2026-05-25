package calendar

import "github.com/google/uuid"

// CalendarItem is the formatted view of a checklist item rendered on the
// Plan Calendar. start_date / due_date are "YYYY-MM-DD" or "" when null.
type CalendarItem struct {
	ID          uuid.UUID
	Description string
	Scope       string
	Done        bool
	StartDate   string
	DueDate     string
}

// GetCalendarOutput is the service-layer response (handler converts to proto).
type GetCalendarOutput struct {
	PlanID      uuid.UUID
	View        string
	WindowStart string // "YYYY-MM-DD"
	WindowEnd   string // "YYYY-MM-DD"
	Items       []CalendarItem
}
