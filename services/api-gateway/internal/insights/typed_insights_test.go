package insights

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/discovery"
	"github.com/google/uuid"
)

type recordingSuggestions struct {
	gotPlanID uuid.UUID
	daily     []string
	single    string
}

func (r *recordingSuggestions) GenerateSuggestions(ctx context.Context, planID uuid.UUID) (string, error) {
	r.gotPlanID = planID
	return r.single, nil
}

func (r *recordingSuggestions) GenerateDailySuggestions(ctx context.Context, planID uuid.UUID) ([]string, error) {
	r.gotPlanID = planID
	return r.daily, nil
}

type recordingVideos struct {
	gotPlanID uuid.UUID
	resources []discovery.Resource
}

func (r *recordingVideos) GenerateSuggestedVideoLinks(ctx context.Context, planID uuid.UUID) ([]discovery.Resource, error) {
	r.gotPlanID = planID
	return r.resources, nil
}

func newInsightsAPI(t *testing.T, s SuggestionsService, v VideoSuggestionsService) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	RegisterInsightsOperations(api, s, v, func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), uuid.New())))
	}, nil)
	return api.(humatest.TestAPI)
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
