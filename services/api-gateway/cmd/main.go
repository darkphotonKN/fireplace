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
// docs/swagger.{json,yaml}. Output is Swagger / OpenAPI 2.0 (swaggo/swag
// default); see docs/api-conventions.md for the governing principle.
//
//go:generate sh -c "cd .. && go run github.com/swaggo/swag/cmd/swag init -g cmd/main.go -o docs --parseDependency --parseInternal"

//	@title			Fireplace API Gateway
//	@version		1.0
//	@description	HTTP edge for the Fireplace microservices. This spec is a
//	@description	code-generated SHAPE contract (OpenAPI 2.0), not a validator:
//	@description	runtime validation lives in Go (gin `binding` tags at the edge,
//	@description	domain invariants in the downstream services). Conditional
//	@description	rules are described in prose on each field, never encoded as
//	@description	schema constraints. See docs/api-conventions.md.
//	@host			localhost:8080
//	@BasePath		/api
//	@schemes		http https
//	@tag.name		checklists
//	@tag.description	Checklist items belonging to a plan (tasks and notes). Reference vertical for the code-first OpenAPI setup.
//	@securityDefinitions.apikey	BearerAuth
//	@in				header
//	@name			Authorization
//	@description	JWT bearer token. Format: "Bearer &lt;token&gt;".

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
