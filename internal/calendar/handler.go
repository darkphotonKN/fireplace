package calendar

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetMonth handles GET /api/plans/:id/calendar?month=YYYY-MM
func (h *Handler) GetMonth(c *gin.Context) {
	planIDStr := c.Param("id")
	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Invalid plan ID",
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	month := c.Query("month")
	if month == "" {
		// Default to current month
		month = time.Now().Format("2006-01")
	}

	resp, err := h.service.GetMonth(planID, month)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      err.Error(),
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Calendar entries retrieved successfully",
		"result":     resp,
		"statusCode": http.StatusOK,
	})
}
