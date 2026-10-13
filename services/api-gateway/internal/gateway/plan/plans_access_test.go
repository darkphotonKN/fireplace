package plangw

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeRelations plays plan-service's GetPlanAccess: one fixed answer.
type fakeRelations struct {
	rel   Relation
	err   error
	asked int
}

func (f *fakeRelations) GetPlanAccess(ctx context.Context, planID, userID uuid.UUID) (Relation, error) {
	f.asked++
	return f.rel, f.err
}

// The real client is the seam's source; it answers the relation alone, so the
// owner id plan-service sends never crosses into the gateway's access policy.
var _ RelationSource = (*Client)(nil)

// recordingPlans plays the plans half of plan-service and records every call
// that reached it, so a test can prove a refused request never got there.
type recordingPlans struct {
	PlansClient
	calls []string
	owner uuid.UUID // userId on every plan it returns
}

func (r *recordingPlans) plan(id uuid.UUID) *PlanResp {
	return &PlanResp{ID: id, UserID: r.owner, Name: "p"}
}

func (r *recordingPlans) GetPlan(ctx context.Context, id, userID uuid.UUID) (*PlanResp, error) {
	r.calls = append(r.calls, "get")
	return r.plan(id), nil
}

func (r *recordingPlans) CreatePlan(ctx context.Context, userID uuid.UUID, req CreatePlanReq) (*PlanResp, error) {
	r.calls = append(r.calls, "create")
	return r.plan(uuid.New()), nil
}

func (r *recordingPlans) UpdatePlan(ctx context.Context, id, userID uuid.UUID, req UpdatePlanReq) (*PlanResp, error) {
	r.calls = append(r.calls, "update")
	return r.plan(id), nil
}

func (r *recordingPlans) ToggleDailyReset(ctx context.Context, id, userID uuid.UUID) (*PlanResp, error) {
	r.calls = append(r.calls, "toggle")
	return r.plan(id), nil
}

func (r *recordingPlans) DeletePlan(ctx context.Context, id, userID uuid.UUID) error {
	r.calls = append(r.calls, "delete")
	return nil
}

func newPlansAPI(t *testing.T, plans PlansClient, access *Access, caller uuid.UUID) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	RegisterPlanOperations(api, plans, access, func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), caller)))
	}, nil)
	return api.(humatest.TestAPI)
}

// planCall is one plan-by-id route.
type planCall struct {
	name string
	do   func(api humatest.TestAPI, planURL string) *httptest.ResponseRecorder
}

var (
	getPlanCall = planCall{"GET", func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
		return api.Get(u)
	}}
	patchPlanCall = planCall{"PATCH", func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
		return api.Patch(u, map[string]any{"name": "renamed"})
	}}
	togglePlanCall = planCall{"PATCH toggle", func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
		return api.Patch(u + "/toggle-daily-reset")
	}}
	deletePlanCall = planCall{"DELETE", func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
		return api.Delete(u)
	}}
	ownerOnlyPlanCalls = []planCall{patchPlanCall, togglePlanCall, deletePlanCall}
	allPlanCalls       = []planCall{getPlanCall, patchPlanCall, togglePlanCall, deletePlanCall}
)

func problemCode(t *testing.T, resp *httptest.ResponseRecorder) string {
	t.Helper()
	var p struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &p); err != nil {
		t.Fatalf("body is not a problem: %v: %s", err, resp.Body.String())
	}
	return p.Code
}

