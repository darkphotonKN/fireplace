package notes

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// recordingService captures what reaches the service, so the tests assert on the
// parsed filters rather than on the HTTP status alone.
type recordingService struct {
	NotesService
	gotFilters *FilterOptions
	called     bool
}

func (r *recordingService) GetNotesByPlanID(planID uuid.UUID, filters *FilterOptions) ([]Note, error) {
	r.called, r.gotFilters = true, filters
	return nil, nil
}

type allowOwnership struct{}

func (allowOwnership) AssertPlanOwnership(ctx context.Context, planID, userID uuid.UUID) error {
	return nil
}

func newNotesAPI(t *testing.T, svc NotesService) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	// No auth middleware: these tests are about parameter handling. The identity
	// bridge is exercised by the config-level tests.
	RegisterNotesOperations(api, svc, allowOwnership{}, func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), uuid.New())))
	}, nil)
	return api.(humatest.TestAPI)
}

// `tags` is read with QueryArray today, so `?tags=a&tags=b` is a two-element
// filter. A scalar typed param would silently collapse this to the last value —
// a regression no contract gate could see.
func TestListNotes_RepeatedTagsReachServiceAsSlice(t *testing.T) {
	rec := &recordingService{}
	api := newNotesAPI(t, rec)
	planID := uuid.New()

	resp := api.Get("/api/plans/" + planID.String() + "/notes?tags=a&tags=b")
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", resp.Code, resp.Body.String())
	}
	if !rec.called {
		t.Fatal("service was never called")
	}
	if got := rec.gotFilters.Tags; len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("Tags = %v, want [a b]", got)
	}
}

// An absent isRead/isDismissed filter is NOT the same as false: omitting it
// means "no filter", while `?isRead=false` means "only unread". The legacy
// parse also treated any non-"true" value as false, so `?isRead=xyz` filters on
// false rather than 422-ing.
func TestListNotes_AbsentBooleanFilterIsDistinctFromFalse(t *testing.T) {
	planID := uuid.New()
	boolPtr := func(b bool) *bool { return &b }

	cases := []struct {
		name        string
		query       string
		wantRead    *bool
		wantDismiss *bool
	}{
		{"both omitted", "", nil, nil},
		{"isRead true", "?isRead=true", boolPtr(true), nil},
		{"isRead false", "?isRead=false", boolPtr(false), nil},
		{"isRead junk reads as false", "?isRead=xyz", boolPtr(false), nil},
		{"isRead empty is no filter", "?isRead=", nil, nil},
		{"isDismissed true", "?isDismissed=true", nil, boolPtr(true)},
		{"both set", "?isRead=false&isDismissed=true", boolPtr(false), boolPtr(true)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &recordingService{}
			api := newNotesAPI(t, rec)

			resp := api.Get("/api/plans/" + planID.String() + "/notes" + tc.query)
			if resp.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body: %s", resp.Code, resp.Body.String())
			}
			assertBoolFilter(t, "IsRead", rec.gotFilters.IsRead, tc.wantRead)
			assertBoolFilter(t, "IsDismissed", rec.gotFilters.IsDismissed, tc.wantDismiss)
		})
	}
}

func assertBoolFilter(t *testing.T, name string, got, want *bool) {
	t.Helper()
	switch {
	case want == nil && got != nil:
		t.Fatalf("%s = %v, want nil (absent must not become a filter)", name, *got)
	case want != nil && got == nil:
		t.Fatalf("%s = nil, want %v", name, *want)
	case want != nil && *got != *want:
		t.Fatalf("%s = %v, want %v", name, *got, *want)
	}
}

// R15: the transport mirror must be faithful, so a POPULATED domain note and
// its mirror must marshal to identical JSON. Populated matters — a zero-valued
// fixture compares {} to {} and proves nothing.
func TestNoteResponse_RoundTripMatchesDomainJSON(t *testing.T) {
	created := time.Date(2026, 8, 19, 10, 30, 0, 0, time.UTC)
	domain := Note{
		ID:             uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		PlanID:         uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Content:        "Ship the notes slice",
		Type:           TypeAI,
		Priority:       PriorityHigh,
		Tags:           pq.StringArray{"go", "contract"},
		RelatedTaskIDs: pq.StringArray{"33333333-3333-3333-3333-333333333333"},
		IsRead:         true,
		IsDismissed:    false,
		AIMetadata: &AIMetadata{
			GeneratedFrom: "plan focus",
			Confidence:    0.87,
			SourceContext: "checklist",
			GeneratedAt:   "2026-08-19T10:30:00Z",
		},
		CreatedAt: created,
		UpdatedAt: created,
	}

	wantJSON, err := json.Marshal(domain)
	if err != nil {
		t.Fatalf("marshal domain: %v", err)
	}
	gotJSON, err := json.Marshal(toNoteResponse(domain))
	if err != nil {
		t.Fatalf("marshal transport: %v", err)
	}

	var want, got any
	if err := json.Unmarshal(wantJSON, &want); err != nil {
		t.Fatalf("unmarshal domain json: %v", err)
	}
	if err := json.Unmarshal(gotJSON, &got); err != nil {
		t.Fatalf("unmarshal transport json: %v", err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("transport mirror drifted from the domain payload:\n domain    = %s\n transport = %s", wantJSON, gotJSON)
	}
}

// FS-0004 §Edge States: an empty list marshals to [] and never null — a
// consumer iterating null breaks.
func TestListNotes_EmptyResultIsArrayNotNull(t *testing.T) {
	rec := &recordingService{}
	api := newNotesAPI(t, rec)

	resp := api.Get("/api/plans/" + uuid.New().String() + "/notes")
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.Code)
	}
	if body := strings.TrimSpace(resp.Body.String()); body != "[]" {
		t.Fatalf("body = %q, want %q", body, "[]")
	}
}

