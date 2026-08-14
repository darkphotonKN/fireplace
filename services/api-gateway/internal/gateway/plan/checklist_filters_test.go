package plangw

import (
	"context"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
)

// recordingClient captures what the filters were converted to, so the tests can
// assert on what reaches plan-service rather than on the HTTP status alone.
type recordingClient struct {
	ChecklistsClient
	gotScope *string
	gotType  *string
	called   bool
}

func (r *recordingClient) ListChecklists(ctx context.Context, planID, userID uuid.UUID, scope, itemType *string) ([]*ChecklistResp, error) {
	r.called, r.gotScope, r.gotType = true, scope, itemType
	return nil, nil
}

func newFilterAPI(t *testing.T, rec *recordingClient) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	// No auth middleware: these tests are about parameter handling, and the
	// identity bridge is exercised by the config-level tests.
	RegisterChecklistOperations(api, rec, func(ctx huma.Context, next func(huma.Context)) {
		// Stand in for the real bridge: put an identity on the context the way
		// BridgeIdentity does, without booting the middleware chain.
		next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), uuid.New())))
	}, nil)
	return api.(humatest.TestAPI)
}

// The legacy handler forwarded a filter only when the query value was non-empty:
//
//	if v := c.Query("scope"); v != "" { scopePtr = &v }
//
// so `?scope=` and omitting scope have ALWAYS meant the same thing. Declaring an
// enum on the parameter would make the empty string illegal and turn a request
// the API has always accepted into a 422.
func TestListChecklists_EmptyFilterIsNotRejected(t *testing.T) {
	planID := uuid.New()
	cases := []struct {
		name string
		url  string
	}{
		{"omitted", "/api/plans/" + planID.String() + "/checklists"},
		{"empty scope", "/api/plans/" + planID.String() + "/checklists?scope="},
		{"empty type", "/api/plans/" + planID.String() + "/checklists?type="},
		{"both empty", "/api/plans/" + planID.String() + "/checklists?scope=&type="},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &recordingClient{}
			api := newFilterAPI(t, rec)

			resp := api.Get(tc.url)

			if resp.Code == http.StatusUnprocessableEntity {
				t.Fatalf("%s was rejected with 422; the legacy API accepted it: %s", tc.url, resp.Body.String())
			}
			if !rec.called {
				t.Fatalf("request never reached the client (status %d): %s", resp.Code, resp.Body.String())
			}
			if rec.gotScope != nil {
				t.Errorf("scope forwarded as %q; an empty/absent filter must forward nil", *rec.gotScope)
			}
			if rec.gotType != nil {
				t.Errorf("type forwarded as %q; an empty/absent filter must forward nil", *rec.gotType)
			}
		})
	}
}

func TestListChecklists_RealFilterIsForwarded(t *testing.T) {
	planID := uuid.New()
	rec := &recordingClient{}
	api := newFilterAPI(t, rec)

	resp := api.Get("/api/plans/" + planID.String() + "/checklists?scope=daily&type=task")

	if !rec.called {
		t.Fatalf("request never reached the client (status %d): %s", resp.Code, resp.Body.String())
	}
	if rec.gotScope == nil || *rec.gotScope != "daily" {
		t.Errorf("scope = %v, want \"daily\"", rec.gotScope)
	}
	if rec.gotType == nil || *rec.gotType != "task" {
		t.Errorf("type = %v, want \"task\"", rec.gotType)
	}
}

// The enum on scope/type is a BEHAVIOUR CHANGE and this test pins which way it
// went. The legacy handler forwarded any non-empty value downstream, so
// `?scope=bogus` reached plan-service. Declaring an enum makes huma reject it at
// the boundary with 422 instead.
//
// That is the correct side of ADR-0005's line — scope and type are SHAPE, and
// shape is validated at the boundary — but it is a change, not a
// transcription, so it is asserted rather than assumed.
func TestListChecklists_InvalidFilterIsRejectedAtTheBoundary(t *testing.T) {
	planID := uuid.New()
	rec := &recordingClient{}
	api := newFilterAPI(t, rec)

	resp := api.Get("/api/plans/" + planID.String() + "/checklists?scope=bogus")

	if resp.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422 for a value outside the declared enum", resp.Code)
	}
	if rec.called {
		t.Error("an invalid scope reached plan-service; shape must be rejected at the boundary")
	}
}
