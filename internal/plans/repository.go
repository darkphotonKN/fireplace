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
		id, user_id, name, description, focus, plan_type, created_at, updated_at
	FROM plans
	WHERE id = $id
	`
	var plan models.Plan

	err := r.db.Get(&plan, query, id)

	if err != nil {
		return nil, errorutils.AnalyzeDBErr(err)
	}

	return &plan, nil
}

