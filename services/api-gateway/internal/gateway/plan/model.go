package plangw

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// --- HTTP request shapes (kept 1:1 with the monolith API the FE expects) ---

type CreatePlanReq struct {
	Name        string `json:"name" binding:"required"`
	Focus       string `json:"focus" binding:"required"`
	Description string `json:"description"`
	PlanType    string `json:"planType" binding:"required"`
}

type UpdatePlanReq struct {
	Name        *string `json:"name,omitempty"`
	Focus       *string `json:"focus,omitempty"`
	Description *string `json:"description,omitempty"`
	DailyReset  *bool   `json:"dailyReset,omitempty"`
}

type SearchParam struct {
	Term   string `form:"term" binding:"required"`
	Limit  string `form:"limit"`
	Offset string `form:"offset"`
}

// CreateChecklistReq + UpdateChecklistReq match the monolith. UpdateChecklistReq
// uses optUUID for three-state parent_id JSON semantics (absent / null / value).
type CreateChecklistReq struct {
	Description string     `json:"description"`
	Scope       *string    `json:"scope,omitempty"`
	Type        *string    `json:"type,omitempty"`
	ParentID    *uuid.UUID `json:"parentId,omitempty"`
}

type UpdateChecklistReq struct {
	Description *string `json:"description,omitempty"`
	Done        *bool   `json:"done,omitempty"`
	Scope       *string `json:"scope,omitempty"`
	Archived    *bool   `json:"archived,omitempty"`
	Type        *string `json:"type,omitempty"`
	ParentID    OptUUID `json:"parentId"`
}

type UpdateDatesReq struct {
	StartDate OptDate `json:"startDate"`
	DueDate   OptDate `json:"dueDate"`
}

type ArchiveReq struct {
	Archived bool `json:"archived"`
}

// OptUUID distinguishes absent / null / value in JSON. Ported from the
// monolith's checklistitems.optUUID so FE indent/outdent semantics survive.
type OptUUID struct {
	Present bool
	Valid   bool
	Value   uuid.UUID
}

func (o *OptUUID) UnmarshalJSON(data []byte) error {
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

// OptDate likewise distinguishes the three JSON states for date fields.
type OptDate struct {
	Present bool
	Valid   bool
	Value   time.Time
}

func (o *OptDate) UnmarshalJSON(data []byte) error {
	o.Present = true
	if string(data) == "null" {
		o.Valid = false
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("date must be a string or null: %w", err)
	}
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

// --- HTTP response shapes (match monolith JSON field names exactly) ---

type PlanResp struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"userId"`
	Name        string    `json:"name"`
	Focus       string    `json:"focus"`
	Description string    `json:"description"`
	PlanType    string    `json:"planType"`
	DailyReset  bool      `json:"dailyReset"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ChecklistResp struct {
	ID          uuid.UUID  `json:"id"`
	Description string     `json:"description"`
	Done        bool       `json:"done"`
	Sequence    string     `json:"sequence"`
	Type        string     `json:"type"`
	ParentID    *uuid.UUID `json:"parentId,omitempty"`
	StartDate   *time.Time `json:"startDate,omitempty"`
	DueDate     *time.Time `json:"dueDate,omitempty"`
	Scope       string     `json:"scope"`
	Archived    bool       `json:"archived"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	PlanID      uuid.UUID  `json:"planId"`
}
