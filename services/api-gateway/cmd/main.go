package main

import (
	"os"
	"strings"

	"github.com/darkphotonKN/fireplace/common/broker"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/common/discovery/consul"
	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/darkphotonKN/fireplace/services/api-gateway/config"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/logger"
	"github.com/joho/godotenv"
)

// The OpenAPI document is generated FROM this code (code-first) and is a build
// artifact — never hand-edited. `make openapi` regenerates it from the typed
// handlers; `make gates` fails if the committed copy has drifted.

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

	// AMQP — the gateway became an event PRODUCER when auth-service folded back
	// in (ADR-0009 §1). It publishes user.created on auth.events; plan-service
	// and insights-service bind queues there for user.deleted cascade-delete,
	// so this exchange must keep being declared and published to.
	amqpCh, closeAmqp := broker.Connect(
		commonhelpers.GetEnvString("RABBITMQ_USER", "fireplace"),
		commonhelpers.GetEnvString("RABBITMQ_PASS", "fireplace"),
		commonhelpers.GetEnvString("RABBITMQ_HOST", "localhost"),
		commonhelpers.GetEnvString("RABBITMQ_PORT", "5683"),
	)
	defer closeAmqp()
	if err := broker.DeclareExchange(amqpCh, commonconstants.AuthEventsExchange, "topic"); err != nil {
		logger.Error("Failed to declare auth.events exchange", "error", err)
		os.Exit(1)
	}
	publisher := broker.NewAmqpPublisher(amqpCh)

	// router setup
	router := config.SetupRouter(db, registry, publisher)

	// The listen address is built with a colon exactly once. The default used to
	// carry its own (":8080") while the address was also built with one, so an
	// unset PORT produced "::8080" — which SplitHostPort rejects as too many
	// colons. It survived because every environment that runs this sets PORT,
	// so the failure only ever showed up as a bind error logged after a startup
	// line that had already claimed success.
	//
	// TrimPrefix accepts PORT in either form.
	const defaultDevPort = "8080"

	port := strings.TrimPrefix(os.Getenv("PORT"), ":")
	if port == "" {
		port = defaultDevPort
	}
	addr := ":" + port

	logger.Info("Starting server", "addr", addr)
	if err := router.Run(addr); err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
