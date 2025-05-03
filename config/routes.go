package config

import (
	"github.com/darkphotonKN/fireplace/internal/booking"
	"github.com/darkphotonKN/fireplace/internal/checklistitems"
	"github.com/darkphotonKN/fireplace/internal/user"
	"github.com/gin-gonic/gin"
)

/**
* Sets up API prefix route and all routers.
**/
func SetupRouter() *gin.Engine {
	router := gin.Default()

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
	userRoutes.GET("/", userHandler.GetAll)
	userRoutes.POST("/signup", userHandler.Create)
	userRoutes.POST("/signin", userHandler.Login)

	// --- Plan Routes ---

	planRoutes := api.Group("/plan")

	// --- CHECKLIST ---

	// -- Checklist Setup --
	checkListRepo := checklistitems.NewRepository(DB)
	checkListService := checklistitems.NewService(checkListRepo)
	checkListHandler := checklistitems.NewHandler(checkListService)

	// -- Checklist Plan-Specific Routes --
	checkListRoutes := planRoutes.Group("/:plan_id/checklist")
	checkListRoutes.GET("/", checkListHandler.GetAll)
	checkListRoutes.POST("/", checkListHandler.Create)
	checkListRoutes.PATCH("/:id", checkListHandler.Update)
	checkListRoutes.DELETE("/:id", checkListHandler.Delete)

	// -- BOOKING --

	// --- Booking Setup ---
	bookingRepo := booking.NewRepository(DB)
	bookingService := booking.NewService(bookingRepo)
	bookingHandler := booking.NewHandler(bookingService)

	// ---  Booking Routes ---
	bookingRoutes := api.Group("/booking")
	bookingRoutes.POST("/:user_id", bookingHandler.Create)
	bookingRoutes.GET("/:id", bookingHandler.GetById)

	return router
}
