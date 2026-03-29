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

// plans including shared plans
type AllPlansResponse struct {
	PlansOwned  []*models.Plan
	SharedPlans []*models.Plan
}
