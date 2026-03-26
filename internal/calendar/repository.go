package calendar

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// CalendarRepository defines the data access interface for calendar entries.
type CalendarRepository interface {
	GetByMonth(planID uuid.UUID, startDate, endDate time.Time) ([]CalendarEntry, error)
}

// Repository implements CalendarRepository with a PostgreSQL backend.
type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByMonth(planID uuid.UUID, startDate, endDate time.Time) ([]CalendarEntry, error) {
	query := `
		SELECT
			ce.id, ce.plan_id, ce.checklist_item_id, ce.entry_type,
			ce.scheduled_date, ce.position, ce.pinned,
			ce.rec_title, ce.rec_url, ce.rec_description,
			ce.created_at, ce.updated_at,
			ci.description, ci.done
		FROM calendar_entries ce
		LEFT JOIN checklist_items ci ON ce.checklist_item_id = ci.id
		WHERE ce.plan_id = $1
		  AND ce.scheduled_date >= $2
		  AND ce.scheduled_date <= $3
		ORDER BY ce.scheduled_date ASC, ce.position ASC
	`

	rows, err := r.db.Queryx(query, planID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar entries: %w", err)
	}
	defer rows.Close()

	var entries []CalendarEntry
	for rows.Next() {
		var entry CalendarEntry
		if err := rows.StructScan(&entry); err != nil {
			return nil, fmt.Errorf("failed to scan calendar entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}
