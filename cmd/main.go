package main

import (
	"fmt"
	"os"

	"github.com/darkphotonKN/fireplace/config"
	"github.com/darkphotonKN/fireplace/internal/logger"
	"github.com/joho/godotenv"
)

/**
* Main entry point to entire application.
* NOTE: Keep code here as clean and little as possible.
**/
func main() {

	// env setup
	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found, using system environment variables")
	}

	// database setup
	db := config.InitDB()
	defer db.Close()

	// router setup
	router := config.SetupRouter(db)

	defaultDevPort := ":8080"

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultDevPort
	}

	// starts server and listen on port
	logger.Info("Starting server", "port", port)
	if err := router.Run(fmt.Sprintf(":%s", port)); err != nil {
		logger.Error("Failed to start server", "error", err)
	}
}
