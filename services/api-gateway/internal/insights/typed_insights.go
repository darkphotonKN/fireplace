package insights

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/discovery"
	"github.com/google/uuid"
)

// SERIALIZED insights operations (FS-0004 §API surface, insights).
//
// This is slice ⓪ for these three endpoints under ADR-0002 serialize-on-touch:
// a behaviour-preserving typed wrap that establishes the contract BEFORE the
// handlers are repointed at insights-service over gRPC. Everything here
// transcribes what ships today so the repoint shows up as a reviewable diff
// rather than an invisible drift.

// SuggestionsService and VideoSuggestionsService are two seams, not one,
// because they are two different objects: the router builds one service around
// the checklist generator and another around the search-term generator. A
// single fat interface would force each to carry a method it does not serve.
type SuggestionsService interface {
	GenerateSuggestions(ctx context.Context, planID uuid.UUID) (string, error)
	GenerateDailySuggestions(ctx context.Context, planID uuid.UUID) ([]string, error)
}

type VideoSuggestionsService interface {
	GenerateSuggestedVideoLinks(ctx context.Context, planID uuid.UUID) ([]discovery.Resource, error)
}

// --- transport mirror ------------------------------------------------------

// VideoSuggestionResponse mirrors discovery.Resource field for field (R6).
//
// `source` and `type` are transcribed even though the current finder never
// populates them — they ship as empty strings today, and dropping them from the
// contract would be a shape change, not a transcription.
type VideoSuggestionResponse struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Source      string `json:"source"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Bodies are BARE — a string and two arrays — because that is exactly what
// ships inside `result` today, and every other collection in this contract is
// published as a bare array. Wrapping them would invent a field name that
// exists nowhere in the current API (R6). The output types carry distinct Go
// names so nothing collides at registration.
type SuggestionOutput struct{ Body string }
type DailySuggestionsOutput struct{ Body []string }
type VideoSuggestionsOutput struct{ Body []VideoSuggestionResponse }

// PlanIDQueryInput is the shape all three operations share. `plan_id` is
// snake_case while notes uses camelCase; both are transcribed as they ship.
type PlanIDQueryInput struct {
	PlanID uuid.UUID `query:"plan_id" doc:"Plan id"`
}

func RegisterInsightsOperations(api huma.API, s SuggestionsService, v VideoSuggestionsService,
	protect func(huma.Context, func(huma.Context)), secured []map[string][]string,
) {
	mw := huma.Middlewares{protect}
	errs := []int{
		http.StatusUnauthorized, http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
	}

	identity := func(ctx context.Context, op string) error {
		if _, ok := auth.UserIDFromCtx(ctx); !ok {
			return apierr.ProblemFor(op, apierr.ErrNoIdentity())
		}
		return nil
	}

	huma.Register(api, huma.Operation{
		OperationID: "getChecklistSuggestion", Method: http.MethodGet,
		Path:        "/api/insights/checklist-suggestion",
		Middlewares: mw, Security: secured,
		Summary: "Suggest the next checklist item",
		Description: "Returns one concrete, verb-first next task derived from the plan's focus " +
			"and current checklist. The body is the suggestion itself.",
		Errors: errs,
	}, func(ctx context.Context, in *PlanIDQueryInput) (*SuggestionOutput, error) {
		if err := identity(ctx, "insights: generate suggestions"); err != nil {
			return nil, err
		}
		res, err := s.GenerateSuggestions(ctx, in.PlanID)
		if err != nil {
			return nil, apierr.ProblemFor("insights: generate suggestions", err)
		}
		return &SuggestionOutput{Body: res}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getDailyChecklistSuggestion", Method: http.MethodGet,
		Path:        "/api/insights/checklist-suggestion-daily",
		Middlewares: mw, Security: secured,
		Summary: "Suggest daily items from long-term work",
		Description: "Returns three suggestions biased toward breaking down long-term checklist " +
			"items, each nudged away from the previous so the set does not repeat itself.",
		Errors: errs,
	}, func(ctx context.Context, in *PlanIDQueryInput) (*DailySuggestionsOutput, error) {
		if err := identity(ctx, "insights: generate daily suggestions"); err != nil {
			return nil, err
		}
		res, err := s.GenerateDailySuggestions(ctx, in.PlanID)
		if err != nil {
			return nil, apierr.ProblemFor("insights: generate daily suggestions", err)
		}
		if res == nil {
			res = []string{}
		}
		return &DailySuggestionsOutput{Body: res}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getSuggestedVideos", Method: http.MethodGet,
		Path:        "/api/insights/suggest-videos",
		Middlewares: mw, Security: secured,
		Summary: "Suggest learning videos for a plan",
		Description: "Generates search terms from the plan's focus and checklist, then returns " +
			"one recommended video per term.",
		Errors: errs,
	}, func(ctx context.Context, in *PlanIDQueryInput) (*VideoSuggestionsOutput, error) {
		if err := identity(ctx, "insights: suggested video links"); err != nil {
			return nil, err
		}
		res, err := v.GenerateSuggestedVideoLinks(ctx, in.PlanID)
		if err != nil {
			return nil, apierr.ProblemFor("insights: suggested video links", err)
		}
		return &VideoSuggestionsOutput{Body: toVideoResponses(res)}, nil
	})
}

// toVideoResponses is explicitly non-nil so an empty result marshals to [] and
// never null (FS-0004 §Edge States).
func toVideoResponses(in []discovery.Resource) []VideoSuggestionResponse {
	out := make([]VideoSuggestionResponse, 0, len(in))
	for _, r := range in {
		out = append(out, VideoSuggestionResponse{
			Title:       r.Title,
			URL:         r.URL,
			Source:      r.Source,
			Type:        string(r.Type),
			Description: r.Description,
		})
	}
	return out
}
