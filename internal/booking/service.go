package booking

import (
	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/google/uuid"
)

type Service struct {
	Repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{
		Repo: repo,
	}
}

func (s *Service) GetById(userId uuid.UUID, id uuid.UUID) (*models.Booking, error) {
	return s.Repo.GetById(userId, id)
}

func (s *Service) Create(userId uuid.UUID, req CreateRequest) error {
	return s.Repo.Create(userId, req)
}

func (s *Service) CreateTwo(userId uuid.UUID, req CreateRequest) error {
	return s.Repo.Create(userId, req)
}
