package checklistitems

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &repository{
		db: db,
	}
}

func (s *repository) Create(req CreateReq) error {
	return nil
}

func (s *repository) Update(req CreateReq) error {
	return nil
}

func (s *repository) Delete(id uuid.UUID) error {
	return nil
}
