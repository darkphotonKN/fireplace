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

// FS-12ZMF R11: an item-by-id route hands plan-service the plan in its path, and
// plan-service answers NotFound for an item that lives in another plan. This
// fake plays plan-service: it holds one item in one plan and scopes every call
// by the plan it is given, so the tests assert on what reached it and on the
// status that came back.
type scopedItems struct {
	ChecklistsClient

	planID, itemID uuid.UUID
	gotPlan        uuid.UUID
	writes         int
}

func (s *scopedItems) find(planID, id uuid.UUID) (*ChecklistResp, error) {
	s.gotPlan = planID
	if planID != s.planID || id != s.itemID {
		return nil, status.Error(codes.NotFound, "not found")
	}
	return &ChecklistResp{ID: id, PlanID: planID}, nil
}

func (s *scopedItems) write(planID, id uuid.UUID) (*ChecklistResp, error) {
	item, err := s.find(planID, id)
	if err == nil {
		s.writes++
	}
	return item, err
}

func (s *scopedItems) GetChecklist(ctx context.Context, planID, id, userID uuid.UUID) (*ChecklistResp, error) {
	return s.find(planID, id)
}

func (s *scopedItems) UpdateChecklist(ctx context.Context, planID, id, userID uuid.UUID, req UpdateChecklistReq) (*ChecklistResp, error) {
	return s.write(planID, id)
}

func (s *scopedItems) UpdateChecklistDates(ctx context.Context, planID, id, userID uuid.UUID, req UpdateDatesReq) (*ChecklistResp, error) {
	return s.write(planID, id)
}

func (s *scopedItems) ArchiveChecklist(ctx context.Context, planID, id, userID uuid.UUID, archived bool) (*ChecklistResp, error) {
	return s.write(planID, id)
}

func (s *scopedItems) DeleteChecklist(ctx context.Context, planID, id, userID uuid.UUID) error {
	_, err := s.write(planID, id)
	return err
}

func newScopedAPI(t *testing.T, items *scopedItems) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	RegisterChecklistOperations(api, items, allowAll(), func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), uuid.New())))
	}, nil)
	return api.(humatest.TestAPI)
}

// itemCall is one item-by-id route, sent at an item URL.
type itemCall struct {
	name       string
	wantStatus int
	do         func(api humatest.TestAPI, itemURL string) *httptest.ResponseRecorder
}

func itemCalls() []itemCall {
	return []itemCall{
		{"GET", http.StatusOK, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Get(u)
		}},
		{"PATCH", http.StatusOK, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Patch(u, map[string]any{"description": "hijacked"})
		}},
		{"PATCH /dates", http.StatusOK, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Patch(u+"/dates", map[string]any{"startDate": "2026-10-20"})
		}},
		{"PATCH /archive", http.StatusOK, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Patch(u+"/archive", map[string]any{"archived": true})
		}},
		{"DELETE", http.StatusNoContent, func(api humatest.TestAPI, u string) *httptest.ResponseRecorder {
			return api.Delete(u)
		}},
	}
}

func TestItemRoutes_ItemOfAnotherPlan_Answers404AndChangesNothing(t *testing.T) {
	planA, planB, itemOfB := uuid.New(), uuid.New(), uuid.New()

	for _, call := range itemCalls() {
		t.Run(call.name, func(t *testing.T) {
			items := &scopedItems{planID: planB, itemID: itemOfB}
			api := newScopedAPI(t, items)

			resp := call.do(api, "/api/plans/"+planA.String()+"/checklists/"+itemOfB.String())

			if resp.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404: %s", resp.Code, resp.Body.String())
			}
			if items.gotPlan != planA {
				t.Errorf("plan-service was asked under plan %s, want the path plan %s", items.gotPlan, planA)
			}
			if items.writes != 0 {
				t.Error("plan B's item was written through plan A's path")
			}
		})
	}
}

func TestItemRoutes_ItemOfThePathPlan_BehavesAsBefore(t *testing.T) {
	planB, itemOfB := uuid.New(), uuid.New()

	for _, call := range itemCalls() {
		t.Run(call.name, func(t *testing.T) {
			items := &scopedItems{planID: planB, itemID: itemOfB}
			api := newScopedAPI(t, items)

			resp := call.do(api, "/api/plans/"+planB.String()+"/checklists/"+itemOfB.String())

			if resp.Code != call.wantStatus {
				t.Fatalf("status = %d, want %d: %s", resp.Code, call.wantStatus, resp.Body.String())
			}
			if items.gotPlan != planB {
				t.Errorf("plan-service was asked under plan %s, want %s", items.gotPlan, planB)
			}
		})
	}
}
