package outbox

import (
	"context"
	"fmt"
	"log/slog"
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

	return res, nil
}

func (r *repository) BatchUpdatePublished(ctx context.Context, ids []uuid.UUID) error {
	unprocessedIds := make([]uuid.UUID, 0)

	query := `
	UPDATE outbox SET
	published_at = NOW()
	WHERE
	`
	var b strings.Builder
	b.WriteString(query)

	// arguments for the ids
	args := make([]uuid.UUID, 0, len(ids))

	for i, id := range ids {
		newId := fmt.Sprintf("id = $%d AND ", i+1)
		_, err := b.WriteString(newId)

		if err != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("error attemptings to build string for id %s", id))
			unprocessedIds = append(unprocessedIds, id)

			// move on to the next query
			continue
		}

		// successful
		args = append(args, id)
	}

	// execute query

}
