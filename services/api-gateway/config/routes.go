package config

import (
	"log"
	"time"

	commondiscovery "github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/ai"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/discovery" // legacy AI search discovery (not service registry)
	authgw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/auth"
	calendargw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/calendar"
	plangw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/plan"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/insights"
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
func SetupRouter(db *sqlx.DB, registry commondiscovery.Registry) *gin.Engine {
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

	// auth-service is remote — the gateway calls it via gRPC. Tokens are still
	// validated locally by auth.AuthMiddleware using the shared JWT_SECRET.
	// No legacy gin handler is constructed for auth: every users endpoint is
	// serialized, so the client is consumed directly by the typed operations.
	authClient := authgw.NewClient(registry)

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
	userAnalyticsHandler := useranalytics.NewHandler(userAnalyticsService)

	checklistGen := ai.NewChecklistGen()
	insightsRepo := insights.NewRepository(db)
	insightsService := insights.NewService(insightsRepo, checklistGen, planAdapter, planAdapter, nil)
	insightsHandler := insights.NewHandler(insightsService)

	searchTermGen := ai.NewSearchTermGenerator()
	youtubeVideoFinder, err := discovery.NewYoutubeVideoFinder()
	if err != nil {
		log.Fatalf("Error when attempting to initialize youtubeVideoFinder, error: %+v\n", err)
	}
	videoInsightsService := insights.NewService(insightsRepo, searchTermGen, planAdapter, planAdapter, youtubeVideoFinder)
	videoInsightsHandler := insights.NewHandler(videoInsightsService)

	notesRepo := notes.NewRepository(db)
	notesGen := ai.NewNotesGenerator()
	notesService := notes.NewService(notesRepo, notesGen, planAdapter, planAdapter)
	notesHandler := notes.NewHandler(notesService)

	// calendar-service is remote — gateway proxies /api/plans/:id/calendar
	// through calendargw via gRPC. Calendar-service calls plan-service on
	// its own for ownership checks + item reads.
	calendarGwClient := calendargw.NewClient(registry)
	calendarGwHandler := calendargw.NewHandler(calendarGwClient)

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
		Profile:    authClient,
		Users:      authClient,
		Plans:      planGwClient,
		Checklists: planGwClient,
	})

	// Plan routes are SERIALIZED (FS-0004, I-0016) and registered above via
	// MountSerialized. Their gin registrations are deleted, not left alongside.

	// Checklist routes are SERIALIZED (FS-0004, I-0017), registered above.

	// -- User Analytics Routes --
	userAnalyticsRoutes := protected.Group("/analytics")
	userAnalyticsRoutes.GET("/user/:userId", userAnalyticsHandler.GetUserAnalytics)

	// -- Insight Routes --
	insightsRoutes := protected.Group("/insights")
	insightsRoutes.GET("/checklist-suggestion", insightsHandler.GenerateSuggestions)
	insightsRoutes.GET("/checklist-suggestion-daily", insightsHandler.GenerateDailySuggestions)
	insightsRoutes.GET("/suggest-videos", videoInsightsHandler.GenerateSuggestedVideoLinks)

	// -- Notes Routes --
	notesRoutes := protected.Group("/plans/:id/notes")
	notesRoutes.GET("", notesHandler.GetAll)
	notesRoutes.GET("/:noteId", notesHandler.GetByID)
	notesRoutes.POST("", notesHandler.Create)
	notesRoutes.PATCH("/:noteId", notesHandler.Update)
	notesRoutes.DELETE("/:noteId", notesHandler.Delete)
	notesRoutes.POST("/generate-ai", notesHandler.GenerateAINotes)

	// -- Calendar Routes (proxied to calendar-service via gRPC) --
	calendarRoutes := protected.Group("/plans/:id/calendar")
	calendarRoutes.GET("", calendarGwHandler.GetCalendar)

	// --- JOBS ---
	// Daily reset is now a gRPC call to plan-service.DailyReset (via the adapter).
	dailyJob := jobs.NewDailyResetJob(planAdapter)

	jobManager := jobs.NewManager()
	jobManager.AddJob(dailyJob)
	jobManager.StartAll()

	return router
}
