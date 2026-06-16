package outbox

import (
	"context"
	"fmt"
	"strings"

	commonmodel "github.com/darkphotonKN/fireplace/common/model"
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

func (r *repository) GetUnpublished(ctx context.Context, tx *sqlx.Tx, limit int) ([]*commonmodel.OutboxEvent, error) {
	query := `
	SELECT
		id,
		routing_key,
		exchange,
		payload,
		published_at,
		created_at
	FROM outbox
	WHERE published_at IS NULL
	ORDER BY created_at ASC
	LIMIT $1
	FOR UPDATE SKIP LOCKED
	`

	var res []*commonmodel.OutboxEvent

	err := tx.SelectContext(ctx, &res, query, limit)

	if err != nil {
		return nil, commonhelpers.WrapDBErr("plans", "GetAllUnpublished", err)
	}

	return res, nil
}

func (r *repository) BatchUpdatePublishedAt(ctx context.Context, tx *sqlx.Tx, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	queryStart := `
	UPDATE outbox SET
	published_at = NOW()
	WHERE id IN (`

	var query strings.Builder
	query.WriteString(queryStart)

	// arguments for the ids
	args := make([]any, 0, len(ids))

	for i, id := range ids {
		if i > 0 {
			query.WriteString(", ")
		}

		fmt.Fprintf(&query, "$%d", i+1)

		// successful
		args = append(args, id)
	}

	// close off query
	query.WriteString(`)`)

	// execute query
	_, err := tx.ExecContext(ctx, query.String(), args...)

	if err != nil {
		return commonhelpers.WrapDBErr("plans", "BatchUpdatePublished", err)
	}

	return nil
}
