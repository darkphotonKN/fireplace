package user

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service Service
}

type Service interface {
	GetById(id uuid.UUID) (*models.User, error)
	GetProfile(id uuid.UUID) (*models.User, error)
	UpdateProfile(id uuid.UUID, req UpdateProfileRequest) (*models.User, error)
	Create(user models.User) error
	HashPassword(password string) (string, error)
	GetAll() ([]*Response, error)
	Login(loginReq LoginRequest) (*LoginResponse, error)
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// parseUserIDFromJWT extracts the user ID from the Authorization header's JWT token.
func parseUserIDFromJWT(c *gin.Context) (uuid.UUID, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return uuid.Nil, fmt.Errorf("missing authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return uuid.Nil, fmt.Errorf("invalid authorization header format")
	}

	token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid token claims")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid sub claim")
	}

	return uuid.Parse(sub)
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID, err := parseUserIDFromJWT(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": fmt.Sprintf("Unauthorized: %s", err.Error())})
		return
	}

	profile, err := h.service.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"statusCode": http.StatusInternalServerError, "message": fmt.Sprintf("Error fetching profile: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully retrieved profile.", "result": profile})
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, err := parseUserIDFromJWT(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": fmt.Sprintf("Unauthorized: %s", err.Error())})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": fmt.Sprintf("Invalid request body: %s", err.Error())})
		return
	}

	profile, err := h.service.UpdateProfile(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": fmt.Sprintf("Error updating profile: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully updated profile.", "result": profile})
}

func (h *Handler) Create(c *gin.Context) {
	var user models.User

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": fmt.Sprintf("Error with parsing payload as JSON.")})
		return
	}

	err := h.service.Create(user)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"statusCode:": http.StatusInternalServerError, "message": fmt.Sprintf("Error when attempting to create user: %s", err.Error())})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"statusCode:": http.StatusCreated, "message": "Successfully created user."})
}

func (h *Handler) GetById(c *gin.Context) {
	idParam := c.Param("id")

	id, err := uuid.Parse(idParam)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": fmt.Sprintf("Error with id %d, not a valid uuid.", id)})
		return
	}

	user, err := h.service.GetById(id)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": fmt.Sprintf("Error when attempting to get user with id %d %s", id, err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "Successfully retreived user.",
		"result": user})
}

func (h *Handler) GetAll(c *gin.Context) {
	users, err := h.service.GetAll()

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode:": http.StatusBadRequest, "message": fmt.Sprintf("Error when attempting to get all users: %s:\n", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode:": http.StatusOK, "message": "Successfully retrieved users.", "result": users})
}

func (h *Handler) Login(c *gin.Context) {
	var loginReq LoginRequest

	err := c.ShouldBindJSON(&loginReq)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": fmt.Sprintf("Error when unmarshalling json payload: %s\n", err)})
		return
	}

	user, err := h.service.Login(loginReq)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"statusCode": http.StatusBadRequest, "message": fmt.Sprintf("Error when attempting to login user: %s\n", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statusCode": http.StatusOK, "message": "Successfully logged in.",
		"result": user})
}
