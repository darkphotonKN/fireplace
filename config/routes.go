package config

import (
	// "context"
	"log"
	"time"

	"github.com/darkphotonKN/fireplace/internal/ai"
	"github.com/darkphotonKN/fireplace/internal/auth"
	"github.com/darkphotonKN/fireplace/internal/calendar"
	"github.com/darkphotonKN/fireplace/internal/checklistitems"
	"github.com/darkphotonKN/fireplace/internal/discovery"
	"github.com/darkphotonKN/fireplace/internal/insights"
	"github.com/darkphotonKN/fireplace/internal/jobs"
	"github.com/darkphotonKN/fireplace/internal/logger"
	"github.com/darkphotonKN/fireplace/internal/notes"
	"github.com/darkphotonKN/fireplace/internal/plans"
	"github.com/darkphotonKN/fireplace/internal/user"
	"github.com/darkphotonKN/fireplace/internal/useranalytics"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

/**
* Sets up API prefix route and all routers.
**/
func SetupRouter(db *sqlx.DB) *gin.Engine {
	router := gin.Default()

	// NOTE: debugging middleware
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

	// base route
	api := router.Group("/api")

	// TODO: testing crawler
	// finder, _ := discovery.NewYoutubeVideoFinder()
	//
	// go finder.FindResources(context.Background(), []concepts.Concept{})

	// --- SERVICE SETUP ---

	userRepo := user.NewRepository(db)
	userService := user.NewService(userRepo)
	userHandler := user.NewHandler(userService)

	planRepo := plans.NewRepository(db)
	planService := plans.NewService(planRepo)
	planHandler := plans.NewHandler(planService)

	checkListRepo := checklistitems.NewRepository(db)
	checkListService := checklistitems.NewService(checkListRepo)
	checkListHandler := checklistitems.NewHandler(checkListService)

	userAnalyticsRepo := useranalytics.NewRepository(db)
	userAnalyticsService := useranalytics.NewService(userAnalyticsRepo, checkListService)
	userAnalyticsHandler := useranalytics.NewHandler(userAnalyticsService)

	checklistGen := ai.NewChecklistGen()
	insightsRepo := insights.NewRepository(db)
	insightsService := insights.NewService(insightsRepo, checklistGen, checkListService, planService, nil)
	insightsHandler := insights.NewHandler(insightsService)

	searchTermGen := ai.NewSearchTermGenerator()
	youtubeVideoFinder, err := discovery.NewYoutubeVideoFinder()
	if err != nil {
		log.Fatalf("Error when attempting to initialize youtubeVideoFinder, error: %+v\n", err)
	}
	videoInsightsRepoService := insights.NewService(insightsRepo, searchTermGen, checkListService, planService, youtubeVideoFinder)
	videoInsightsHandler := insights.NewHandler(videoInsightsRepoService)

	notesRepo := notes.NewRepository(db)
	notesGen := ai.NewNotesGenerator()
	notesService := notes.NewService(notesRepo, notesGen, checkListService, planService)
	notesHandler := notes.NewHandler(notesService)

	calendarRepo := calendar.NewRepository(db)
	calendarService := calendar.NewService(calendarRepo)
	calendarHandler := calendar.NewHandler(calendarService)

	// --- PUBLIC ROUTES (no auth) ---

	publicUsers := api.Group("/users")
	publicUsers.POST("/signup", userHandler.Create)
	publicUsers.POST("/signin", userHandler.Login)

	// --- PROTECTED ROUTES (auth middleware) ---

	protected := api.Group("")
	protected.Use(auth.AuthMiddleware())

	// -- User Routes --
	protectedUsers := protected.Group("/users")
	protectedUsers.GET("/profile", userHandler.GetProfile)
	protectedUsers.PATCH("/profile", userHandler.UpdateProfile)
	protectedUsers.GET("/:id", userHandler.GetById)
	protectedUsers.GET("", userHandler.GetAll)

	// -- Plan Routes --
	planRoutes := protected.Group("/plans")
	planRoutes.GET("/:id", planHandler.GetById)
	planRoutes.GET("", planHandler.GetAll)
	planRoutes.GET("/shared", planHandler.GetAllShared)
	planRoutes.POST("", planHandler.Create)
	planRoutes.PATCH("/:id", planHandler.Update)
	planRoutes.PATCH("/:id/toggle-daily-reset", planHandler.ToggleDailyReset)
	planRoutes.DELETE("/:id", planHandler.Delete)

	// -- Checklist Routes --
	checkListRoutes := protected.Group("/plans/:id/checklists")
	checkListRoutes.GET("", checkListHandler.GetAll)
	checkListRoutes.GET("/archived", checkListHandler.GetAllArchived)
	checkListRoutes.GET("/upcoming", checkListHandler.GetUpcoming)
	checkListRoutes.GET("/:checklist_id", checkListHandler.GetByID)
	checkListRoutes.POST("", checkListHandler.Create)
	checkListRoutes.PATCH("/:checklist_id", checkListHandler.Update)
	checkListRoutes.DELETE("/:checklist_id", checkListHandler.Delete)
	checkListRoutes.PATCH("/:checklist_id/schedule", checkListHandler.SetSchedule)
	checkListRoutes.PATCH("/:checklist_id/archive", checkListHandler.Archive)

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

	// -- Calendar Routes --
	calendarRoutes := protected.Group("/plans/:id/calendar")
	calendarRoutes.GET("", calendarHandler.GetMonth)

	// --- JOBS ---
	// TODO: write a job manager for graceful shutdown
	dailyJob := jobs.NewDailyResetJob(checkListService)
	scheduledItemsJob := jobs.NewScheduledItemsJob(checkListService)

	jobManager := jobs.NewManager()
	jobManager.AddJob(dailyJob)
	jobManager.AddJob(scheduledItemsJob)
	jobManager.StartAll()

	return router
}
