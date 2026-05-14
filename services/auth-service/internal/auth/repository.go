package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type repository struct {
	DB *sqlx.DB
}

func NewRepository(db *sqlx.DB) *repository {
	return &repository{DB: db}
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User

	query := `SELECT id, email, name, hashed_password, created_at, updated_at FROM users WHERE id = $1`

	err := r.DB.GetContext(ctx, &u, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, commonconstants.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &u, nil
}
