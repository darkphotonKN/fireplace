package checklistitems

import (
	"context"
	"fmt"

	"github.com/darkphotonKN/fireplace/internal/constants"
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

func (s *repository) GetAllByPlanId(ctx context.Context, planId uuid.UUID, scope *string, upcoming *string) ([]*models.ChecklistItem, error) {
	query := `
	SELECT id, description, done, sequence, scope, scheduled_time, archived, created_at, updated_at, plan_id
	FROM checklist_items
	WHERE plan_id = $1
	AND archived = false
	`

	// Add scope filtering if provided
	args := []interface{}{planId}
	if scope != nil {
		query += `AND scope = $2
	`
		args = append(args, *scope)
	}

	if upcoming != nil {
		interval := fmt.Sprintf("'1 %s'", *upcoming)

		query += fmt.Sprintf(`
			AND scheduled_time IS NOT NULL 
	    AND scheduled_time >= CURRENT_TIMESTAMP
			AND scheduled_time <= CURRENT_TIMESTAMP + INTERVAL %s
		`, interval)
	}

	// Always add ordering
	query += `ORDER BY sequence ASC`

	fmt.Printf("constructed query: %s\n", query)
	fmt.Printf("constructed args: %+v\n", args)

	var items []*models.ChecklistItem
	err := s.db.SelectContext(ctx, &items, query, args...)
	if err != nil {
		return nil, errorutils.AnalyzeDBErr(err)
	}

	return items, nil
}

func (s *repository) GetAllArchivedByPlanId(ctx context.Context, planId uuid.UUID, scope *string) ([]*models.ChecklistItem, error) {
	baseQuery := `
	SELECT id, description, done, sequence, scope, scheduled_time, archived, created_at, updated_at, plan_id
	FROM checklist_items
	WHERE plan_id = $1
	AND archived = true
	`

	// Add scope filtering if provided
	args := []interface{}{planId}
	if scope != nil {
		baseQuery += `AND scope = $2
	`
		args = append(args, *scope)
	}

	fmt.Printf("Args for archived items: %v\n", args)

	// Always add ordering
	baseQuery += `ORDER BY sequence ASC`

	var items []*models.ChecklistItem
	err := s.db.SelectContext(ctx, &items, baseQuery, args...)
	if err != nil {
		return nil, errorutils.AnalyzeDBErr(err)
	}

	return items, nil
}

func (s *repository) GetAll(ctx context.Context, scope *string) ([]*models.ChecklistItem, error) {
	query := `
	SELECT 
		id, 
		description,
		done,
		sequence,
		scope, 
		scheduled_time,
		created_at,
		updated_at,
		plan_id
	FROM checklist_items
	`

	var items []*models.ChecklistItem

	args := []interface{}{}

	if scope != nil {
		query += "\nWHERE scope = $1"
		args = append(args, *scope)
		err := s.db.SelectContext(ctx, &items, query, args...)

		if err != nil {
			return nil, errorutils.AnalyzeDBErr(err)
		}
	} else {

		err := s.db.SelectContext(ctx, &items, query)

		if err != nil {
			return nil, errorutils.AnalyzeDBErr(err)
		}
	}

	fmt.Printf("Final constructed query: \n%s\n\n", query)

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
	INSERT INTO checklist_items (description, done, sequence, scope, plan_id)
	VALUES(:description, :done, :sequence, :scope, :plan_id)
	RETURNING id, description, done, sequence, plan_id, scope, created_at, updated_at
	`

	scope := constants.ScopeLongterm
	if req.Scope != nil {
		scope = constants.ChecklistItemScope(*req.Scope)
	}

	item := struct {
		PlanID      uuid.UUID                    `db:"plan_id"`
		Description string                       `db:"description"`
		Done        bool                         `db:"done"`
		Sequence    int                          `db:"sequence"`
		Scope       constants.ChecklistItemScope `db:"scope"`
	}{
		PlanID:      planID,
		Description: req.Description,
		Done:        false,
		Sequence:    sequenceNo,
		Scope:       scope,
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
		return nil, constants.ErrNotFound
	}

	return newItem, nil
}

func (s *repository) Update(ctx context.Context, id uuid.UUID, req UpdateReq) error {
	query := `
	UPDATE checklist_items
	SET
		description = COALESCE(:description, description),
		done = COALESCE(:done, done),
		scope = COALESCE(:scope, scope),
		archived = COALESCE(:archived, archived),`

	// check if scheduled time exists, otherwise set it to nil to remove scheduled time
	if req.ScheduledTime == nil {
		query += `
		scheduled_time = NULL`
	} else {
		query += `
		scheduled_time = :scheduled_time`
	}

	// always add where clause
	query += `
	WHERE id = :id`

	item := map[string]interface{}{
		"id":             id,
		"description":    req.Description,
		"done":           req.Done,
		"scope":          req.Scope,
		"scheduled_time": req.ScheduledTime,
		"archived":       req.Archived,
	}

	fmt.Printf("Updating id: %+v\n")
	fmt.Printf("Updating checklist_items with item: %+v\n", item)
	fmt.Printf("constructed query: %s\n", query)

	result, err := s.db.NamedExecContext(ctx, query, item)

	return errorutils.AnalyzeDBResults(err, result)
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

func (s *repository) GetByID(ctx context.Context, id uuid.UUID) (*models.ChecklistItem, error) {
	query := `
	SELECT id, description, done, sequence, scope, scheduled_time, created_at, updated_at, plan_id
	FROM checklist_items
	WHERE id = $1
	`

	var item models.ChecklistItem
	err := s.db.GetContext(ctx, &item, query, id)
	if err != nil {
		return nil, errorutils.AnalyzeDBErr(err)
	}

	return &item, nil
}

func (s *repository) BatchUpdate(ctx context.Context, planId uuid.UUID, done *bool, scope *constants.ChecklistItemScope) error {
	query := `
	UPDATE checklist_items
	SET
		done = COALESCE(:done, done)
	WHERE plan_id = :planId
	AND scope = :scope
	`

	params := map[string]interface{}{
		"done":  *done,
		"scope": *scope,
	}

	_, err := s.db.NamedExecContext(ctx, query, params)

	if err != nil {
		fmt.Printf("Error when updating all checklist items: %s\n", err.Error())

		return errorutils.AnalyzeDBErr(err)
	}

	return nil
}

/**
* Reset all checklist items with daily reset column set as true for all plans under a user.
**/
func (r *repository) BulkResetDailyItems(ctx context.Context) error {
	query := `
	WITH items_to_update AS (
	SELECT 
		checklist_items.id as id,
		checklist_items.done as done,
		checklist_items.scope as scope,
		plans.daily_reset as daily_reset
	FROM checklist_items
	JOIN plans ON checklist_items.plan_id = plans.id
	WHERE done = true 
	AND daily_reset = true
	AND scope = 'daily'
	)

	UPDATE checklist_items SET
		done = false
	WHERE id IN (SELECT id FROM items_to_update)
	`

	_, err := r.db.ExecContext(ctx, query)

	if err != nil {
		return errorutils.AnalyzeDBErr(err)
	}

	return nil
}
