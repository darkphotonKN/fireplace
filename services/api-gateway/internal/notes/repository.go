package notes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Repository struct {
	db *sqlx.DB
}

// wrapDBErr delegates to the shared WrapDBErr helper, translating infrastructure
// errors into domain sentinels (or wrapping with repo + operation context). This
// notes repo is slated to move into a dedicated microservice under the
// strangler-fig migration; this local shim just keeps existing call sites
// compiling in the gateway in the meantime.
func wrapDBErr(op string, err error) error {
	return commonhelpers.WrapDBErr("notes repo", op, err)
}

// NewRepository creates a new notes repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new note into the database
func (r *Repository) Create(note *Note) (*Note, error) {
	var aiMetadataJSON []byte
	var err error

	if note.AIMetadata != nil {
		aiMetadataJSON, err = json.Marshal(note.AIMetadata)
		if err != nil {
			return nil, fmt.Errorf("notes repo: create note: marshal ai metadata: %w", err)
		}
	}

	var aiMetadataParam interface{} = nil
	if note.AIMetadata != nil {
		aiMetadataParam = aiMetadataJSON
	}

	query := `
		INSERT INTO notes (plan_id, content, type, priority, tags, related_task_ids, ai_metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, plan_id, content, type, priority, tags, related_task_ids, is_read, is_dismissed, ai_metadata, created_at, updated_at
	`

	var createdNote Note
	var aiMetadataRaw sql.NullString

	err = r.db.QueryRowx(
		query,
		note.PlanID,
		note.Content,
		note.Type,
		note.Priority,
		pq.Array(note.Tags),
		pq.Array(note.RelatedTaskIDs),
		aiMetadataParam,
	).Scan(
		&createdNote.ID,
		&createdNote.PlanID,
		&createdNote.Content,
		&createdNote.Type,
		&createdNote.Priority,
		&createdNote.Tags,
		&createdNote.RelatedTaskIDs,
		&createdNote.IsRead,
		&createdNote.IsDismissed,
		&aiMetadataRaw,
		&createdNote.CreatedAt,
		&createdNote.UpdatedAt,
	)

	if err != nil {
		return nil, wrapDBErr("create note", err)
	}

	// Unmarshal AI metadata if present
	if aiMetadataRaw.Valid && aiMetadataRaw.String != "" {
		var aiMetadata AIMetadata
		if err := json.Unmarshal([]byte(aiMetadataRaw.String), &aiMetadata); err == nil {
			createdNote.AIMetadata = &aiMetadata
		}
	}

	return &createdNote, nil
}

// GetByID retrieves a note by its ID
func (r *Repository) GetByID(id uuid.UUID) (*Note, error) {
	query := `
		SELECT id, plan_id, content, type, priority, tags, related_task_ids,
		       is_read, is_dismissed, ai_metadata, created_at, updated_at
		FROM notes
		WHERE id = $1
	`

	var note Note
	var aiMetadataRaw sql.NullString

	err := r.db.QueryRowx(query, id).Scan(
		&note.ID,
		&note.PlanID,
		&note.Content,
		&note.Type,
		&note.Priority,
		&note.Tags,
		&note.RelatedTaskIDs,
		&note.IsRead,
		&note.IsDismissed,
		&aiMetadataRaw,
		&note.CreatedAt,
		&note.UpdatedAt,
	)

	if err != nil {
		return nil, wrapDBErr("get note by id "+id.String(), err)
	}

	// Unmarshal AI metadata if present
	if aiMetadataRaw.Valid && aiMetadataRaw.String != "" {
		var aiMetadata AIMetadata
		if err := json.Unmarshal([]byte(aiMetadataRaw.String), &aiMetadata); err == nil {
			note.AIMetadata = &aiMetadata
		}
	}

	return &note, nil
}

