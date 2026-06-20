package insights

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *repository {
	return &repository{db: db}
}

// Create is a stub — persistence logic is not implemented yet.
func (r *repository) Create(ctx context.Context) error {
	// TODO: implement
	return nil
}
