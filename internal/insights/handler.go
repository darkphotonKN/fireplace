package insights

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}
type Service interface {
	AutocompleteChecklistSuggestion(currentTxt string) (string, error)
	GenerateChecklistSuggestion() (string, error)
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GenerateChecklistSuggestionHandler(c *gin.Context) {
	res, err := h.service.GenerateChecklistSuggestion()

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": "error when generating completion for checklist: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "successfully generated completion", "result": res})
}
