package plangw

import (
	"context"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
)

// reorderRecorder captures what a reorder request was converted to, so the tests
// assert on what reaches plan-service rather than on the HTTP status alone.
type reorderRecorder struct {
	ChecklistsClient

	called       bool
	updateCalled bool
	gotReq       ReorderChecklistReq
	out          []*ChecklistResp
	err          error
}

func (r *reorderRecorder) ReorderChecklists(ctx context.Context, planID, userID uuid.UUID, req ReorderChecklistReq) ([]*ChecklistResp, error) {
	r.called, r.gotReq = true, req
	return r.out, r.err
}

// UpdateChecklist is recorded too: `order` is a static segment sitting beside
// `{checklist_id}` on the same prefix, and a router that resolved it to the
// wildcard would hand a reorder to the single-item update.
func (r *reorderRecorder) UpdateChecklist(ctx context.Context, id, userID uuid.UUID, req UpdateChecklistReq) (*ChecklistResp, error) {
	r.updateCalled = true
	return &ChecklistResp{ID: id}, nil
}

func newReorderAPI(t *testing.T, rec *reorderRecorder) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	RegisterChecklistOperations(api, rec, func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), uuid.New())))
	}, nil)
	return api.(humatest.TestAPI)
}

// The whole request is the order, so every part of it has to survive the
// boundary: the scope, the parent that addresses the set, and the ids IN ORDER.
func TestReorderChecklists_ForwardsTheWholeSiblingSet(t *testing.T) {
	planID, parentID := uuid.New(), uuid.New()
	first, second := uuid.New(), uuid.New()

	rec := &reorderRecorder{out: []*ChecklistResp{
		{ID: first, Sequence: "1", PlanID: planID},
		{ID: second, Sequence: "2", PlanID: planID},
	}}
	api := newReorderAPI(t, rec)

	resp := api.Patch("/api/plans/"+planID.String()+"/checklists/order", map[string]any{
		"scope":    "daily",
		"parentId": parentID.String(),
		"ids":      []string{first.String(), second.String()},
	})

	if !rec.called {
		t.Fatalf("request never reached the client (status %d): %s", resp.Code, resp.Body.String())
	}
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.Code, resp.Body.String())
	}
	if rec.gotReq.Scope != "daily" {
		t.Errorf("scope = %q, want \"daily\"", rec.gotReq.Scope)
	}
	if rec.gotReq.ParentID == nil || *rec.gotReq.ParentID != parentID {
		t.Errorf("parentId = %v, want %s", rec.gotReq.ParentID, parentID)
	}
	if len(rec.gotReq.IDs) != 2 || rec.gotReq.IDs[0] != first || rec.gotReq.IDs[1] != second {
		t.Errorf("ids = %v, want [%s %s]", rec.gotReq.IDs, first, second)
	}
	if rec.updateCalled {
		t.Error("PATCH .../checklists/order was routed to the single-item update")
	}
}

// R2.1: a top-level set is addressed with parentId null. Null is a VALUE here,
// not an omission, so it has to arrive as one.
func TestReorderChecklists_NullParentAddressesTheTopLevelSet(t *testing.T) {
	planID, itemID := uuid.New(), uuid.New()
	rec := &reorderRecorder{}
	api := newReorderAPI(t, rec)

	resp := api.Patch("/api/plans/"+planID.String()+"/checklists/order", map[string]any{
		"scope":    "longterm",
		"parentId": nil,
		"ids":      []string{itemID.String()},
	})

	if !rec.called {
		t.Fatalf("request never reached the client (status %d): %s", resp.Code, resp.Body.String())
	}
	if rec.gotReq.ParentID != nil {
		t.Errorf("parentId = %s, want nil for the top-level set", *rec.gotReq.ParentID)
	}
}

// The body is the whole order, so a body that is not fully specified is refused
// at the boundary rather than guessed at (ADR-0005: shape is the gateway's job).
func TestReorderChecklists_IncompleteOrUnknownBodyIsRejectedAtTheBoundary(t *testing.T) {
	planID, itemID := uuid.New(), uuid.New()

	cases := []struct {
		name string
		body map[string]any
	}{
		{"no parentId", map[string]any{
			"scope": "daily", "ids": []string{itemID.String()},
		}},
		{"no scope", map[string]any{
			"parentId": nil, "ids": []string{itemID.String()},
		}},
		{"no ids", map[string]any{
			"scope": "daily", "parentId": nil,
		}},
		{"empty ids", map[string]any{
			"scope": "daily", "parentId": nil, "ids": []string{},
		}},
		{"repeated id", map[string]any{
			"scope": "daily", "parentId": nil, "ids": []string{itemID.String(), itemID.String()},
		}},
		{"scope outside the enum", map[string]any{
			"scope": "weekly", "parentId": nil, "ids": []string{itemID.String()},
		}},
		{"unknown member", map[string]any{
			"scope": "daily", "parentId": nil, "ids": []string{itemID.String()}, "sequence": "3",
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &reorderRecorder{}
			api := newReorderAPI(t, rec)

			resp := api.Patch("/api/plans/"+planID.String()+"/checklists/order", tc.body)

			if resp.Code != http.StatusUnprocessableEntity {
				t.Errorf("status = %d, want 422: %s", resp.Code, resp.Body.String())
			}
			if rec.called {
				t.Error("an ill-shaped body reached plan-service")
			}
		})
	}
}
