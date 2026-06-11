package outbox

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

func (r *repository) CreateTx(ctx context.Context, tx *sqlx.Tx, params CreateOutboxParams) error {
	query := `
	INSERT INTO outbox (event_id, routing_key, exchange, payload)
	VALUES (:event_id, :routing_key, :exchange, :payload)
	`

	_, err := tx.NamedExecContext(ctx, query, params)
	if err != nil {
		return commonhelpers.WrapDBErr("outbox repo", "create outbox", err)
	}

	return nil
}
