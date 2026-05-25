package calendargw

import (
	"net/http"
	"time"

	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		if view == "week" {
			date = time.Now().UTC().Format("2006-01-02")
		} else {
			date = time.Now().UTC().Format("2006-01")
		}
	}

	resp, err := h.client.GetCalendar(c.Request.Context(), planID, userID, view, date)
	if err != nil {
		code, msg := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"statusCode": http.StatusOK,
		"message":    "Calendar entries retrieved successfully",
		"result":     resp,
	})
}

func httpStatusFromGRPC(err error) (int, string) {
	s, ok := status.FromError(err)
	if !ok {
		return http.StatusInternalServerError, err.Error()
	}
	switch s.Code() {
	case codes.NotFound:
		return http.StatusNotFound, s.Message()
	case codes.PermissionDenied:
		return http.StatusForbidden, s.Message()
	case codes.InvalidArgument:
		return http.StatusBadRequest, s.Message()
	case codes.Unauthenticated:
		return http.StatusUnauthorized, s.Message()
	default:
		return http.StatusInternalServerError, s.Message()
	}
}
