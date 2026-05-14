package useranalytics

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	GetByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (*UserAnalytics, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (*UserAnalytics, error) {
	// TODO: Implement
	return nil, nil
}