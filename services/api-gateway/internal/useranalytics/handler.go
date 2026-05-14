package useranalytics

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service Service
}

type Service interface {
	GetUserAnalytics(ctx context.Context, userID uuid.UUID, date time.Time) (*UserAnalytics, error)
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetUserAnalytics(c *gin.Context) {
	// TODO: Implement
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

