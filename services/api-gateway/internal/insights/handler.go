package insights

import (
	"context"
	"fmt"
	"net/http"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/discovery"
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
	planId, err := uuid.Parse(c.Query("plan_id"))
	if err != nil {
		apierr.Fail(c, "insights: generate suggestions", fmt.Errorf("%w: plan_id %q", commonconstants.ErrUUIDCouldNotBeParsed, c.Query("plan_id")))
		return
	}

	res, err := h.service.GenerateSuggestions(c.Request.Context(), planId)
	if err != nil {
		apierr.Fail(c, "insights: generate suggestions", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "successfully generated completion", "result": res})
}

func (h *Handler) GenerateDailySuggestions(c *gin.Context) {
	planId, err := uuid.Parse(c.Query("plan_id"))
	if err != nil {
		apierr.Fail(c, "insights: generate daily suggestions", fmt.Errorf("%w: plan_id %q", commonconstants.ErrUUIDCouldNotBeParsed, c.Query("plan_id")))
		return
	}

	res, err := h.service.GenerateDailySuggestions(c.Request.Context(), planId)
	if err != nil {
		apierr.Fail(c, "insights: generate daily suggestions", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "successfully generated completion", "result": res})
}

func (h *Handler) GenerateSuggestedVideoLinks(c *gin.Context) {
	planId, err := uuid.Parse(c.Query("plan_id"))
	if err != nil {
		apierr.Fail(c, "insights: suggested video links", fmt.Errorf("%w: plan_id %q", commonconstants.ErrUUIDCouldNotBeParsed, c.Query("plan_id")))
		return
	}

	res, err := h.service.GenerateSuggestedVideoLinks(c.Request.Context(), planId)
	if err != nil {
		apierr.Fail(c, "insights: suggested video links", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully generated suggested video links.", "result": res})
}
