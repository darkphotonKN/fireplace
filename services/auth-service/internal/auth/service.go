package auth

import (
	"context"

	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	"github.com/google/uuid"
)

// Repository is the narrow contract the service depends on (consumer owns interface).
type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}

type service struct {
	repo      Repository
	publishCh commonbroker.Publisher
}

func NewService(repo Repository, publishCh commonbroker.Publisher) *service {
	return &service{repo: repo, publishCh: publishCh}
}

func (s *service) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetByID(ctx, id)
}
