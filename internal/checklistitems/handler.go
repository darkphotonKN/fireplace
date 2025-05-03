package checklistitems

import "github.com/google/uuid"

type Handler struct {
	service Service
}

type Service interface {
	Create(req CreateReq) error
	Update(req CreateReq) error
	Delete(id uuid.UUID) error
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}
