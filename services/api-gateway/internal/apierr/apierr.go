// Package apierr is the api-gateway's single point for turning errors into HTTP
// responses. It is the gateway analogue of the per-service status mapper: one
// StatusFor, one Fail helper, so individual handlers never repeat status
// switches or log/format error bodies by hand.
//
// The gateway sees two kinds of error:
//   - gRPC status errors propagated from downstream services (auth/plan/
//     calendar) — the wire status is authoritative and mapped to HTTP.
//   - gateway-local domain errors (notes, insights, useranalytics) expressed as
//     common/constants sentinels.
//
// Both are handled here; anything unrecognised is a 500.
package apierr

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StatusFor maps an error to an HTTP status code and a client-safe message.
// The message is intentionally generic — the full error chain is logged
// server-side by Fail, never returned to the client.
func StatusFor(err error) (int, string) {
	if err == nil {
		return http.StatusOK, ""
	}

	// A gRPC status propagated from a downstream service is authoritative —
	// map its code directly. status.FromError unwraps %w chains.
	if st, ok := status.FromError(err); ok && st.Code() != codes.OK {
		return httpForCode(st.Code())
	}

	switch {
	case errors.Is(err, commonconstants.ErrNotFound):
		return http.StatusNotFound, "not found"
	case errors.Is(err, commonconstants.ErrDuplicateResource):
		return http.StatusConflict, "already exists"
	case errors.Is(err, commonconstants.ErrInvalidInput),
		errors.Is(err, commonconstants.ErrConstraintViolation),
		errors.Is(err, commonconstants.ErrUUIDCouldNotBeParsed):
		return http.StatusBadRequest, "invalid request"
	case errors.Is(err, commonconstants.ErrUnauthorized):
		return http.StatusUnauthorized, "unauthorized"
	case errors.Is(err, commonconstants.ErrForbidden):
		return http.StatusForbidden, "forbidden"
	default:
		return http.StatusInternalServerError, "internal error"
	}
}

func httpForCode(code codes.Code) (int, string) {
	switch code {
	case codes.NotFound:
		return http.StatusNotFound, "not found"
	case codes.AlreadyExists:
		return http.StatusConflict, "already exists"
	case codes.InvalidArgument:
		return http.StatusBadRequest, "invalid request"
	case codes.Unauthenticated:
		return http.StatusUnauthorized, "unauthorized"
	case codes.PermissionDenied:
		return http.StatusForbidden, "forbidden"
	default:
		return http.StatusInternalServerError, "internal error"
	}
}

// Fail writes a JSON error response and logs the error exactly once. op
// identifies the operation for the log line; 4xx logs at Warn (client fault),
// 5xx at Error (server fault). The request context is passed to the *Context
// slog variants so any request/trace IDs propagate. The response body keeps the
// gateway's established { statusCode, message } shape.
func Fail(c *gin.Context, op string, err error) {
	code, msg := StatusFor(err)
	ctx := c.Request.Context()
	attrs := []any{"err", err, "status", code, "method", c.Request.Method, "path", c.FullPath()}
	if code >= http.StatusInternalServerError {
		slog.ErrorContext(ctx, op+" failed", attrs...)
	} else {
		slog.WarnContext(ctx, op+" rejected", attrs...)
	}
	c.JSON(code, gin.H{"statusCode": code, "message": msg})
}

// FailCtx is Fail for callers that aren't inside a gin handler but still need
// the shared mapping (rare). Kept minimal on purpose.
func FailCtx(ctx context.Context, op string, err error) (int, string) {
	code, msg := StatusFor(err)
	if code >= http.StatusInternalServerError {
		slog.ErrorContext(ctx, op+" failed", "err", err, "status", code)
	} else {
		slog.WarnContext(ctx, op+" rejected", "err", err, "status", code)
	}
	return code, msg
}
