package calendargw

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
)

// recordingCalendar captures what reaches calendar-service.
type recordingCalendar struct {
	gotView string
	gotDate string
	called  bool
}

func (r *recordingCalendar) GetCalendar(ctx context.Context, planID, userID uuid.UUID, view, date string) (*GetCalendarResp, error) {
	r.called, r.gotView, r.gotDate = true, view, date
	return &GetCalendarResp{PlanID: planID.String(), View: view, Items: nil}, nil
}

func newCalendarAPI(t *testing.T, rec *recordingCalendar) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	RegisterCalendarOperations(api, rec, func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), uuid.New())))
	}, nil)
	return api.(humatest.TestAPI)
}

// `view` is NOT validated today — it is forwarded verbatim and calendar-service
// decides. Declaring an enum would turn a request the API has always accepted
// into a 422, so the typed param transcribes the current handling instead of
// inventing validation.
func TestGetCalendar_UnrecognisedViewIsForwardedNotRejected(t *testing.T) {
	rec := &recordingCalendar{}
	api := newCalendarAPI(t, rec)
	planID := uuid.New()

	resp := api.Get("/api/plans/" + planID.String() + "/calendar?view=fortnight&date=2026-08-20")
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", resp.Code, resp.Body.String())
	}
	if !rec.called {
		t.Fatal("calendar-service was never called")
	}
	if rec.gotView != "fortnight" {
		t.Fatalf("view = %q, want %q forwarded verbatim", rec.gotView, "fortnight")
	}
}

// An absent `date` defaults to the current window, formatted to match `view` —
// YYYY-MM-DD for week, YYYY-MM for anything else. Transcribed from the legacy
// handler, which did this before calling calendar-service.
func TestGetCalendar_AbsentDateDefaultsToCurrentWindow(t *testing.T) {
	planID := uuid.New()
	cases := []struct {
		name       string
		query      string
		wantFormat string
	}{
		{"week view defaults to a day", "?view=week", "2006-01-02"},
		{"month view defaults to a month", "?view=month", "2006-01"},
		{"absent view defaults to a month", "", "2006-01"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &recordingCalendar{}
			api := newCalendarAPI(t, rec)

			resp := api.Get("/api/plans/" + planID.String() + "/calendar" + tc.query)
			if resp.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body: %s", resp.Code, resp.Body.String())
			}
			want := time.Now().UTC().Format(tc.wantFormat)
			if rec.gotDate != want {
				t.Fatalf("date = %q, want %q", rec.gotDate, want)
			}
		})
	}
}

// An explicit date is forwarded untouched.
func TestGetCalendar_ExplicitDateIsForwarded(t *testing.T) {
	rec := &recordingCalendar{}
	api := newCalendarAPI(t, rec)

	resp := api.Get("/api/plans/" + uuid.New().String() + "/calendar?view=week&date=2026-01-05")
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.Code)
	}
	if rec.gotDate != "2026-01-05" {
		t.Fatalf("date = %q, want %q", rec.gotDate, "2026-01-05")
	}
}

// An empty window marshals to [] and never null (FS-0004 §Edge States).
func TestGetCalendar_EmptyItemsIsArrayNotNull(t *testing.T) {
	rec := &recordingCalendar{}
	api := newCalendarAPI(t, rec)

	resp := api.Get("/api/plans/" + uuid.New().String() + "/calendar?view=week")
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.Code)
	}
	if body := resp.Body.String(); !strings.Contains(body, `"items":[]`) {
		t.Fatalf("body = %s, want items to be []", body)
	}
}
