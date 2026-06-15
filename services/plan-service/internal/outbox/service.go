package outbox

import (
	"context"

	commonmodel "github.com/darkphotonKN/fireplace/common/model"
	"github.com/jmoiron/sqlx"
)

type service struct {
	repo Repository
}

type Repository interface {
	CreateTx(ctx context.Context, tx *sqlx.Tx, params CreateOutboxParams) error
}

func NewService(repo Repository) *service {
	return &service{repo: repo}
}

func (s *service) CreateTx(ctx context.Context, tx *sqlx.Tx, params CreateOutboxParams) error {
	return s.repo.CreateTx(ctx, tx, params)
}

func (s *service) Drain(ctx context.Context) ([]*commonmodel.OutboxEvent, error) {

	return nil, nil
}
