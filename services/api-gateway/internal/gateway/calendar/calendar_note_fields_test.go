package calendargw

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	planpb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	plangw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/plan"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FS-PGG2N R11 (G20): a dated note in the calendar shows its card text's first
// line, summary before body. So each calendar item carries the item's type and,
// when set, its summary; a task and a note without one carry no summary.

type fixedWindow struct{ items []*planpb.ChecklistItem }

func (f fixedWindow) ListItemsInDateWindow(ctx context.Context, planID, userID uuid.UUID, windowStart, windowEnd time.Time) ([]*planpb.ChecklistItem, error) {
	return f.items, nil
}

func TestGetCalendar_ItemsCarryTypeAndSummary(t *testing.T) {
	day := timestamppb.New(time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC))
	gist := "the gist"
	note := &planpb.ChecklistItem{Id: uuid.NewString(), Description: "a long body\nline two",
		Type: "note", Summary: &gist, Scope: "longterm", StartDate: day, DueDate: day}
	bare := &planpb.ChecklistItem{Id: uuid.NewString(), Description: "first\nsecond",
		Type: "note", Scope: "longterm", StartDate: day, DueDate: day}
	task := &planpb.ChecklistItem{Id: uuid.NewString(), Description: "a task",
		Type: "task", Scope: "longterm", StartDate: day, DueDate: day}

	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	access := plangw.NewAccess(relations{rel: plangw.RelationMember})
	RegisterCalendarOperations(api, NewLocalClient(fixedWindow{[]*planpb.ChecklistItem{note, bare, task}}, access),
		func(ctx huma.Context, next func(huma.Context)) {
			next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), uuid.New())))
		}, nil)

	resp := api.(humatest.TestAPI).Get("/api/plans/" + uuid.NewString() + "/calendar?view=week&date=2026-10-14")
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.Code, resp.Body.String())
	}
	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 3 {
		t.Fatalf("items = %d, want 3: %s", len(body.Items), resp.Body.String())
	}

	want := []struct {
		typ     string
		summary any // nil means the member is absent
	}{{"note", gist}, {"note", nil}, {"task", nil}}
	for i, w := range want {
		got := body.Items[i]
		if got["type"] != w.typ {
			t.Errorf("item %d type = %v, want %q", i, got["type"], w.typ)
		}
		summary, present := got["summary"]
		if w.summary == nil && present {
			t.Errorf("item %d summary = %v, want it omitted", i, summary)
		}
		if w.summary != nil && summary != w.summary {
			t.Errorf("item %d summary = %v, want %q", i, summary, w.summary)
		}
	}
}
