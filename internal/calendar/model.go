package calendar

import "github.com/google/uuid"

// CalendarItem is a checklist item rendered on the Plan Calendar.
// Dates are formatted as "YYYY-MM-DD"; an empty string means the column was null.
type CalendarItem struct {
	ID          uuid.UUID `json:"id"`
	Description string    `json:"description"`
	Scope       string    `json:"scope"`
	Done        bool      `json:"done"`
	StartDate   string    `json:"startDate"`
	DueDate     string    `json:"dueDate"`
}

// GetCalendarResp is the API response shape for GET /api/plans/:id/calendar.
type GetCalendarResp struct {
	PlanID      string         `json:"planId"`
	View        string         `json:"view"`
	WindowStart string         `json:"windowStart"`
	WindowEnd   string         `json:"windowEnd"`
	Items       []CalendarItem `json:"items"`
}
