package config

import (
	"context"
	"net/http"
	"testing"

	calendargw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/calendar"
	plangw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/plan"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// strangerRelations answers "no relation" for every plan.
type strangerRelations struct{}

func (strangerRelations) GetPlanAccess(ctx context.Context, planID, userID uuid.UUID) (plangw.Relation, error) {
	return plangw.RelationNone, nil
}

// planScopedRoutes is one request per plan-scoped domain, sent through the real
// registration path. Every downstream client is nil, so a route that skipped
// the access seam would panic rather than pass.
func planScopedRoutes(planID string) []struct{ method, path string } {
	return []struct{ method, path string }{
		{http.MethodGet, "/api/plans/" + planID},
		{http.MethodPatch, "/api/plans/" + planID + "/toggle-daily-reset"},
		{http.MethodDelete, "/api/plans/" + planID},
		{http.MethodGet, "/api/plans/" + planID + "/checklists"},
		{http.MethodDelete, "/api/plans/" + planID + "/checklists/" + uuid.NewString()},
		{http.MethodGet, "/api/plans/" + planID + "/notes"},
		{http.MethodDelete, "/api/plans/" + planID + "/notes/" + uuid.NewString()},
		{http.MethodGet, "/api/plans/" + planID + "/members"},
		{http.MethodDelete, "/api/plans/" + planID + "/members/" + uuid.NewString()},
		{http.MethodGet, "/api/plans/" + planID + "/calendar?view=week"},
		{http.MethodGet, "/api/insights/checklist-suggestion?plan_id=" + planID},
		{http.MethodGet, "/api/insights/checklist-suggestion-daily?plan_id=" + planID},
		{http.MethodGet, "/api/insights/suggest-videos?plan_id=" + planID},
	}
}

func engineWithAccess(access *plangw.Access) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	MountSerialized(engine, APIDeps{
		PlanAccess: access,
		Calendar:   calendargw.NewLocalClient(nil, access),
	})
	return engine
}

// FS-12ZMF R4, R5: the seam is wired into every plan-scoped domain, and a
// stranger gets 404 from each of them.
func TestPlanScopedRoutes_Stranger_Is404Everywhere(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	e := engineWithAccess(plangw.NewAccess(strangerRelations{}))

	for _, r := range planScopedRoutes(uuid.NewString()) {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			rec := do(t, e, r.method, r.path, token(t))
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

// FS-12ZMF R8: wiring no access seam refuses every plan-scoped route with 403.
func TestPlanScopedRoutes_NoAccessSeam_Is403Everywhere(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	e := engineWithAccess(nil)

	for _, r := range planScopedRoutes(uuid.NewString()) {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			rec := do(t, e, r.method, r.path, token(t))
			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403: %s", rec.Code, rec.Body.String())
			}
		})
	}
}
