package config

import (
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	plangw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/plan"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FS-02JBZ R54: the per-plan daily-reset toggle is gone from the contract and
// from the router. Nothing nightly reads the flag any more (R53).
func TestToggleDailyReset_IsNotRegistered(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := MountSerialized(engine, APIDeps{PlanAccess: plangw.NewAccess(strangerRelations{})})

	for path, item := range api.OpenAPI().Paths {
		if strings.Contains(path, "toggle-daily-reset") {
			t.Errorf("openapi still documents %s", path)
		}
		for _, op := range []string{opID(item.Patch), opID(item.Put), opID(item.Post)} {
			if op == "toggleDailyReset" {
				t.Errorf("openapi still carries operation toggleDailyReset at %s", path)
			}
		}
	}
	for _, r := range engine.Routes() {
		if strings.Contains(r.Path, "toggle-daily-reset") {
			t.Errorf("router still registers %s %s", r.Method, r.Path)
		}
	}

	rec := do(t, engine, http.MethodPatch, "/api/plans/"+uuid.NewString()+"/toggle-daily-reset", token(t))
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 404 or 405: %s", rec.Code, rec.Body.String())
	}
}

func opID(op *huma.Operation) string {
	if op == nil {
		return ""
	}
	return op.OperationID
}
