package main

import (
	"fmt"
	"os"

	"github.com/darkphotonKN/fireplace/common/discovery/consul"
	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/darkphotonKN/fireplace/services/api-gateway/config"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/logger"
	"github.com/joho/godotenv"
)

// The OpenAPI spec is generated FROM this code (code-first) and is a build
// artifact — never hand-edited. `make gen` (or `go generate ./...`) regenerates
// default); see docs/api-conventions.md for the governing principle.


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

	// consul registry — used by gateway/* clients to discover downstream
	// services (auth-service, plan-service, etc.). The gateway itself does
	// not register, since it's only invoked externally over HTTP.
	consulAddr := commonhelpers.GetEnvString("CONSUL_ADDR", "localhost:8520")
	registry, err := consul.NewRegistry(consulAddr, "api-gateway")
	if err != nil {
		logger.Error("Failed to connect to Consul", "error", err)
		os.Exit(1)
	}

	// router setup
	router := config.SetupRouter(db, registry)

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
