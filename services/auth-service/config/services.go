package config

import (
	"log/slog"
	"os"
	"time"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/auth"
	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/darkphotonKN/fireplace/services/auth-service/internal/auth"
	"github.com/jmoiron/sqlx"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

// parseDurationOr returns the parsed env var or fallback on error/empty.
// Accepts standard Go duration strings: "30m", "24h", "720h", etc.
// (No "d" suffix — use hours: 24h = 1 day, 720h = 30 days, 2160h = 90 days.)
func parseDurationOr(env string, fallback time.Duration) time.Duration {
	v := os.Getenv(env)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		slog.Warn("invalid duration in env, using fallback", "env", env, "value", v, "error", err)
		return fallback
	}
	return d
}

// SetupServices wires the auth-service dependencies and returns a configured
// gRPC server with handlers registered. The JWT secret is read from the env
// here and threaded into the service so the issuer and the api-gateway
// middleware can use the same key without coordinating.
func SetupServices(db *sqlx.DB, amqpChannel *amqp.Channel, _ discovery.Registry) *grpc.Server {
	repo := auth.NewRepository(db)
	publisher := commonbroker.NewAmqpPublisher(amqpChannel)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		slog.Warn("JWT_SECRET is empty — tokens will fail validation in the api-gateway")
	}

	// Token lifetimes — env-driven so each env can tune without code changes.
	// Defaults match the .env (1d access / 7d refresh) — a standard short-lived
	// access + medium-lived refresh split. Override in prod .env.
	accessTTL := parseDurationOr("ACCESS_TOKEN_TTL", 24*time.Hour)    // 1 day
	refreshTTL := parseDurationOr("REFRESH_TOKEN_TTL", 168*time.Hour) // 7 days
	slog.Info("token lifetimes configured", "access", accessTTL, "refresh", refreshTTL)

	service := auth.NewService(repo, publisher, jwtSecret, accessTTL, refreshTTL)
	handler := auth.NewHandler(service)

	if err := auth.SetupAMQPInfrastructure(amqpChannel); err != nil {
		slog.Error("Failed to setup AMQP infrastructure", "error", err)
	}
	consumer := auth.NewConsumer(service, amqpChannel)
	consumer.Listen()

	grpcServer := grpc.NewServer()
	pb.RegisterAuthServiceServer(grpcServer, handler)

	slog.Info("auth-service initialized successfully")
	return grpcServer
}
