package outbox

import (
	"context"

	commonmodel "github.com/darkphotonKN/fireplace/common/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type service struct {
	repo Repository
}

// defaultUnpublishedLimit caps how many outbox rows a single drain pulls.
const defaultUnpublishedLimit = 10

type Repository interface {
	CreateTx(ctx context.Context, tx *sqlx.Tx, params CreateOutboxParams) error
	GetUnpublished(ctx context.Context, tx *sqlx.Tx, limit int) ([]*commonmodel.OutboxEvent, error)
	BatchUpdatePublishedAt(ctx context.Context, tx *sqlx.Tx, ids []uuid.UUID) error
}

func NewService(repo Repository) *service {
	return &service{repo: repo}
}

func (s *service) CreateTx(ctx context.Context, tx *sqlx.Tx, params CreateOutboxParams) error {
	return s.repo.CreateTx(ctx, tx, params)
}

func (s *service) GetUnpublished(ctx context.Context, tx *sqlx.Tx) ([]*commonmodel.OutboxEvent, error) {
	return s.repo.GetUnpublished(ctx, tx, defaultUnpublishedLimit)
}

func (s *service) MarkPublished(ctx context.Context, tx *sqlx.Tx, ids []uuid.UUID) error {
	return s.repo.BatchUpdatePublishedAt(ctx, tx, ids)
}
