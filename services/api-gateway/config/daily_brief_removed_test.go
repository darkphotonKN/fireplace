package config

import (
	"net/http"
	"strings"
	"testing"

	plangw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/plan"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FS-02JBZ R92, A2: the daily brief's HTTP surface is retired now the client
// reads the day plan. Its read generated on a miss, which ADR-0019 clause 2
// forbids, and nothing calls it any more.
func TestDailyBrief_IsNotRegistered(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := MountSerialized(engine, APIDeps{PlanAccess: plangw.NewAccess(strangerRelations{})})

	for path, item := range api.OpenAPI().Paths {
		if strings.Contains(path, "daily-brief") {
			t.Errorf("openapi still documents %s", path)
		}
		for _, op := range []string{opID(item.Get), opID(item.Post)} {
			if op == "getDailyBrief" || op == "refreshDailyBrief" {
				t.Errorf("openapi still carries operation %s at %s", op, path)
			}
		}
	}
	for _, r := range engine.Routes() {
		if strings.Contains(r.Path, "daily-brief") {
			t.Errorf("router still registers %s %s", r.Method, r.Path)
		}
	}

	for _, req := range []struct{ method, path string }{
		{http.MethodGet, "/api/insights/daily-brief?planId=" + uuid.NewString() + "&date=2026-10-25"},
		{http.MethodPost, "/api/insights/daily-brief/refresh"},
	} {
		rec := do(t, engine, req.method, req.path, token(t))
		if rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: status = %d, want 404 or 405: %s", req.method, req.path, rec.Code, rec.Body.String())
		}
	}
}
