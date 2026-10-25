package insights

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/darkphotonKN/fireplace/common/errcode"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeDayPlans struct {
	plan    DayPlanResponse
	err     error
	calls   int
	gotPlan uuid.UUID
	gotUser uuid.UUID
	gotDate string
	// drew records that PlanMyDay, not GetDayPlan, was called.
	drew bool

	entry    DayPlanEntryResponse
	gotEntry uuid.UUID
	gotItem  *uuid.UUID
	unticked bool

	gotEnabled *bool
}

func (f *fakeDayPlans) SetItemRevisions(ctx context.Context, planID, userID, itemID uuid.UUID, enabled bool) error {
	f.calls++
	f.gotPlan, f.gotUser, f.gotItem, f.gotEnabled = planID, userID, &itemID, &enabled
	return f.err
}

func (f *fakeDayPlans) TickDayPlanEntry(ctx context.Context, entryID, userID uuid.UUID, itemID *uuid.UUID) (DayPlanEntryResponse, error) {
	f.calls++
	f.gotEntry, f.gotUser, f.gotItem = entryID, userID, itemID
	return f.entry, f.err
}

func (f *fakeDayPlans) UntickDayPlanEntry(ctx context.Context, entryID, userID uuid.UUID) error {
	f.calls++
	f.unticked = true
	f.gotEntry, f.gotUser = entryID, userID
	return f.err
}

func (f *fakeDayPlans) GetDayPlan(ctx context.Context, planID, userID uuid.UUID, date string) (DayPlanResponse, error) {
	f.calls++
	f.gotPlan, f.gotUser, f.gotDate = planID, userID, date
	return f.plan, f.err
}

func (f *fakeDayPlans) PlanMyDay(ctx context.Context, planID, userID uuid.UUID, date string) (DayPlanResponse, error) {
	f.calls++
	f.drew = true
	f.gotPlan, f.gotUser, f.gotDate = planID, userID, date
	return f.plan, f.err
}

func newDayPlanAPI(t *testing.T, d DayPlanService, userID uuid.UUID) humatest.TestAPI {
	t.Helper()
	cfg := huma.DefaultConfig("test", "1")
	cfg.CreateHooks = nil // as in production: no injected $schema member
	_, api := humatest.New(t, cfg)
	RegisterDayPlanOperations(api, d, allowAccess{}, func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), userID)))
	}, nil)
	return api.(humatest.TestAPI)
}

const dayPlanPath = "/api/insights/day-plan"

func plannedDayPlan(planID uuid.UUID) (DayPlanResponse, uuid.UUID, uuid.UUID) {
	item, parent := uuid.New(), uuid.New()
	at := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)
	minutes := 45
	return DayPlanResponse{
		PlanID: planID, Date: "2026-10-25", Status: "planned", Source: "ai", GeneratedAt: &at, RefreshesRemaining: 3,
		Groups: []DayPlanGroupResponse{{Label: "Graphs", Entries: []DayPlanEntryResponse{
			{ID: uuid.New(), Kind: "existing_item", ItemID: &item, ParentItemID: &parent, Text: "Solve 3 BFS problems", Reason: "due", Minutes: &minutes},
			{ID: uuid.New(), Kind: "new_step", ParentItemID: &parent, Text: "Do the knapsack problem", Reason: "start"},
		}}},
	}, item, parent
}

func TestGetDayPlan_ForwardsIdentityAndPublishesThePlan(t *testing.T) {
	caller, planID := uuid.New(), uuid.New()
	plan, item, parent := plannedDayPlan(planID)
	d := &fakeDayPlans{plan: plan}

	resp := newDayPlanAPI(t, d, caller).Get(dayPlanPath + "?planId=" + planID.String() + "&date=2026-10-25")

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", resp.Code, resp.Body.String())
	}
	if d.drew || d.gotUser != caller || d.gotPlan != planID || d.gotDate != "2026-10-25" {
		t.Fatalf("service drew %v with user %s plan %s date %q", d.drew, d.gotUser, d.gotPlan, d.gotDate)
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"planId", "date", "status", "source", "generatedAt", "refreshesRemaining", "groups"} {
		if _, ok := body[k]; !ok {
			t.Fatalf("body lacks %q: %s", k, resp.Body.String())
		}
	}
	if len(body) != 7 {
		t.Fatalf("body has %d members, want exactly 7: %s", len(body), resp.Body.String())
	}
	group := body["groups"].([]any)[0].(map[string]any)
	existing, fresh := group["entries"].([]any)[0].(map[string]any), group["entries"].([]any)[1].(map[string]any)
	if group["label"] != "Graphs" || existing["itemId"] != item.String() || existing["parentItemId"] != parent.String() ||
		existing["minutes"] != float64(45) || existing["ticked"] != false {
		t.Fatalf("existing entry = %v", existing)
	}
	for _, absent := range []string{"itemId", "minutes", "revision", "revisionsEnabled"} {
		if _, has := fresh[absent]; has {
			t.Fatalf("new step carries %q: %v", absent, fresh)
		}
	}
	if fresh["ticked"] != false {
		t.Fatalf("new step = %v, want ticked false", fresh)
	}
}

