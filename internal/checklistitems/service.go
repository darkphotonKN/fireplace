package checklistitems

import "github.com/google/uuid"

type service struct {
	repo Repository
}

func NewService() Service {
	return &service{}
}

func (s *service) Create(req CreateReq) error {
	return nil
}

func (s *service) Update(req CreateReq) error {
	return nil
}

func (s *service) Delete(id uuid.UUID) error {
	return nil
}
