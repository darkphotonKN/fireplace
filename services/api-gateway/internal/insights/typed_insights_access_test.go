package insights

import (
	"context"
	"net/http"
	"slices"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	plangw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/plan"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// relations plays plan-service's relation answer behind the gateway's real
// access seam.
type relations struct {
	rel plangw.Relation
	err error
}

func (r relations) GetPlanAccess(ctx context.Context, planID, userID uuid.UUID) (plangw.Relation, error) {
	return r.rel, r.err
}

var insightsPaths = []string{
	"/api/insights/checklist-suggestion",
	"/api/insights/checklist-suggestion-daily",
	"/api/insights/suggest-videos",
}

// R4, R7, R5, R8: the three ?plan_id= routes were open to any signed-in caller.
// They are now owner-or-member, a stranger gets 404, and a refused request
// never reaches insights-service.
func TestInsightsOperations_Access(t *testing.T) {
	cases := []struct {
		name       string
		access     PlanAccess
		wantStatus int
	}{
		{"owner", plangw.NewAccess(relations{rel: plangw.RelationOwner}), http.StatusOK},
		{"member", plangw.NewAccess(relations{rel: plangw.RelationMember}), http.StatusOK},
		{"stranger", plangw.NewAccess(relations{rel: plangw.RelationNone}), http.StatusNotFound},
		{"plan-service down", plangw.NewAccess(relations{err: status.Error(codes.Unavailable, "down")}), http.StatusServiceUnavailable},
		{"no checker", nil, http.StatusForbidden},
	}
	for _, tc := range cases {
		for _, path := range insightsPaths {
			t.Run(tc.name+path, func(t *testing.T) {
				s, v := &recordingSuggestions{}, &recordingVideos{}
				_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
				RegisterInsightsOperations(api, s, v, tc.access, func(ctx huma.Context, next func(huma.Context)) {
					next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), uuid.New())))
				}, nil)

				resp := api.(humatest.TestAPI).Get(path + "?plan_id=" + uuid.New().String())

				if resp.Code != tc.wantStatus {
					t.Fatalf("status = %d, want %d: %s", resp.Code, tc.wantStatus, resp.Body.String())
				}
				reached := s.gotPlanID != uuid.Nil || v.gotPlanID != uuid.Nil
				if allowed := tc.wantStatus == http.StatusOK; reached != allowed {
					t.Errorf("insights-service reached = %v, want %v", reached, allowed)
				}
			})
		}
	}
}

// The three operations declare the 404 they now produce.
func TestInsightsOperations_Declare404(t *testing.T) {
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	RegisterInsightsOperations(api, nil, nil, nil, func(ctx huma.Context, next func(huma.Context)) {}, nil)

	for _, path := range insightsPaths {
		op := api.OpenAPI().Paths[path].Get
		if op == nil {
			t.Fatalf("%s: no GET operation", path)
		}
		if _, ok := op.Responses["404"]; !ok {
			keys := make([]string, 0, len(op.Responses))
			for k := range op.Responses {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			t.Errorf("%s declares %v, want 404 among them", path, keys)
		}
	}
}
