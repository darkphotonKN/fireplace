package checklistitems

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CreateReq struct {
	Description string     `json:"description"`
	Scope       *string    `json:"scope,omitempty"`
	Type        *string    `json:"type,omitempty"`
	ParentID    *uuid.UUID `json:"parentId,omitempty"`
}

type UpdateReq struct {
	Description *string `json:"description,omitempty"`
	Done        *bool   `json:"done,omitempty"`
	Sequence    *bool   `json:"sequence,omitempty"`
	Scope       *string `json:"scope,omitempty"`
	Archived    *bool   `json:"archived,omitempty"`
	Type        *string `json:"type,omitempty"`
	// ParentID is three-state: absent leaves the column alone, explicit
	// null clears it (outdent), a UUID sets it (indent).
	ParentID optUUID `json:"parentId"`
}

// optUUID mirrors optDate: distinguishes absent / null / value in JSON.
type optUUID struct {
	Present bool
	Valid   bool
	Value   uuid.UUID
}

func (o *optUUID) UnmarshalJSON(data []byte) error {
	o.Present = true
	if string(data) == "null" {
		o.Valid = false
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("parentId must be a UUID string or null: %w", err)
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return fmt.Errorf("parentId is not a valid UUID: %w", err)
	}
	o.Valid = true
	o.Value = id
	return nil
}

type BatchUpdateReq struct {
	list []UpdateReq
}

// optDate distinguishes three JSON states for a date field:
//
//	{}                       → Present=false, Valid=false  (leave column alone)
//	{"startDate": null}      → Present=true,  Valid=false  (clear column)
//	{"startDate": "2026-..."} → Present=true,  Valid=true   (set column)
type optDate struct {
	Present bool
	Valid   bool
	Value   time.Time
}

func (o *optDate) UnmarshalJSON(data []byte) error {
	o.Present = true
	if string(data) == "null" {
		o.Valid = false
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("date must be a string or null: %w", err)
	}
	// Accept YYYY-MM-DD; tolerate full RFC3339 by truncating to date.
	if i := strings.Index(s, "T"); i > 0 {
		s = s[:i]
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
	if err != nil {
		return fmt.Errorf("date must be YYYY-MM-DD: %w", err)
	}
	o.Valid = true
	o.Value = t
	return nil
}

// UpdateDatesReq is the body of PATCH /checklists/:id/dates.
// Both fields are optional; either may be null to clear.
type UpdateDatesReq struct {
	StartDate optDate `json:"startDate"`
	DueDate   optDate `json:"dueDate"`
}
