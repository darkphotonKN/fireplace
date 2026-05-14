package authgw

import (
	"fmt"
	"net/http"

	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler is the gateway's HTTP shim for auth-service. Each handler mirrors
// the monolith's previous response shape exactly so existing frontend clients
// don't need to change.
type Handler struct {
	client *Client
}

func NewHandler(client *Client) *Handler {
	return &Handler{client: client}
}

func (h *Handler) Create(c *gin.Context) {
	var req SignUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": "Error with parsing payload as JSON."})
		return
	}
	if _, err := h.client.SignUp(c.Request.Context(), &req); err != nil {
		code := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"statusCode:": code, "message": fmt.Sprintf("Error when attempting to create user: %s", err.Error())})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"statusCode:": http.StatusCreated, "message": "Successfully created user."})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": fmt.Sprintf("Error when unmarshalling json payload: %s\n", err)})
		return
	}
	resp, err := h.client.SignIn(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": fmt.Sprintf("Error when attempting to login user: %s\n", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully logged in.", "result": resp})
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": fmt.Sprintf("Unauthorized: %s", err.Error())})
		return
	}
	profile, err := h.client.GetProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"statusCode": http.StatusInternalServerError, "message": fmt.Sprintf("Error fetching profile: %s", err.Error())})
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully retrieved profile.", "result": profile})
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": fmt.Sprintf("Unauthorized: %s", err.Error())})
		return
	}
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": fmt.Sprintf("Invalid request body: %s", err.Error())})
		return
	}
	profile, err := h.client.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": fmt.Sprintf("Error updating profile: %s", err.Error())})
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully updated profile.", "result": profile})
}

func (h *Handler) GetById(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": fmt.Sprintf("Error with id %s, not a valid uuid.", idParam)})
		return
	}
	user, err := h.client.GetById(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": fmt.Sprintf("Error when attempting to get user with id %s %s", idParam, err.Error())})
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "Successfully retreived user.", "result": user})
}

func (h *Handler) GetAll(c *gin.Context) {
	users, err := h.client.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": fmt.Sprintf("Error when attempting to get all users: %s:\n", err.Error())})
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "Successfully retrieved users.", "result": users})
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
