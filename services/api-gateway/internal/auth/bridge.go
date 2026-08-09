package auth

import (
	"context"
	"encoding/json"
	"log/slog"
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
		userID, reason, err := authenticate(c)
		if reason != "" {
			// Same server-side logging as the legacy path; the client still
			// only ever learns "unauthorized".
			slog.WarnContext(c.Request.Context(), "auth middleware: request rejected",
				"reason", reason, "err", err, "method", c.Request.Method, "path", c.FullPath())

			body, _ := json.Marshal(map[string]any{
				"type":   "about:blank",
				"title":  http.StatusText(http.StatusUnauthorized),
				"status": http.StatusUnauthorized,
				"detail": "unauthorized",
				"code":   errcode.Unauthenticated,
				"errors": []any{},
			})
			c.Abort()
			// c.Data, not c.JSON — c.JSON forces application/json and would
			// silently drop the RFC 9457 media type.
			c.Data(http.StatusUnauthorized, "application/problem+json", body)
			return
		}

		c.Set("userId", userID)
		c.Next()
	}
}
