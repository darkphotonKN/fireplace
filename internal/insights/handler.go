package insights

type Handler struct {
	service Service
}
type Service interface {
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}
