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

// CreateChecklistReq is the body for POST /plans/{id}/checklists.
//
// Validation at this edge is shape-only (gin binding). The conditional/domain
// rules below are enforced downstream by plan-service and are described here in
// prose — they are deliberately NOT encoded in the schema (see
// docs/api-conventions.md):
//   - A parent referenced by parentId must be a top-level item in the SAME plan
//     (two-tier maximum) — see plan-service checklistitem.validateParent.
type CreateChecklistReq struct {
	// Free-text item description.
	Description string `json:"description" example:"Write the integration test plan"`
	// Item scope. Optional; plan-service defaults it to "longterm".
	Scope *string `json:"scope,omitempty" enums:"daily,longterm" example:"daily"`
	// Item type. Optional; plan-service defaults it to "task".
	Type *string `json:"type,omitempty" enums:"task,note" example:"task"`
	// Optional parent item id for nesting. Must reference a top-level item in the
	// same plan (enforced downstream).
	ParentID *uuid.UUID `json:"parentId,omitempty" format:"uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// UpdateChecklistReq is the body for PATCH /plans/{id}/checklists/{checklist_id}.
// Every field is optional (partial update). parentId uses three-state JSON
// semantics (omit = leave, null = clear, value = set).
//
// Conditional/domain rules enforced downstream (prose, not schema):
//   - Converting a parent item to type "note" is rejected if it has children.
//   - Re-parenting is rejected unless the target is a top-level item in the same
//     plan and this item has no children of its own (two-tier maximum).
type UpdateChecklistReq struct {
	Description *string `json:"description,omitempty" example:"Updated description"`
	Done        *bool   `json:"done,omitempty" example:"true"`
	Scope       *string `json:"scope,omitempty" enums:"daily,longterm" example:"longterm"`
	Archived    *bool   `json:"archived,omitempty" example:"false"`
	Type        *string `json:"type,omitempty" enums:"task,note" example:"task"`
	// Parent item id: send a UUID to re-parent, null to clear, or omit to leave
	// unchanged. swaggertype renders the three-state OptUUID as a plain string.
	ParentID OptUUID `json:"parentId" swaggertype:"string" format:"uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// UpdateDatesReq is the body for PATCH /plans/{id}/checklists/{checklist_id}/dates.
//
// Both fields are OPTIONAL and use three-state JSON semantics: omit the key to
// leave a date unchanged, send null to clear it, or send a "YYYY-MM-DD" string
// to set it.
//
// Conditional domain rule (enforced downstream by plan-service, NOT in this
// schema): when both dates are present, startDate must be on or before dueDate.
// This rule is intentionally expressed in prose rather than as a schema
// constraint — the spec is a shape contract, not a validator. See
// docs/api-conventions.md.
type UpdateDatesReq struct {
	// New start date "YYYY-MM-DD", null to clear, or omit to leave unchanged.
	StartDate OptDate `json:"startDate" swaggertype:"string" format:"date" example:"2026-06-01"`
	// New due date "YYYY-MM-DD", null to clear, or omit to leave unchanged. Must
	// be on or after startDate when both are present (enforced downstream).
	DueDate OptDate `json:"dueDate" swaggertype:"string" format:"date" example:"2026-06-15"`
}

type ArchiveReq struct {
	Archived bool `json:"archived" example:"true"`
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
	ID          uuid.UUID  `json:"id" format:"uuid" example:"3f1a2b3c-4d5e-6f70-8192-a3b4c5d6e7f8"`
	Description string     `json:"description" example:"Write the integration test plan"`
	Done        bool       `json:"done" example:"false"`
	Sequence    string     `json:"sequence" example:"1"`
	Type        string     `json:"type" enums:"task,note" example:"task"`
	ParentID    *uuid.UUID `json:"parentId,omitempty" format:"uuid"`
	StartDate   *time.Time `json:"startDate,omitempty" format:"date-time"`
	DueDate     *time.Time `json:"dueDate,omitempty" format:"date-time"`
	Scope       string     `json:"scope" enums:"daily,longterm" example:"daily"`
	Archived    bool       `json:"archived" example:"false"`
	CreatedAt   time.Time  `json:"created_at" format:"date-time"`
	UpdatedAt   time.Time  `json:"updated_at" format:"date-time"`
	PlanID      uuid.UUID  `json:"planId" format:"uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// --- Swagger response envelopes (doc-only) ---
//
// The handlers emit the gateway's standard gin.H envelope
// ({ "statusCode": <int>, "result": <payload> } on success, and
// { "statusCode": <int>, "message": <string> } on error via apierr.Fail).
// These structs exist purely so the generated spec mirrors that wire shape;
// they are never constructed in code.

// ChecklistResponse is the success envelope for a single checklist item.
type ChecklistResponse struct {
	StatusCode int           `json:"statusCode" example:"200"`
	Result     ChecklistResp `json:"result"`
}

// ChecklistListResponse is the success envelope for a list of checklist items.
type ChecklistListResponse struct {
	StatusCode int             `json:"statusCode" example:"200"`
	Result     []ChecklistResp `json:"result"`
}

// MessageResponse is the success envelope for endpoints that return only a
// status message (e.g. delete).
type MessageResponse struct {
	StatusCode int    `json:"statusCode" example:"200"`
	Message    string `json:"message" example:"Item deleted."`
}

// ErrorResponse is the standard gateway error body produced by apierr.Fail.
type ErrorResponse struct {
	StatusCode int    `json:"statusCode" example:"404"`
	Message    string `json:"message" example:"not found"`
}
