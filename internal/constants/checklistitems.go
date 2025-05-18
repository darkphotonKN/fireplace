package constants

// Scopes of checklist items
type ChecklistItemScope string

const (
	ScopeLongterm ChecklistItemScope = "longterm"
	ScopeDaily    ChecklistItemScope = "daily"
)

type ChecklistUpcoming string

var (
	UpcomingToday ChecklistUpcoming = "today"
	UpcomingWeek  ChecklistUpcoming = "week"
	UpcomingMonth ChecklistUpcoming = "month"
	UpcomingYear  ChecklistUpcoming = "year"
)
