package plangw

import (
	"fmt"
	"net/http"
	"strconv"

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

// --- Plan routes ---

func (h *Handler) CreatePlan(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		writeUnauthorized(c, err)
		return
	}
	var req CreatePlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": err.Error()})
		return
	}
	plan, err := h.client.CreatePlan(c.Request.Context(), userID, req)
	if err != nil {
		writeGRPCError(c, err, "create plan")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"statusCode": http.StatusCreated, "message": "Plan created.", "result": plan})
}

func (h *Handler) GetPlanByID(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		writeUnauthorized(c, err)
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": "invalid plan id"})
		return
	}
	plan, err := h.client.GetPlan(c.Request.Context(), id, userID)
	if err != nil {
		writeGRPCError(c, err, "get plan")
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": plan})
}

func (h *Handler) ListPlans(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		writeUnauthorized(c, err)
		return
	}
	plans, err := h.client.ListPlans(c.Request.Context(), userID)
	if err != nil {
		writeGRPCError(c, err, "list plans")
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": plans})
}

func (h *Handler) ListSharedPlans(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		writeUnauthorized(c, err)
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.Query("offset"))
	plans, err := h.client.ListSharedPlans(c.Request.Context(), userID, limit, offset)
	if err != nil {
		writeGRPCError(c, err, "list shared plans")
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": plans})
}

func (h *Handler) SearchPlans(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		writeUnauthorized(c, err)
		return
	}
	var params SearchParam
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": err.Error()})
		return
	}
	results, err := h.client.SearchPlans(c.Request.Context(), userID, params)
	if err != nil {
		writeGRPCError(c, err, "search plans")
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": results})
}

func (h *Handler) UpdatePlan(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		writeUnauthorized(c, err)
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": "invalid plan id"})
		return
	}
	var req UpdatePlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": err.Error()})
		return
	}
	plan, err := h.client.UpdatePlan(c.Request.Context(), id, userID, req)
	if err != nil {
		writeGRPCError(c, err, "update plan")
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": plan})
}

func (h *Handler) ToggleDailyReset(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		writeUnauthorized(c, err)
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": "invalid plan id"})
		return
	}
	plan, err := h.client.ToggleDailyReset(c.Request.Context(), id, userID)
	if err != nil {
		writeGRPCError(c, err, "toggle daily_reset")
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": plan})
}

func (h *Handler) DeletePlan(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		writeUnauthorized(c, err)
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": "invalid plan id"})
		return
	}
	if err := h.client.DeletePlan(c.Request.Context(), id, userID); err != nil {
		writeGRPCError(c, err, "delete plan")
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Plan deleted."})
}

// --- Checklist routes ---

func (h *Handler) CreateChecklist(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		writeUnauthorized(c, err)
		return
	}
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": "invalid plan id"})
		return
	}
	var req CreateChecklistReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": err.Error()})
		return
	}
	item, err := h.client.CreateChecklist(c.Request.Context(), planID, userID, req)
	if err != nil {
		writeGRPCError(c, err, "create checklist item")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"statusCode": http.StatusCreated, "result": item})
}

func (h *Handler) GetChecklist(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		writeUnauthorized(c, err)
		return
	}
	id, err := uuid.Parse(c.Param("checklist_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": "invalid checklist id"})
		return
	}
	item, err := h.client.GetChecklist(c.Request.Context(), id, userID)
	if err != nil {
		writeGRPCError(c, err, "get checklist item")
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": item})
}

func (h *Handler) ListChecklists(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		writeUnauthorized(c, err)
		return
	}
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": "invalid plan id"})
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
		writeGRPCError(c, err, "list checklist items")
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": items})
}

func (h *Handler) ListArchivedChecklists(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		writeUnauthorized(c, err)
		return
	}
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": "invalid plan id"})
		return
	}
	items, err := h.client.ListArchivedChecklists(c.Request.Context(), planID, userID)
	if err != nil {
		writeGRPCError(c, err, "list archived checklist items")
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": items})
}

func (h *Handler) ListUpcomingChecklists(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		writeUnauthorized(c, err)
		return
	}
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": "invalid plan id"})
		return
	}
	items, err := h.client.ListUpcomingChecklists(c.Request.Context(), planID, userID)
	if err != nil {
		writeGRPCError(c, err, "list upcoming checklist items")
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": items})
}

func (h *Handler) UpdateChecklist(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		writeUnauthorized(c, err)
		return
	}
	id, err := uuid.Parse(c.Param("checklist_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": "invalid checklist id"})
		return
	}
	var req UpdateChecklistReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": err.Error()})
		return
	}
	item, err := h.client.UpdateChecklist(c.Request.Context(), id, userID, req)
	if err != nil {
		writeGRPCError(c, err, "update checklist item")
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": item})
}

func (h *Handler) UpdateChecklistDates(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		writeUnauthorized(c, err)
		return
	}
	id, err := uuid.Parse(c.Param("checklist_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": "invalid checklist id"})
		return
	}
	var req UpdateDatesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": err.Error()})
		return
	}
	item, err := h.client.UpdateChecklistDates(c.Request.Context(), id, userID, req)
	if err != nil {
		writeGRPCError(c, err, "update checklist dates")
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": item})
}

func (h *Handler) ArchiveChecklist(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		writeUnauthorized(c, err)
		return
	}
	id, err := uuid.Parse(c.Param("checklist_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": "invalid checklist id"})
		return
	}
	// The monolith's Archive endpoint always set archived=true; preserve that
	// behaviour by ignoring the request body. (A future "unarchive" endpoint
	// could accept {"archived": false}.)
	item, err := h.client.ArchiveChecklist(c.Request.Context(), id, userID, true)
	if err != nil {
		writeGRPCError(c, err, "archive checklist item")
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "result": item})
}

func (h *Handler) DeleteChecklist(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		writeUnauthorized(c, err)
		return
	}
	id, err := uuid.Parse(c.Param("checklist_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": "invalid checklist id"})
		return
	}
	if err := h.client.DeleteChecklist(c.Request.Context(), id, userID); err != nil {
		writeGRPCError(c, err, "delete checklist item")
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Item deleted."})
}

// --- error helpers ---

func writeUnauthorized(c *gin.Context, err error) {
	c.JSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": fmt.Sprintf("Unauthorized: %s", err.Error())})
}

func writeGRPCError(c *gin.Context, err error, op string) {
	code := httpStatusFromGRPC(err)
	c.JSON(code, gin.H{"statusCode": code, "message": fmt.Sprintf("Error during %s: %s", op, err.Error())})
}

func httpStatusFromGRPC(err error) int {
	if err == nil {
		return http.StatusOK
	}
	s, ok := status.FromError(err)
	if !ok {
		return http.StatusInternalServerError
	}
	switch s.Code() {
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
