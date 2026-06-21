package inbox

import (
	"context"

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

func (r *repository) Create(ctx context.Context, eventID uuid.UUID) error {
	query := `
	INSERT INTO processed_events(event_id, consumer)
	VALUES ($1, $2)
	`
	// let unique constraint catch the error on duplicate
	_, err := r.db.ExecContext(ctx, query, eventID, "insights")
	if err != nil {
		return commonhelpers.WrapDBErr("insights_inbox", "create", err)
	}

	return nil
}

// CreateTx is the transactional sibling of Create — stub for now.
func (r *repository) CreateTx(ctx context.Context, tx *sqlx.Tx, eventID uuid.UUID) error {
	query := `
	INSERT INTO processed_events(event_id, consumer)
	VALUES ($1, $2)
	`
	// let unique constraint catch the error on duplicate
	_, err := tx.ExecContext(ctx, query, eventID, "insights")
	if err != nil {
		return commonhelpers.WrapDBErr("insights_inbox", "create", err)
	}

	return nil
}
