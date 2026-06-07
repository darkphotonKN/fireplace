package config

import (
	"log/slog"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/orchestrator"
	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/darkphotonKN/fireplace/services/orchestrator-service/internal/orchestrator"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

// SetupServices wires the orchestrator-service dependencies and returns a
// configured gRPC server. orchestrator-service owns NO database — it
// coordinates other services over gRPC + AMQP, so the only injected
// infrastructure is the AMQP channel (for publishing) and the service registry
// (for discovering downstream services once real orchestration lands).
func SetupServices(amqpChannel *amqp.Channel, _ discovery.Registry) *grpc.Server {
	// Publisher injected into the service so it can emit orchestrator.events.
	publisher := commonbroker.NewAmqpPublisher(amqpChannel)

	service := orchestrator.NewService(publisher)

	handler := orchestrator.NewHandler(service)

	// Declare the exchange + per-service inbound queue, then start the consumer
	// with the service injected (so future event handlers can call into it).
	if err := orchestrator.SetupAMQPInfrastructure(amqpChannel); err != nil {
		slog.Error("Failed to setup AMQP infrastructure", "error", err)
	}
	consumer := orchestrator.NewConsumer(service, amqpChannel)
	consumer.Listen()

	grpcServer := grpc.NewServer()
	pb.RegisterOrchestratorServiceServer(grpcServer, handler)

	slog.Info("orchestrator-service initialized successfully")
	return grpcServer
}
