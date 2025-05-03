package plans

type service struct {
	repo Repository
}

type Repository interface {
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}
