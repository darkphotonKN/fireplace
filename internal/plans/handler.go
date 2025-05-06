package plans

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetById(c *gin.Context) {
	// get id from param
	idParam := c.Param("id")

	// check that its a valid uuid
	id, err := uuid.Parse(idParam)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": fmt.Sprintf("Error with id %s, not a valid uuid.", idParam)})
		// return to stop flow of function after error response
		return
	}

	plan, err := h.service.GetById(c.Request.Context(), id)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": fmt.Sprintf("Error when attempting to get a plan with id %s: %s", idParam, err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "Successfully retrieved plan.",
		"result": plan})
}

// Create adds a new plan
func (h *Handler) Create(c *gin.Context) {
	// TODO: static now, will come from jwt in future
	userId, err := uuid.Parse("11111111-1111-1111-1111-111111111111")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"statusCode": http.StatusInternalServerError, "message": "Failed to parse user ID", "error": err.Error()})
		return
	}

	var req CreatePlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": "Invalid request body", "error": err.Error()})
		return
	}

	// Create the plan with the user ID from static source (future: JWT)
	newPlan, err := h.service.Create(c.Request.Context(), req, userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"statusCode": http.StatusInternalServerError, "message": "Failed to create plan", "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"statusCode": http.StatusCreated, "message": "Successfully created plan", "result": newPlan})
}

// Update modifies an existing plan (only name, description, and focus fields)
func (h *Handler) Update(c *gin.Context) {
	// TODO: static now, will come from jwt in future
	userId, err := uuid.Parse("11111111-1111-1111-1111-111111111111")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"statusCode": http.StatusInternalServerError, "message": "Failed to parse user ID", "error": err.Error()})
		return
	}

	// Get plan ID from URL parameter
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": fmt.Sprintf("Error with id %s, not a valid uuid.", idParam)})
		return
	}

	// Parse request body
	var req UpdatePlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": "Invalid request body", "error": err.Error()})
		return
	}

	// Update the plan
	if err := h.service.Update(c.Request.Context(), id, req, userId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"statusCode": http.StatusInternalServerError, "message": "Failed to update plan", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully updated plan"})
}

// GetAll returns all plans
func (h *Handler) GetAll(c *gin.Context) {
	// TODO: static now, will come from jwt in future
	userId, err := uuid.Parse("11111111-1111-1111-1111-111111111111")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"statusCode": http.StatusInternalServerError, "message": "Failed to parse user ID", "error": err.Error()})
		return
	}

	plans, err := h.service.GetAll(c.Request.Context(), userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"statusCode": http.StatusInternalServerError, "message": "Failed to get plans", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully retrieved all plans", "result": plans})
}
