package guided

import (
	"context"
	"time"

	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/ai"
	"github.com/jmoiron/sqlx"
)

// Repository is the usage ledger (generations) in the insights database: the
// ai.Ledger every metered generator records into and the token budget reads.
// The retired daily brief's tables (daily_briefs, suggestions) stay in the
// schema but nothing here reads or writes them (FS-02JBZ R92).
type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// RecordAttempt appends one model call's usage to the ledger (ai.Ledger).
func (r *Repository) RecordAttempt(ctx context.Context, a ai.Attempt) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO generations (kind, plan_id, user_id, model, input_tokens, cached_input_tokens, output_tokens, succeeded)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		string(a.Kind), a.PlanID, a.UserID, a.Model,
		a.Usage.InputTokens, a.Usage.CachedInputTokens, a.Usage.OutputTokens, a.Succeeded)
	if err != nil {
		return commonhelpers.WrapDBErr("generations", "insert", err)
	}
	return nil
}

// TokensSince sums input plus output tokens over every ledger row recorded at
// or after since, across all kinds, plans and users (ai.Ledger; FS-02JBZ R32).
// Cached input is part of input_tokens, so it is not added again.
func (r *Repository) TokensSince(ctx context.Context, since time.Time) (int64, error) {
	var sum int64
	if err := r.db.GetContext(ctx, &sum, `
		SELECT COALESCE(SUM(input_tokens::bigint + output_tokens::bigint), 0)
		FROM generations WHERE created_at >= $1`, since); err != nil {
		return 0, commonhelpers.WrapDBErr("generations", "sum tokens", err)
	}
	return sum, nil
}
