package checklistitems

import (
	"context"

	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/darkphotonKN/fireplace/internal/utils/errorutils"
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

func (s *repository) GetAll(ctx context.Context) ([]*models.ChecklistItem, error) {
	return nil, nil
}

func (s *repository) CountItems(ctx context.Context) (int, error) {
	var count int
	query := `
	SELECT COUNT(id)
	FROM checklist_items
	`

	err := s.db.QueryRowxContext(ctx, query).Scan(&count)

	if err != nil {
		return 0, errorutils.AnalyzeDBErr(err)
	}

	return count, nil
}

func (s *repository) Create(ctx context.Context, req CreateReq, sequenceNo int) error {
	query := `
	INSERT INTO checklist_items (description, done, sequence)
	VALUES(:description, :done, :sequence)
	`

	item := struct {
		description string `db:"description"`
		done        bool   `db:"done"`
		sequence    int    `db:"sequence"`
	}{
		description: req.Description,
		done:        req.Done,
		sequence:    sequenceNo,
	}

	_, err := s.db.NamedExecContext(ctx, query, item)

	if err != nil {
		return errorutils.AnalyzeDBErr(err)
	}

	return nil
}

func (s *repository) Update(ctx context.Context, id uuid.UUID, req CreateReq) error {
	return nil
}

func (s *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}
