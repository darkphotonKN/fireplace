package config

import (
	"log/slog"
	"os"
	"time"

	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	commondiscovery "github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/ai"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	authgw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/auth"
	calendargw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/calendar"
	insightsgw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/insights"
	plangw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/plan"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/jobs"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/logger"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/notes"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/useranalytics"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// SetupRouter wires every domain handler, mounts public + protected route
// groups, and starts background jobs. The registry is used by `gateway/*`
// clients to discover downstream services (auth-service, plan-service).
//
// After Phase 4c, no plan/checklist data lives in this DB — plan-service owns
// it, reached via the planGw client + adapter.
func SetupRouter(db *sqlx.DB, registry commondiscovery.Registry, publisher commonbroker.Publisher) *gin.Engine {
	router := gin.Default()

	router.Use(func(c *gin.Context) {
		logger.Debug("Incoming request", "method", c.Request.Method, "path", c.Request.URL.Path, "host", c.Request.Host)
		c.Next()
	})

	// TODO: CORS for development, remove in PROD
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3010"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := router.Group("/api")

	// --- SERVICE SETUP ---

	// The auth domain runs IN-PROCESS (ADR-0009 §1) — auth-service was folded
	// back into the gateway, so there is no gRPC hop. Tokens are still validated
	// by auth.AuthMiddleware using the shared JWT_SECRET, and verification stays
	// in common/auth (ADR-0009 §5) because plan-service and insights-service
	// verify independently. Do not duplicate the verifier here.
	//
	// No legacy gin handler is constructed for auth: every users endpoint is
	// serialized, so LocalClient is consumed directly by the typed operations.
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		slog.Warn("JWT_SECRET is empty — issued tokens will fail validation")
	}
	// Token lifetimes are env-driven so each environment tunes without a code
	// change. Defaults match the .env: 1 day access, 7 day refresh.
	accessTTL := parseDurationOr("ACCESS_TOKEN_TTL", 24*time.Hour)
	refreshTTL := parseDurationOr("REFRESH_TOKEN_TTL", 168*time.Hour)

	authService := authgw.NewService(authgw.NewRepository(db), publisher, jwtSecret, accessTTL, refreshTTL)
	authClient := authgw.NewLocalClient(authService)

	// plan-service is remote — HTTP routes for /plans and /plans/:id/checklists
	// proxy through plangw via gRPC. The adapter satisfies the in-process
	// interfaces that cross-domain consumers (insights, notes, calendar,
	// useranalytics, jobs) depend on.
	// No legacy gin handler: every plan and checklist endpoint is serialized, so
	// client is consumed directly by the typed operations.
	planGwClient := plangw.NewClient(registry)
	planAdapter := plangw.NewAdapter(planGwClient)

	userAnalyticsRepo := useranalytics.NewRepository(db)
	userAnalyticsService := useranalytics.NewService(userAnalyticsRepo, planAdapter)

	// insights-service is remote — /api/insights/* proxies through insightsgw
	// via gRPC. The in-process implementation (internal/insights service +
	// repository, internal/discovery, and the checklist/search-term generators)
	// was deleted with this move rather than left alongside.
	insightsGwClient := insightsgw.NewClient(registry)

	notesRepo := notes.NewRepository(db)
	notesGen := ai.NewNotesGenerator()
	notesService := notes.NewService(notesRepo, notesGen, planAdapter, planAdapter)

	// calendar-service is remote — gateway proxies /api/plans/:id/calendar
	// through calendargw via gRPC. Calendar-service calls plan-service on
	// its own for ownership checks + item reads.
	calendarGwClient := calendargw.NewClient(registry)

	// --- PROTECTED ROUTES (auth middleware) ---

	protected := api.Group("")
	protected.Use(auth.AuthMiddleware())

	// --- SERIALIZED (typed) SURFACE — ADR-0002 plane 1 ---
	//
	// The ENTIRE users group — signup, signin, get-by-id, list — plus the profile
	// pair is served by typed huma handlers. Their legacy gin registrations are
	// deleted rather than left alongside: gin panics on duplicate route
	// registration, and serialization is replacement, not coexistence per path.
	//
	// Every other group here is still enveloped and is retrofitted group by group
	// under FS-0004. Until a group lands it is reachable but undocumented — a
	// stated trade, since the swaggo document that used to describe it was
	// removed outright (ADR-0006 §5).
	MountSerialized(router, APIDeps{
		Profile:        authClient,
		Users:          authClient,
		Plans:          planGwClient,
		Checklists:     planGwClient,
		Notes:          notesService,
		NotesOwnership: planAdapter,

		Suggestions:      insightsGwClient,
		VideoSuggestions: insightsGwClient,
		Analytics:        userAnalyticsService,
		Calendar:         calendarGwClient,
	})

	// Plan routes are SERIALIZED (FS-0004, I-0016) and registered above via
	// MountSerialized. Their gin registrations are deleted, not left alongside.

	// Checklist routes are SERIALIZED (FS-0004, I-0017), registered above.

	// Analytics and insights routes are SERIALIZED (FS-0004, I-0019), registered
	// above via MountSerialized. Their gin registrations are deleted, not left
	// alongside: gin panics on duplicate route registration.

	// Notes routes are SERIALIZED (FS-0004, I-0018), registered above via
	// MountSerialized. Their gin registrations are deleted, not left alongside:
	// gin panics on duplicate route registration, and serialization is
	// replacement, not coexistence per path.

	// Calendar routes are SERIALIZED (FS-0004, I-0019), registered above.

	// --- JOBS ---
	// Daily reset is now a gRPC call to plan-service.DailyReset (via the adapter).
	dailyJob := jobs.NewDailyResetJob(planAdapter)

	jobManager := jobs.NewManager()
	jobManager.AddJob(dailyJob)
	jobManager.StartAll()

	return router
}

// parseDurationOr returns the parsed env var or fallback on error/empty.
// Accepts standard Go duration strings: "30m", "24h", "720h". No "d" suffix —
// use hours: 24h = 1 day, 168h = 7 days.
func parseDurationOr(env string, fallback time.Duration) time.Duration {
	v := os.Getenv(env)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		slog.Warn("invalid duration in env, using fallback", "env", env, "value", v, "error", err)
		return fallback
	}
	return d
}
