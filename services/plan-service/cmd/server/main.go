package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/darkphotonKN/fireplace/common/broker"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/darkphotonKN/fireplace/common/discovery/static"
	commontelemetry "github.com/darkphotonKN/fireplace/common/telemetry"
	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/darkphotonKN/fireplace/services/plan-service/config"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
)

var (
	environment       = commonhelpers.GetEnvString("ENVIRONMENT", "development")
	collectorEndpoint = commonhelpers.GetEnvString("COLLECTOR_ENDPOINT", "localhost:4317")

	serviceName    = "plan"
	grpcAddr       = commonhelpers.GetEnvString("GRPC_PLAN_ADDR", "7103")
	serviceVersion = commonhelpers.GetEnvString("SERVICE_VERSION", "1.0.0")

	// The address OTHER services reach this one on. "localhost" is correct when
	// everything runs on the host under air, and wrong inside a container — a
	// process binding or advertising localhost there is reachable only by
	// itself. Compose sets this to the container name (ADR-0012 §2).
	advertiseAddr = commonhelpers.GetEnvString("ADVERTISE_HOST", "localhost")
	otelEnabled    = commonhelpers.GetEnvString("OTEL_ENABLED", "true") == "true"

	amqpUser     = commonhelpers.GetEnvString("RABBITMQ_USER", "fireplace")
	amqpPassword = commonhelpers.GetEnvString("RABBITMQ_PASS", "fireplace")
	amqpHost     = commonhelpers.GetEnvString("RABBITMQ_HOST", "localhost")
	amqpPort     = commonhelpers.GetEnvString("RABBITMQ_PORT", "5683")
)

func main() {
	commonhelpers.SetupLogger(environment)

	db := config.InitDB()
	defer db.Close()

	// Static discovery (ADR-0012 §4) — Consul is gone. Same discovery.Registry
	// interface, so no gRPC call site changed.
	registry := static.NewRegistry()

	ctx := context.Background()

	shutdown, err := commontelemetry.Init(ctx, commontelemetry.Config{
		ServiceName:       serviceName,
		ServiceVersion:    serviceVersion,
		Environment:       environment,
		CollectorEndpoint: collectorEndpoint,
		Enabled:           otelEnabled,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer shutdown(ctx)

	instanceID := discovery.GenerateInstanceID(serviceName)

	if err := registry.Register(ctx, instanceID, serviceName, advertiseAddr+":"+grpcAddr); err != nil {
		log.Printf("\nError when registering service:\n\n%s\n\n", err)
		panic(err)
	}

	go func() {
		for {
			if err := registry.HealthCheck(instanceID, serviceName); err != nil {
				log.Fatal("Health check failed.")
			}
			time.Sleep(time.Second * 1)
		}
	}()
	defer registry.Deregister(ctx, instanceID, serviceName)

	listener, err := net.Listen("tcp", ":"+grpcAddr)
	if err != nil {
		log.Fatalf("Failed to listen at port %s: %v", grpcAddr, err)
	}
	defer listener.Close()

	ch, closeCh := broker.Connect(amqpUser, amqpPassword, amqpHost, amqpPort)
	broker.DeclareExchange(ch, commonconstants.PlanEventsExchange, "topic")
	// plan-service also publishes/consumes against auth.events for cascade-delete on user.deleted.
	broker.DeclareExchange(ch, commonconstants.AuthEventsExchange, "topic")
	defer func() {
		closeCh()
		ch.Close()
	}()

	// for graceful shutdown
	// returns a context that cancels AUTOMATICALLY when SIGTERM or SIGINT arrives.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// setup DI of services and grpc server

	// setup separate context for cancelling to give grace window for final drain
	// NOTE: don't tie this to sigterm or everything downstream is torn down when
	// sigterm fires
	wg := sync.WaitGroup{}
	workerCtx, workerCtxCnl := context.WithCancel(context.Background())
	grpcServer := config.SetupServices(workerCtx, db, ch, registry, &wg)

	go func() {
		slog.Info("plan-service gRPC server starting", "port", grpcAddr)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal("Can't connect to grpc server. Error:", err.Error())
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received, stopping gRPC server")
	grpcServer.GracefulStop()

	slog.Info("grpc server stopped")

	// gracefully kill workers, giving it time to close
	workerCtxCnl()

	// block until given workers time to clean up
	wg.Wait()
}
