package outbox

type service struct {
	repo Repository
}

type Repository interface {
	CreateTx(params CreateOutboxParams) error
}

func NewService(repo Repository) *service {
	return &service{repo: repo}
}

func (s *service) CreateTx(params CreateOutboxParams) error {
	return s.repo.CreateTx(params)
}