func TestGetDayPlan_NotPlannedHasNoSourceNoDrawTimeAndEmptyGroups(t *testing.T) {
	d := &fakeDayPlans{plan: DayPlanResponse{PlanID: uuid.New(), Date: "2026-10-25", Status: "not_planned", RefreshesRemaining: 3}}

	resp := newDayPlanAPI(t, d, uuid.New()).Get(dayPlanPath + "?planId=" + uuid.NewString() + "&date=2026-10-25")

	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"groups":[]`) {
		t.Fatalf("status %d body %s, want groups: []", resp.Code, resp.Body.String())
	}
	var body map[string]any
	json.Unmarshal(resp.Body.Bytes(), &body)
	if _, has := body["source"]; has {
		t.Fatalf("not_planned carries source: %s", resp.Body.String())
	}
	if _, has := body["generatedAt"]; has {
		t.Fatalf("not_planned carries generatedAt: %s", resp.Body.String())
	}
}

func TestGetDayPlan_ShapeFailuresAre422AtTheBoundary(t *testing.T) {
	plan := uuid.NewString()
	for name, query := range map[string]string{
		"missing planId":   "date=2026-10-25",
		"missing date":     "planId=" + plan,
		"malformed planId": "planId=nope&date=2026-10-25",
		"malformed date":   "planId=" + plan + "&date=25-10-2026",
		"snake_case param": "plan_id=" + plan + "&date=2026-10-25",
	} {
		t.Run(name, func(t *testing.T) {
			d := &fakeDayPlans{}
			resp := newDayPlanAPI(t, d, uuid.New()).Get(dayPlanPath + "?" + query)

			if resp.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422: %s", resp.Code, resp.Body.String())
			}
			if d.calls != 0 {
				t.Fatal("a malformed request reached insights-service")
			}
		})
	}
}

func TestPlanMyDay_ForwardsIdentityAndPublishesThePlan(t *testing.T) {
	caller, planID := uuid.New(), uuid.New()
	plan, _, _ := plannedDayPlan(planID)
	d := &fakeDayPlans{plan: plan}

	resp := newDayPlanAPI(t, d, caller).Post(dayPlanPath, map[string]any{"planId": planID.String(), "date": "2026-10-25"})

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", resp.Code, resp.Body.String())
	}
	if !d.drew || d.gotUser != caller || d.gotPlan != planID || d.gotDate != "2026-10-25" {
		t.Fatalf("service drew %v with user %s plan %s date %q", d.drew, d.gotUser, d.gotPlan, d.gotDate)
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "planned" || body["source"] != "ai" || len(body["groups"].([]any)) != 1 {
		t.Fatalf("body = %s", resp.Body.String())
	}
}

func TestPlanMyDay_ShapeFailuresAre422AtTheBoundary(t *testing.T) {
	plan := uuid.NewString()
	for name, body := range map[string]map[string]any{
		"missing planId":   {"date": "2026-10-25"},
		"missing date":     {"planId": plan},
		"malformed planId": {"planId": "nope", "date": "2026-10-25"},
		"malformed date":   {"planId": plan, "date": "25-10-2026"},
		"snake_case field": {"plan_id": plan, "date": "2026-10-25"},
		"identity in body": {"planId": plan, "date": "2026-10-25", "userId": uuid.NewString()},
	} {
		t.Run(name, func(t *testing.T) {
			d := &fakeDayPlans{}
			resp := newDayPlanAPI(t, d, uuid.New()).Post(dayPlanPath, body)

			if resp.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422: %s", resp.Code, resp.Body.String())
			}
			if d.calls != 0 {
				t.Fatal("a malformed request reached insights-service")
			}
		})
	}
}

func TestDayPlan_DomainFailuresCarryTheirCode(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   errcode.Code
	}{
		{"refresh limit reached", commongrpc.WithCode(codes.ResourceExhausted, errcode.BriefRefreshLimitReached, "3 of 3 spent"), http.StatusTooManyRequests, errcode.BriefRefreshLimitReached},
		{"date out of range", commongrpc.WithCode(codes.InvalidArgument, errcode.BriefDateOutOfRange, "too far"), http.StatusBadRequest, errcode.BriefDateOutOfRange},
		{"generation failed", status.Error(codes.Unavailable, "model down"), http.StatusServiceUnavailable, errcode.ServiceUnavailable},
		{"plan not found", status.Error(codes.NotFound, "no plan"), http.StatusNotFound, errcode.NotFound},
	}
	ops := map[string]func(api humatest.TestAPI) *httptest.ResponseRecorder{
		"getDayPlan": func(api humatest.TestAPI) *httptest.ResponseRecorder {
			return api.Get(dayPlanPath + "?planId=" + uuid.NewString() + "&date=2026-10-25")
		},
		"planMyDay": func(api humatest.TestAPI) *httptest.ResponseRecorder {
			return api.Post(dayPlanPath, map[string]any{"planId": uuid.NewString(), "date": "2026-10-25"})
		},
	}
	for _, tt := range tests {
		for op, call := range ops {
			t.Run(op+"/"+tt.name, func(t *testing.T) {
				resp := call(newDayPlanAPI(t, &fakeDayPlans{err: tt.err}, uuid.New()))

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
}

func tickPath(entryID string) string { return dayPlanPath + "/entries/" + entryID + "/tick" }

func TestTickDayPlanEntry_ForwardsIdentityAndItemAndPublishesTheEntry(t *testing.T) {
	caller, entryID, item, parent := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	d := &fakeDayPlans{entry: DayPlanEntryResponse{
		ID: entryID, Kind: "existing_item", ItemID: &item, ParentItemID: &parent, Text: "Solve the climbing stairs problem", Reason: "start", Ticked: true,
	}}

	resp := newDayPlanAPI(t, d, caller).Put(tickPath(entryID.String()), map[string]any{"itemId": item.String()})

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", resp.Code, resp.Body.String())
	}
	if d.gotEntry != entryID || d.gotUser != caller || d.gotItem == nil || *d.gotItem != item {
		t.Fatalf("service got entry %s user %s item %v", d.gotEntry, d.gotUser, d.gotItem)
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["id"] != entryID.String() || body["kind"] != "existing_item" || body["itemId"] != item.String() || body["ticked"] != true {
		t.Fatalf("body = %s, want the ticked entry", resp.Body.String())
	}
}

func TestTickDayPlanEntry_ItemIDIsOptional(t *testing.T) {
	for name, send := range map[string]func(api humatest.TestAPI, path string) *httptest.ResponseRecorder{
		"empty object": func(api humatest.TestAPI, path string) *httptest.ResponseRecorder {
			return api.Put(path, map[string]any{})
		},
		"no body": func(api humatest.TestAPI, path string) *httptest.ResponseRecorder { return api.Put(path) },
	} {
		t.Run(name, func(t *testing.T) {
			entryID := uuid.New()
			d := &fakeDayPlans{entry: DayPlanEntryResponse{ID: entryID, Kind: "existing_item", Ticked: true}}

			resp := send(newDayPlanAPI(t, d, uuid.New()), tickPath(entryID.String()))

			if resp.Code != http.StatusOK || d.gotEntry != entryID || d.gotItem != nil {
				t.Fatalf("status %d entry %s item %v: %s", resp.Code, d.gotEntry, d.gotItem, resp.Body.String())
			}
		})
	}
}

func TestUntickDayPlanEntry_ForwardsIdentityAndAnswers204(t *testing.T) {
	caller, entryID := uuid.New(), uuid.New()
	d := &fakeDayPlans{}

	resp := newDayPlanAPI(t, d, caller).Delete(tickPath(entryID.String()))

	if resp.Code != http.StatusNoContent || resp.Body.Len() != 0 {
		t.Fatalf("status = %d body %q, want 204 with no body", resp.Code, resp.Body.String())
	}
	if !d.unticked || d.gotEntry != entryID || d.gotUser != caller {
		t.Fatalf("service unticked %v entry %s user %s", d.unticked, d.gotEntry, d.gotUser)
	}
}

func TestDayTicks_ShapeFailuresAre422AtTheBoundary(t *testing.T) {
	entry := uuid.NewString()
	for name, call := range map[string]func(api humatest.TestAPI) *httptest.ResponseRecorder{
		"tick malformed entry id": func(api humatest.TestAPI) *httptest.ResponseRecorder {
			return api.Put(tickPath("nope"), map[string]any{})
		},
		"tick malformed itemId": func(api humatest.TestAPI) *httptest.ResponseRecorder {
			return api.Put(tickPath(entry), map[string]any{"itemId": "nope"})
		},
		"tick snake_case field": func(api humatest.TestAPI) *httptest.ResponseRecorder {
			return api.Put(tickPath(entry), map[string]any{"item_id": uuid.NewString()})
		},
		"tick identity in body": func(api humatest.TestAPI) *httptest.ResponseRecorder {
			return api.Put(tickPath(entry), map[string]any{"userId": uuid.NewString()})
		},
		"untick malformed entry id": func(api humatest.TestAPI) *httptest.ResponseRecorder { return api.Delete(tickPath("nope")) },
	} {
		t.Run(name, func(t *testing.T) {
			d := &fakeDayPlans{}
			resp := call(newDayPlanAPI(t, d, uuid.New()))

			if resp.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422: %s", resp.Code, resp.Body.String())
			}
			if d.calls != 0 {
				t.Fatal("a malformed request reached insights-service")
			}
		})
	}
}

func TestDayTicks_DomainFailuresCarryTheirCode(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   errcode.Code
		tickOnly   bool
	}{
		{"not tickable", commongrpc.WithCode(codes.FailedPrecondition, errcode.DayPlanEntryNotTickable, "archived"), http.StatusBadRequest, errcode.DayPlanEntryNotTickable, true},
		{"date out of range", commongrpc.WithCode(codes.InvalidArgument, errcode.BriefDateOutOfRange, "too far"), http.StatusBadRequest, errcode.BriefDateOutOfRange, false},
		{"entry not found", status.Error(codes.NotFound, "no entry"), http.StatusNotFound, errcode.NotFound, false},
		{"insights down", status.Error(codes.Unavailable, "down"), http.StatusServiceUnavailable, errcode.ServiceUnavailable, false},
	}
	ops := map[string]func(api humatest.TestAPI) *httptest.ResponseRecorder{
		"tickDayPlanEntry": func(api humatest.TestAPI) *httptest.ResponseRecorder {
			return api.Put(tickPath(uuid.NewString()), map[string]any{})
		},
		"untickDayPlanEntry": func(api humatest.TestAPI) *httptest.ResponseRecorder {
			return api.Delete(tickPath(uuid.NewString()))
		},
	}
	for _, tt := range tests {
		for op, call := range ops {
			if tt.tickOnly && op != "tickDayPlanEntry" {
				continue
			}
			t.Run(op+"/"+tt.name, func(t *testing.T) {
				resp := call(newDayPlanAPI(t, &fakeDayPlans{err: tt.err}, uuid.New()))

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
}

func revisionsPath(itemID string) string { return dayPlanPath + "/items/" + itemID + "/revisions" }

func TestSetItemRevisions_ForwardsIdentityPlanItemAndSwitchAndAnswers204(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		caller, planID, item := uuid.New(), uuid.New(), uuid.New()
		d := &fakeDayPlans{}

		resp := newDayPlanAPI(t, d, caller).Put(revisionsPath(item.String()), map[string]any{"planId": planID.String(), "enabled": enabled})

		if resp.Code != http.StatusNoContent || resp.Body.Len() != 0 {
			t.Fatalf("enabled %v: status = %d body %q, want 204 with no body", enabled, resp.Code, resp.Body.String())
		}
		if d.gotUser != caller || d.gotPlan != planID || d.gotItem == nil || *d.gotItem != item || d.gotEnabled == nil || *d.gotEnabled != enabled {
			t.Fatalf("service got user %s plan %s item %v enabled %v", d.gotUser, d.gotPlan, d.gotItem, d.gotEnabled)
		}
	}
}

func TestSetItemRevisions_ShapeFailuresAre422AtTheBoundary(t *testing.T) {
	plan, item := uuid.NewString(), uuid.NewString()
	for name, tc := range map[string]struct {
		item string
		body map[string]any
	}{
		"malformed item id": {"nope", map[string]any{"planId": plan, "enabled": true}},
		"missing planId":    {item, map[string]any{"enabled": true}},
		"missing enabled":   {item, map[string]any{"planId": plan}},
		"malformed planId":  {item, map[string]any{"planId": "nope", "enabled": true}},
		"enabled not bool":  {item, map[string]any{"planId": plan, "enabled": "yes"}},
		"identity in body":  {item, map[string]any{"planId": plan, "enabled": true, "userId": uuid.NewString()}},
	} {
		t.Run(name, func(t *testing.T) {
			d := &fakeDayPlans{}
			resp := newDayPlanAPI(t, d, uuid.New()).Put(revisionsPath(tc.item), tc.body)

			if resp.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422: %s", resp.Code, resp.Body.String())
			}
			if d.calls != 0 {
				t.Fatal("a malformed request reached insights-service")
			}
		})
	}
}

func TestSetItemRevisions_ItemNotInThePlanIs404NotFound(t *testing.T) {
	d := &fakeDayPlans{err: status.Error(codes.NotFound, "plan item not found")}

	resp := newDayPlanAPI(t, d, uuid.New()).Put(revisionsPath(uuid.NewString()), map[string]any{"planId": uuid.NewString(), "enabled": false})

	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", resp.Code, resp.Body.String())
	}
	var p struct {
		Code errcode.Code `json:"code"`
	}
	json.Unmarshal(resp.Body.Bytes(), &p)
	if p.Code != errcode.NotFound {
		t.Fatalf("code = %q, want %q", p.Code, errcode.NotFound)
	}
}
