package insights

import (
	"context"
	"errors"
	"testing"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/ai"
	"github.com/google/uuid"
)

type stubPlans struct{}

func (stubPlans) GetPlanContext(ctx context.Context, planID, userID uuid.UUID) (*PlanContext, error) {
	return &PlanContext{Focus: "Learn Go"}, nil
}

// callRecorder is a ContentGenerator that keeps the Call of every request.
type callRecorder struct {
	out   string
	err   error
	calls []ai.Call
}

func (r *callRecorder) Generate(ctx context.Context, call ai.Call, prompt string) (string, error) {
	r.calls = append(r.calls, call)
	return r.out, r.err
}

type stubVideos struct{ asked int }

func (s *stubVideos) FindVideos(ctx context.Context, terms []string) ([]Video, error) {
	s.asked++
	return []Video{{Title: "v"}}, nil
}

// Every legacy model call names its ledger kind, plan and user, so the
// metered generator can record it (FS-02JBZ R31, A4).
func TestService_LegacyGeneratorsAttributeTheirCalls(t *testing.T) {
	planID, userID := uuid.New(), uuid.New()
	tests := []struct {
		name      string
		run       func(s *Service) error
		gen       func(checklist, terms *callRecorder) *callRecorder
		wantKind  ai.Kind
		wantCalls int
	}{
		{"single checklist suggestion", func(s *Service) error {
			_, err := s.GenerateSuggestion(context.Background(), planID, userID)
			return err
		}, func(c, _ *callRecorder) *callRecorder { return c }, ai.KindChecklistSuggestion, 1},
		{"video search terms", func(s *Service) error {
			_, err := s.SuggestVideos(context.Background(), planID, userID)
			return err
		}, func(_, v *callRecorder) *callRecorder { return v }, ai.KindVideoTerms, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checklist, terms := &callRecorder{out: "Do it"}, &callRecorder{out: "go tutorial"}
			svc := NewService(stubPlans{}, checklist, terms, &stubVideos{}, nil, nil, nil)

			if err := tt.run(svc); err != nil {
				t.Fatal(err)
			}

			got := tt.gen(checklist, terms).calls
			want := ai.Call{Kind: tt.wantKind, PlanID: planID, UserID: userID}
			if len(got) != tt.wantCalls {
				t.Fatalf("calls = %d, want %d", len(got), tt.wantCalls)
			}
			for _, c := range got {
				if c != want {
					t.Fatalf("call = %+v, want %+v", c, want)
				}
			}
		})
	}
}

func TestService_BudgetExhaustionReachesTheCaller(t *testing.T) {
	refused := &callRecorder{err: ai.ErrBudgetExhausted}
	videos := &stubVideos{}
	svc := NewService(stubPlans{}, refused, refused, videos, nil, nil, nil)
	ctx := context.Background()

	if _, err := svc.GenerateSuggestion(ctx, uuid.New(), uuid.New()); !errors.Is(err, ai.ErrBudgetExhausted) {
		t.Fatalf("suggestion err = %v, want ErrBudgetExhausted", err)
	}
	if _, err := svc.SuggestVideos(ctx, uuid.New(), uuid.New()); !errors.Is(err, ai.ErrBudgetExhausted) {
		t.Fatalf("videos err = %v, want ErrBudgetExhausted", err)
	}
	if videos.asked != 0 {
		t.Fatal("the video finder ran after the search terms were refused")
	}
}
