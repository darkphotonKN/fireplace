package auth

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	commonauth "github.com/darkphotonKN/fireplace/common/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthMiddleware validates the bearer JWT locally against JWT_SECRET — no
// remote call to auth-service is needed. auth-service issues tokens with the
// same shared secret; the gateway validates and extracts the user id only.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": "Missing authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": "Invalid authorization header format"})
			return
		}

		claims, err := commonauth.ParseToken(parts[1], os.Getenv("JWT_SECRET"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": "Invalid or expired token"})
			return
		}

		userID, err := commonauth.UserIDFromClaims(claims)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": "Invalid user ID in token"})
			return
		}

		c.Set("userId", userID)
		c.Next()
	}
}

// GetUserID extracts the authenticated user ID from gin.Context.
func GetUserID(c *gin.Context) (uuid.UUID, error) {
	val, exists := c.Get("userId")
	if !exists {
		return uuid.Nil, fmt.Errorf("userId not found in context")
	}
	userID, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("userId is not a valid UUID")
	}
	return userID, nil
}