func relationOf(t *testing.T, resp *httptest.ResponseRecorder) string {
	t.Helper()
	var p struct {
		Relation string `json:"relation"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &p); err != nil {
		t.Fatalf("body is not a plan: %v: %s", err, resp.Body.String())
	}
	return p.Relation
}

// R6, R9: a member's PATCH is refused before plan-service is asked to update,
// which is what ends FS-0005's silent 200 no-op.
func TestUpdatePlan_Member_Is403AndNeverReachesUpdate(t *testing.T) {
	plans := &recordingPlans{}
	api := newPlansAPI(t, plans, NewAccess(&fakeRelations{rel: RelationMember}), uuid.New())

	resp := patchPlanCall.do(api, "/api/plans/"+uuid.New().String())

	if resp.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", resp.Code, resp.Body.String())
	}
	if got := problemCode(t, resp); got != "FORBIDDEN" {
		t.Errorf("code = %q, want FORBIDDEN", got)
	}
	if len(plans.calls) != 0 {
		t.Errorf("plan-service was called: %v", plans.calls)
	}
}

// R6: every owner-only operation refuses a member, and nothing changes.
func TestOwnerOnlyPlanOperations_Member_Is403(t *testing.T) {
	for _, call := range ownerOnlyPlanCalls {
		t.Run(call.name, func(t *testing.T) {
			plans := &recordingPlans{}
			api := newPlansAPI(t, plans, NewAccess(&fakeRelations{rel: RelationMember}), uuid.New())

			resp := call.do(api, "/api/plans/"+uuid.New().String())

			if resp.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403: %s", resp.Code, resp.Body.String())
			}
			if len(plans.calls) != 0 {
				t.Errorf("plan-service was called: %v", plans.calls)
			}
		})
	}
}

// R5: a stranger (or a plan that does not exist) gets 404 on every route, and
// the body carries nothing of the plan, not even its owner.
func TestPlanOperations_Stranger_Is404(t *testing.T) {
	for _, call := range allPlanCalls {
		t.Run(call.name, func(t *testing.T) {
			plans := &recordingPlans{}
			api := newPlansAPI(t, plans, NewAccess(&fakeRelations{rel: RelationNone}), uuid.New())

			resp := call.do(api, "/api/plans/"+uuid.New().String())

			if resp.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404: %s", resp.Code, resp.Body.String())
			}
			if got := problemCode(t, resp); got != "NOT_FOUND" {
				t.Errorf("code = %q, want NOT_FOUND", got)
			}
			if len(plans.calls) != 0 {
				t.Errorf("plan-service was called: %v", plans.calls)
			}
		})
	}
}

// R7, R23: a member reads the plan, and the body says they are a member.
func TestGetPlan_Member_Is200WithRelationMember(t *testing.T) {
	plans := &recordingPlans{owner: uuid.New()}
	api := newPlansAPI(t, plans, NewAccess(&fakeRelations{rel: RelationMember}), uuid.New())

	resp := getPlanCall.do(api, "/api/plans/"+uuid.New().String())

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.Code, resp.Body.String())
	}
	if got := relationOf(t, resp); got != "member" {
		t.Errorf("relation = %q, want member", got)
	}
}

// The owner sees no change apart from relation: "owner" on plan bodies.
func TestPlanOperations_Owner_Succeed(t *testing.T) {
	cases := []struct {
		call         planCall
		wantStatus   int
		wantRelation bool
	}{
		{getPlanCall, http.StatusOK, true},
		{patchPlanCall, http.StatusOK, true},
		{togglePlanCall, http.StatusOK, true},
		{deletePlanCall, http.StatusNoContent, false},
	}
	for _, tc := range cases {
		t.Run(tc.call.name, func(t *testing.T) {
			caller := uuid.New()
			plans := &recordingPlans{owner: caller}
			api := newPlansAPI(t, plans, NewAccess(&fakeRelations{rel: RelationOwner}), caller)

			resp := tc.call.do(api, "/api/plans/"+uuid.New().String())

			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d: %s", resp.Code, tc.wantStatus, resp.Body.String())
			}
			if len(plans.calls) != 1 {
				t.Errorf("plan-service calls = %v, want exactly one", plans.calls)
			}
			if tc.wantRelation {
				if got := relationOf(t, resp); got != "owner" {
					t.Errorf("relation = %q, want owner", got)
				}
			}
		})
	}
}

// R8: no answer means no access. An unreachable plan-service fails the request
// with its own status and the operation never runs.
func TestPlanOperations_AccessCheckFails_RouteFailsAndNeverProceeds(t *testing.T) {
	for _, call := range allPlanCalls {
		t.Run(call.name, func(t *testing.T) {
			plans := &recordingPlans{}
			down := &fakeRelations{err: status.Error(codes.Unavailable, "plan-service down")}
			api := newPlansAPI(t, plans, NewAccess(down), uuid.New())

			resp := call.do(api, "/api/plans/"+uuid.New().String())

			if resp.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want 503: %s", resp.Code, resp.Body.String())
			}
			if len(plans.calls) != 0 {
				t.Errorf("plan-service was called: %v", plans.calls)
			}
		})
	}
}

// R8: a missing checker is a 403, never an allow.
func TestPlanOperations_NilAccess_Is403(t *testing.T) {
	for _, call := range allPlanCalls {
		t.Run(call.name, func(t *testing.T) {
			plans := &recordingPlans{}
			api := newPlansAPI(t, plans, nil, uuid.New())

			resp := call.do(api, "/api/plans/"+uuid.New().String())

			if resp.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403: %s", resp.Code, resp.Body.String())
			}
			if len(plans.calls) != 0 {
				t.Errorf("plan-service was called: %v", plans.calls)
			}
		})
	}
}

// listedPlans returns one plan the caller owns and one somebody else owns.
type listedPlans struct {
	PlansClient
	caller, other uuid.UUID
}

func (l *listedPlans) both() []*PlanResp {
	return []*PlanResp{
		{ID: uuid.New(), UserID: l.caller, Name: "mine"},
		{ID: uuid.New(), UserID: l.other, Name: "theirs"},
	}
}

func (l *listedPlans) ListPlans(ctx context.Context, userID uuid.UUID) ([]*PlanResp, error) {
	return l.both(), nil
}

func (l *listedPlans) ListSharedPlans(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*PlanResp, error) {
	return l.both(), nil
}

// R23, D14: in a list, relation is owner exactly when userId is the caller.
// Lists ask no per-plan access question: they only ever hold plans the caller
// owns or shares.
func TestListOperations_RelationIsOwnerIffCallerOwns(t *testing.T) {
	for _, path := range []string{"/api/plans", "/api/plans/shared"} {
		t.Run(path, func(t *testing.T) {
			caller := uuid.New()
			plans := &listedPlans{caller: caller, other: uuid.New()}
			rels := &fakeRelations{rel: RelationNone}
			api := newPlansAPI(t, plans, NewAccess(rels), caller)

			resp := api.Get(path)

			if resp.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", resp.Code, resp.Body.String())
			}
			var got []struct {
				Name     string `json:"name"`
				Relation string `json:"relation"`
			}
			if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
				t.Fatalf("body: %v", err)
			}
			want := map[string]string{"mine": "owner", "theirs": "member"}
			if len(got) != len(want) {
				t.Fatalf("got %d plans, want %d", len(got), len(want))
			}
			for _, p := range got {
				if p.Relation != want[p.Name] {
					t.Errorf("plan %q: relation = %q, want %q", p.Name, p.Relation, want[p.Name])
				}
			}
			if rels.asked != 0 {
				t.Errorf("a list asked the access seam %d times", rels.asked)
			}
		})
	}
}

// R23: the creator of a plan is its owner.
func TestCreatePlan_RelationIsOwner(t *testing.T) {
	plans := &recordingPlans{}
	api := newPlansAPI(t, plans, NewAccess(&fakeRelations{rel: RelationNone}), uuid.New())

	resp := api.Post("/api/plans", map[string]any{"name": "n", "focus": "f", "planType": "project"})

	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", resp.Code, resp.Body.String())
	}
	if got := relationOf(t, resp); got != "owner" {
		t.Errorf("relation = %q, want owner", got)
	}
}
