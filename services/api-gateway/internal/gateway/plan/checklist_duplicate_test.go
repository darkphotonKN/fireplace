package plangw

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// duplicateRecorder plays plan-service for the duplicate operation: it records
// the plan and item the path addressed and answers with whatever it is given.
type duplicateRecorder struct {
	ChecklistsClient

	called         bool
	gotPlan, gotID uuid.UUID
	out            []*ChecklistResp
	err            error
}

func (d *duplicateRecorder) DuplicateChecklist(ctx context.Context, planID, id, userID uuid.UUID) ([]*ChecklistResp, error) {
	d.called, d.gotPlan, d.gotID = true, planID, id
	return d.out, d.err
}

func newDuplicateAPI(t *testing.T, rec *duplicateRecorder) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	RegisterChecklistOperations(api, rec, allowAll(), func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), uuid.New())))
	}, nil)
	return api.(humatest.TestAPI)
}

func duplicateURL(planID, itemID string) string {
	return "/api/plans/" + planID + "/checklists/" + itemID + "/duplicate"
}

// R11, R7: no body, 201, and the created items in plan-service's order — the
// copy first, then its children — each item complete with its intention.
func TestDuplicateChecklist_Returns201WithTheCreatedItemsInOrder(t *testing.T) {
	planID, source := uuid.New(), uuid.New()
	copyID, childID := uuid.New(), uuid.New()
	intention := "ship the beta"
	rec := &duplicateRecorder{out: []*ChecklistResp{
		{ID: copyID, PlanID: planID, Type: "task", Scope: "longterm", Intention: &intention},
		{ID: childID, PlanID: planID, Type: "task", Scope: "longterm", ParentID: &copyID},
	}}
	api := newDuplicateAPI(t, rec)

	resp := api.Post(duplicateURL(planID.String(), source.String()))

	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", resp.Code, resp.Body.String())
	}
	if rec.gotPlan != planID || rec.gotID != source {
		t.Errorf("plan-service asked for (%s, %s), want the path's (%s, %s)", rec.gotPlan, rec.gotID, planID, source)
	}
	var body []ChecklistResp
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v — %s", err, resp.Body.String())
	}
	if len(body) != 2 || body[0].ID != copyID || body[1].ID != childID {
		t.Fatalf("body = %+v, want [copy child]", body)
	}
	if body[0].Intention == nil || *body[0].Intention != intention {
		t.Errorf("copy intention = %v, want %q", body[0].Intention, intention)
	}
	if body[1].ParentID == nil || *body[1].ParentID != copyID {
		t.Errorf("child parentId = %v, want %s", body[1].ParentID, copyID)
	}
}

// R13: plan-service's refusals pass through the shared mapping, and a malformed
// path uuid never leaves the boundary.
func TestDuplicateChecklist_ErrorsMapToProblems(t *testing.T) {
	cases := []struct {
		name     string
		itemID   string
		err      error
		want     int
		wantCode string
	}{
		{"archived source", uuid.NewString(),
			status.Error(codes.InvalidArgument, "invalid input: an archived item cannot be duplicated"),
			http.StatusBadRequest, "VALIDATION_FAILED"},
		{"missing or other-plan source", uuid.NewString(),
			status.Error(codes.NotFound, "not found"), http.StatusNotFound, "NOT_FOUND"},
		{"plan-service unreachable", uuid.NewString(),
			status.Error(codes.Unavailable, "down"), http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE"},
		// The 422 body's `code` is installed by the real mount (config/api.go),
		// not by humatest; config/api_test.go asserts it.
		{"malformed item uuid", "not-a-uuid", nil, http.StatusUnprocessableEntity, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &duplicateRecorder{err: tc.err}
			api := newDuplicateAPI(t, rec)

			resp := api.Post(duplicateURL(uuid.NewString(), tc.itemID))

			if resp.Code != tc.want {
				t.Fatalf("status = %d, want %d: %s", resp.Code, tc.want, resp.Body.String())
			}
			if tc.wantCode != "" {
				var body struct {
					Code string `json:"code"`
				}
				if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
					t.Fatalf("decode: %v — %s", err, resp.Body.String())
				}
				if body.Code != tc.wantCode {
					t.Errorf("code = %q, want %q", body.Code, tc.wantCode)
				}
			}
			if tc.err == nil && rec.called {
				t.Error("a malformed path reached plan-service")
			}
		})
	}
}
