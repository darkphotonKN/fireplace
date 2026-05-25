package calendargw

import "github.com/google/uuid"

// CalendarItem mirrors the proto shape; we keep a separate Go struct so the
// HTTP response field names ("startDate" / "dueDate") match the monolith's
// API exactly without leaking proto field naming.
type CalendarItem struct {
	ID          uuid.UUID `json:"id"`
	Description string    `json:"description"`
	Scope       string    `json:"scope"`
	Done        bool      `json:"done"`
	StartDate   string    `json:"startDate"`
	DueDate     string    `json:"dueDate"`
}

// GetCalendarResp is the JSON shape returned by GET /api/plans/:id/calendar.
type GetCalendarResp struct {
	PlanID      string         `json:"planId"`
	View        string         `json:"view"`
	WindowStart string         `json:"windowStart"`
	WindowEnd   string         `json:"windowEnd"`
	Items       []CalendarItem `json:"items"`
}
