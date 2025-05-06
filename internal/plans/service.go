package plans

import (
	"context"

	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/google/uuid"
)

type service struct {
	repo Repository
}

type Repository interface {
	GetById(ctx context.Context, id uuid.UUID) (*models.Plan, error)
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) GetById(ctx context.Context, id uuid.UUID) (*models.Plan, error) {
	return s.repo.GetById(ctx, id)
}
