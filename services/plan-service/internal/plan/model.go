package plan

import (
	"time"

	"github.com/google/uuid"
)

// Plan mirrors the plans table.
type Plan struct {
	ID          uuid.UUID `db:"id" json:"id"`
	UserID      uuid.UUID `db:"user_id" json:"user_id"`
	Name        string    `db:"name" json:"name"`
	Focus       string    `db:"focus" json:"focus"`
	Description string    `db:"description" json:"description"`
	PlanType    string    `db:"plan_type" json:"plan_type"`
	DailyReset  bool      `db:"daily_reset" json:"daily_reset"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// SearchResult is the projection returned by SearchPlan — same shape as Plan
// minus the user_id (search is already user-scoped).
type SearchResult struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `db:"name"`
	Focus       string    `db:"focus"`
	Description string    `db:"description"`
	PlanType    string    `db:"plan_type"`
	DailyReset  bool      `db:"daily_reset"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// Service-layer inputs (proto request → these, then service → repo).

type CreatePlanInput struct {
	UserID      uuid.UUID
	Name        string
	Focus       string
	Description string
	PlanType    string
	DailyReset  *bool // explicit nil means "service picks the default by plan type"
}

type UpdatePlanInput struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Name        *string
	Focus       *string
	Description *string
	PlanType    *string
	DailyReset  *bool
}

type SearchInput struct {
	UserID uuid.UUID
	Term   string
	Limit  int
	Offset int
}

// Plan type constants — ported from monolith internal/constants/plans.go so
// the default-daily-reset rule stays put.
const (
	PlanTypeProject  = "project"
	PlanTypeLearning = "learning"
)
