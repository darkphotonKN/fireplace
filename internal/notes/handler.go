package notes

import (
	"net/http"

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
	planIDStr := c.Param("id")
	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Invalid plan ID",
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	var req CreateNoteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      err.Error(),
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	note, err := h.service.CreateNote(planID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      err.Error(),
			"statusCode": http.StatusInternalServerError,
		})
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
	noteIDStr := c.Param("noteId")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Invalid note ID",
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	note, err := h.service.GetNoteByID(noteID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "note not found" {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{
			"error":      err.Error(),
			"statusCode": status,
		})
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
	planIDStr := c.Param("id")
	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Invalid plan ID",
			"statusCode": http.StatusBadRequest,
		})
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      err.Error(),
			"statusCode": http.StatusInternalServerError,
		})
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
	noteIDStr := c.Param("noteId")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Invalid note ID",
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	var req UpdateNoteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      err.Error(),
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	note, err := h.service.UpdateNote(noteID, &req)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "note not found" {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{
			"error":      err.Error(),
			"statusCode": status,
		})
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
	noteIDStr := c.Param("noteId")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Invalid note ID",
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	err = h.service.DeleteNote(noteID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "note not found" {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{
			"error":      err.Error(),
			"statusCode": status,
		})
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
	planIDStr := c.Param("id")
	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Invalid plan ID",
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	var req GenerateAINotesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no request body, generate all types
		req.RequestType = "all"
	}

	notes, err := h.service.GenerateAINotes(planID, req.RequestType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      err.Error(),
			"statusCode": http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "AI notes generated successfully",
		"result":     notes,
		"statusCode": http.StatusOK,
	})
}