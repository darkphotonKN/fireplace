package outbox

import (
	"context"

	commonmodel "github.com/darkphotonKN/fireplace/common/model"
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
	INSERT INTO outbox (routing_key, exchange, payload)
	VALUES (:routing_key, :exchange, :payload)
	`

	_, err := tx.NamedExecContext(ctx, query, params)
	if err != nil {
		return commonhelpers.WrapDBErr("outbox repo", "create outbox", err)
	}

	return nil
}

func (r *repository) GetAllUnpublished(ctx context.Context) ([]*commonmodel.OutboxEvent, error) {
	query := `
	SELECT 
		id,
		routing_key,
		exchange,
		payload,
		published_at,
		created_at
	FROM outbox
	WHERE published IS NULL
	`

	var res []*commonmodel.OutboxEvent

	err := r.db.SelectContext(ctx, &res, query)

	if err != nil {
		return nil, commonhelpers.WrapDBErr("plans", "GetAllUnpublished", err)
	}

	return nil, nil
}
