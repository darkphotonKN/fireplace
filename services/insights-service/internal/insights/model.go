package insights

import (
	"github.com/google/uuid"
)

// Video is the formatted view of a recommended learning video. Mirrors the
// shape the frontend already consumes (and the insights.Video proto message).
type Video struct {
	Title       string
	URL         string
	Source      string
	Type        string
	Description string
}

// PlanContext is the plan-side data insights needs to build an LLM prompt: the
// plan's focus plus a flattened view of its checklist items. Fetched from
// plan-service over gRPC (insights owns none of this data itself).
type PlanContext struct {
	PlanID         uuid.UUID
	Focus          string
	ChecklistItems []ChecklistItem
}

// ChecklistItem is the slim projection of a plan-service checklist item used
// purely as prompt context.
type ChecklistItem struct {
	Description string
	Scope       string
}

// create from req
type PlanCreatedParam struct {
	PlanID  uuid.UUID
	UserID  uuid.UUID
	EventID uuid.UUID
}

// create param for interfacing with the repo
type CreateInsightParam struct {
	PlanID      uuid.UUID `db:"plan_id"`
	UserID      uuid.UUID `db:"user_id"`
	InsightType string    `db:"insight_type"`
	Content     string    `db:"content"`
}