// GetByPlanID retrieves all notes for a specific plan
func (r *Repository) GetByPlanID(planID uuid.UUID, filters *FilterOptions) ([]Note, error) {
	query := `
		SELECT id, plan_id, content, type, priority, tags, related_task_ids,
		       is_read, is_dismissed, ai_metadata, created_at, updated_at
		FROM notes
		WHERE plan_id = $1
	`

	args := []interface{}{planID}
	argCount := 1

	// Apply filters
	if filters != nil {
		var conditions []string

		if filters.Type != "" {
			argCount++
			conditions = append(conditions, fmt.Sprintf("type = $%d", argCount))
			args = append(args, filters.Type)
		}

		if filters.Priority != "" {
			argCount++
			conditions = append(conditions, fmt.Sprintf("priority = $%d", argCount))
			args = append(args, filters.Priority)
		}

		if filters.IsRead != nil {
			argCount++
			conditions = append(conditions, fmt.Sprintf("is_read = $%d", argCount))
			args = append(args, *filters.IsRead)
		}

		if filters.IsDismissed != nil {
			argCount++
			conditions = append(conditions, fmt.Sprintf("is_dismissed = $%d", argCount))
			args = append(args, *filters.IsDismissed)
		}

		if len(filters.Tags) > 0 {
			argCount++
			conditions = append(conditions, fmt.Sprintf("tags && $%d", argCount))
			args = append(args, pq.Array(filters.Tags))
		}

		if filters.RelatedTaskID != "" {
			argCount++
			conditions = append(conditions, fmt.Sprintf("$%d = ANY(related_task_ids)", argCount))
			args = append(args, filters.RelatedTaskID)
		}

		if len(conditions) > 0 {
			query += " AND " + strings.Join(conditions, " AND ")
		}
	}

	query += " ORDER BY priority DESC, created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, wrapDBErr("list notes by plan "+planID.String(), err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		var aiMetadataRaw sql.NullString

		err := rows.Scan(
			&note.ID,
			&note.PlanID,
			&note.Content,
			&note.Type,
			&note.Priority,
			&note.Tags,
			&note.RelatedTaskIDs,
			&note.IsRead,
			&note.IsDismissed,
			&aiMetadataRaw,
			&note.CreatedAt,
			&note.UpdatedAt,
		)

		if err != nil {
			return nil, wrapDBErr("list notes by plan "+planID.String()+": scan", err)
		}

		// Unmarshal AI metadata if present
		if aiMetadataRaw.Valid && aiMetadataRaw.String != "" {
			var aiMetadata AIMetadata
			if err := json.Unmarshal([]byte(aiMetadataRaw.String), &aiMetadata); err == nil {
				note.AIMetadata = &aiMetadata
			}
		}

		notes = append(notes, note)
	}

	return notes, nil
}

// Update updates an existing note
func (r *Repository) Update(id uuid.UUID, updates *UpdateNoteReq) (*Note, error) {
	var setClauses []string
	var args []interface{}
	argCount := 0

	if updates.Content != nil {
		argCount++
		setClauses = append(setClauses, fmt.Sprintf("content = $%d", argCount))
		args = append(args, *updates.Content)
	}

	if updates.Priority != nil {
		argCount++
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argCount))
		args = append(args, *updates.Priority)
	}

	if len(updates.Tags) > 0 {
		argCount++
		setClauses = append(setClauses, fmt.Sprintf("tags = $%d", argCount))
		args = append(args, pq.Array(updates.Tags))
	}

	if updates.IsRead != nil {
		argCount++
		setClauses = append(setClauses, fmt.Sprintf("is_read = $%d", argCount))
		args = append(args, *updates.IsRead)
	}

	if updates.IsDismissed != nil {
		argCount++
		setClauses = append(setClauses, fmt.Sprintf("is_dismissed = $%d", argCount))
		args = append(args, *updates.IsDismissed)
	}

	if len(setClauses) == 0 {
		return nil, fmt.Errorf("%w: no fields to update", commonconstants.ErrInvalidInput)
	}

	argCount++
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE notes
		SET %s, updated_at = CURRENT_TIMESTAMP
		WHERE id = $%d
		RETURNING id, plan_id, content, type, priority, tags, related_task_ids,
		          is_read, is_dismissed, ai_metadata, created_at, updated_at
	`, strings.Join(setClauses, ", "), argCount)

	var note Note
	var aiMetadataRaw sql.NullString

	err := r.db.QueryRowx(query, args...).Scan(
		&note.ID,
		&note.PlanID,
		&note.Content,
		&note.Type,
		&note.Priority,
		&note.Tags,
		&note.RelatedTaskIDs,
		&note.IsRead,
		&note.IsDismissed,
		&aiMetadataRaw,
		&note.CreatedAt,
		&note.UpdatedAt,
	)

	if err != nil {
		return nil, wrapDBErr("update note "+id.String(), err)
	}

	// Unmarshal AI metadata if present
	if aiMetadataRaw.Valid && aiMetadataRaw.String != "" {
		var aiMetadata AIMetadata
		if err := json.Unmarshal([]byte(aiMetadataRaw.String), &aiMetadata); err == nil {
			note.AIMetadata = &aiMetadata
		}
	}

	return &note, nil
}

// Delete removes a note from the database
func (r *Repository) Delete(id uuid.UUID) error {
	query := "DELETE FROM notes WHERE id = $1"

	result, err := r.db.Exec(query, id)
	if err != nil {
		return wrapDBErr("delete note "+id.String(), err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return wrapDBErr("delete note "+id.String()+": rows affected", err)
	}

	if rowsAffected == 0 {
		return commonconstants.ErrNotFound
	}

	return nil
}

// DeleteByPlanID removes all notes for a specific plan
func (r *Repository) DeleteByPlanID(planID uuid.UUID) error {
	query := "DELETE FROM notes WHERE plan_id = $1"

	_, err := r.db.Exec(query, planID)
	if err != nil {
		return wrapDBErr("delete notes by plan "+planID.String(), err)
	}

	return nil
}
