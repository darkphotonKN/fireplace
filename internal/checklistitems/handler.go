package checklistitems

import (
	"context"
	"net/http"

	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service Service
}

type Service interface {
	GetAll(ctx context.Context) ([]*models.ChecklistItem, error)
	Create(ctx context.Context, req CreateReq, planID uuid.UUID) error
	Update(ctx context.Context, id uuid.UUID, req UpdateReq) error
	Delete(ctx context.Context, id uuid.UUID) error
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

type UpdateStatus string

const (
	failure UpdateStatus = "failure"
	success UpdateStatus = "success"
)

// GetAll returns all checklist items
func (h *Handler) GetAll(c *gin.Context) {
	items, err := h.service.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get checklist items"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "successfully retrieved all checklist items.", "result": items})
}

// Create adds a new checklist item
func (h *Handler) Create(c *gin.Context) {
	var req CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	planIDParam := c.Param("plan_id")

	planID, err := uuid.Parse(planIDParam)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect uuid format provided plan id in the param."})
		return
	}

	if err := h.service.Create(c.Request.Context(), req, planID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create checklist item. Error: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"statusCode:": http.StatusOK, "message": "successfully created checklist item.", "result": success})
}

// Update modifies an existing checklist item
func (h *Handler) Update(c *gin.Context) {
	idParam := c.Param("id")

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

	c.JSON(http.StatusCreated, gin.H{"statusCode:": http.StatusOK, "message": "successfully update checklist item.", "result": success})
}

// Delete removes a checklist item by ID
func (h *Handler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format", "result": failure})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete checklist item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "successfully deleted checklist item.", "result": success})
}
