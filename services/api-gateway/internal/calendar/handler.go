package calendar

import (
	"errors"
	"net/http"
	"time"

	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/constants"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetCalendar handles GET /api/plans/:id/calendar?view=<week|month>&date=<...>
func (h *Handler) GetCalendar(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan ID"})
		return
	}

	userID, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	view := c.Query("view")
	date := c.Query("date")
	if date == "" {
		// Default: current month for month view, today's date for week view.
		if view == "week" {
			date = time.Now().UTC().Format("2006-01-02")
		} else {
			date = time.Now().UTC().Format("2006-01")
		}
	}

	resp, err := h.service.GetCalendar(c.Request.Context(), planID, userID, view, date)
	if err != nil {
		switch {
		case errors.Is(err, constants.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, constants.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"statusCode": http.StatusOK,
		"message":    "Calendar entries retrieved successfully",
		"result":     resp,
	})
}
