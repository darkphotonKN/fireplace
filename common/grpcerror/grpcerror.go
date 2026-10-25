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
//
// Domain codes (FS-DF4ZB R52): when a service refuses a request for a reason the
// client must tell apart from other failures of the same class, it returns
// WithCode, which attaches the errcode as a google.rpc.ErrorInfo detail. The
// gateway reads it back with DomainCode. Message text is never parsed for
// codes: prose is allowed to change, a detail is contract.
package commongrpc

import (
	"context"
	"errors"
	"log/slog"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/common/errcode"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
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

// ErrorDomain is the ErrorInfo.domain every Fireplace service stamps on a domain
// code. DomainCode only trusts a reason under this domain, so an ErrorInfo some
// library attaches with its own vocabulary is never mistaken for an errcode.
const ErrorDomain = "fireplace"

// WithCode builds a gRPC status error of the given class carrying code as an
// ErrorInfo detail. The class still decides the HTTP status at the gateway; the
// code is what the client switches on. Adding a new domain code therefore needs
// only its errcode constant: no mapping changes on either side.
func WithCode(class codes.Code, code errcode.Code, msg string) error {
	st := status.New(class, msg)
	withInfo, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason: string(code),
		Domain: ErrorDomain,
	})
	if err != nil {
		// Only reachable if ErrorInfo fails to marshal. The class still reaches
		// the caller, so the gateway falls back to the class mapping.
		return st.Err()
	}
	return withInfo.Err()
}

// DomainCode returns the errcode an error carries as a Fireplace ErrorInfo
// detail, unwrapping %w chains. ok is false for a nil error, a non-status error,
// a status without such a detail, or an empty reason.
func DomainCode(err error) (code errcode.Code, ok bool) {
	if err == nil {
		return "", false
	}
	st, isStatus := status.FromError(err)
	if !isStatus {
		return "", false
	}
	for _, d := range st.Details() {
		info, isInfo := d.(*errdetails.ErrorInfo)
		if isInfo && info.GetDomain() == ErrorDomain && info.GetReason() != "" {
			return errcode.Code(info.GetReason()), true
		}
	}
	return "", false
}
