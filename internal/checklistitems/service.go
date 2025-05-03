package checklistitems

import (
	"context"

	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/google/uuid"
)

type service struct {
	repo Repository
}

type Repository interface {
	Create(ctx context.Context, req CreateReq, sequenceNo int) error
	Update(ctx context.Context, id uuid.UUID, req CreateReq) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetAll(ctx context.Context) ([]*models.ChecklistItem, error)
	CountItems(ctx context.Context) (int, error)
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) GetAll(ctx context.Context) ([]*models.ChecklistItem, error) {
	return s.repo.GetAll(ctx)
}

func (s *service) Create(ctx context.Context, req CreateReq) error {
	// count number of current items in table
	count, err := s.repo.CountItems(ctx)

	if err != nil {
		return err
	}

	// add 1 to make new sequence
	return s.repo.Create(ctx, req, count+1)
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req CreateReq) error {
	return s.repo.Update(ctx, id, req)
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
