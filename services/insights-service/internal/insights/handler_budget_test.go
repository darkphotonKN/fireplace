package insights

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/insights"
	"github.com/darkphotonKN/fireplace/common/errcode"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/ai"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// failingInsights answers every legacy RPC with err.
type failingInsights struct{ err error }

func (f failingInsights) GenerateSuggestion(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", f.err
}
func (f failingInsights) SuggestVideos(context.Context, uuid.UUID, uuid.UUID) ([]Video, error) {
	return nil, f.err
}

// Over budget, the legacy generators answer ResourceExhausted carrying
// LLM_BUDGET_EXHAUSTED as ErrorInfo, which the gateway maps to 429 (R33).
func TestHandler_BudgetExhaustedIsResourceExhaustedWithItsCode(t *testing.T) {
	h := NewHandler(failingInsights{err: fmt.Errorf("insights: generate suggestion: %w", ai.ErrBudgetExhausted)}, nil)
	ctx := context.Background()
	plan, user := uuid.NewString(), uuid.NewString()

	rpcs := map[string]func() error{
		"GenerateSuggestion": func() error {
			_, err := h.GenerateSuggestion(ctx, &pb.GenerateSuggestionRequest{PlanId: plan, UserId: user})
			return err
		},
		"SuggestVideos": func() error {
			_, err := h.SuggestVideos(ctx, &pb.SuggestVideosRequest{PlanId: plan, UserId: user})
			return err
		},
	}
	for name, call := range rpcs {
		t.Run(name, func(t *testing.T) {
			err := call()
			if status.Code(err) != codes.ResourceExhausted {
				t.Fatalf("code = %v, want ResourceExhausted (%v)", status.Code(err), err)
			}
			if code, ok := commongrpc.DomainCode(err); !ok || code != errcode.LLMBudgetExhausted {
				t.Fatalf("domain code = %q (ok %v), want LLM_BUDGET_EXHAUSTED", code, ok)
			}
		})
	}
}
