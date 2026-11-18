package insights

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/darkphotonKN/fireplace/common/errcode"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	plangw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/plan"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeNoteSummaries struct {
	summary string
	err     error

	calls                     int
	gotPlan, gotUser, gotItem uuid.UUID
}

func (f *fakeNoteSummaries) SummarizeNote(_ context.Context, planID, userID, itemID uuid.UUID) (string, error) {
	f.calls++
	f.gotPlan, f.gotUser, f.gotItem = planID, userID, itemID
	return f.summary, f.err
}

const noteSummaryPath = "/api/insights/note-summary"

func newNoteSummaryAPI(t *testing.T, n NoteSummaryService, access PlanAccess, userID uuid.UUID) humatest.TestAPI {
	t.Helper()
	cfg := huma.DefaultConfig("test", "1")
	cfg.CreateHooks = nil
	_, api := humatest.New(t, cfg)
	RegisterNoteSummaryOperations(api, n, access, func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), userID)))
	}, nil)
	return api.(humatest.TestAPI)
}

// FS-PGG2N R38: the caller's identity and both ids reach insights-service, and
// the answer is { summary }.
func TestSummarizeNote_ForwardsIdentityAndAnswersTheSummary(t *testing.T) {
	caller, planID, itemID := uuid.New(), uuid.New(), uuid.New()
	n := &fakeNoteSummaries{summary: "A short summary."}

	resp := newNoteSummaryAPI(t, n, allowAccess{}, caller).Post(noteSummaryPath,
		map[string]any{"planId": planID.String(), "itemId": itemID.String()})

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", resp.Code, resp.Body.String())
	}
	if n.gotUser != caller || n.gotPlan != planID || n.gotItem != itemID {
		t.Fatalf("service got user %s plan %s item %s", n.gotUser, n.gotPlan, n.gotItem)
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 1 || body["summary"] != "A short summary." {
		t.Fatalf("body = %s, want exactly { summary }", resp.Body.String())
	}
}

// R38, FS-12ZMF: the access seam runs first; a stranger is 404 and never
// reaches insights-service.
func TestSummarizeNote_Access(t *testing.T) {
	cases := []struct {
		name       string
		access     PlanAccess
		wantStatus int
	}{
		{"owner", plangw.NewAccess(relations{rel: plangw.RelationOwner}), http.StatusOK},
		{"member", plangw.NewAccess(relations{rel: plangw.RelationMember}), http.StatusOK},
		{"stranger", plangw.NewAccess(relations{rel: plangw.RelationNone}), http.StatusNotFound},
		{"no checker", nil, http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := &fakeNoteSummaries{summary: "x"}
			resp := newNoteSummaryAPI(t, n, tc.access, uuid.New()).Post(noteSummaryPath,
				map[string]any{"planId": uuid.NewString(), "itemId": uuid.NewString()})

			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d: %s", resp.Code, tc.wantStatus, resp.Body.String())
			}
			if reached := n.calls > 0; reached != (tc.wantStatus == http.StatusOK) {
				t.Fatalf("insights-service reached = %v", reached)
			}
		})
	}
}

// ADR-0005: malformed or missing ids and unknown members are 422 at the
// boundary, never forwarded.
func TestSummarizeNote_ShapeFailuresAre422AtTheBoundary(t *testing.T) {
	id := uuid.NewString()
	for name, body := range map[string]map[string]any{
		"missing planId":   {"itemId": id},
		"missing itemId":   {"planId": id},
		"malformed planId": {"planId": "nope", "itemId": id},
		"malformed itemId": {"planId": id, "itemId": "nope"},
		"identity in body": {"planId": id, "itemId": id, "userId": id},
		"snake_case field": {"plan_id": id, "itemId": id},
	} {
		t.Run(name, func(t *testing.T) {
			n := &fakeNoteSummaries{}
			resp := newNoteSummaryAPI(t, n, allowAccess{}, uuid.New()).Post(noteSummaryPath, body)

			if resp.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422: %s", resp.Code, resp.Body.String())
			}
			if n.calls != 0 {
				t.Fatal("a malformed request reached insights-service")
			}
		})
	}
}

// FS-PGG2N §API surface: each refusal from insights-service leaves with its
// status and code. The two 503s differ only by code.
func TestSummarizeNote_DomainFailuresCarryTheirCode(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   errcode.Code
	}{
		{"item missing", status.Error(codes.NotFound, "no item"), http.StatusNotFound, errcode.NotFound},
		{"a task", commongrpc.WithCode(codes.FailedPrecondition, errcode.ItemNotANote, "task"), http.StatusBadRequest, errcode.ItemNotANote},
		{"budget spent", commongrpc.WithCode(codes.ResourceExhausted, errcode.LLMBudgetExhausted, "spent"), http.StatusTooManyRequests, errcode.LLMBudgetExhausted},
		{"model failed", commongrpc.WithCode(codes.Unavailable, errcode.NoteSummaryFailed, "boom"), http.StatusServiceUnavailable, errcode.NoteSummaryFailed},
		{"insights unreachable", status.Error(codes.Unavailable, "connection refused"), http.StatusServiceUnavailable, errcode.ServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := newNoteSummaryAPI(t, &fakeNoteSummaries{err: tt.err}, allowAccess{}, uuid.New()).Post(noteSummaryPath,
				map[string]any{"planId": uuid.NewString(), "itemId": uuid.NewString()})

			if resp.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
			var p struct {
				Code errcode.Code `json:"code"`
			}
			json.Unmarshal(resp.Body.Bytes(), &p)
			if p.Code != tt.wantCode {
				t.Fatalf("code = %q, want %q", p.Code, tt.wantCode)
			}
		})
	}
}
