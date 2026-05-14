package notes

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Note types
const (
	TypeUser       = "user"
	TypeAI         = "ai"
	TypeWarning    = "warning"
	TypeInsight    = "insight"
	TypeSuggestion = "suggestion"
)

// Note priorities
const (
	PriorityLow      = "low"
	PriorityMedium   = "medium"
	PriorityHigh     = "high"
	PriorityCritical = "critical"
)

// AIMetadata contains information about AI-generated notes
type AIMetadata struct {
	GeneratedFrom string  `json:"generatedFrom"`
	Confidence    float64 `json:"confidence"`
	SourceContext string  `json:"sourceContext"`
	GeneratedAt   string  `json:"generatedAt"`
}

// Note represents a note entity in the database
type Note struct {
	ID             uuid.UUID      `db:"id" json:"id"`
	PlanID         uuid.UUID      `db:"plan_id" json:"planId"`
	Content        string         `db:"content" json:"content"`
	Type           string         `db:"type" json:"type"`
	Priority       string         `db:"priority" json:"priority"`
	Tags           pq.StringArray `db:"tags" json:"tags"`
	RelatedTaskIDs pq.StringArray `db:"related_task_ids" json:"relatedTaskIds"`
	IsRead         bool           `db:"is_read" json:"isRead"`
	IsDismissed    bool           `db:"is_dismissed" json:"isDismissed"`
	AIMetadata     *AIMetadata    `db:"ai_metadata" json:"aiMetadata,omitempty"`
	CreatedAt      time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time      `db:"updated_at" json:"updatedAt"`
}

// CreateNoteReq represents a request to create a new note
type CreateNoteReq struct {
	Content        string     `json:"content" binding:"required"`
	Type           string     `json:"type,omitempty"`
	Priority       string     `json:"priority,omitempty"`
	Tags           []string   `json:"tags,omitempty"`
	RelatedTaskIDs []string   `json:"relatedTaskIds,omitempty"`
	AIMetadata     *AIMetadata `json:"aiMetadata,omitempty"`
}

// UpdateNoteReq represents a request to update an existing note
type UpdateNoteReq struct {
	Content     *string  `json:"content,omitempty"`
	Priority    *string  `json:"priority,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	IsRead      *bool    `json:"isRead,omitempty"`
	IsDismissed *bool    `json:"isDismissed,omitempty"`
}

// GenerateAINotesReq represents a request to generate AI notes
type GenerateAINotesReq struct {
	RequestType string `json:"requestType" binding:"required"` // "suggestion", "warning", "insight"
}

// FilterOptions for querying notes
type FilterOptions struct {
	Type           string   `json:"type,omitempty"`
	Priority       string   `json:"priority,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	IsRead         *bool    `json:"isRead,omitempty"`
	IsDismissed    *bool    `json:"isDismissed,omitempty"`
	RelatedTaskID  string   `json:"relatedTaskId,omitempty"`
}