package commongrpc

import (
	"context"
	"errors"
	"fmt"
	"testing"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/common/errcode"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// The code a producer attaches must come back out of the status, whatever the
// status class: the class is the coarse signal, the code the precise one.
func TestWithCode_RoundTrip_CodeAndClassSurvive(t *testing.T) {
	tests := []struct {
		name  string
		class codes.Code
		code  errcode.Code
	}{
		{"failed precondition", codes.FailedPrecondition, errcode.MemberIsOwner},
		{"resource exhausted", codes.ResourceExhausted, errcode.Code("SOME_LIMIT_REACHED")},
		{"invalid argument", codes.InvalidArgument, errcode.ValidationFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WithCode(tt.class, tt.code, "refused")

			if got := status.Code(err); got != tt.class {
				t.Fatalf("status class = %v, want %v", got, tt.class)
			}
			got, ok := DomainCode(err)
			if !ok || got != tt.code {
				t.Fatalf("DomainCode = (%q, %v), want (%q, true)", got, ok, tt.code)
			}
		})
	}
}

// The detail must survive the wire: a gRPC server re-reads the status from the
// returned error, and the client decodes the proto. Round-tripping through the
// proto form is what the gateway actually sees.
func TestWithCode_SurvivesProtoRoundTrip(t *testing.T) {
	err := WithCode(codes.ResourceExhausted, errcode.Code("SOME_LIMIT_REACHED"), "slow down")

	wire := status.FromProto(status.Convert(err).Proto()).Err()

	if got, ok := DomainCode(wire); !ok || got != "SOME_LIMIT_REACHED" {
		t.Fatalf("DomainCode after proto round trip = (%q, %v)", got, ok)
	}
}

// Every gateway client wraps the downstream error before returning it.
func TestDomainCode_WrappedStatus_StillFound(t *testing.T) {
	err := fmt.Errorf("calling insights: %w", WithCode(codes.FailedPrecondition, errcode.MemberIsOwner, "x"))

	if got, ok := DomainCode(err); !ok || got != errcode.MemberIsOwner {
		t.Fatalf("DomainCode(wrapped) = (%q, %v)", got, ok)
	}
}

func TestDomainCode_NoDomainCode_ReportsAbsent(t *testing.T) {
	foreign, _ := status.New(codes.InvalidArgument, "x").WithDetails(&errdetails.ErrorInfo{
		Reason: "API_KEY_INVALID",
		Domain: "googleapis.com",
	})
	tests := []struct {
		name string
		err  error
	}{
		{"nil", nil},
		{"plain error", errors.New("boom")},
		{"status without detail", status.Error(codes.InvalidArgument, "name is required")},
		{"error info from another domain", foreign.Err()},
		{"empty reason", WithCode(codes.InvalidArgument, "", "x")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, ok := DomainCode(tt.err); ok {
				t.Fatalf("DomainCode = (%q, true), want absent", got)
			}
		})
	}
}

// Services hand handler errors to Fail. A coded status is not a sentinel, so it
// must pass through untouched with its detail rather than be rebuilt (which
// would drop the detail).
func TestFail_CodedStatus_PassesThroughWithDetail(t *testing.T) {
	err := fmt.Errorf("refresh brief: %w", WithCode(codes.ResourceExhausted, errcode.Code("SOME_LIMIT_REACHED"), "cap"))

	got := Fail(context.Background(), "insights: refresh", err)

	if status.Code(got) != codes.ResourceExhausted {
		t.Fatalf("Fail class = %v, want ResourceExhausted", status.Code(got))
	}
	if code, ok := DomainCode(got); !ok || code != "SOME_LIMIT_REACHED" {
		t.Fatalf("Fail dropped the detail: (%q, %v)", code, ok)
	}
}

// Sentinel mapping is unaffected by the new helper.
func TestStatus_Sentinel_StillMapped(t *testing.T) {
	if got := status.Code(Status(commonconstants.ErrNotFound)); got != codes.NotFound {
		t.Fatalf("Status(ErrNotFound) = %v, want NotFound", got)
	}
}
