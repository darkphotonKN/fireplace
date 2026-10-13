package plangw

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FS-043WP R38/R39: an item's intention is set only through the item PATCH and
// is returned on every item response, omitted when null.

// intentionItems plays plan-service for the PATCH and create routes: it keeps
// what reached it and answers with the item it was told to return.
type intentionItems struct {
	ChecklistsClient

	gotUpdate *UpdateChecklistReq
	created   bool
	reply     ChecklistResp
	err       error
}

func (f *intentionItems) UpdateChecklist(ctx context.Context, planID, id, userID uuid.UUID, req UpdateChecklistReq) (*ChecklistResp, error) {
	f.gotUpdate = &req
	if f.err != nil {
		return nil, f.err
	}
	out := f.reply
	out.ID, out.PlanID = id, planID
	return &out, nil
}

func (f *intentionItems) CreateChecklist(ctx context.Context, planID, userID uuid.UUID, req CreateChecklistReq) (*ChecklistResp, error) {
	f.created = true
	return &ChecklistResp{ID: uuid.New(), PlanID: planID}, nil
}

func newIntentionAPI(t *testing.T, items *intentionItems) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	RegisterChecklistOperations(api, items, allowAll(), func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), uuid.New())))
	}, nil)
	return api.(humatest.TestAPI)
}

func itemURL() string {
	return "/api/plans/" + uuid.NewString() + "/checklists/" + uuid.NewString()
}

func strp(s string) *string { return &s }

func TestUpdateChecklist_Intention_IsForwardedAndReturned(t *testing.T) {
	items := &intentionItems{reply: ChecklistResp{Intention: strp("ship it")}}
	api := newIntentionAPI(t, items)

	resp := api.Patch(itemURL(), map[string]any{"intention": "  ship it  "})

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.Code, resp.Body.String())
	}
	if items.gotUpdate.Intention == nil || *items.gotUpdate.Intention != "  ship it  " {
		t.Errorf("forwarded intention = %v, want the string as sent (plan-service trims)", items.gotUpdate.Intention)
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["intention"] != "ship it" {
		t.Errorf("response intention = %v, want \"ship it\"", body["intention"])
	}
}

func TestUpdateChecklist_Intention_OmittedIsLeftAloneAndNullIsOmitted(t *testing.T) {
	items := &intentionItems{}
	api := newIntentionAPI(t, items)

	resp := api.Patch(itemURL(), map[string]any{"description": "renamed"})

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.Code, resp.Body.String())
	}
	if items.gotUpdate.Intention != nil {
		t.Errorf("forwarded intention = %q, want nil (leave unchanged)", *items.gotUpdate.Intention)
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, present := body["intention"]; present {
		t.Errorf("a null intention was serialised: %s", resp.Body.String())
	}
}

func TestUpdateChecklist_Intention_EmptyStringIsForwardedToClear(t *testing.T) {
	items := &intentionItems{}
	api := newIntentionAPI(t, items)

	resp := api.Patch(itemURL(), map[string]any{"intention": ""})

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.Code, resp.Body.String())
	}
	if items.gotUpdate.Intention == nil || *items.gotUpdate.Intention != "" {
		t.Errorf("forwarded intention = %v, want \"\" so plan-service clears it", items.gotUpdate.Intention)
	}
}

// plan-service's over-280 refusal arrives as InvalidArgument with its own
// message; it must read as VALIDATION_FAILED, not a profile-name code.
func TestUpdateChecklist_Intention_TooLongIs400ValidationFailed(t *testing.T) {
	items := &intentionItems{err: status.Error(codes.InvalidArgument,
		"invalid input: intention must be at most 280 characters (got 281)")}
	api := newIntentionAPI(t, items)

	resp := api.Patch(itemURL(), map[string]any{"intention": "x"})

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", resp.Code, resp.Body.String())
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != "VALIDATION_FAILED" {
		t.Errorf("code = %q, want VALIDATION_FAILED", body.Code)
	}
}

func TestCreateChecklist_Intention_IsAnUnknownMember(t *testing.T) {
	items := &intentionItems{}
	api := newIntentionAPI(t, items)

	resp := api.Post("/api/plans/"+uuid.NewString()+"/checklists",
		map[string]any{"description": "new", "intention": "why"})

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422: %s", resp.Code, resp.Body.String())
	}
	if items.created {
		t.Error("a create carrying intention reached plan-service")
	}
}

// client.go maps both directions between the wire types and the proto.
func TestUpdateItemRequest_Intention(t *testing.T) {
	planID, id, userID := uuid.New(), uuid.New(), uuid.New()

	if got := updateItemRequest(planID, id, userID, UpdateChecklistReq{}); got.Intention != nil {
		t.Errorf("omitted intention reached the proto as %q", *got.Intention)
	}
	got := updateItemRequest(planID, id, userID, UpdateChecklistReq{Intention: strp("")})
	if got.Intention == nil || *got.Intention != "" {
		t.Errorf("empty intention = %v, want a set \"\"", got.Intention)
	}
	if got.PlanId != planID.String() || got.Id != id.String() || got.UserId != userID.String() {
		t.Errorf("ids not carried: %+v", got)
	}
}

func TestChecklistFromProto_Intention(t *testing.T) {
	if got := checklistFromProto(&pb.ChecklistItem{}); got.Intention != nil {
		t.Errorf("unset intention mapped to %q", *got.Intention)
	}
	got := checklistFromProto(&pb.ChecklistItem{Intention: strp("why")})
	if got.Intention == nil || *got.Intention != "why" {
		t.Errorf("intention = %v, want \"why\"", got.Intention)
	}
}
