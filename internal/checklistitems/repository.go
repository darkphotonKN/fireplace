package checklistitems

import (
	"context"
	"fmt"
	"time"

	"github.com/darkphotonKN/fireplace/internal/constants"
	"github.com/darkphotonKN/fireplace/internal/logger"
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

func (s *repository) GetAllByPlanId(ctx context.Context, planId uuid.UUID, scope *string, itemType *string, upcoming *string) ([]*models.ChecklistItem, error) {
	query := `
	SELECT id, description, done, sequence, scope, type, parent_id, start_date, due_date, archived, created_at, updated_at, plan_id
	FROM checklist_items
	WHERE plan_id = $1
	AND archived = false
	`

	args := []interface{}{planId}
	if scope != nil {
		args = append(args, *scope)
		query += fmt.Sprintf(" AND scope = $%d\n", len(args))
	}
	if itemType != nil {
		args = append(args, *itemType)
		query += fmt.Sprintf(" AND type = $%d\n", len(args))
	}

	if upcoming != nil {
		// Filter on start_date — items "starting in the next week/month".
		// scheduled_time was replaced by start_date/due_date; start_date is the
		// closest semantic match for the original "upcoming" query.
		interval := fmt.Sprintf("'1 %s'", *upcoming)

		query += fmt.Sprintf(`
			AND start_date IS NOT NULL
			AND start_date >= CURRENT_DATE
			AND start_date <= CURRENT_DATE + INTERVAL %s
		`, interval)
	}

	// Always add ordering
	query += `ORDER BY sequence ASC`

	logger.Debug("Executing GetAllByPlanId query", "query", query, "args", args)

	var items []*models.ChecklistItem
	err := s.db.SelectContext(ctx, &items, query, args...)
	if err != nil {
		return nil, errorutils.AnalyzeDBErr(err)
	}

	return items, nil
}

// HasChildren returns true if any non-archived row has parent_id = id.
func (s *repository) HasChildren(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM checklist_items WHERE parent_id = $1)`
	if err := s.db.GetContext(ctx, &exists, query, id); err != nil {
		return false, errorutils.AnalyzeDBErr(err)
	}
	return exists, nil
}

func (s *repository) GetAllArchivedByPlanId(ctx context.Context, planId uuid.UUID, scope *string) ([]*models.ChecklistItem, error) {
	baseQuery := `
	SELECT id, description, done, sequence, scope, type, parent_id, start_date, due_date, archived, created_at, updated_at, plan_id
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

	logger.Debug("Fetching archived items", "args", args)

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
		type,
		parent_id,
		start_date,
		due_date,
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
	INSERT INTO checklist_items (description, done, sequence, scope, type, parent_id, plan_id)
	VALUES(:description, :done, :sequence, :scope, :type, :parent_id, :plan_id)
	RETURNING id, description, done, sequence, plan_id, scope, type, parent_id, created_at, updated_at
	`

	scope := constants.ScopeLongterm
	if req.Scope != nil {
		scope = constants.ChecklistItemScope(*req.Scope)
	}

	itemType := "task"
	if req.Type != nil {
		itemType = *req.Type
	}

	item := struct {
		PlanID      uuid.UUID                    `db:"plan_id"`
		Description string                       `db:"description"`
		Done        bool                         `db:"done"`
		Sequence    int                          `db:"sequence"`
		Scope       constants.ChecklistItemScope `db:"scope"`
		Type        string                       `db:"type"`
		ParentID    *uuid.UUID                   `db:"parent_id"`
	}{
		PlanID:      planID,
		Description: req.Description,
		Done:        false,
		Sequence:    sequenceNo,
		Scope:       scope,
		Type:        itemType,
		ParentID:    req.ParentID,
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
	// Build the SET clause: COALESCE for fields where nil means "leave alone";
	// parent_id needs explicit handling because nil-valid means "clear".
	setClause := `
		description = COALESCE(:description, description),
		done = COALESCE(:done, done),
		scope = COALESCE(:scope, scope),
		archived = COALESCE(:archived, archived),
		type = COALESCE(:type, type)`

	item := map[string]interface{}{
		"id":          id,
		"description": req.Description,
		"done":        req.Done,
		"scope":       req.Scope,
		"archived":    req.Archived,
		"type":        req.Type,
	}

	if req.ParentID.Present {
		setClause += `, parent_id = :parent_id`
		if req.ParentID.Valid {
			pid := req.ParentID.Value
			item["parent_id"] = pid
		} else {
			item["parent_id"] = nil
		}
	}

	query := `UPDATE checklist_items SET ` + setClause + ` WHERE id = :id`

	result, err := s.db.NamedExecContext(ctx, query, item)
	return errorutils.AnalyzeDBResults(err, result)
}

// UpdateDates persists final start_date / due_date values. Pass nil to clear a column.
// The DB CHECK constraint enforces start_date <= due_date as a final guard.
func (s *repository) UpdateDates(ctx context.Context, id uuid.UUID, startDate, dueDate *time.Time) error {
	query := `
	UPDATE checklist_items
	SET start_date = $2, due_date = $3
	WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id, startDate, dueDate)
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
	SELECT id, description, done, sequence, scope, type, parent_id, start_date, due_date, created_at, updated_at, plan_id
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

func (r *repository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.ChecklistItem, error) {
	query := `
	SELECT
		ci.id,
		ci.description,
		ci.done,
		ci.sequence,
		ci.scope,
		ci.type,
		ci.parent_id,
		ci.start_date,
		ci.due_date,
		ci.archived,
		ci.created_at,
		ci.updated_at,
		ci.plan_id
	FROM checklist_items ci
	JOIN plans p ON ci.plan_id = p.id
	WHERE p.user_id = $1
	AND ci.archived = false
	ORDER BY ci.created_at DESC
	`

	var items []*models.ChecklistItem
	err := r.db.SelectContext(ctx, &items, query, userID)
	if err != nil {
		return nil, errorutils.AnalyzeDBErr(err)
	}

	return items, nil
}
