package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"time"

	"github.com/darkphotonKN/fireplace/common/broker"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/darkphotonKN/fireplace/common/discovery/consul"
	commontelemetry "github.com/darkphotonKN/fireplace/common/telemetry"
	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/darkphotonKN/fireplace/services/calendar-service/config"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
)

var (
	environment       = commonhelpers.GetEnvString("ENVIRONMENT", "development")
	collectorEndpoint = commonhelpers.GetEnvString("COLLECTOR_ENDPOINT", "localhost:4317")

	serviceName    = "calendar"
	grpcAddr       = commonhelpers.GetEnvString("GRPC_CALENDAR_ADDR", "7104")
	consulAddr     = commonhelpers.GetEnvString("CONSUL_ADDR", "localhost:8520")
	serviceVersion = commonhelpers.GetEnvString("SERVICE_VERSION", "1.0.0")
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

	grpcServer := config.SetupServices(db, ch, registry)

	slog.Info("calendar-service gRPC server starting", "port", grpcAddr)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal("Can't connect to grpc server. Error:", err.Error())
	}
}
