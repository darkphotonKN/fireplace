package plans

import (
	"context"
	"fmt"
	"net/http"

	"github.com/darkphotonKN/fireplace/internal/auth"
	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service Service
}

type Service interface {
	GetById(ctx context.Context, id uuid.UUID) (*models.Plan, error)
	Create(ctx context.Context, req CreatePlanReq, userID uuid.UUID) (*models.Plan, error)
	Update(ctx context.Context, id uuid.UUID, req UpdatePlanReq, userID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	GetAll(ctx context.Context, userID uuid.UUID) ([]*models.Plan, error)
	ToggleDailyReset(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetById(c *gin.Context) {
	idParam := c.Param("id")

	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": fmt.Sprintf("Error with id %s, not a valid uuid.", idParam)})
		return
	}

	plan, err := h.service.GetById(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": fmt.Sprintf("Error when attempting to get a plan with id %s: %s", idParam, err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "Successfully retrieved plan.", "result": plan})
}

func (h *Handler) Create(c *gin.Context) {
	userId, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": "Unauthorized"})
		return
	}

	var req CreatePlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": "Invalid request body", "error": err.Error()})
		return
	}

	newPlan, err := h.service.Create(c.Request.Context(), req, userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"statusCode": http.StatusInternalServerError, "message": "Failed to create plan", "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"statusCode": http.StatusCreated, "message": "Successfully created plan", "result": newPlan})
}

func (h *Handler) Update(c *gin.Context) {
	userId, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": "Unauthorized"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": fmt.Sprintf("Error with id %s, not a valid uuid.", idParam)})
		return
	}

	var req UpdatePlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": "Invalid request body", "error": err.Error()})
		return
	}

	if err := h.service.Update(c.Request.Context(), id, req, userId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"statusCode": http.StatusInternalServerError, "message": "Failed to update plan", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully updated plan"})
}

func (h *Handler) GetAll(c *gin.Context) {
	userId, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": "Unauthorized"})
		return
	}

	plans, err := h.service.GetAll(c.Request.Context(), userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"statusCode": http.StatusInternalServerError, "message": "Failed to get plans", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully retrieved all plans", "result": plans})
}

func (h *Handler) Delete(c *gin.Context) {
	userId, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": "Unauthorized"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": fmt.Sprintf("Error with id %s, not a valid uuid.", idParam)})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id, userId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"statusCode": http.StatusInternalServerError, "message": "Failed to delete plan", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully deleted plan"})
}

func (h *Handler) ToggleDailyReset(c *gin.Context) {
	userId, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": "Unauthorized"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": fmt.Sprintf("Error with id %s, not a valid uuid.", idParam)})
		return
	}

	if err := h.service.ToggleDailyReset(c.Request.Context(), id, userId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"statusCode": http.StatusInternalServerError, "message": "Failed to toggle daily reset", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully toggled daily reset"})
}
