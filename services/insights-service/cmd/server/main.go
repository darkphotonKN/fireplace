package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os/signal"
	"syscall"
	"time"

	"github.com/darkphotonKN/fireplace/common/broker"
	"github.com/darkphotonKN/fireplace/common/cache"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/darkphotonKN/fireplace/common/discovery/consul"
	commontelemetry "github.com/darkphotonKN/fireplace/common/telemetry"
	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/darkphotonKN/fireplace/services/insights-service/config"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
)

var (
	environment       = commonhelpers.GetEnvString("ENVIRONMENT", "development")
	collectorEndpoint = commonhelpers.GetEnvString("COLLECTOR_ENDPOINT", "localhost:4317")

	serviceName = "insights"
	// gRPC port — ordinal 6 in the 710X scheme (auth 7101, example 7102,
	// plan 7103, calendar 7104, orchestrator 7105, insights 7106).
	grpcAddr       = commonhelpers.GetEnvString("GRPC_INSIGHTS_ADDR", "7106")
	consulAddr     = commonhelpers.GetEnvString("CONSUL_ADDR", "localhost:8520")
	serviceVersion = commonhelpers.GetEnvString("SERVICE_VERSION", "1.0.0")
	otelEnabled    = commonhelpers.GetEnvString("OTEL_ENABLED", "true") == "true"

	amqpUser     = commonhelpers.GetEnvString("RABBITMQ_USER", "fireplace")
	amqpPassword = commonhelpers.GetEnvString("RABBITMQ_PASS", "fireplace")
	amqpHost     = commonhelpers.GetEnvString("RABBITMQ_HOST", "localhost")
	amqpPort     = commonhelpers.GetEnvString("RABBITMQ_PORT", "5683")

	redisAddr     = commonhelpers.GetEnvString("REDIS_ADDR", "localhost:6380")
	redisPassword = commonhelpers.GetEnvString("REDIS_PASSWORD", "")
)

func main() {
	commonhelpers.SetupLogger(environment)

	db := config.InitDB()
	defer db.Close()

	registry, err := consul.NewRegistry(consulAddr, serviceName)
	if err != nil {
		log.Fatal("Failed to create Consul registry")
	}

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

	if err := registry.Register(ctx, instanceID, serviceName, "localhost:"+grpcAddr); err != nil {
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

	listener, err := net.Listen("tcp", "localhost:"+grpcAddr)
	if err != nil {
		log.Fatalf("Failed to listen at port %s: %v", grpcAddr, err)
	}
	defer listener.Close()

	ch, closeCh := broker.Connect(amqpUser, amqpPassword, amqpHost, amqpPort)
	defer func() {
		closeCh()
		ch.Close()
	}()

	redisClient, err := cache.Connect(ctx, cache.Config{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer redisClient.Close()

	// for graceful shutdown
	// returns a context that cancels AUTOMATICALLY when SIGTERM or SIGINT arrives.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	grpcServer, err := config.SetupServices(db, ch, registry, redisClient)
	if err != nil {
		log.Fatalf("Failed to set up services: %v", err)
	}

	go func() {
		slog.Info("insights-service gRPC server starting", "port", grpcAddr)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal("Can't connect to grpc server. Error:", err.Error())
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received, stopping gRPC server")
	grpcServer.GracefulStop()

	slog.Info("grpc server stopped")
}
