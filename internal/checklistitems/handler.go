package checklistitems

import (
	"context"
	"fmt"
	"net/http"

	"github.com/darkphotonKN/fireplace/internal/constants"
	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service Service
}

type Service interface {
	GetAllByPlanId(ctx context.Context, planId uuid.UUID, scope *string, upcoming *string) ([]*models.ChecklistItem, error)
	GetAllArchivedByPlanId(ctx context.Context, planId uuid.UUID, scope *string) ([]*models.ChecklistItem, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.ChecklistItem, error)
	Create(ctx context.Context, req CreateReq, planID uuid.UUID) (*models.ChecklistItem, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateReq) error
	Delete(ctx context.Context, id uuid.UUID) error
	SetSchedule(ctx context.Context, id uuid.UUID, req SetScheduleReq) error
	Archive(ctx context.Context, id uuid.UUID) error
	GetUpcoming(ctx context.Context, planId uuid.UUID) ([]*models.ChecklistItem, error)
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetAll(c *gin.Context) {
	planIdParam := c.Param("id")

	planId, err := uuid.Parse(planIdParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect uuid format."})
		return
	}

	scope := c.Query("scope")
	var scopePtr *string
	if scope != "" {
		scopePtr = &scope
	}

	items, err := h.service.GetAllByPlanId(c.Request.Context(), planId, scopePtr, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get checklist items. Error:" + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "successfully retrieved all checklist items.", "result": items})
}

func (h *Handler) GetAllArchived(c *gin.Context) {
	planIdParam := c.Param("id")

	planId, err := uuid.Parse(planIdParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect uuid format."})
		return
	}

	scope := c.Query("scope")
	var scopePtr *string
	if scope != "" {
		scopePtr = &scope
	}

	items, err := h.service.GetAllArchivedByPlanId(c.Request.Context(), planId, scopePtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get archived checklist items. Error:" + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "successfully retrieved archived checklist items.", "result": items})
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	planIDParam := c.Param("id")

	planID, err := uuid.Parse(planIDParam)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect uuid format provided plan id in the param."})
		return
	}

	newItem, err := h.service.Create(c.Request.Context(), req, planID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create checklist item. Error: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"statusCode:": http.StatusOK, "message": "successfully created checklist item.", "result": newItem})
}

func (h *Handler) Update(c *gin.Context) {
	idParam := c.Param("checklist_id")

	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect uuid format."})
		return
	}

	var req UpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.service.Update(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update checklist item. Error: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"statusCode:": http.StatusOK, "message": "successfully update checklist item.", "result": constants.UpdateStatusSuccess})
}

func (h *Handler) Delete(c *gin.Context) {
	idStr := c.Param("checklist_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format", "result": constants.UpdateStatusFailure})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete checklist item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "successfully deleted checklist item.", "result": constants.UpdateStatusSuccess})
}

func (h *Handler) GetByID(c *gin.Context) {
	idStr := c.Param("checklist_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	item, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get checklist item. Error: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "Successfully retrieved checklist item.", "result": item})
}

func (h *Handler) SetSchedule(c *gin.Context) {
	idStr := c.Param("checklist_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format", "result": constants.UpdateStatusFailure})
		return
	}

	var req SetScheduleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body. Error was: " + err.Error()})
		return
	}

	if err := h.service.SetSchedule(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set schedule on checklist item. Error: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "Successfully set schedule on checklist item.", "result": constants.UpdateStatusSuccess})
}

func (h *Handler) Archive(c *gin.Context) {
	idStr := c.Param("checklist_id")
	id, err := uuid.Parse(idStr)
	fmt.Println("Checklist Id was:", idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format", "result": constants.UpdateStatusFailure})
		return
	}

	if err := h.service.Archive(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to archive checklist item. Error: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "Successfully archived checklist item.", "result": constants.UpdateStatusSuccess})
}

// GetUpcoming returns all upcoming tasks for a plan
func (h *Handler) GetUpcoming(c *gin.Context) {
	planIdParam := c.Param("id")

	planId, err := uuid.Parse(planIdParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect uuid format."})
		return
	}

	items, err := h.service.GetUpcoming(c.Request.Context(), planId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get upcoming tasks. Error:" + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "Successfully retrieved upcoming tasks.", "result": items})
}
