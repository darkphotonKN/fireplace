// Package grpcerror centralizes how the gRPC services translate domain errors
// into gRPC status codes and log them. It is the single shared mapper referred
// to in the error-handling guidelines: services do not repeat status switches
// in every handler — they call Fail (or Status) from here.
//
// Mapping rules:
//   - domain sentinels from common/constants map to the closest gRPC code
//   - an error that already carries a gRPC status (e.g. propagated from a
//     downstream service) is passed through untouched so its code survives
//   - everything else is codes.Internal
package commongrpc

import (
	"context"
	"errors"
	"log/slog"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// codeFor maps a domain error to the gRPC code that best represents it.
// Unknown errors default to codes.Internal.
func codeFor(err error) codes.Code {
	switch {
	case errors.Is(err, commonconstants.ErrNotFound):
		return codes.NotFound
	case errors.Is(err, commonconstants.ErrDuplicateResource):
		return codes.AlreadyExists
	case errors.Is(err, commonconstants.ErrInvalidInput),
		errors.Is(err, commonconstants.ErrConstraintViolation),
		errors.Is(err, commonconstants.ErrUUIDCouldNotBeParsed):
		return codes.InvalidArgument
	case errors.Is(err, commonconstants.ErrForbidden):
		return codes.PermissionDenied
	case errors.Is(err, commonconstants.ErrUnauthorized):
		return codes.Unauthenticated
	default:
		return codes.Internal
	}
}

// Status converts a domain error into a gRPC status error. A nil error returns
// nil. If err already carries a gRPC status that we don't recognise as one of
// our sentinels (typically a status propagated from a downstream service), it
// is returned unchanged so the original code flows through to the caller.
func Status(err error) error {
	if err == nil {
		return nil
	}
	code := codeFor(err)
	if code == codes.Internal {
		// Preserve a real downstream gRPC status instead of flattening it to
		// Internal. status.FromError unwraps %w chains.
		if _, ok := status.FromError(err); ok {
			return err
		}
	}
	return status.Error(code, err.Error())
}

// Fail logs the error exactly once at the appropriate level and returns the
// gRPC status error to hand back to the client. This is the only place gRPC
// handlers should log an error path: server-fault codes (Internal/Unknown) log
// at Error, client-fault codes log at Warn. The full error chain is passed as a
// structured field; op identifies the operation (e.g. "auth: sign up").
//
// Extra structured attributes (key/value pairs) may be appended via attrs.
func Fail(ctx context.Context, op string, err error, attrs ...any) error {
	st := Status(err)
	code := status.Code(st)
	args := append([]any{"err", err, "code", code.String()}, attrs...)
	if code == codes.Internal || code == codes.Unknown {
		slog.ErrorContext(ctx, op+" failed", args...)
	} else {
		slog.WarnContext(ctx, op+" rejected", args...)
	}
	return st
}
