package insights

import "github.com/darkphotonKN/fireplace/internal/ai"

type service struct {
	repo       Repository
	contentGen ContentGenAI
}

type Repository interface {
}

type ContentGenAI interface {
	ChatCompletion(message string) (string, error)
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}
