package apierr

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/darkphotonKN/fireplace/common/errcode"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// A downstream service being unreachable is not a bug in the gateway, and a
// caller can do something about it — retry. Mapping it to 500 tells the caller
// "we are broken", which is both wrong and unactionable (FS-0004 R13).
func TestStatusFor_DownstreamUnavailable_Is503(t *testing.T) {
	err := status.Error(codes.Unavailable, "auth-service: connection refused")

	got, msg := StatusFor(err)

	if got != http.StatusServiceUnavailable {
		t.Fatalf("StatusFor(Unavailable) = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if msg == "" {
		t.Fatal("StatusFor(Unavailable) returned an empty message")
	}
}

func TestCodeFor_DownstreamUnavailable_IsServiceUnavailable(t *testing.T) {
	err := status.Error(codes.Unavailable, "auth-service: connection refused")

	if got := CodeFor(err); got != errcode.ServiceUnavailable {
		t.Fatalf("CodeFor(Unavailable) = %q, want %q", got, errcode.ServiceUnavailable)
	}
}

// A wrapped gRPC status must map the same way — status.FromError unwraps %w
// chains, and every gateway client wraps before returning.
func TestStatusFor_WrappedUnavailable_Is503(t *testing.T) {
	err := fmt.Errorf("calling auth-service: %w", status.Error(codes.Unavailable, "down"))

	if got, _ := StatusFor(err); got != http.StatusServiceUnavailable {
		t.Fatalf("StatusFor(wrapped Unavailable) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

// The mappings that already worked must keep working — this test exists so a
// change to the Unavailable case cannot silently reorder the switch.
func TestStatusFor_ExistingMappingsUnchanged(t *testing.T) {
	tests := []struct {
		name string
		code codes.Code
		want int
	}{
		{"not found", codes.NotFound, http.StatusNotFound},
		{"already exists", codes.AlreadyExists, http.StatusConflict},
		{"invalid argument", codes.InvalidArgument, http.StatusBadRequest},
		{"unauthenticated", codes.Unauthenticated, http.StatusUnauthorized},
		{"permission denied", codes.PermissionDenied, http.StatusForbidden},
		{"unknown falls through", codes.Unknown, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := StatusFor(status.Error(tt.code, "x")); got != tt.want {
				t.Fatalf("StatusFor(%v) = %d, want %d", tt.code, got, tt.want)
			}
		})
	}
}

// ProblemFor is the single adapter every serialized handler routes through, so
// the status and the code must travel together. A 503 carrying INTERNAL_ERROR
// would pass any test that asserted only on status.
func TestProblemFor_Unavailable_CarriesBothStatusAndCode(t *testing.T) {
	p := ProblemFor("list users", status.Error(codes.Unavailable, "down"))

	if p.Status != http.StatusServiceUnavailable {
		t.Fatalf("Problem.Status = %d, want %d", p.Status, http.StatusServiceUnavailable)
	}
	if p.Code != errcode.ServiceUnavailable {
		t.Fatalf("Problem.Code = %q, want %q", p.Code, errcode.ServiceUnavailable)
	}
	if p.Errors == nil {
		t.Fatal("Problem.Errors is nil; it must be present and empty so clients never null-check")
	}
}

// FS-DF4ZB R52: the seam reads the domain code from the status detail before
// it falls back to the status class. The status itself still comes from the
// class, so a detailed status and an undetailed one of the same class share an
// HTTP status and differ only by code.
func TestProblemFor_ErrorSeam(t *testing.T) {
	const limitCode = errcode.Code("SOME_LIMIT_REACHED") // stands in for a later slice's code

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   errcode.Code
	}{
		// Detail first, whatever the class.
		{"detail on ResourceExhausted", commongrpc.WithCode(codes.ResourceExhausted, limitCode, "cap reached"),
			http.StatusTooManyRequests, limitCode},
		{"detail on FailedPrecondition", commongrpc.WithCode(codes.FailedPrecondition, errcode.MemberIsOwner, "refused"),
			http.StatusBadRequest, errcode.MemberIsOwner},
		{"detail on NotFound", commongrpc.WithCode(codes.NotFound, limitCode, "gone"),
			http.StatusNotFound, limitCode},
		{"wrapped detail", fmt.Errorf("calling insights: %w", commongrpc.WithCode(codes.ResourceExhausted, limitCode, "x")),
			http.StatusTooManyRequests, limitCode},

		// The "name" rule must not capture a status that carries a detail.
		{"InvalidArgument with detail mentioning name", commongrpc.WithCode(codes.InvalidArgument, errcode.ValidationFailed, "parent item name is invalid"),
			http.StatusBadRequest, errcode.ValidationFailed},
		{"InvalidArgument with empty-reason detail mentioning name", commongrpc.WithCode(codes.InvalidArgument, "", "display name of the owner"),
			http.StatusBadRequest, errcode.ValidationFailed},

		// auth-service's profile-name failure sends no detail: unchanged.
		{"auth profile name, no detail", status.Error(codes.InvalidArgument, "display name must not be empty"),
			http.StatusBadRequest, errcode.ProfileNameEmpty},

		// New class fallbacks (no detail).
		{"ResourceExhausted, no detail", status.Error(codes.ResourceExhausted, "x"),
			http.StatusTooManyRequests, errcode.Internal},
		{"FailedPrecondition, no detail", status.Error(codes.FailedPrecondition, "x"),
			http.StatusBadRequest, errcode.ValidationFailed},

		// Existing class mappings, unchanged.
		{"NotFound", status.Error(codes.NotFound, "x"), http.StatusNotFound, errcode.NotFound},
		{"AlreadyExists", status.Error(codes.AlreadyExists, "x"), http.StatusConflict, errcode.AlreadyExists},
		{"InvalidArgument", status.Error(codes.InvalidArgument, "x"), http.StatusBadRequest, errcode.ValidationFailed},
		{"Unauthenticated", status.Error(codes.Unauthenticated, "x"), http.StatusUnauthorized, errcode.Unauthenticated},
		{"PermissionDenied", status.Error(codes.PermissionDenied, "x"), http.StatusForbidden, errcode.Forbidden},
		{"Unavailable", status.Error(codes.Unavailable, "x"), http.StatusServiceUnavailable, errcode.ServiceUnavailable},
		{"Unknown", status.Error(codes.Unknown, "x"), http.StatusInternalServerError, errcode.Internal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ProblemFor("op", tt.err)

			if p.Status != tt.wantStatus {
				t.Errorf("Problem.Status = %d, want %d", p.Status, tt.wantStatus)
			}
			if p.Code != tt.wantCode {
				t.Errorf("Problem.Code = %q, want %q", p.Code, tt.wantCode)
			}
		})
	}
}
