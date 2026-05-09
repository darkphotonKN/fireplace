package calendar

import (
	"context"
	"fmt"
	"time"

	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository reads checklist items overlapping a date window.
type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// GetItemsInWindow returns non-archived checklist items for the plan whose
// [start_date, due_date] range (treating null on one side as the other side)
// intersects [windowStart, windowEnd]. Items with both dates null are excluded.
func (r *Repository) GetItemsInWindow(ctx context.Context, planID uuid.UUID, windowStart, windowEnd time.Time) ([]*models.ChecklistItem, error) {
	query := `
	SELECT id, description, done, sequence, scope, start_date, due_date, archived, created_at, updated_at, plan_id
	FROM checklist_items
	WHERE plan_id = $1
	  AND archived = false
	  AND (start_date IS NOT NULL OR due_date IS NOT NULL)
	  AND COALESCE(start_date, due_date) <= $3
	  AND COALESCE(due_date,   start_date) >= $2
	ORDER BY COALESCE(start_date, due_date) ASC, sequence ASC`

	var items []*models.ChecklistItem
	if err := r.db.SelectContext(ctx, &items, query, planID, windowStart, windowEnd); err != nil {
		return nil, fmt.Errorf("failed to query calendar items: %w", err)
	}
	return items, nil
}
