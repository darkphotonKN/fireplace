package insights

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
)

// SERIALIZED insights operations (FS-0004 §API surface, insights).
//
// These endpoints were serialized first (I-0019) and repointed at
// insights-service afterwards, in that order, because ADR-0002 §6 forbids a
// handler rewrite preceding its endpoints' serialization. The contract below
// was established while the in-process implementation still served it, so the
// move to gRPC was a diff the gates could see rather than an invisible drift.
//
// The response shapes are unchanged by the move: the HTTP surface is identical
// either side of it, which is exactly what slice ⓪ bought.

// SuggestionsService and VideoSuggestionsService are two seams, not one, so a
// caller depends only on what it consumes. Both are satisfied by the
// insights-service gRPC client (internal/gateway/insights).
//
// Both carry userID, which the in-process implementation did not. Neither is
// where access is decided: plan-service's GetPlan does not check the caller, so
// these routes ask the gateway's access seam first (PlanAccess below).
type SuggestionsService interface {
	GenerateSuggestions(ctx context.Context, planID, userID uuid.UUID) (string, error)
	GenerateDailySuggestions(ctx context.Context, planID, userID uuid.UUID) ([]string, error)
}

type VideoSuggestionsService interface {
	SuggestVideos(ctx context.Context, planID, userID uuid.UUID) ([]VideoSuggestionResponse, error)
}

// PlanAccess is the gateway's one plan-access seam (plangw.Access, ADR-0015),
// declared at the consumer. All three operations are owner-or-member on their
// plan_id (FS-12ZMF R4, R7): a caller with no relation gets 404.
type PlanAccess interface {
	RequireMember(ctx context.Context, planID, userID uuid.UUID) error
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
	access PlanAccess,
	protect func(huma.Context, func(huma.Context)), secured []map[string][]string,
) {
	mw := huma.Middlewares{protect}
	errs := []int{
		http.StatusUnauthorized, http.StatusNotFound, http.StatusUnprocessableEntity,
		http.StatusServiceUnavailable,
	}

	// authorize resolves the caller, then asks whether they may read the plan.
	// A nil access checker fails CLOSED (R8).
	authorize := func(ctx context.Context, op string, planID uuid.UUID) (uuid.UUID, error) {
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return uuid.Nil, apierr.ProblemFor(op, apierr.ErrNoIdentity())
		}
		if access == nil {
			return uuid.Nil, apierr.ProblemFor(op,
				fmt.Errorf("%w: plan access checker not configured", commonconstants.ErrForbidden))
		}
		if err := access.RequireMember(ctx, planID, userID); err != nil {
			return uuid.Nil, apierr.ProblemFor(op, err)
		}
		return userID, nil
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
		userID, err := authorize(ctx, "insights: generate suggestions", in.PlanID)
		if err != nil {
			return nil, err
		}
		res, err := s.GenerateSuggestions(ctx, in.PlanID, userID)
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
		userID, err := authorize(ctx, "insights: generate daily suggestions", in.PlanID)
		if err != nil {
			return nil, err
		}
		res, err := s.GenerateDailySuggestions(ctx, in.PlanID, userID)
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
		userID, err := authorize(ctx, "insights: suggested video links", in.PlanID)
		if err != nil {
			return nil, err
		}
		res, err := v.SuggestVideos(ctx, in.PlanID, userID)
		if err != nil {
			return nil, apierr.ProblemFor("insights: suggested video links", err)
		}
		if res == nil {
			res = []VideoSuggestionResponse{}
		}
		return &VideoSuggestionsOutput{Body: res}, nil
	})
}
