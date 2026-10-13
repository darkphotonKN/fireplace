package notes

import (
	"context"
	"net/http"
	"net/http/httptest"
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
// access seam, so these tests exercise the policy notes actually runs under.
type relations struct {
	rel plangw.Relation
	err error
}

func (r relations) GetPlanAccess(ctx context.Context, planID, userID uuid.UUID) (plangw.Relation, error) {
	return r.rel, r.err
}

// everyNote answers every notes operation with a note in whichever plan it is
// asked about, and records that it was reached.
type everyNote struct {
	planID uuid.UUID
	calls  []string
}

func (e *everyNote) note() *Note { return &Note{ID: uuid.New(), PlanID: e.planID} }

func (e *everyNote) CreateNote(planID uuid.UUID, req *CreateNoteReq) (*Note, error) {
	e.calls = append(e.calls, "create")
	return e.note(), nil
}

func (e *everyNote) GetNoteByID(id uuid.UUID) (*Note, error) {
	e.calls = append(e.calls, "get")
	return e.note(), nil
}

func (e *everyNote) GetNotesByPlanID(planID uuid.UUID, filters *FilterOptions) ([]Note, error) {
	e.calls = append(e.calls, "list")
	return nil, nil
}

func (e *everyNote) UpdateNote(id uuid.UUID, updates *UpdateNoteReq) (*Note, error) {
	e.calls = append(e.calls, "update")
	return e.note(), nil
}

func (e *everyNote) DeleteNote(id uuid.UUID) error {
	e.calls = append(e.calls, "delete")
	return nil
}

func (e *everyNote) GenerateAINotes(planID uuid.UUID, requestType string) ([]Note, error) {
	e.calls = append(e.calls, "generate")
	return nil, nil
}

type noteCall struct {
	name       string
	wantStatus int
	do         func(api humatest.TestAPI, planURL string) *httptest.ResponseRecorder
}

func noteCalls() []noteCall {
	note := "/notes/" + uuid.New().String()
	return []noteCall{
		{"list", http.StatusOK, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Get(u + "/notes")
		}},
		{"create", http.StatusCreated, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Post(u+"/notes", map[string]any{"content": "c"})
		}},
		{"generate ai", http.StatusOK, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Post(u+"/notes/generate-ai", map[string]any{})
		}},
		{"get", http.StatusOK, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Get(u + note)
		}},
		{"update", http.StatusOK, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Patch(u+note, map[string]any{"isRead": true})
		}},
		{"delete", http.StatusNoContent, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Delete(u + note)
		}},
	}
}

// R7, R5, R8, R10 across every notes route: owner and member are served, a
// stranger gets 404 (no longer the undeclared 403), a failed check fails the
// route, and a missing checker refuses with 403. A refused request never
// reaches the notes service.
func TestNotesOperations_Access(t *testing.T) {
	cases := []struct {
		name       string
		access     PlanAccess
		wantStatus int // 0: the route's own success status
	}{
		{"owner", plangw.NewAccess(relations{rel: plangw.RelationOwner}), 0},
		{"member", plangw.NewAccess(relations{rel: plangw.RelationMember}), 0},
		{"stranger", plangw.NewAccess(relations{rel: plangw.RelationNone}), http.StatusNotFound},
		{"plan-service down", plangw.NewAccess(relations{err: status.Error(codes.Unavailable, "down")}), http.StatusServiceUnavailable},
		{"no checker", nil, http.StatusForbidden},
	}
	for _, tc := range cases {
		for _, call := range noteCalls() {
			t.Run(tc.name+"/"+call.name, func(t *testing.T) {
				planID := uuid.New()
				svc := &everyNote{planID: planID}
				_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
				RegisterNotesOperations(api, svc, tc.access, func(ctx huma.Context, next func(huma.Context)) {
					next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), uuid.New())))
				}, nil)

				resp := call.do(api.(humatest.TestAPI), "/api/plans/"+planID.String())

				want := tc.wantStatus
				if want == 0 {
					want = call.wantStatus
				}
				if resp.Code != want {
					t.Fatalf("status = %d, want %d: %s", resp.Code, want, resp.Body.String())
				}
				reached := len(svc.calls) > 0
				if allowed := tc.wantStatus == 0; reached != allowed {
					t.Errorf("notes service reached = %v, want %v (calls %v)", reached, allowed, svc.calls)
				}
			})
		}
	}
}
