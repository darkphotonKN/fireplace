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

type Repository interface {
	CreateTx(ctx context.Context, tx *sqlx.Tx, params CreateOutboxParams) error
	GetAllUnpublished(ctx context.Context) ([]*commonmodel.OutboxEvent, error)
	BatchMarkPublished(ctx context.Context, ids []uuid.UUID) error
}

func NewService(repo Repository) *service {
	return &service{repo: repo}
}

func (s *service) CreateTx(ctx context.Context, tx *sqlx.Tx, params CreateOutboxParams) error {
	return s.repo.CreateTx(ctx, tx, params)
}

func (s *service) GetUnpublished(ctx context.Context) ([]*commonmodel.OutboxEvent, error) {
	return s.repo.GetAllUnpublished(ctx)
}

func (s *service) MarkUnpublished(ctx context.Context, ids []uuid.UUID) error {
	return s.repo.BatchMarkPublished(ctx, ids)
}
