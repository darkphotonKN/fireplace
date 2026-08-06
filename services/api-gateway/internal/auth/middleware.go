package auth

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	commonauth "github.com/darkphotonKN/fireplace/common/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// abortUnauthorized logs the rejection once (client fault ⇒ Warn) and aborts
// with a generic 401. The specific reason / underlying error stays server-side;
// the client only ever sees "unauthorized".
func abortUnauthorized(c *gin.Context, reason string, err error) {
	slog.WarnContext(c.Request.Context(), "auth middleware: request rejected",
		"reason", reason, "err", err, "method", c.Request.Method, "path", c.FullPath())
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"statusCode": http.StatusUnauthorized, "message": "unauthorized"})
}

// authenticate runs the token check and WRITES NOTHING. Both the legacy
// middleware and the problem+json fork call it, so there is exactly one place
// where "is this caller authenticated" is decided and two places where the
// answer is rendered.
//
// This split exists because a middleware cannot un-write a response: the fork
// originally called AuthMiddleware and then tried to replace its body, which
// emitted two concatenated JSON documents.
func authenticate(c *gin.Context) (uuid.UUID, string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return uuid.Nil, "missing authorization header", nil
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return uuid.Nil, "malformed authorization header", nil
	}

	claims, err := commonauth.ParseToken(parts[1], os.Getenv("JWT_SECRET"))
	if err != nil {
		return uuid.Nil, "invalid or expired token", err
	}

	userID, err := commonauth.UserIDFromClaims(claims)
	if err != nil {
		return uuid.Nil, "invalid user id in token", err
	}

	return userID, "", nil
}

// AuthMiddleware validates the bearer JWT locally against JWT_SECRET — no
// remote call to auth-service is needed. auth-service issues tokens with the
// same shared secret; the gateway validates and extracts the user id only.
//
// LEGACY RENDERING: {statusCode, message}. Serialized routes use
// ProblemMiddleware instead; this one must stay byte-identical.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, reason, err := authenticate(c)
		if reason != "" {
			abortUnauthorized(c, reason, err)
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
