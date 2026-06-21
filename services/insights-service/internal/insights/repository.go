package insights

import (
	"context"

	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/jmoiron/sqlx"
)

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, param CreateInsightParam) error {
	query := `
	INSERT INTO generated_insights(user_id, plan_id, insight_type, content)
	VALUES ($1, $2, $3, $4)
	`
	// let unique constraint catch the error on duplicate
	_, err := r.db.ExecContext(ctx, query, param.UserID, param.PlanID, param.InsightType, param.Content)
	if err != nil {
		return commonhelpers.WrapDBErr("insights", "create", err)
	}

	return nil
}

// CreateTx is the transactional sibling of Create — stub for now.
func (r *repository) CreateTx(ctx context.Context, tx *sqlx.Tx, param CreateInsightParam) error {
	query := `
	INSERT INTO generated_insights(user_id, plan_id, insight_type, content)
	VALUES ($1, $2, $3, $4)
	`
	// let unique constraint catch the error on duplicate
	_, err := tx.ExecContext(ctx, query, param.UserID, param.PlanID, param.InsightType, param.Content)
	if err != nil {
		return commonhelpers.WrapDBErr("insights", "create", err)
	}

	return nil
}
