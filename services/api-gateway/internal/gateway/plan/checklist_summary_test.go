package plangw

import (
	"encoding/json"
	"net/http"
	"testing"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FS-PGG2N R2, R3, R5: a note's summary is set only through the item PATCH and
// is returned on every item response, omitted when null. intentionItems plays
// plan-service for these routes too.

func TestUpdateChecklist_Summary_IsForwardedAndReturned(t *testing.T) {
	items := &intentionItems{reply: ChecklistResp{Type: "note", Summary: strp("gist")}}
	api := newIntentionAPI(t, items)

	resp := api.Patch(itemURL(), map[string]any{"summary": "  gist  "})

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.Code, resp.Body.String())
	}
	if items.gotUpdate.Summary == nil || *items.gotUpdate.Summary != "  gist  " {
		t.Errorf("forwarded summary = %v, want the string as sent (plan-service trims)", items.gotUpdate.Summary)
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["summary"] != "gist" {
		t.Errorf("response summary = %v, want \"gist\"", body["summary"])
	}
}

func TestUpdateChecklist_Summary_OmittedIsLeftAloneAndNullIsOmitted(t *testing.T) {
	items := &intentionItems{}
	api := newIntentionAPI(t, items)

	resp := api.Patch(itemURL(), map[string]any{"description": "a longer body"})

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.Code, resp.Body.String())
	}
	if items.gotUpdate.Summary != nil {
		t.Errorf("forwarded summary = %q, want nil (leave unchanged)", *items.gotUpdate.Summary)
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, present := body["summary"]; present {
		t.Errorf("a null summary was serialised: %s", resp.Body.String())
	}
}

func TestUpdateChecklist_Summary_EmptyStringIsForwardedToClear(t *testing.T) {
	items := &intentionItems{}
	api := newIntentionAPI(t, items)

	resp := api.Patch(itemURL(), map[string]any{"summary": ""})

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.Code, resp.Body.String())
	}
	if items.gotUpdate.Summary == nil || *items.gotUpdate.Summary != "" {
		t.Errorf("forwarded summary = %v, want \"\" so plan-service clears it", items.gotUpdate.Summary)
	}
}

// plan-service's refusals (too long, or on a task) arrive as InvalidArgument
// and must read as VALIDATION_FAILED.
func TestUpdateChecklist_Summary_RefusalIs400ValidationFailed(t *testing.T) {
	for _, msg := range []string{
		"invalid input: summary must be at most 280 characters (got 281)",
		"invalid input: only a note can carry a summary",
	} {
		t.Run(msg, func(t *testing.T) {
			items := &intentionItems{err: status.Error(codes.InvalidArgument, msg)}
			api := newIntentionAPI(t, items)

			resp := api.Patch(itemURL(), map[string]any{"summary": "x"})

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
		})
	}
}

func TestCreateChecklist_Summary_IsAnUnknownMember(t *testing.T) {
	items := &intentionItems{}
	api := newIntentionAPI(t, items)

	resp := api.Post("/api/plans/"+uuid.NewString()+"/checklists",
		map[string]any{"description": "new", "summary": "gist"})

	// The 422's VALIDATION_FAILED code comes from the gateway's huma.NewError
	// hook (config/api.go), which this bare fixture does not install.
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422: %s", resp.Code, resp.Body.String())
	}
	if items.created {
		t.Error("a create carrying summary reached plan-service")
	}
}

// client.go maps both directions between the wire types and the proto.
func TestUpdateItemRequest_Summary(t *testing.T) {
	planID, id, userID := uuid.New(), uuid.New(), uuid.New()

	if got := updateItemRequest(planID, id, userID, UpdateChecklistReq{}); got.Summary != nil {
		t.Errorf("omitted summary reached the proto as %q", *got.Summary)
	}
	got := updateItemRequest(planID, id, userID, UpdateChecklistReq{Summary: strp("")})
	if got.Summary == nil || *got.Summary != "" {
		t.Errorf("empty summary = %v, want a set \"\"", got.Summary)
	}
}

func TestChecklistFromProto_Summary(t *testing.T) {
	if got := checklistFromProto(&pb.ChecklistItem{}); got.Summary != nil {
		t.Errorf("unset summary mapped to %q", *got.Summary)
	}
	got := checklistFromProto(&pb.ChecklistItem{Summary: strp("gist")})
	if got.Summary == nil || *got.Summary != "gist" {
		t.Errorf("summary = %v, want \"gist\"", got.Summary)
	}
}
