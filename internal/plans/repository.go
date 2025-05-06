package plans

import (
	"context"

	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/darkphotonKN/fireplace/internal/utils/errorutils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
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
		plan_type
	) VALUES (
		:user_id, 
		:name, 
		:description,
		:focus,
		:plan_type
	) RETURNING 
		id, 
		user_id, 
		name, 
		description, 
		focus, 
		plan_type, 
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
		name = :name, 
		description = :description,
		focus = :focus
	WHERE id = :id AND user_id = :user_id
	`

	// Map for named parameters
	params := map[string]interface{}{
		"id":          id,
		"name":        req.Name,
		"description": req.Description,
		"focus":       req.Focus,
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
