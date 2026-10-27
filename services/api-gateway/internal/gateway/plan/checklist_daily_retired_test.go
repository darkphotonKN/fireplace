package plangw

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
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
)

// FS-02JBZ R51: plan-service refuses a daily create or update with
// DAILY_SCOPE_RETIRED as an ErrorInfo detail on InvalidArgument. The gateway
// adds no mapping of its own: the detail alone must surface as
// 400 · DAILY_SCOPE_RETIRED, and the request itself (scope enum unchanged)
// must still be well-formed, so it reaches plan-service instead of failing 422.

// dailyRetiredItems plays plan-service answering every write with the refusal.
type dailyRetiredItems struct {
	ChecklistsClient

	gotCreate *CreateChecklistReq
	gotUpdate *UpdateChecklistReq
}

func retired() error {
	return commongrpc.WithCode(codes.InvalidArgument, errcode.DailyScopeRetired,
		"the daily scope is retired: items are long-term")
}

func (f *dailyRetiredItems) CreateChecklist(ctx context.Context, planID, userID uuid.UUID, req CreateChecklistReq) (*ChecklistResp, error) {
	f.gotCreate = &req
	return nil, retired()
}

func (f *dailyRetiredItems) UpdateChecklist(ctx context.Context, planID, id, userID uuid.UUID, req UpdateChecklistReq) (*ChecklistResp, error) {
	f.gotUpdate = &req
	return nil, retired()
}

func newDailyAPI(t *testing.T, items ChecklistsClient) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	RegisterChecklistOperations(api, items, allowAll(), func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), uuid.New())))
	}, nil)
	return api.(humatest.TestAPI)
}

func TestChecklistWrites_DailyScope_Is400DailyScopeRetired(t *testing.T) {
	tests := []struct {
		name    string
		send    func(f *dailyRetiredItems) (int, []byte)
		reached func(f *dailyRetiredItems) bool
	}{
		{
			name: "create as daily",
			send: func(f *dailyRetiredItems) (int, []byte) {
				resp := newDailyAPI(t, f).Post("/api/plans/"+uuid.NewString()+"/checklists",
					map[string]any{"description": "habit", "scope": "daily"})
				return resp.Code, resp.Body.Bytes()
			},
			reached: func(f *dailyRetiredItems) bool {
				return f.gotCreate != nil && f.gotCreate.Scope != nil && *f.gotCreate.Scope == "daily"
			},
		},
		{
			name: "update to daily",
			send: func(f *dailyRetiredItems) (int, []byte) {
				resp := newDailyAPI(t, f).Patch(itemURL(), map[string]any{"scope": "daily"})
				return resp.Code, resp.Body.Bytes()
			},
			reached: func(f *dailyRetiredItems) bool {
				return f.gotUpdate != nil && f.gotUpdate.Scope != nil && *f.gotUpdate.Scope == "daily"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &dailyRetiredItems{}

			status, body := tt.send(f)

			if !tt.reached(f) {
				t.Fatal("the daily request never reached plan-service: the contract refused it, " +
					"but the scope enum must stay well-formed (ADR-0005)")
			}
			if status != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", status, body)
			}
			var problem struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(body, &problem); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if problem.Code != string(errcode.DailyScopeRetired) {
				t.Errorf("code = %q, want %q", problem.Code, errcode.DailyScopeRetired)
			}
		})
	}
}
