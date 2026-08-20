package apierr

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/common/errcode"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RFC 9457 problem+json for the SERIALIZED surface (ADR-0004).
//
// Legacy gin handlers keep using Fail / StatusFor and the {statusCode, message}
// envelope. Serialized handlers return Problem. Both read the SAME mapping —
// there is one error boundary in this gateway, and this file only adds a second
// serialization of it.

// Problem is the wire shape of an error on a serialized endpoint.
//
// It replaces huma.ErrorModel because that type has no extension member for
// `code`. It MUST implement ContentType — that method is what emits
// application/problem+json; omitting it silently degrades to application/json
// while still passing any test that asserts only on status and body.
type Problem struct {
	Type   string       `json:"type,omitempty" doc:"URI reference identifying the problem type"`
	Title  string       `json:"title,omitempty" doc:"Short, stable summary of the problem type"`
	Status int          `json:"status,omitempty" doc:"HTTP status code"`
	Detail string       `json:"detail,omitempty" doc:"Explanation specific to this occurrence"`
	Code   errcode.Code `json:"code" doc:"Stable domain error code — switch on THIS, not on detail" example:"NOT_FOUND"`
	// NOT omitempty: FS-0002 R16 requires errors[] to be PRESENT and empty for
	// downstream failures, so the FE never has to null-check it. omitempty would
	// drop a zero-length slice entirely.
	Errors []*huma.ErrorDetail `json:"errors" doc:"Field-level detail; empty for downstream failures"`
}

func (p *Problem) Error() string  { return p.Detail }
func (p *Problem) GetStatus() int { return p.Status }

// ContentType is what makes this RFC 9457 rather than ordinary JSON.
func (p *Problem) ContentType(ct string) string {
	if ct == "application/json" {
		return "application/problem+json"
	}
	return ct
}

// CodeFor maps an error to its domain code, using the same precedence as
// StatusFor: a gRPC status from downstream is authoritative, then local
// sentinels, then a catch-all.
//
// NOTE (FS-0002 R15): the FS describes StatusFor itself growing a code
// dimension. Implemented as a sibling function instead so the ~20 existing
// legacy call sites keep compiling unchanged — same boundary, same precedence,
// no behavioural difference. Flagged for code-review rather than done silently.
func CodeFor(err error) errcode.Code {
	if err == nil {
		return ""
	}

	if st, ok := status.FromError(err); ok && st.Code() != codes.OK {
		switch st.Code() {
		case codes.NotFound:
			return errcode.NotFound
		case codes.AlreadyExists:
			return errcode.AlreadyExists
		case codes.InvalidArgument:
			// auth-service's only profile validation rule. The wire carries no
			// structured field detail, so the precision must live in the code.
			if msg := st.Message(); msg != "" && containsFold(msg, "name") {
				return errcode.ProfileNameEmpty
			}
			return errcode.ValidationFailed
		case codes.Unauthenticated:
			return errcode.Unauthenticated
		case codes.PermissionDenied:
			return errcode.Forbidden
		case codes.Unavailable:
			return errcode.ServiceUnavailable
		default:
			return errcode.Internal
		}
	}

	switch {
	case errors.Is(err, commonconstants.ErrNotFound):
		return errcode.NotFound
	case errors.Is(err, commonconstants.ErrDuplicateResource):
		return errcode.AlreadyExists
	case errors.Is(err, commonconstants.ErrInvalidInput),
		errors.Is(err, commonconstants.ErrConstraintViolation),
		errors.Is(err, commonconstants.ErrUUIDCouldNotBeParsed):
		return errcode.ValidationFailed
	case errors.Is(err, commonconstants.ErrNotImplemented):
		return errcode.NotImplemented
	case errors.Is(err, commonconstants.ErrUnauthorized):
		return errcode.Unauthenticated
	case errors.Is(err, commonconstants.ErrForbidden):
		return errcode.Forbidden
	default:
		return errcode.Internal
	}
}

// ProblemFor is the single adapter from any gateway error to problem+json.
// It performs NO mapping of its own — StatusFor and CodeFor own that decision
// for the whole gateway; this only re-serializes it.
// ErrNoIdentity is the sentinel for "the identity bridge yielded nothing".
// Unreachable behind the auth middleware, but every serialized operation still
// maps it rather than dereferencing a missing id — the failure mode of an
// identity lookup must be a contract-shaped 401, never a panic.
func ErrNoIdentity() error {
	return fmt.Errorf("%w: no identity in context", commonconstants.ErrUnauthorized)
}

func ProblemFor(op string, err error) *Problem {
	code, msg := StatusFor(err)
	return &Problem{
		Type:   "about:blank",
		Title:  http.StatusText(code),
		Status: code,
		Detail: op + ": " + msg,
		Code:   CodeFor(err),
		Errors: []*huma.ErrorDetail{},
	}
}

func containsFold(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			a, b := s[i+j], sub[j]
			if a >= 'A' && a <= 'Z' {
				a += 32
			}
			if b >= 'A' && b <= 'Z' {
				b += 32
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// CodeForStatus maps a bare HTTP status to a domain code. Used for errors huma
// raises itself (request validation, unmatched routes) where there is no
// underlying error to classify.
func CodeForStatus(status int) errcode.Code {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return errcode.ValidationFailed
	case http.StatusUnauthorized:
		return errcode.Unauthenticated
	case http.StatusForbidden:
		return errcode.Forbidden
	case http.StatusNotFound:
		return errcode.NotFound
	case http.StatusConflict:
		return errcode.AlreadyExists
	case http.StatusNotImplemented:
		return errcode.NotImplemented
	default:
		return errcode.Internal
	}
}