// crossPlanService owns a note that belongs to a DIFFERENT plan than the one in
// the path.
type crossPlanService struct {
	NotesService
	note    Note
	deleted bool
}

func (c *crossPlanService) GetNoteByID(id uuid.UUID) (*Note, error) { return &c.note, nil }
func (c *crossPlanService) DeleteNote(id uuid.UUID) error           { c.deleted = true; return nil }

// Owning a plan does not imply owning an arbitrary note claimed to be in it.
// The answer is 404 rather than 403 on purpose: 403 would confirm the note
// exists somewhere else.
func TestNoteOperations_NoteFromAnotherPlanIs404AndDoesNotMutate(t *testing.T) {
	pathPlan := uuid.New()
	noteID := uuid.New()
	svc := &crossPlanService{note: Note{ID: noteID, PlanID: uuid.New()}} // different plan

	api := newNotesAPI(t, svc)
	base := "/api/plans/" + pathPlan.String() + "/notes/" + noteID.String()

	t.Run("get", func(t *testing.T) {
		if resp := api.Get(base); resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404; body: %s", resp.Code, resp.Body.String())
		}
	})
	t.Run("delete does not reach the service", func(t *testing.T) {
		if resp := api.Delete(base); resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404; body: %s", resp.Code, resp.Body.String())
		}
		if svc.deleted {
			t.Fatal("DeleteNote was called for a note outside the path's plan")
		}
	})
}

// A nil ownership checker must fail CLOSED. The failure mode of an
// authorization seam is never "allow".
func TestNotesOperations_NilOwnershipFailsClosed(t *testing.T) {
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	RegisterNotesOperations(api, &recordingService{}, nil, func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), uuid.New())))
	}, nil)

	resp := api.(humatest.TestAPI).Get("/api/plans/" + uuid.New().String() + "/notes")
	if resp.Code == http.StatusOK {
		t.Fatalf("status = 200 with no ownership checker; must fail closed")
	}
}

// generateAIService records the requestType that reached the service.
type generateAIService struct {
	NotesService
	gotRequestType string
	called         bool
}

func (g *generateAIService) GenerateAINotes(planID uuid.UUID, requestType string) ([]Note, error) {
	g.called, g.gotRequestType = true, requestType
	return nil, nil
}

// The legacy handler defaulted requestType to "all" whenever the body was
// absent or unparseable. Empty and explicit values must keep their meaning.
func TestGenerateAINotes_BodyHandling(t *testing.T) {
	planID := uuid.New()
	cases := []struct {
		name     string
		body     any
		wantCode int
		wantType string
	}{
		{"explicit type", map[string]any{"requestType": "warning"}, http.StatusOK, "warning"},
		{"empty object defaults to all", map[string]any{}, http.StatusOK, "all"},
		{"empty string defaults to all", map[string]any{"requestType": ""}, http.StatusOK, "all"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &generateAIService{}
			api := newNotesAPI(t, svc)

			resp := api.Post("/api/plans/"+planID.String()+"/notes/generate-ai", tc.body)
			if resp.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d; body: %s", resp.Code, tc.wantCode, resp.Body.String())
			}
			if svc.gotRequestType != tc.wantType {
				t.Fatalf("requestType = %q, want %q", svc.gotRequestType, tc.wantType)
			}
		})
	}
}

// A completely absent body is 400, where the legacy handler fell back to "all".
// This is the one behaviour difference in the slice and it is deliberate: huma
// derives "body required" from the typed Body field, and the alternative —
// RawBody plus a hand-rolled unmarshal — would erase the schema this feature
// exists to publish. `{}` still means "all", and the only consumer
// (notesService.generateContextualNotes) always sends a body.
func TestGenerateAINotes_AbsentBodyIs400(t *testing.T) {
	svc := &generateAIService{}
	api := newNotesAPI(t, svc)

	resp := api.Post("/api/plans/" + uuid.New().String() + "/notes/generate-ai")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", resp.Code, resp.Body.String())
	}
	if svc.called {
		t.Fatal("service was called despite a rejected body")
	}
}

// The legacy handler guarded the WHOLE tag filter on `c.Query("tags")`, which
// returns only the FIRST value — so `?tags=&tags=b` applied no tag filter at
// all, even though a non-empty value was supplied. That is transcribed, not
// fixed (R6), and it is exactly the kind of silent difference the contract
// gates cannot see — so it is pinned here rather than left to a comment.
func TestListNotes_TagFilterEdgeCasesMatchLegacy(t *testing.T) {
	planID := uuid.New()
	cases := []struct {
		name  string
		query string
		want  []string
	}{
		{"both values", "?tags=a&tags=b", []string{"a", "b"}},
		{"leading empty suppresses the whole filter", "?tags=&tags=b", nil},
		{"trailing empty is dropped", "?tags=a&tags=", []string{"a"}},
		{"single empty is no filter", "?tags=", nil},
		{"omitted is no filter", "", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &recordingService{}
			api := newNotesAPI(t, rec)

			resp := api.Get("/api/plans/" + planID.String() + "/notes" + tc.query)
			if resp.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body: %s", resp.Code, resp.Body.String())
			}
			got := rec.gotFilters.Tags
			if len(got) != len(tc.want) {
				t.Fatalf("Tags = %#v, want %#v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("Tags = %#v, want %#v", got, tc.want)
				}
			}
		})
	}
}
