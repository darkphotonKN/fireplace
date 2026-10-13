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

// Relation is a user's relation to one plan: exactly one of owner, member or
// none (ADR-0015 §1). A plan that does not exist is none.
type Relation string

const (
	RelationNone   Relation = "none"
	RelationOwner  Relation = "owner"
	RelationMember Relation = "member"
)

// Access is the answer to "what is this user to this plan". OwnerID is set
// whenever the plan exists, whatever the relation, and is uuid.Nil otherwise.
type Access struct {
	Relation Relation
	OwnerID  uuid.UUID
}

// AccessRow is what the database holds for one (plan, user) pair: who owns the
// plan, and whether a plan_shares row exists for the user. The share's role is
// not read: it does not change the relation in v1 (FS-12ZMF R3).
type AccessRow struct {
	OwnerID uuid.UUID `db:"owner_id"`
	Shared  bool      `db:"shared"`
}

// Member is one share on a plan: the user it names, and when it was written
// (FS-12ZMF R15). plan-service never holds a member's name or email; the
// gateway resolves those.
type Member struct {
	UserID uuid.UUID `db:"user_id"`
	Since  time.Time `db:"created_at"`
}

// Members is a plan's people as plan-service knows them: the owner, when the
// plan was made (the owner's "since"), and the members in the order they were
// added. The owner is never among Members.
type Members struct {
	OwnerID       uuid.UUID
	PlanCreatedAt time.Time
	Members       []*Member
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
