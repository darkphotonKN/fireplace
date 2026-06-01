package notes

import (
	"fmt"
	"net/http"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

// NewHandler creates a new notes handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create handles POST /api/plans/:id/notes
func (h *Handler) Create(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierr.Fail(c, "notes: create", fmt.Errorf("%w: plan id %q", commonconstants.ErrUUIDCouldNotBeParsed, c.Param("id")))
		return
	}

	var req CreateNoteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, "notes: create", fmt.Errorf("%w: malformed request body: %v", commonconstants.ErrInvalidInput, err))
		return
	}

	note, err := h.service.CreateNote(planID, &req)
	if err != nil {
		apierr.Fail(c, "notes: create", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Note created successfully",
		"result":     note,
		"statusCode": http.StatusCreated,
	})
}

// GetByID handles GET /api/plans/:id/notes/:noteId
func (h *Handler) GetByID(c *gin.Context) {
	noteID, err := uuid.Parse(c.Param("noteId"))
	if err != nil {
		apierr.Fail(c, "notes: get", fmt.Errorf("%w: note id %q", commonconstants.ErrUUIDCouldNotBeParsed, c.Param("noteId")))
		return
	}

	note, err := h.service.GetNoteByID(noteID)
	if err != nil {
		apierr.Fail(c, "notes: get", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Note retrieved successfully",
		"result":     note,
		"statusCode": http.StatusOK,
	})
}

// GetAll handles GET /api/plans/:id/notes
func (h *Handler) GetAll(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierr.Fail(c, "notes: list", fmt.Errorf("%w: plan id %q", commonconstants.ErrUUIDCouldNotBeParsed, c.Param("id")))
		return
	}

	// Parse filter options from query parameters
	filters := &FilterOptions{}

	if typeParam := c.Query("type"); typeParam != "" {
		filters.Type = typeParam
	}

	if priorityParam := c.Query("priority"); priorityParam != "" {
		filters.Priority = priorityParam
	}

	if isReadParam := c.Query("isRead"); isReadParam != "" {
		isRead := isReadParam == "true"
		filters.IsRead = &isRead
	}

	if isDismissedParam := c.Query("isDismissed"); isDismissedParam != "" {
		isDismissed := isDismissedParam == "true"
		filters.IsDismissed = &isDismissed
	}

	if taskIdParam := c.Query("relatedTaskId"); taskIdParam != "" {
		filters.RelatedTaskID = taskIdParam
	}

	// Get tags from query (comma-separated)
	if tagsParam := c.Query("tags"); tagsParam != "" {
		// Split tags by comma
		tags := []string{}
		for _, tag := range c.QueryArray("tags") {
			if tag != "" {
				tags = append(tags, tag)
			}
		}
		if len(tags) > 0 {
			filters.Tags = tags
		}
	}

	notes, err := h.service.GetNotesByPlanID(planID, filters)
	if err != nil {
		apierr.Fail(c, "notes: list", err)
		return
	}

	// Return empty array if no notes found
	if notes == nil {
		notes = []Note{}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Notes retrieved successfully",
		"result":     notes,
		"statusCode": http.StatusOK,
	})
}

// Update handles PATCH /api/plans/:id/notes/:noteId
func (h *Handler) Update(c *gin.Context) {
	noteID, err := uuid.Parse(c.Param("noteId"))
	if err != nil {
		apierr.Fail(c, "notes: update", fmt.Errorf("%w: note id %q", commonconstants.ErrUUIDCouldNotBeParsed, c.Param("noteId")))
		return
	}

	var req UpdateNoteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, "notes: update", fmt.Errorf("%w: malformed request body: %v", commonconstants.ErrInvalidInput, err))
		return
	}

	note, err := h.service.UpdateNote(noteID, &req)
	if err != nil {
		apierr.Fail(c, "notes: update", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Note updated successfully",
		"result":     note,
		"statusCode": http.StatusOK,
	})
}

// Delete handles DELETE /api/plans/:id/notes/:noteId
func (h *Handler) Delete(c *gin.Context) {
	noteID, err := uuid.Parse(c.Param("noteId"))
	if err != nil {
		apierr.Fail(c, "notes: delete", fmt.Errorf("%w: note id %q", commonconstants.ErrUUIDCouldNotBeParsed, c.Param("noteId")))
		return
	}

	if err := h.service.DeleteNote(noteID); err != nil {
		apierr.Fail(c, "notes: delete", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Note deleted successfully",
		"result":     "success",
		"statusCode": http.StatusOK,
	})
}

// GenerateAINotes handles POST /api/plans/:id/notes/generate-ai
func (h *Handler) GenerateAINotes(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierr.Fail(c, "notes: generate ai", fmt.Errorf("%w: plan id %q", commonconstants.ErrUUIDCouldNotBeParsed, c.Param("id")))
		return
	}

	var req GenerateAINotesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no request body, generate all types
		req.RequestType = "all"
	}

	notes, err := h.service.GenerateAINotes(planID, req.RequestType)
	if err != nil {
		apierr.Fail(c, "notes: generate ai", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "AI notes generated successfully",
		"result":     notes,
		"statusCode": http.StatusOK,
	})
}
