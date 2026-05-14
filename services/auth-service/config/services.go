package config

import (
	"log/slog"
	"os"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/auth"
	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/darkphotonKN/fireplace/services/auth-service/internal/auth"
	"github.com/jmoiron/sqlx"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

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

	service := auth.NewService(repo, publisher, jwtSecret)
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
