package insights

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
)

type recordingSuggestions struct {
	gotPlanID uuid.UUID
	gotUserID uuid.UUID
	daily     []string
	single    string
}

func (r *recordingSuggestions) GenerateSuggestions(ctx context.Context, planID, userID uuid.UUID) (string, error) {
	r.gotPlanID, r.gotUserID = planID, userID
	return r.single, nil
}

func (r *recordingSuggestions) GenerateDailySuggestions(ctx context.Context, planID, userID uuid.UUID) ([]string, error) {
	r.gotPlanID, r.gotUserID = planID, userID
	return r.daily, nil
}

type recordingVideos struct {
	gotPlanID uuid.UUID
	gotUserID uuid.UUID
	videos    []VideoSuggestionResponse
}

func (r *recordingVideos) SuggestVideos(ctx context.Context, planID, userID uuid.UUID) ([]VideoSuggestionResponse, error) {
	r.gotPlanID, r.gotUserID = planID, userID
	return r.videos, nil
}

func newInsightsAPI(t *testing.T, s SuggestionsService, v VideoSuggestionsService) humatest.TestAPI {
	return newInsightsAPIAs(t, s, v, uuid.New())
}

func newInsightsAPIAs(t *testing.T, s SuggestionsService, v VideoSuggestionsService, userID uuid.UUID) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	RegisterInsightsOperations(api, s, v, func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), userID)))
	}, nil)
	return api.(humatest.TestAPI)
}

// The in-process implementation took a plan id and NOTHING else — no user id,
// no ownership assertion, so any authenticated caller could read suggestions
// for any plan. insights-service fetches plan context through plan-service,
// which enforces ownership, so the caller's identity has to reach it. This is
// the behaviour change the strangler repoint carries.
func TestInsights_CallerIdentityReachesTheService(t *testing.T) {
	caller := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	planID := uuid.New()

	t.Run("suggestion", func(t *testing.T) {
		s := &recordingSuggestions{}
		api := newInsightsAPIAs(t, s, &recordingVideos{}, caller)
		api.Get("/api/insights/checklist-suggestion?plan_id=" + planID.String())
		if s.gotUserID != caller {
			t.Fatalf("userID = %s, want %s", s.gotUserID, caller)
		}
	})
	t.Run("daily", func(t *testing.T) {
		s := &recordingSuggestions{}
		api := newInsightsAPIAs(t, s, &recordingVideos{}, caller)
		api.Get("/api/insights/checklist-suggestion-daily?plan_id=" + planID.String())
		if s.gotUserID != caller {
			t.Fatalf("userID = %s, want %s", s.gotUserID, caller)
		}
	})
	t.Run("videos", func(t *testing.T) {
		v := &recordingVideos{}
		api := newInsightsAPIAs(t, &recordingSuggestions{}, v, caller)
		api.Get("/api/insights/suggest-videos?plan_id=" + planID.String())
		if v.gotUserID != caller {
			t.Fatalf("userID = %s, want %s", v.gotUserID, caller)
		}
	})
}

// plan_id is snake_case here while notes uses camelCase. Both are transcribed
// as they ship (R6) — normalizing is a later feature's job.
func TestInsights_PlanIDIsSnakeCaseQueryParam(t *testing.T) {
	planID := uuid.New()
	s := &recordingSuggestions{single: "Do the thing"}
	v := &recordingVideos{}
	api := newInsightsAPI(t, s, v)

	resp := api.Get("/api/insights/checklist-suggestion?plan_id=" + planID.String())
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", resp.Code, resp.Body.String())
	}
	if s.gotPlanID != planID {
		t.Fatalf("planID = %s, want %s", s.gotPlanID, planID)
	}
}

// The single suggestion ships as a bare string inside `result` today, so with
// the envelope gone the body IS the string. No wrapper object is invented (R6).
func TestGetChecklistSuggestion_BodyIsBareString(t *testing.T) {
	s := &recordingSuggestions{single: "Review your current project priorities"}
	api := newInsightsAPI(t, s, &recordingVideos{})

	resp := api.Get("/api/insights/checklist-suggestion?plan_id=" + uuid.New().String())
	if got := strings.TrimSpace(resp.Body.String()); got != `"Review your current project priorities"` {
		t.Fatalf("body = %s, want a bare JSON string", got)
	}
}

// The daily suggestions ship as an array of strings — bare, matching every
// other collection this contract publishes.
func TestGetDailyChecklistSuggestion_BodyIsBareStringArray(t *testing.T) {
	s := &recordingSuggestions{daily: []string{"a", "b", "c"}}
	api := newInsightsAPI(t, s, &recordingVideos{})

	resp := api.Get("/api/insights/checklist-suggestion-daily?plan_id=" + uuid.New().String())
	if got := strings.TrimSpace(resp.Body.String()); got != `["a","b","c"]` {
		t.Fatalf("body = %s, want a bare JSON string array", got)
	}
}

// Empty results marshal to [] and never null (FS-0004 §Edge States).
func TestInsights_EmptyCollectionsAreArraysNotNull(t *testing.T) {
	api := newInsightsAPI(t, &recordingSuggestions{}, &recordingVideos{})
	planID := uuid.New().String()

	for _, path := range []string{
		"/api/insights/checklist-suggestion-daily?plan_id=" + planID,
		"/api/insights/suggest-videos?plan_id=" + planID,
	} {
		if got := strings.TrimSpace(api.Get(path).Body.String()); got != "[]" {
			t.Fatalf("%s body = %s, want []", path, got)
		}
	}
}
