package plans

type CreatePlanReq struct {
	Name        string    `json:"name" binding:"required"`
	Focus       string    `json:"focus" binding:"required"`
	Description string    `json:"description"`
	PlanType    string    `json:"planType" binding:"required"`
}

type UpdatePlanReq struct {
	Name        string `json:"name" binding:"required"`
	Focus       string `json:"focus"`
	Description string `json:"description"`
}

