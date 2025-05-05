package checklistitems

import (
	"context"
	"fmt"

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
	query := `
	SELECT id, description, done, sequence, created_at, updated_at, plan_id
	FROM checklist_items
	ORDER BY sequence ASC
	`

	var items []*models.ChecklistItem
	err := s.db.SelectContext(ctx, &items, query)
	if err != nil {
		return nil, errorutils.AnalyzeDBErr(err)
	}

	return items, nil
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

func (s *repository) Create(ctx context.Context, req CreateReq, planID uuid.UUID, sequenceNo int) (*models.ChecklistItem, error) {
	query := `
	INSERT INTO checklist_items (description, done, sequence, plan_id)
	VALUES(:description, :done, :sequence, :plan_id)
	RETURNING id, description, done, sequence, plan_id, created_at, updated_at
	`

	item := struct {
		Description string    `db:"description"`
		Done        bool      `db:"done"`
		Sequence    int       `db:"sequence"`
		PlanID      uuid.UUID `db:"plan_id"`
	}{
		Description: req.Description,
		Done:        false,
		Sequence:    sequenceNo,
		PlanID:      planID,
	}

	newItem := &models.ChecklistItem{}

	rows, err := s.db.NamedQueryContext(ctx, query, item)

	if err != nil {
		fmt.Printf("Error from db when attempting to create item: %v\n", err)
		return nil, errorutils.AnalyzeDBErr(err)
	}
	defer rows.Close()

	// acquire the first item
	if rows.Next() {
		if err := rows.StructScan(newItem); err != nil {
			fmt.Printf("Error from db when attempting to scan created item: %v\n", err)
			return nil, errorutils.AnalyzeDBErr(err)
		}
	} else {
		return nil, errorutils.ErrNotFound
	}

	return newItem, nil
}

func (s *repository) Update(ctx context.Context, id uuid.UUID, req UpdateReq) error {
	query := `
	UPDATE checklist_items
	SET 
		description = COALESCE(:description, description),
		done = COALESCE(:done, done)	
	WHERE id = :id
	`

	item := struct {
		ID          uuid.UUID `db:"id"`
		Description *string   `db:"description"`
		Done        *bool     `db:"done"`
	}{
		ID:          id,
		Description: req.Description,
		Done:        req.Done,
	}

	_, err := s.db.NamedExecContext(ctx, query, item)
	if err != nil {
		return errorutils.AnalyzeDBErr(err)
	}

	return nil
}

func (s *repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
	DELETE FROM checklist_items
	WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return errorutils.AnalyzeDBErr(err)
	}

	return nil
}
