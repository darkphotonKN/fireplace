package plans

import (
	"github.com/darkphotonKN/fireplace/internal/models"
)

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

type SharePlanReq struct {
	UserID string `json:"user_id"`
}

type SearchParam struct {
	Term   string `form:"term" binding:"required"`
	Limit  string `form:"limit"`
	Offset string `form:"offset"`
}

type SearchPlanRes struct {
	models.BaseDBDateModel
	Name        string `db:"name" json:"name"`
	Focus       string `db:"focus" json:"focus"`
	Description string `db:"description" json:"description"`
	PlanType    string `db:"plan_type" json:"planType"`
	DailyReset  bool   `db:"daily_reset" json:"dailyReset"`
}
