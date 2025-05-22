package insights

import (
	"context"
	"net/http"

	"github.com/darkphotonKN/fireplace/internal/discovery"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service Service
}

type Service interface {
	AutocompleteChecklistSuggestion(currentTxt string) (string, error)
	GenerateSuggestions(ctx context.Context, planId uuid.UUID) (string, error)
	GenerateDailySuggestions(ctx context.Context, planId uuid.UUID) ([]string, error)
	GenerateSuggestedVideoLinks(ctx context.Context, planId uuid.UUID) ([]discovery.Resource, error)
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GenerateSuggestions(c *gin.Context) {
	planIdQuery := c.Query("plan_id")
	planId, err := uuid.Parse(planIdQuery)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": "error when parsing planId from query string: " + err.Error()})
		return
	}

	res, err := h.service.GenerateSuggestions(c.Request.Context(), planId)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": "error when generating completion for checklist: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "successfully generated completion", "result": res})
}

func (h *Handler) GenerateDailySuggestions(c *gin.Context) {
	planIdQuery := c.Query("plan_id")
	planId, err := uuid.Parse(planIdQuery)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": "error when parsing planId from query string: " + err.Error()})
		return
	}

	res, err := h.service.GenerateDailySuggestions(c.Request.Context(), planId)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": "error when generating completion for checklist: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "successfully generated completion", "result": res})
}

func (h *Handler) GenerateSuggestedVideoLinks(c *gin.Context) {
	planIdQuery := c.Query("plan_id")
	planId, err := uuid.Parse(planIdQuery)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": "error when parsing planId from query string: " + err.Error()})
		return
	}

	res, err := h.service.GenerateSuggestedVideoLinks(c.Request.Context(), planId)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": "error when generating suggested video links." + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "Successfully generated suggested video links.", "result": res})
}
