package calendargw

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	planpb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	plangw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/plan"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// relations plays plan-service's relation answer behind the gateway's real
// access seam.
type relations struct {
	rel plangw.Relation
	err error
}

func (r relations) GetPlanAccess(ctx context.Context, planID, userID uuid.UUID) (plangw.Relation, error) {
	return r.rel, r.err
}

// windowItems plays plan-service's item window and records that it was asked.
type windowItems struct {
	asked bool
}

func (w *windowItems) ListItemsInDateWindow(ctx context.Context, planID, userID uuid.UUID, windowStart, windowEnd time.Time) ([]*planpb.ChecklistItem, error) {
	w.asked = true
	return nil, nil
}

// R7, R5, R8, R10: the calendar is owner-or-member through the access seam,
// not AssertPlanOwnership. A stranger gets 404 (was 403), and a refused request
// never fetches the plan's items.
func TestGetCalendar_Access(t *testing.T) {
	cases := []struct {
		name       string
		access     PlanAccess
		wantStatus int
	}{
		{"owner", plangw.NewAccess(relations{rel: plangw.RelationOwner}), http.StatusOK},
		{"member", plangw.NewAccess(relations{rel: plangw.RelationMember}), http.StatusOK},
		{"stranger", plangw.NewAccess(relations{rel: plangw.RelationNone}), http.StatusNotFound},
		{"plan-service down", plangw.NewAccess(relations{err: status.Error(codes.Unavailable, "down")}), http.StatusServiceUnavailable},
		{"no checker", nil, http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := &windowItems{}
			_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
			RegisterCalendarOperations(api, NewLocalClient(items, tc.access), func(ctx huma.Context, next func(huma.Context)) {
				next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), uuid.New())))
			}, nil)

			resp := api.(humatest.TestAPI).Get("/api/plans/" + uuid.New().String() + "/calendar?view=week&date=2026-10-14")

			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d: %s", resp.Code, tc.wantStatus, resp.Body.String())
			}
			if allowed := tc.wantStatus == http.StatusOK; items.asked != allowed {
				t.Errorf("items fetched = %v, want %v", items.asked, allowed)
			}
		})
	}
}

// The access question comes before anything else, input validation included:
// a stranger learns nothing from the calendar, not even that its query was bad.
func TestGetCalendar_Stranger_404EvenWithABadWindow(t *testing.T) {
	items := &windowItems{}
	c := NewLocalClient(items, plangw.NewAccess(relations{rel: plangw.RelationNone}))
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	RegisterCalendarOperations(api, c, func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), uuid.New())))
	}, nil)

	resp := api.(humatest.TestAPI).Get("/api/plans/" + uuid.New().String() + "/calendar?view=fortnight&date=x")

	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", resp.Code, resp.Body.String())
	}
}
