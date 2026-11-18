package insights

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/insights"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/common/errcode"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/ai"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/notesummary"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeNoteSummaries struct {
	summary string
	err     error

	calls                  int
	planID, userID, itemID uuid.UUID
}

func (f *fakeNoteSummaries) Summarize(_ context.Context, planID, userID, itemID uuid.UUID) (string, error) {
	f.calls++
	f.planID, f.userID, f.itemID = planID, userID, itemID
	return f.summary, f.err
}

func TestHandler_SummarizeNote_ReturnsTheSummary(t *testing.T) {
	notes := &fakeNoteSummaries{summary: "A short summary."}
	planID, userID, itemID := uuid.New(), uuid.New(), uuid.New()

	resp, err := NewHandler(nil, nil, notes).SummarizeNote(context.Background(), &pb.SummarizeNoteRequest{
		PlanId: planID.String(), UserId: userID.String(), ItemId: itemID.String(),
	})

	if err != nil || resp.GetSummary() != "A short summary." {
		t.Fatalf("resp = %v, err = %v", resp, err)
	}
	if notes.planID != planID || notes.userID != userID || notes.itemID != itemID {
		t.Fatalf("service got (%s, %s, %s)", notes.planID, notes.userID, notes.itemID)
	}
}

// FS-PGG2N §API surface: each refusal leaves with its class and, where it has
// one, its domain code as ErrorInfo.
func TestHandler_SummarizeNote_Refusals(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		wantClass codes.Code
		wantCode  errcode.Code // "" = no domain code
	}{
		{"not found", fmt.Errorf("x: %w", commonconstants.ErrNotFound), codes.NotFound, ""},
		{"not a note", fmt.Errorf("x: %w", notesummary.ErrNotANote), codes.FailedPrecondition, errcode.ItemNotANote},
		{"budget spent", fmt.Errorf("x: %w", ai.ErrBudgetExhausted), codes.ResourceExhausted, errcode.LLMBudgetExhausted},
		{"model failed", fmt.Errorf("%w: boom", notesummary.ErrFailed), codes.Unavailable, errcode.NoteSummaryFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewHandler(nil, nil, &fakeNoteSummaries{err: tc.err}).SummarizeNote(context.Background(),
				&pb.SummarizeNoteRequest{PlanId: uuid.NewString(), UserId: uuid.NewString(), ItemId: uuid.NewString()})

			if status.Code(err) != tc.wantClass {
				t.Fatalf("class = %v, want %v (%v)", status.Code(err), tc.wantClass, err)
			}
			code, ok := commongrpc.DomainCode(err)
			if tc.wantCode == "" && ok {
				t.Fatalf("domain code = %q, want none", code)
			}
			if tc.wantCode != "" && code != tc.wantCode {
				t.Fatalf("domain code = %q (ok %v), want %q", code, ok, tc.wantCode)
			}
		})
	}
}

// Malformed ids are InvalidArgument and never reach the service.
func TestHandler_SummarizeNote_MalformedIds(t *testing.T) {
	ok := uuid.NewString()
	cases := map[string]*pb.SummarizeNoteRequest{
		"plan": {PlanId: "nope", UserId: ok, ItemId: ok},
		"user": {PlanId: ok, UserId: "nope", ItemId: ok},
		"item": {PlanId: ok, UserId: ok, ItemId: "nope"},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			notes := &fakeNoteSummaries{summary: "x"}
			_, err := NewHandler(nil, nil, notes).SummarizeNote(context.Background(), req)
			if status.Code(err) != codes.InvalidArgument || notes.calls != 0 {
				t.Fatalf("class = %v, calls = %d; want InvalidArgument and no call", status.Code(err), notes.calls)
			}
		})
	}
}
