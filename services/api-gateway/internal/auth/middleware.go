package auth

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthMiddleware extracts user ID from JWT and sets it in gin.Context as "userId".
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

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": "Invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": "Invalid token claims"})
			return
		}

		sub, ok := claims["sub"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": "Invalid sub claim"})
			return
		}

		userID, err := uuid.Parse(sub)
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
