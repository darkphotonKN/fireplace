package plangw

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"

	"github.com/danielgtaylor/huma/v2"
	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
)

// SERIALIZED plans operations (FS-0004 §API surface, plans).
//
// Ownership checks live in plan-service and surface as 403. The gateway
// supplies the caller identity and never restates a downstream rule (ADR-0005).

// SearchResult is the transport mirror for a search hit.
//
// It exists because the client returns []*pb.SearchPlanResult directly, and a
// protobuf message must never reach components.schemas (ADR-0003 §3). Every
// other plans response already had a local type; this was the one gap.
type SearchResult struct {
	ID          string  `json:"id" doc:"Plan id"`
	Name        string  `json:"name" doc:"Plan name"`
	Description string  `json:"description,omitempty" doc:"Plan description"`
	Similarity  float32 `json:"similarity" doc:"Relevance score for the query term"`
}

func searchResultFromProto(r *pb.SearchPlanResult) SearchResult {
	return SearchResult{
		ID:          r.Id,
		Name:        r.Name,
		Description: r.Description,
		Similarity:  r.Similarity,
	}
}

// PlansClient is the narrow seam these operations depend on.
type PlansClient interface {
	CreatePlan(ctx context.Context, userID uuid.UUID, req CreatePlanReq) (*PlanResp, error)
	GetPlan(ctx context.Context, id, userID uuid.UUID) (*PlanResp, error)
	ListPlans(ctx context.Context, userID uuid.UUID) ([]*PlanResp, error)
	ListSharedPlans(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*PlanResp, error)
	SearchPlans(ctx context.Context, userID uuid.UUID, params SearchParam) ([]*pb.SearchPlanResult, error)
	UpdatePlan(ctx context.Context, id, userID uuid.UUID, req UpdatePlanReq) (*PlanResp, error)
	ToggleDailyReset(ctx context.Context, id, userID uuid.UUID) (*PlanResp, error)
	DeletePlan(ctx context.Context, id, userID uuid.UUID) error
}

// --- huma input/output wrappers -------------------------------------------

type PlanOutput struct{ Body PlanResp }
type PlanListOutput struct{ Body []PlanResp }

type PlanIDInput struct {
	ID uuid.UUID `path:"id" doc:"Plan id"`
}

type CreatePlanInput struct{ Body CreatePlanReq }

type UpdatePlanInput struct {
	ID   uuid.UUID `path:"id" doc:"Plan id"`
	Body UpdatePlanReq
}

// SearchPlansInput mirrors the legacy `form`-bound SearchParam. limit and
// offset are STRINGS there and are parsed with a silent fallback; typed as
// integers here so the contract states what they are, with the same defaults.
type SearchPlansInput struct {
	Term   string `query:"term" required:"true" doc:"Search term"`
	Limit  int    `query:"limit" default:"20" doc:"Maximum results"`
	Offset int    `query:"offset" doc:"Results to skip"`
}

type SearchOutput struct{ Body []SearchResult }

type ListSharedInput struct {
	Limit  int `query:"limit" default:"20" doc:"Maximum results"`
	Offset int `query:"offset" doc:"Results to skip"`
}

// DeleteOutput carries no body: the operation answers 204.
type DeleteOutput struct{}

