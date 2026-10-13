package plangw

import (
	"context"
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

// recordingChecklists answers every checklist operation and records that it
// was reached.
type recordingChecklists struct {
	calls []string
}

func (r *recordingChecklists) item(planID, id uuid.UUID) *ChecklistResp {
	return &ChecklistResp{ID: id, PlanID: planID}
}

func (r *recordingChecklists) CreateChecklist(ctx context.Context, planID, userID uuid.UUID, req CreateChecklistReq) (*ChecklistResp, error) {
	r.calls = append(r.calls, "create")
	return r.item(planID, uuid.New()), nil
}

func (r *recordingChecklists) GetChecklist(ctx context.Context, planID, id, userID uuid.UUID) (*ChecklistResp, error) {
	r.calls = append(r.calls, "get")
	return r.item(planID, id), nil
}

func (r *recordingChecklists) ListChecklists(ctx context.Context, planID, userID uuid.UUID, scope, itemType *string) ([]*ChecklistResp, error) {
	r.calls = append(r.calls, "list")
	return nil, nil
}

func (r *recordingChecklists) ListArchivedChecklists(ctx context.Context, planID, userID uuid.UUID) ([]*ChecklistResp, error) {
	r.calls = append(r.calls, "list archived")
	return nil, nil
}

func (r *recordingChecklists) ListUpcomingChecklists(ctx context.Context, planID, userID uuid.UUID) ([]*ChecklistResp, error) {
	r.calls = append(r.calls, "list upcoming")
	return nil, nil
}

func (r *recordingChecklists) UpdateChecklist(ctx context.Context, planID, id, userID uuid.UUID, req UpdateChecklistReq) (*ChecklistResp, error) {
	r.calls = append(r.calls, "update")
	return r.item(planID, id), nil
}

func (r *recordingChecklists) ReorderChecklists(ctx context.Context, planID, userID uuid.UUID, req ReorderChecklistReq) ([]*ChecklistResp, error) {
	r.calls = append(r.calls, "reorder")
	return nil, nil
}

func (r *recordingChecklists) UpdateChecklistDates(ctx context.Context, planID, id, userID uuid.UUID, req UpdateDatesReq) (*ChecklistResp, error) {
	r.calls = append(r.calls, "dates")
	return r.item(planID, id), nil
}

func (r *recordingChecklists) ArchiveChecklist(ctx context.Context, planID, id, userID uuid.UUID, archived bool) (*ChecklistResp, error) {
	r.calls = append(r.calls, "archive")
	return r.item(planID, id), nil
}

func (r *recordingChecklists) DeleteChecklist(ctx context.Context, planID, id, userID uuid.UUID) error {
	r.calls = append(r.calls, "delete")
	return nil
}

func newChecklistsAPI(t *testing.T, c ChecklistsClient, access *Access) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	RegisterChecklistOperations(api, c, access, func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), uuid.New())))
	}, nil)
	return api.(humatest.TestAPI)
}

// allowAll is an access seam that answers owner for every plan, for tests
// about something other than access.
func allowAll() *Access {
	return NewAccess(&fakeRelations{rel: RelationOwner})
}

// checklistCall is one checklist route, sent under a plan URL.
type checklistCall struct {
	name       string
	wantStatus int
	do         func(api humatest.TestAPI, planURL string) *httptest.ResponseRecorder
}

func checklistCalls() []checklistCall {
	item := "/checklists/" + uuid.New().String()
	return []checklistCall{
		{"list", http.StatusOK, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Get(u + "/checklists")
		}},
		{"list archived", http.StatusOK, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Get(u + "/checklists/archived")
		}},
		{"list upcoming", http.StatusOK, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Get(u + "/checklists/upcoming")
		}},
		{"create", http.StatusCreated, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Post(u+"/checklists", map[string]any{"description": "d"})
		}},
		{"reorder", http.StatusOK, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Patch(u+"/checklists/order", map[string]any{
				"scope": "daily", "parentId": nil, "ids": []string{uuid.New().String()},
			})
		}},
		{"get", http.StatusOK, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Get(u + item)
		}},
		{"update", http.StatusOK, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Patch(u+item, map[string]any{"done": true})
		}},
		{"dates", http.StatusOK, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Patch(u+item+"/dates", map[string]any{"startDate": "2026-10-20"})
		}},
		{"archive", http.StatusOK, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Patch(u+item+"/archive", map[string]any{"archived": true})
		}},
		{"delete", http.StatusNoContent, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Delete(u + item)
		}},
	}
}

// R7, R5, R8 across every checklist route: a member and an owner are served, a
// stranger gets 404, a failed check fails the route, and a missing checker
// refuses with 403. A refused request never reaches plan-service.
func TestChecklistOperations_Access(t *testing.T) {
	cases := []struct {
		name       string
		access     *Access
		wantStatus int // 0: the route's own success status
	}{
		{"owner", NewAccess(&fakeRelations{rel: RelationOwner}), 0},
		{"member", NewAccess(&fakeRelations{rel: RelationMember}), 0},
		{"stranger", NewAccess(&fakeRelations{rel: RelationNone}), http.StatusNotFound},
		{"plan-service down", NewAccess(&fakeRelations{err: status.Error(codes.Unavailable, "down")}), http.StatusServiceUnavailable},
		{"no checker", nil, http.StatusForbidden},
	}
	for _, tc := range cases {
		for _, call := range checklistCalls() {
			t.Run(tc.name+"/"+call.name, func(t *testing.T) {
				rec := &recordingChecklists{}
				api := newChecklistsAPI(t, rec, tc.access)

				resp := call.do(api, "/api/plans/"+uuid.New().String())

				want := tc.wantStatus
				if want == 0 {
					want = call.wantStatus
				}
				if resp.Code != want {
					t.Fatalf("status = %d, want %d: %s", resp.Code, want, resp.Body.String())
				}
				reached := len(rec.calls) > 0
				if allowed := tc.wantStatus == 0; reached != allowed {
					t.Errorf("plan-service reached = %v, want %v (calls %v)", reached, allowed, rec.calls)
				}
			})
		}
	}
}
