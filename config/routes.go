package config

import (
	"fmt"
	"time"

	"github.com/darkphotonKN/fireplace/internal/ai"
	"github.com/darkphotonKN/fireplace/internal/checklistitems"
	"github.com/darkphotonKN/fireplace/internal/insights"
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

	// --- USER ---

	// -- User Setup --
	userRepo := user.NewRepository(DB)
	userService := user.NewService(userRepo)
	userHandler := user.NewHandler(userService)

	// -- User Routes --
	userRoutes := api.Group("/user")
	userRoutes.GET("/:id", userHandler.GetById)
	userRoutes.GET("", userHandler.GetAll)
	userRoutes.POST("/signup", userHandler.Create)
	userRoutes.POST("/signin", userHandler.Login)

	// --- Plan Routes ---

	planRoutes := api.Group("/plan")

	// -- Plan Setup --
	planRepo := plans.NewRepository(DB)
	planService := plans.NewService(planRepo)
	planHandler := plans.NewHandler(planService)

	// -- Plan Routes --
	planRoutes.GET("/:id", planHandler.GetById)

	// --- CHECKLIST ---

	// -- Checklist Setup --
	checkListRepo := checklistitems.NewRepository(DB)
	checkListService := checklistitems.NewService(checkListRepo)
	checkListHandler := checklistitems.NewHandler(checkListService)

	// -- Checklist Plan-Specific Routes --
	checkListRoutes := planRoutes.Group("/:plan_id/checklist")
	checkListRoutes.GET("", checkListHandler.GetAll)
	checkListRoutes.POST("", checkListHandler.Create)
	checkListRoutes.PATCH("/:id", checkListHandler.Update)
	checkListRoutes.DELETE("/:id", checkListHandler.Delete)

	// --- INSIGHTS ---

	// -- Insights Setup --
	contentGen := ai.NewContentGen()
	insightsRepo := insights.NewRepository
	insightsService := insights.NewService(insightsRepo, contentGen, checkListService)
	insightsHandler := insights.NewHandler(insightsService)

	// -- User Routes --
	insightsRoutes := api.Group("/insights")
	insightsRoutes.GET("/checklist-suggestion", insightsHandler.GenerateChecklistSuggestionHandler)

	return router
}