func RegisterPlanOperations(api huma.API, c PlansClient,
	protect func(huma.Context, func(huma.Context)), secured []map[string][]string,
) {
	mw := huma.Middlewares{protect}
	errs := []int{
		http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
		http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
	}

	huma.Register(api, huma.Operation{
		OperationID: "listPlans", Method: http.MethodGet, Path: "/api/plans",
		Middlewares: mw, Security: secured,
		Summary:     "List the caller's plans",
		Description: "Returns every plan owned by the authenticated user.",
		Errors:      []int{http.StatusUnauthorized, http.StatusServiceUnavailable},
	}, func(ctx context.Context, _ *struct{}) (*PlanListOutput, error) {
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return nil, apierr.ProblemFor("list plans", errNoIdentity())
		}
		plans, err := c.ListPlans(ctx, userID)
		if err != nil {
			return nil, apierr.ProblemFor("list plans", err)
		}
		return &PlanListOutput{Body: derefPlans(plans)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "searchPlans", Method: http.MethodGet, Path: "/api/plans/search",
		Middlewares: mw, Security: secured,
		Summary:     "Search plans",
		Description: "Full-text search across the caller's plans, ranked by similarity.",
		Errors: []int{
			http.StatusUnauthorized, http.StatusUnprocessableEntity,
			http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *SearchPlansInput) (*SearchOutput, error) {
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return nil, apierr.ProblemFor("search plans", errNoIdentity())
		}
		results, err := c.SearchPlans(ctx, userID, SearchParam{
			Term:   in.Term,
			Limit:  itoa(in.Limit),
			Offset: itoa(in.Offset),
		})
		if err != nil {
			return nil, apierr.ProblemFor("search plans", err)
		}
		out := make([]SearchResult, 0, len(results))
		for _, r := range results {
			out = append(out, searchResultFromProto(r))
		}
		return &SearchOutput{Body: out}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listSharedPlans", Method: http.MethodGet, Path: "/api/plans/shared",
		Middlewares: mw, Security: secured,
		Summary:     "List plans shared with the caller",
		Description: "Returns plans other users have shared with the authenticated user.",
		Errors: []int{
			http.StatusUnauthorized, http.StatusUnprocessableEntity,
			http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *ListSharedInput) (*PlanListOutput, error) {
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return nil, apierr.ProblemFor("list shared plans", errNoIdentity())
		}
		plans, err := c.ListSharedPlans(ctx, userID, in.Limit, in.Offset)
		if err != nil {
			return nil, apierr.ProblemFor("list shared plans", err)
		}
		return &PlanListOutput{Body: derefPlans(plans)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getPlan", Method: http.MethodGet, Path: "/api/plans/{id}",
		Middlewares: mw, Security: secured,
		Summary:     "Get a plan",
		Description: "Returns one plan by id. Ownership is enforced by plan-service, which answers 403 when the caller does not own it.",
		Errors:      errs,
	}, func(ctx context.Context, in *PlanIDInput) (*PlanOutput, error) {
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return nil, apierr.ProblemFor("get plan", errNoIdentity())
		}
		plan, err := c.GetPlan(ctx, in.ID, userID)
		if err != nil {
			return nil, apierr.ProblemFor("get plan", err)
		}
		return &PlanOutput{Body: *plan}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "createPlan", Method: http.MethodPost, Path: "/api/plans",
		DefaultStatus: http.StatusCreated,
		Middlewares:   mw, Security: secured,
		Summary:     "Create a plan",
		Description: "Creates a plan owned by the authenticated user.",
		Errors: []int{
			http.StatusBadRequest, http.StatusUnauthorized,
			http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *CreatePlanInput) (*PlanOutput, error) {
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return nil, apierr.ProblemFor("create plan", errNoIdentity())
		}
		plan, err := c.CreatePlan(ctx, userID, in.Body)
		if err != nil {
			return nil, apierr.ProblemFor("create plan", err)
		}
		return &PlanOutput{Body: *plan}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "updatePlan", Method: http.MethodPatch, Path: "/api/plans/{id}",
		Middlewares: mw, Security: secured,
		Summary:     "Update a plan",
		Description: "Partial update. Omitted fields are left unchanged.",
		Errors:      append([]int{http.StatusBadRequest}, errs...),
	}, func(ctx context.Context, in *UpdatePlanInput) (*PlanOutput, error) {
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return nil, apierr.ProblemFor("update plan", errNoIdentity())
		}
		plan, err := c.UpdatePlan(ctx, in.ID, userID, in.Body)
		if err != nil {
			return nil, apierr.ProblemFor("update plan", err)
		}
		return &PlanOutput{Body: *plan}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "toggleDailyReset", Method: http.MethodPatch,
		Path:        "/api/plans/{id}/toggle-daily-reset",
		Middlewares: mw, Security: secured,
		Summary:     "Toggle a plan's daily-reset flag",
		Description: "Flips dailyReset and returns the updated plan. Takes no body.",
		Errors:      errs,
	}, func(ctx context.Context, in *PlanIDInput) (*PlanOutput, error) {
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return nil, apierr.ProblemFor("toggle daily_reset", errNoIdentity())
		}
		plan, err := c.ToggleDailyReset(ctx, in.ID, userID)
		if err != nil {
			return nil, apierr.ProblemFor("toggle daily_reset", err)
		}
		return &PlanOutput{Body: *plan}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "deletePlan", Method: http.MethodDelete, Path: "/api/plans/{id}",
		DefaultStatus: http.StatusNoContent,
		Middlewares:   mw, Security: secured,
		Summary:     "Delete a plan",
		Description: "Deletes a plan. Answers 204 with no body.",
		Errors:      errs,
	}, func(ctx context.Context, in *PlanIDInput) (*DeleteOutput, error) {
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return nil, apierr.ProblemFor("delete plan", errNoIdentity())
		}
		if err := c.DeletePlan(ctx, in.ID, userID); err != nil {
			return nil, apierr.ProblemFor("delete plan", err)
		}
		return &DeleteOutput{}, nil
	})
}

// derefPlans copies to values and is explicitly non-nil, so an empty result
// marshals to [] and never null (FS-0004 §Edge States).
func derefPlans(in []*PlanResp) []PlanResp {
	out := make([]PlanResp, 0, len(in))
	for _, p := range in {
		if p != nil {
			out = append(out, *p)
		}
	}
	return out
}

func itoa(i int) string { return strconv.Itoa(i) }

// errNoIdentity is the sentinel for "the identity bridge yielded nothing",
// which should be unreachable behind the auth middleware but must still map to
// a contract-shaped error rather than a nil-pointer panic.
func errNoIdentity() error {
	return fmt.Errorf("%w: no identity in context", commonconstants.ErrUnauthorized)
}
