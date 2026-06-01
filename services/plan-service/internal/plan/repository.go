package plan

import (
	"context"
	"fmt"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *repository {
	return &repository{db: db}
}

// wrapDBErr is the repo boundary translation point: it converts infrastructure
// errors (sql.ErrNoRows, duplicate keys, constraint violations, transient
// failures) into domain sentinels via AnalyzeDBErr, and wraps anything else
// with the repo name + operation for context. It never logs and never decides
// transport status.
func wrapDBErr(op string, err error) error {
	if err == nil {
		return nil
	}
	if mapped := commonhelpers.AnalyzeDBErr(err); mapped != err {
		return mapped
	}
	return fmt.Errorf("plan repo: %s: %w", op, err)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Plan, error) {
	query := `
	SELECT id, user_id, name, description, focus, plan_type, daily_reset, created_at, updated_at
	FROM plans
	WHERE id = $1`
	var p Plan
	if err := r.db.GetContext(ctx, &p, query, id); err != nil {
		return nil, wrapDBErr("get plan by id "+id.String(), err)
	}
	return &p, nil
}

func (r *repository) Create(ctx context.Context, p *Plan) (*Plan, error) {
	query := `
	INSERT INTO plans (user_id, name, description, focus, plan_type, daily_reset)
	VALUES (:user_id, :name, :description, :focus, :plan_type, :daily_reset)
	RETURNING id, user_id, name, description, focus, plan_type, daily_reset, created_at, updated_at`

	rows, err := r.db.NamedQueryContext(ctx, query, p)
	if err != nil {
		return nil, wrapDBErr("create plan", err)
	}
	defer rows.Close()

	var created Plan
	if rows.Next() {
		if err := rows.StructScan(&created); err != nil {
			return nil, wrapDBErr("create plan: scan", err)
		}
	}
	return &created, nil
}

func (r *repository) Update(ctx context.Context, in *UpdatePlanInput) error {
	query := `
	UPDATE plans SET
		name        = COALESCE(:name, name),
		description = COALESCE(:description, description),
		focus       = COALESCE(:focus, focus),
		daily_reset = COALESCE(:daily_reset, daily_reset),
		updated_at  = NOW()
	WHERE id = :id AND user_id = :user_id`

	params := map[string]interface{}{
		"id":          in.ID,
		"user_id":     in.UserID,
		"name":        in.Name,
		"description": in.Description,
		"focus":       in.Focus,
		"daily_reset": in.DailyReset,
	}
	if _, err := r.db.NamedExecContext(ctx, query, params); err != nil {
		return wrapDBErr("update plan "+in.ID.String(), err)
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM plans WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return wrapDBErr("delete plan "+id.String(), err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return wrapDBErr("delete plan "+id.String()+": rows affected", err)
	}
	if rows == 0 {
		return commonconstants.ErrNotFound
	}
	return nil
}

func (r *repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*Plan, error) {
	query := `
	SELECT id, user_id, name, description, focus, plan_type, daily_reset, created_at, updated_at
	FROM plans
	WHERE user_id = $1
	ORDER BY created_at DESC`

	plans := []*Plan{}
	if err := r.db.SelectContext(ctx, &plans, query, userID); err != nil {
		return nil, wrapDBErr("list plans by user "+userID.String(), err)
	}
	return plans, nil
}

func (r *repository) ListShared(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Plan, error) {
	query := `
	SELECT id, user_id, name, description, focus, plan_type, daily_reset, created_at, updated_at
	FROM plans
	WHERE plans.user_id = $1

	UNION ALL

	SELECT plans.id, plans.user_id, name, description, focus, plan_type, daily_reset,
	       plans.created_at, plans.updated_at
	FROM plans
	JOIN plan_shares ON plan_shares.plan_id = plans.id
	WHERE plan_shares.user_id = $1

	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3`

	plans := []*Plan{}
	if err := r.db.SelectContext(ctx, &plans, query, userID, limit, offset); err != nil {
		return nil, wrapDBErr("list shared plans for user "+userID.String(), err)
	}
	return plans, nil
}

func (r *repository) Search(ctx context.Context, in SearchInput) ([]*SearchResult, error) {
	wildCard := "%" + in.Term + "%"
	query := `
	SELECT id, name, focus, description, plan_type, daily_reset, created_at, updated_at
	FROM plans
	WHERE name ILIKE $1 AND user_id = $2
	LIMIT $3 OFFSET $4`

	var out []*SearchResult
	if err := r.db.SelectContext(ctx, &out, query, wildCard, in.UserID, in.Limit, in.Offset); err != nil {
		return nil, wrapDBErr("search plans", err)
	}
	return out, nil
}

// CreateShare inserts a plan_shares row binding planID to userID. Idempotency
// is handled by the PK constraint on (user_id, plan_id).
func (r *repository) CreateShare(ctx context.Context, planID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO plan_shares (plan_id, user_id) VALUES ($1, $2)
		 ON CONFLICT (user_id, plan_id) DO NOTHING`, planID, userID)
	if err != nil {
		return wrapDBErr("create share for plan "+planID.String(), err)
	}
	return nil
}
