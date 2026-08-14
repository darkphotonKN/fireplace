package plangw

import (
	"fmt"
	"net/http"

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

// --- error helpers ---
//
// Each helper turns a specific failure into a domain error and hands it to
// apierr.Fail, the gateway's single map-once / log-once boundary. op is a short
// operation label used in both the log line and (via the package prefix) for
// traceability.

func unauthorized(c *gin.Context, op string, err error) {
	apierr.Fail(c, "plangw: "+op, fmt.Errorf("%w: %v", commonconstants.ErrUnauthorized, err))
}

func badID(c *gin.Context, op, field, value string) {
	apierr.Fail(c, "plangw: "+op, fmt.Errorf("%w: %s %q", commonconstants.ErrUUIDCouldNotBeParsed, field, value))
}

func badBody(c *gin.Context, op string, err error) {
	apierr.Fail(c, "plangw: "+op, fmt.Errorf("%w: malformed request body: %v", commonconstants.ErrInvalidInput, err))
}

func fail(c *gin.Context, op string, err error) {
	apierr.Fail(c, "plangw: "+op, err)
}

// Plan routes are SERIALIZED (FS-0004, I-0016) and live in typed_plans.go.
// Only the checklist handlers below are still gin — they serialize in I-0017.

// --- Checklist routes ---

// CreateChecklist creates a checklist item (task or note) under a plan.
//
// `scope` and `type` are optional shape enums. Parent nesting is validated
// downstream by plan-service — a parent must be a top-level item in the same
// plan, two tiers maximum — and that rule is enforced in code, never as a
// schema constraint.
func (h *Handler) CreateChecklist(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		unauthorized(c, "create checklist item", err)
		return
	}
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badID(c, "create checklist item", "plan id", c.Param("id"))
		return
	}
	var req CreateChecklistReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badBody(c, "create checklist item", err)
		return
	}
	item, err := h.client.CreateChecklist(c.Request.Context(), planID, userID, req)
	if err != nil {
		fail(c, "create checklist item", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"statusCode": http.StatusCreated, "result": item})
}

// GetChecklist returns a single checklist item.
func (h *Handler) GetChecklist(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		unauthorized(c, "get checklist item", err)
		return
	}
	id, err := uuid.Parse(c.Param("checklist_id"))
	if err != nil {
		badID(c, "get checklist item", "checklist id", c.Param("checklist_id"))
		return
	}
	item, err := h.client.GetChecklist(c.Request.Context(), id, userID)
	if err != nil {
		fail(c, "get checklist item", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": item})
}

// ListChecklists lists non-archived checklist items for a plan, optionally
// filtered by scope and/or type.
//
// scope is one of daily | longterm; type is one of task | note. Both are
// optional — an absent filter is not the same as an empty one.
func (h *Handler) ListChecklists(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		unauthorized(c, "list checklist items", err)
		return
	}
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badID(c, "list checklist items", "plan id", c.Param("id"))
		return
	}
	var scopePtr, typePtr *string
	if v := c.Query("scope"); v != "" {
		scopePtr = &v
	}
	if v := c.Query("type"); v != "" {
		typePtr = &v
	}
	items, err := h.client.ListChecklists(c.Request.Context(), planID, userID, scopePtr, typePtr)
	if err != nil {
		fail(c, "list checklist items", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": items})
}

// ListArchivedChecklists lists archived checklist items for a plan.
func (h *Handler) ListArchivedChecklists(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		unauthorized(c, "list archived checklist items", err)
		return
	}
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badID(c, "list archived checklist items", "plan id", c.Param("id"))
		return
	}
	items, err := h.client.ListArchivedChecklists(c.Request.Context(), planID, userID)
	if err != nil {
		fail(c, "list archived checklist items", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": items})
}

// ListUpcomingChecklists lists items starting within the next week for a plan.
func (h *Handler) ListUpcomingChecklists(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		unauthorized(c, "list upcoming checklist items", err)
		return
	}
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badID(c, "list upcoming checklist items", "plan id", c.Param("id"))
		return
	}
	items, err := h.client.ListUpcomingChecklists(c.Request.Context(), planID, userID)
	if err != nil {
		fail(c, "list upcoming checklist items", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": items})
}

// UpdateChecklist applies a partial update to a checklist item.
//
// All fields are optional, and `parentId` is three-state: omitted leaves it
// unchanged, null clears it, a value sets it. Domain rules — task→note
// conversion is blocked when the item has children, and re-parenting is
// constrained to two tiers within the same plan — are enforced downstream by
// plan-service, not by this schema.
func (h *Handler) UpdateChecklist(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		unauthorized(c, "update checklist item", err)
		return
	}
	id, err := uuid.Parse(c.Param("checklist_id"))
	if err != nil {
		badID(c, "update checklist item", "checklist id", c.Param("checklist_id"))
		return
	}
	var req UpdateChecklistReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badBody(c, "update checklist item", err)
		return
	}
	item, err := h.client.UpdateChecklist(c.Request.Context(), id, userID, req)
	if err != nil {
		fail(c, "update checklist item", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": item})
}

// UpdateChecklistDates sets/clears the start and/or due dates of an item.
func (h *Handler) UpdateChecklistDates(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		unauthorized(c, "update checklist dates", err)
		return
	}
	id, err := uuid.Parse(c.Param("checklist_id"))
	if err != nil {
		badID(c, "update checklist dates", "checklist id", c.Param("checklist_id"))
		return
	}
	var req UpdateDatesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badBody(c, "update checklist dates", err)
		return
	}
	item, err := h.client.UpdateChecklistDates(c.Request.Context(), id, userID, req)
	if err != nil {
		fail(c, "update checklist dates", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": item})
}

// ArchiveChecklist archives a checklist item.
func (h *Handler) ArchiveChecklist(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		unauthorized(c, "archive checklist item", err)
		return
	}
	id, err := uuid.Parse(c.Param("checklist_id"))
	if err != nil {
		badID(c, "archive checklist item", "checklist id", c.Param("checklist_id"))
		return
	}
	// The monolith's Archive endpoint always set archived=true; preserve that
	// behaviour by ignoring the request body. (A future "unarchive" endpoint
	// could accept {"archived": false}.)
	item, err := h.client.ArchiveChecklist(c.Request.Context(), id, userID, true)
	if err != nil {
		fail(c, "archive checklist item", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": item})
}

// DeleteChecklist deletes a checklist item.
func (h *Handler) DeleteChecklist(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		unauthorized(c, "delete checklist item", err)
		return
	}
	id, err := uuid.Parse(c.Param("checklist_id"))
	if err != nil {
		badID(c, "delete checklist item", "checklist id", c.Param("checklist_id"))
		return
	}
	if err := h.client.DeleteChecklist(c.Request.Context(), id, userID); err != nil {
		fail(c, "delete checklist item", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Item deleted."})
}
