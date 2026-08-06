package auth

import (
	"context"
	"net/http"

	"github.com/darkphotonKN/fireplace/common/errcode"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// The identity bridge (FS-0002 §Requirements 18-19).
//
// humagin's Context() returns c.Request.Context(), NOT the *gin.Context. So
// AuthMiddleware's c.Set("userId", …) — gin's own KV store — is invisible to a
// typed huma handler. This copies identity across the gap exactly once.
//
// This is the SINGLE identity seam for serialized routes. When ADR-0001's
// metadata-and-context identity lands, it converges HERE. Do not build a
// second path.

type ctxKey struct{}

// BridgeIdentity must be mounted AFTER AuthMiddleware and BEFORE any huma route.
func BridgeIdentity() gin.HandlerFunc {
	return func(c *gin.Context) {
		if userID, err := GetUserID(c); err == nil {
			c.Request = c.Request.WithContext(
				context.WithValue(c.Request.Context(), ctxKey{}, userID))
		}
		c.Next()
	}
}

// UserIDFromCtx reads the authenticated user inside a typed handler.
func UserIDFromCtx(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ctxKey{}).(uuid.UUID)
	return id, ok
}

// ProblemMiddleware is the problem+json-emitting auth variant, mounted on the
// SERIALIZED group only (FS-0002 §Requirements 17).
//
// AuthMiddleware aborts with the legacy {statusCode, message} shape before any
// huma handler runs, which would leave a hole in the contract on its most
// common error. The legacy middleware and every legacy protected route are left
// byte-identical — this is a fork, not a replacement.
func ProblemMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		AuthMiddleware()(c)
		if !c.IsAborted() {
			return
		}
		// Replace the legacy body the middleware just wrote.
		c.Writer.Header().Set("Content-Type", "application/problem+json")
		c.JSON(http.StatusUnauthorized, gin.H{
			"type":   "about:blank",
			"title":  http.StatusText(http.StatusUnauthorized),
			"status": http.StatusUnauthorized,
			"detail": "unauthorized",
			"code":   errcode.Unauthenticated,
			"errors": []any{},
		})
	}
}
