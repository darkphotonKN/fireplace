package config

import (
	"context"
	"fmt"
	"time"

	"github.com/darkphotonKN/fireplace/internal/ai"
	"github.com/darkphotonKN/fireplace/internal/checklistitems"
	"github.com/darkphotonKN/fireplace/internal/concepts"
	"github.com/darkphotonKN/fireplace/internal/discovery"
	"github.com/darkphotonKN/fireplace/internal/insights"
	"github.com/darkphotonKN/fireplace/internal/jobs"
	"github.com/darkphotonKN/fireplace/internal/plans"
	"github.com/darkphotonKN/fireplace/internal/user"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

/**
* Sets up API prefix route and all routers.
**/
func SetupRouter() *gin.Engine {
	router := gin.Default()

	// NOTE: debugging middleware
	router.Use(func(c *gin.Context) {
		fmt.Println("Incoming request to:", c.Request.Method, c.Request.URL.Path, "from", c.Request.Host)
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
	finder, _ := discovery.NewYoutubeVideoFinder()

	go finder.FindResources(context.Background(), []concepts.Concept{})

	// --- USER ---

	// -- User Setup --
	userRepo := user.NewRepository(DB)
	userService := user.NewService(userRepo)
	userHandler := user.NewHandler(userService)

	// -- User Routes --
	userRoutes := api.Group("/users")
	userRoutes.GET("/:id", userHandler.GetById)
	userRoutes.GET("", userHandler.GetAll)
	userRoutes.POST("/signup", userHandler.Create)
	userRoutes.POST("/signin", userHandler.Login)

	// --- Plan Routes ---

	// -- Plan Setup --
	planRepo := plans.NewRepository(DB)
	planService := plans.NewService(planRepo)
	planHandler := plans.NewHandler(planService)

	// -- Plan Routes --
	planRoutes := api.Group("/plans")
	planRoutes.GET("/:id", planHandler.GetById)
	planRoutes.GET("", planHandler.GetAll)
	planRoutes.POST("", planHandler.Create)
	planRoutes.PATCH("/:id", planHandler.Update)
	planRoutes.PATCH("/:id/toggle-daily-reset", planHandler.ToggleDailyReset)
	planRoutes.DELETE("/:id", planHandler.Delete)

	// --- CHECKLIST ---

	// -- Checklist Setup --
	checkListRepo := checklistitems.NewRepository(DB)
	checkListService := checklistitems.NewService(checkListRepo)
	checkListHandler := checklistitems.NewHandler(checkListService)

	// -- Checklist Plan-Specific Routes --
	// TODO: remove after test
	checkListRoutes := api.Group("/plans/:id/checklists")
	checkListRoutes.GET("", checkListHandler.GetAll)
	checkListRoutes.GET("/archived", checkListHandler.GetAllArchived)
	checkListRoutes.GET("/upcoming", checkListHandler.GetUpcoming)
	checkListRoutes.GET("/:checklist_id", checkListHandler.GetByID)
	checkListRoutes.POST("", checkListHandler.Create)
	checkListRoutes.PATCH("/:checklist_id", checkListHandler.Update)
	checkListRoutes.DELETE("/:checklist_id", checkListHandler.Delete)
	checkListRoutes.PATCH("/:checklist_id/schedule", checkListHandler.SetSchedule)
	checkListRoutes.PATCH("/:checklist_id/archive", checkListHandler.Archive)

	// --- INSIGHTS ---

	// -- Insights Setup --
	checklistGen := ai.NewChecklistGen()
	insightsRepo := insights.NewRepository(DB)
	insightsService := insights.NewService(insightsRepo, checklistGen, checkListService, planService)
	insightsHandler := insights.NewHandler(insightsService)

	// -- User Routes --
	insightsRoutes := api.Group("/insights")
	insightsRoutes.GET("/checklist-suggestion", insightsHandler.GenerateSuggestions)
	insightsRoutes.GET("/checklist-suggestion-daily", insightsHandler.GenerateDailySuggestions)

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
