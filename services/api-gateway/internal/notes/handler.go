package notes

import (
	"context"
	"fmt"
	"net/http"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PlanOwnership answers "may this caller act on this plan?".
//
// It is the seam this package was missing entirely. plan-service deliberately
// does NOT enforce ownership on direct reads — it treats the gateway as a
// trusted caller and expects the gateway to assert ownership where it matters
// (plangw.Adapter's own comment says exactly this). Notes never asserted it, so
// every notes endpoint was reachable by any authenticated user for any plan id:
// a two-account probe confirmed a stranger could both READ and WRITE notes on
// someone else's plan.
type PlanOwnership interface {
	AssertPlanOwnership(ctx context.Context, planID, userID uuid.UUID) error
}

type Handler struct {
	service   *Service
	ownership PlanOwnership
}

// NewHandler creates a new notes handler.
//
// ownership is required, not optional. A nil checker fails every request CLOSED
// rather than quietly restoring the unauthorized behaviour this replaced —
// the failure mode of an authorization seam must never be "allow".
func NewHandler(service *Service, ownership PlanOwnership) *Handler {
	return &Handler{service: service, ownership: ownership}
}

// authorize parses the plan id, resolves the caller, and asserts the caller may
// act on that plan. It writes the error response itself, so callers return on
// ok == false.
func (h *Handler) authorize(c *gin.Context, op string) (uuid.UUID, bool) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierr.Fail(c, "notes: "+op, fmt.Errorf("%w: plan id %q", commonconstants.ErrUUIDCouldNotBeParsed, c.Param("id")))
		return uuid.Nil, false
	}
	userID, err := auth.GetUserID(c)
	if err != nil {
		apierr.Fail(c, "notes: "+op, fmt.Errorf("%w: %v", commonconstants.ErrUnauthorized, err))
		return uuid.Nil, false
	}
	if h.ownership == nil {
		apierr.Fail(c, "notes: "+op, fmt.Errorf("%w: ownership checker not configured", commonconstants.ErrForbidden))
		return uuid.Nil, false
	}
	if err := h.ownership.AssertPlanOwnership(c.Request.Context(), planID, userID); err != nil {
		apierr.Fail(c, "notes: "+op, err)
		return uuid.Nil, false
	}
	return planID, true
}

// authorizeNote authorizes the PLAN in the path and then confirms the note
// actually belongs to it.
//
// The plan check alone is not enough. These handlers are keyed by note id and
// the service looks a note up by that id ALONE, so without the second check a
// caller who owns any plan could read or delete a note belonging to someone
// else's plan simply by putting their own plan id in the path. Ownership of the
// container does not imply ownership of an arbitrary item claimed to be in it.
func (h *Handler) authorizeNote(c *gin.Context, op string) (uuid.UUID, bool) {
	planID, ok := h.authorize(c, op)
	if !ok {
		return uuid.Nil, false
	}
	noteID, err := uuid.Parse(c.Param("noteId"))
	if err != nil {
		apierr.Fail(c, "notes: "+op, fmt.Errorf("%w: note id %q", commonconstants.ErrUUIDCouldNotBeParsed, c.Param("noteId")))
		return uuid.Nil, false
	}
	note, err := h.service.GetNoteByID(noteID)
	if err != nil {
		apierr.Fail(c, "notes: "+op, err)
		return uuid.Nil, false
	}
	if note == nil || note.PlanID != planID {
		// Not found rather than forbidden: the note is not in the plan the
		// caller asked about, and saying "wrong plan" would confirm it exists.
		apierr.Fail(c, "notes: "+op, fmt.Errorf("%w: note %s is not in plan %s", commonconstants.ErrNotFound, noteID, planID))
		return uuid.Nil, false
	}
	return noteID, true
}

// Create handles POST /api/plans/:id/notes
func (h *Handler) Create(c *gin.Context) {
	planID, ok := h.authorize(c, "create")
	if !ok {
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
	noteID, ok := h.authorizeNote(c, "get")
	if !ok {
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
	planID, ok := h.authorize(c, "list")
	if !ok {
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
	noteID, ok := h.authorizeNote(c, "update")
	if !ok {
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
	noteID, ok := h.authorizeNote(c, "delete")
	if !ok {
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
	planID, ok := h.authorize(c, "generate ai")
	if !ok {
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
