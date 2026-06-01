package authgw

import (
	"fmt"
	"net/http"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler is the gateway's HTTP shim for auth-service. Each handler mirrors
// the monolith's previous response shape exactly so existing frontend clients
// don't need to change. Error mapping + logging is delegated to apierr.Fail —
// the single place the gateway inspects errors and writes error responses.
type Handler struct {
	client *Client
}

func NewHandler(client *Client) *Handler {
	return &Handler{client: client}
}

func (h *Handler) Create(c *gin.Context) {
	var req SignUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, "authgw: create user", fmt.Errorf("%w: malformed request body: %v", commonconstants.ErrInvalidInput, err))
		return
	}
	if _, err := h.client.SignUp(c.Request.Context(), &req); err != nil {
		apierr.Fail(c, "authgw: create user", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"statusCode": http.StatusCreated, "message": "Successfully created user."})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, "authgw: login", fmt.Errorf("%w: malformed request body: %v", commonconstants.ErrInvalidInput, err))
		return
	}
	resp, err := h.client.SignIn(c.Request.Context(), &req)
	if err != nil {
		apierr.Fail(c, "authgw: login", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully logged in.", "result": resp})
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		apierr.Fail(c, "authgw: get profile", fmt.Errorf("%w: %v", commonconstants.ErrUnauthorized, err))
		return
	}
	profile, err := h.client.GetProfile(c.Request.Context(), userID)
	if err != nil {
		apierr.Fail(c, "authgw: get profile", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully retrieved profile.", "result": profile})
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		apierr.Fail(c, "authgw: update profile", fmt.Errorf("%w: %v", commonconstants.ErrUnauthorized, err))
		return
	}
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, "authgw: update profile", fmt.Errorf("%w: malformed request body: %v", commonconstants.ErrInvalidInput, err))
		return
	}
	profile, err := h.client.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		apierr.Fail(c, "authgw: update profile", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully updated profile.", "result": profile})
}

func (h *Handler) GetById(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		apierr.Fail(c, "authgw: get user by id", fmt.Errorf("%w: %q", commonconstants.ErrUUIDCouldNotBeParsed, idParam))
		return
	}
	user, err := h.client.GetById(c.Request.Context(), id)
	if err != nil {
		apierr.Fail(c, "authgw: get user by id", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully retreived user.", "result": user})
}

func (h *Handler) GetAll(c *gin.Context) {
	users, err := h.client.ListUsers(c.Request.Context())
	if err != nil {
		apierr.Fail(c, "authgw: list users", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully retrieved users.", "result": users})
}
