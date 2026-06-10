package checklistitem

import (
	"context"
	"fmt"
	"time"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *repository {
	return &repository{db: db}
}

// wrapDBErr is the repo boundary translation point: it delegates to the shared
// WrapDBErr helper, which converts infrastructure errors (sql.ErrNoRows,
// duplicate keys, constraint violations, transient failures) into domain
// sentinels and wraps anything else with the repo name + operation for context.
// It never logs and never decides transport status.
func wrapDBErr(op string, err error) error {
	return commonhelpers.WrapDBErr("checklistitem repo", op, err)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Item, error) {
	query := `
	SELECT id, description, done, sequence, scope, type, parent_id, start_date, due_date,
	       archived, created_at, updated_at, plan_id
	FROM checklist_items WHERE id = $1`
	var item Item
	if err := r.db.GetContext(ctx, &item, query, id); err != nil {
		return nil, wrapDBErr("get item by id "+id.String(), err)
	}
	return &item, nil
}

func (r *repository) ListByPlanID(ctx context.Context, in ListItemsInput) ([]*Item, error) {
	query := `
	SELECT id, description, done, sequence, scope, type, parent_id, start_date, due_date,
	       archived, created_at, updated_at, plan_id
	FROM checklist_items
	WHERE plan_id = $1 AND archived = false`

	args := []interface{}{in.PlanID}
	if in.Scope != nil {
		args = append(args, *in.Scope)
		query += fmt.Sprintf(" AND scope = $%d", len(args))
	}
	if in.Type != nil {
		args = append(args, *in.Type)
		query += fmt.Sprintf(" AND type = $%d", len(args))
	}
	if in.Upcoming != nil {
		// Filter on start_date — items "starting in the next week/month".
		query += fmt.Sprintf(`
			AND start_date IS NOT NULL
			AND start_date >= CURRENT_DATE
			AND start_date <= CURRENT_DATE + INTERVAL '1 %s'`, *in.Upcoming)
	}
	query += " ORDER BY sequence ASC"

	var items []*Item
	if err := r.db.SelectContext(ctx, &items, query, args...); err != nil {
		return nil, wrapDBErr("list by plan "+in.PlanID.String(), err)
	}
	return items, nil
}

func (r *repository) ListArchivedByPlanID(ctx context.Context, planID uuid.UUID, scope *string) ([]*Item, error) {
	query := `
	SELECT id, description, done, sequence, scope, type, parent_id, start_date, due_date,
	       archived, created_at, updated_at, plan_id
	FROM checklist_items
	WHERE plan_id = $1 AND archived = true`
	args := []interface{}{planID}
	if scope != nil {
		args = append(args, *scope)
		query += " AND scope = $2"
	}
	query += " ORDER BY sequence ASC"

	var items []*Item
	if err := r.db.SelectContext(ctx, &items, query, args...); err != nil {
		return nil, wrapDBErr("list archived by plan "+planID.String(), err)
	}
	return items, nil
}

func (r *repository) HasChildren(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM checklist_items WHERE parent_id = $1)`
	if err := r.db.GetContext(ctx, &exists, query, id); err != nil {
		return false, wrapDBErr("has children "+id.String(), err)
	}
	return exists, nil
}

func (r *repository) CountItems(ctx context.Context) (int, error) {
	var count int
	if err := r.db.QueryRowxContext(ctx, `SELECT COUNT(id) FROM checklist_items`).Scan(&count); err != nil {
		return 0, wrapDBErr("count items", err)
	}
	return count, nil
}

func (r *repository) Create(ctx context.Context, in CreateItemInput, sequenceNo int) (*Item, error) {
	query := `
	INSERT INTO checklist_items (description, done, sequence, scope, type, parent_id, plan_id)
	VALUES (:description, :done, :sequence, :scope, :type, :parent_id, :plan_id)
	RETURNING id, description, done, sequence, plan_id, scope, type, parent_id, start_date,
	          due_date, archived, created_at, updated_at`

	scope := ScopeLongterm
	if in.Scope != nil {
		scope = *in.Scope
	}
	itemType := TypeTask
	if in.Type != nil {
		itemType = *in.Type
	}

	row := struct {
		PlanID      uuid.UUID  `db:"plan_id"`
		Description string     `db:"description"`
		Done        bool       `db:"done"`
		Sequence    int        `db:"sequence"`
		Scope       string     `db:"scope"`
		Type        string     `db:"type"`
		ParentID    *uuid.UUID `db:"parent_id"`
	}{
		PlanID:      in.PlanID,
		Description: in.Description,
		Done:        false,
		Sequence:    sequenceNo,
		Scope:       scope,
		Type:        itemType,
		ParentID:    in.ParentID,
	}

	rows, err := r.db.NamedQueryContext(ctx, query, row)
	if err != nil {
		return nil, wrapDBErr("create item", err)
	}
	defer rows.Close()

	out := &Item{}
	if rows.Next() {
		if err := rows.StructScan(out); err != nil {
			return nil, wrapDBErr("create item: scan", err)
		}
	} else {
		return nil, commonconstants.ErrNotFound
	}
	return out, nil
}

func (r *repository) Update(ctx context.Context, in UpdateItemInput) error {
	setClause := `
		description = COALESCE(:description, description),
		done        = COALESCE(:done, done),
		scope       = COALESCE(:scope, scope),
		archived    = COALESCE(:archived, archived),
		type        = COALESCE(:type, type)`

	params := map[string]interface{}{
		"id":          in.ID,
		"description": in.Description,
		"done":        in.Done,
		"scope":       in.Scope,
		"archived":    in.Archived,
		"type":        in.Type,
	}

	if in.SetParent {
		setClause += `, parent_id = :parent_id`
		if in.ParentID != nil {
			params["parent_id"] = *in.ParentID
		} else {
			params["parent_id"] = nil
		}
	}

	query := `UPDATE checklist_items SET ` + setClause + ` WHERE id = :id`
	res, err := r.db.NamedExecContext(ctx, query, params)
	if err != nil {
		return wrapDBErr("update item "+in.ID.String(), err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return commonconstants.ErrNotFound
	}
	return nil
}

func (r *repository) UpdateDates(ctx context.Context, id uuid.UUID, startDate, dueDate *time.Time) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE checklist_items SET start_date = $2, due_date = $3 WHERE id = $1`,
		id, startDate, dueDate)
	if err != nil {
		return wrapDBErr("update dates "+id.String(), err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return commonconstants.ErrNotFound
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM checklist_items WHERE id = $1`, id)
	if err != nil {
		return wrapDBErr("delete item "+id.String(), err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return commonconstants.ErrNotFound
	}
	return nil
}

// BulkResetDailyItems uncompletes every "daily" item belonging to a plan with
// daily_reset=true. Called by the nightly job's DailyReset RPC.
func (r *repository) BulkResetDailyItems(ctx context.Context) (int64, error) {
	query := `
	WITH items_to_update AS (
		SELECT checklist_items.id AS id
		FROM checklist_items
		JOIN plans ON checklist_items.plan_id = plans.id
		WHERE done = true AND daily_reset = true AND scope = 'daily'
	)
	UPDATE checklist_items SET done = false
	WHERE id IN (SELECT id FROM items_to_update)`

	res, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, wrapDBErr("bulk reset daily items", err)
	}
	rows, _ := res.RowsAffected()
	return rows, nil
}

// GetByUserID returns non-archived items across all of a user's plans —
// used by useranalytics in 4c via gRPC.
func (r *repository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Item, error) {
	query := `
	SELECT ci.id, ci.description, ci.done, ci.sequence, ci.scope, ci.type, ci.parent_id,
	       ci.start_date, ci.due_date, ci.archived, ci.created_at, ci.updated_at, ci.plan_id
	FROM checklist_items ci
	JOIN plans p ON ci.plan_id = p.id
	WHERE p.user_id = $1 AND ci.archived = false
	ORDER BY ci.created_at DESC`

	var items []*Item
	if err := r.db.SelectContext(ctx, &items, query, userID); err != nil {
		return nil, wrapDBErr("get items by user "+userID.String(), err)
	}
	return items, nil
}

// ListInDateWindow returns non-archived items for a plan whose
// [start_date, due_date] range intersects [windowStart, windowEnd].
// Items with both dates null are excluded. Used by calendar-service.
func (r *repository) ListInDateWindow(ctx context.Context, planID uuid.UUID, windowStart, windowEnd time.Time) ([]*Item, error) {
	query := `
	SELECT id, description, done, sequence, scope, type, parent_id, start_date, due_date,
	       archived, created_at, updated_at, plan_id
	FROM checklist_items
	WHERE plan_id = $1
	  AND archived = false
	  AND (start_date IS NOT NULL OR due_date IS NOT NULL)
	  AND COALESCE(start_date, due_date) <= $3
	  AND COALESCE(due_date,   start_date) >= $2
	ORDER BY COALESCE(start_date, due_date) ASC, sequence ASC`

	var items []*Item
	if err := r.db.SelectContext(ctx, &items, query, planID, windowStart, windowEnd); err != nil {
		return nil, wrapDBErr("list in date window for plan "+planID.String(), err)
	}
	return items, nil
}
