package plans

import (
	"context"

	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/constants"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/models"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/utils/errorutils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *repository {
	return &repository{db: db}
}

func (r *repository) GetById(ctx context.Context, id uuid.UUID) (*models.Plan, error) {
	query := `
	SELECT 
		id, 
		user_id, 
		name, 
		description, 
		focus, 
		plan_type, 
		daily_reset,
		created_at, 
		updated_at
	FROM plans
	WHERE id = $1
	`
	var plan models.Plan

	err := r.db.GetContext(ctx, &plan, query, id)

	if err != nil {
		return nil, errorutils.AnalyzeDBErr(err)
	}

	return &plan, nil
}

func (r *repository) Create(ctx context.Context, plan models.Plan) (*models.Plan, error) {
	query := `
	INSERT INTO plans (
		user_id, 
		name, 
		description,
		focus,
		plan_type,
		daily_reset
	) VALUES (
		:user_id, 
		:name, 
		:description,
		:focus,
		:plan_type,
		:daily_reset
	) RETURNING 
		id, 
		user_id, 
		name, 
		description, 
		focus, 
		plan_type, 
		daily_reset,
		created_at, 
		updated_at
	`

	rows, err := r.db.NamedQueryContext(ctx, query, plan)
	if err != nil {
		return nil, errorutils.AnalyzeDBErr(err)
	}
	defer rows.Close()

	// Get the created plan with full details
	var createdPlan models.Plan
	if rows.Next() {
		if err := rows.StructScan(&createdPlan); err != nil {
			return nil, errorutils.AnalyzeDBErr(err)
		}
	}

	return &createdPlan, nil
}

func (r *repository) Update(ctx context.Context, id uuid.UUID, req UpdatePlanReq, userID uuid.UUID) error {
	query := `
	UPDATE plans SET 
		name = COALESCE(:name, name), 
		description = COALESCE(:description, description),
		focus = COALESCE(:focus, focus),
		daily_reset = COALESCE(:daily_reset, daily_reset)
	WHERE id = :id AND user_id = :user_id
	`

	// Map for named parameters
	params := map[string]interface{}{
		"id":          id,
		"name":        req.Name,
		"description": req.Description,
		"focus":       req.Focus,
		"daily_reset": req.DailyReset,
		"user_id":     userID,
	}

	_, err := r.db.NamedExecContext(ctx, query, params)
	if err != nil {
		return errorutils.AnalyzeDBErr(err)
	}

	return nil
}

// GetAll returns all plans from the database for a specific user
func (r *repository) GetAll(ctx context.Context, userID uuid.UUID) ([]*models.Plan, error) {
	query := `
	SELECT
		id,
		user_id,
		name,
		description,
		focus,
		plan_type,
		daily_reset,
		created_at,
		updated_at
	FROM plans
	WHERE user_id = $1
	ORDER BY created_at DESC
	`

	plans := []*models.Plan{}
	err := r.db.SelectContext(ctx, &plans, query, userID)

	if err != nil {
		return nil, errorutils.AnalyzeDBErr(err)
	}

	return plans, nil
}

// Share a single plan under a user to another user
func (r *repository) CreateSharedPlans(ctx context.Context, planID uuid.UUID, userID uuid.UUID) error {
	query := `
	INSERT INTO plan_shares (plan_id, user_id)
	VALUES (:plan_id, :user_id)
	`

	insertParams := struct {
		PlanID uuid.UUID `json:"plan_id" db:"plan_id"`
		UserID uuid.UUID `json:"user_id" db:"user_id"`
	}{
		PlanID: planID,
		UserID: userID,
	}

	_, err := r.db.NamedExecContext(ctx, query, insertParams)

	if err != nil {
		return errorutils.AnalyzeDBErr(err)
	}

	return nil
}

// Share a single plan under a user to another user
func (r *repository) UpdateSharedPlans(ctx context.Context, planID uuid.UUID, userID uuid.UUID) error {
	query := `
	UPDATE plans_shares SET 
		user_id = $2, 
	WHERE plans.id = $1
	`

	_, err := r.db.ExecContext(ctx, query, planID, userID)

	if err != nil {
		return errorutils.AnalyzeDBErr(err)
	}

	return nil
}

// Get all but including plans shared to user
func (r *repository) GetAllShared(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Plan, error) {

	query := `
	SELECT
		id,
		user_id,
		name,
		description,
		focus,
		plan_type,
		daily_reset,
		created_at,
		updated_at
	FROM plans
	WHERE plans.user_id = $1
	
	UNION ALL

	SELECT
		plans.id,
		plans.user_id,
		name, 
		description,
		focus,
		plan_type,
		daily_reset,
		plans.created_at,
		plans.updated_at
	FROM plans
	JOIN plan_shares
	ON plan_shares.plan_id = plans.id
	WHERE plan_shares.user_id = $1

	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3
	`

	plans := []*models.Plan{}
	err := r.db.SelectContext(ctx, &plans, query, userID, limit, offset)

	if err != nil {
		return nil, errorutils.AnalyzeDBErr(err)
	}

	return plans, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `
	DELETE FROM plans
	WHERE id = $1 AND user_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return errorutils.AnalyzeDBErr(err)
	}

	// Check if any rows were affected (plan exists and belongs to user)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errorutils.AnalyzeDBErr(err)
	}

	if rowsAffected == 0 {
		return constants.ErrNotFound
	}

	return nil
}

func (r *repository) SearchPlan(ctx context.Context, userID uuid.UUID, params SearchParam) ([]*SearchPlanRes, error) {

	var plans []*SearchPlanRes

	wildCard := "%" + params.Term + "%"

	query := `
	SELECT
		id,
		name,
		focus,
		description,
		plan_type,
		daily_reset,
		created_at,
		updated_at
	FROM plans
	WHERE name ILIKE $1
	AND user_id = $2
	LIMIT $3
	OFFSET $4
	`

	err := r.db.SelectContext(ctx, &plans, query, wildCard, userID, params.Limit, params.Offset)

	if err != nil {
		return nil, errorutils.AnalyzeDBErr(err)
	}

	return plans, nil
}
