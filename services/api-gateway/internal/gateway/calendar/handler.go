package calendargw

import (
	"fmt"
	"net/http"
	"time"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	client *Client
}

func NewHandler(client *Client) *Handler {
	return &Handler{client: client}
}

// GetCalendar handles GET /api/plans/:id/calendar?view=<week|month>&date=<...>.
// Same query param + response shape as the pre-extraction monolith handler,
// so the FE doesn't notice the move to a remote service.
func (h *Handler) GetCalendar(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierr.Fail(c, "calendargw: get calendar", fmt.Errorf("%w: plan id %q", commonconstants.ErrUUIDCouldNotBeParsed, c.Param("id")))
		return
	}

	userID, err := auth.GetUserID(c)
	if err != nil {
		apierr.Fail(c, "calendargw: get calendar", fmt.Errorf("%w: %v", commonconstants.ErrUnauthorized, err))
		return
	}

	view := c.Query("view")
	date := c.Query("date")
	if date == "" {
		if view == "week" {
			date = time.Now().UTC().Format("2006-01-02")
		} else {
			date = time.Now().UTC().Format("2006-01")
		}
	}

	resp, err := h.client.GetCalendar(c.Request.Context(), planID, userID, view, date)
	if err != nil {
		apierr.Fail(c, "calendargw: get calendar", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"statusCode": http.StatusOK,
		"message":    "Calendar entries retrieved successfully",
		"result":     resp,
	})
}
