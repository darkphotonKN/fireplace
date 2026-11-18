// Package itemtype names a checklist item's type, the values of the
// checklist_items.type column. It imports nothing, so checklistitem and
// notecomment can both depend on it without depending on each other.
package itemtype

const (
	Task = "task"
	Note = "note"
)
