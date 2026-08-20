package useranalytics

import (
	"time"

	"github.com/google/uuid"
)

type UserAnalytics struct {
	ID               uuid.UUID `json:"id" db:"id"`
	UserID           uuid.UUID `json:"userId" db:"user_id"`
	Date             time.Time `json:"date" db:"date"`
	TasksCompleted   int       `json:"tasksCompleted" db:"tasks_completed"`
	TasksTotal       int       `json:"tasksTotal" db:"tasks_total"`
	CompletionRate   float64   `json:"completionRate" db:"completion_rate"`
	CurrentStreak    int       `json:"currentStreak" db:"current_streak"`
	ActivePlansCount int       `json:"activePlansCount" db:"active_plans_count"`
	CreatedAt        time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time `json:"updatedAt" db:"updated_at"`
}

type GetUserAnalyticsReq struct {
	UserID uuid.UUID `json:"userId" binding:"required"`
	Date   time.Time `json:"date" binding:"required"`
}
