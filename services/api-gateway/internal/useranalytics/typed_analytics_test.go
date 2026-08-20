package useranalytics

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
)

func newAnalyticsAPI(t *testing.T) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	RegisterAnalyticsOperations(api, nil, func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, auth.WithUserID(ctx.Context(), uuid.New())))
	}, nil)
	return api.(humatest.TestAPI)
}

// The handler and repository are documented stubs. The operation is published
// with its success shape declared, but it answers 501 today — and that 501 must
// carry a domain code, because ADR-0004 says clients switch on `code`, never on
// `detail`. A 501 with no code is a contract break wearing a passing status.
func TestGetUserAnalytics_IsNotImplementedWithDomainCode(t *testing.T) {
	api := newAnalyticsAPI(t)

	resp := api.Get("/api/analytics/user/" + uuid.New().String())
	if resp.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501; body: %s", resp.Code, resp.Body.String())
	}

	var problem struct {
		Status int    `json:"status"`
		Code   string `json:"code"`
		Errors []any  `json:"errors"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &problem); err != nil {
		t.Fatalf("body is not problem+json: %v (%s)", err, resp.Body.String())
	}
	if problem.Code != "NOT_IMPLEMENTED" {
		t.Fatalf("code = %q, want NOT_IMPLEMENTED", problem.Code)
	}
	if problem.Errors == nil {
		t.Fatal("errors[] must always be present, empty when there is no field detail (R10)")
	}
}
