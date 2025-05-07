package constants

// Scopes of checklist items
type ChecklistItemScope string

const (
	ScopeLongterm ChecklistItemScope = "longterm"
	ScopeDaily    ChecklistItemScope = "daily"
)

type UpdateStatus string

const (
	UpdateStatusFailure UpdateStatus = "failure"
	UpdateStatusSuccess UpdateStatus = "success"
)
