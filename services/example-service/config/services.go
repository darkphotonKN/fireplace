package config

import (
	"log/slog"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/example"
	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/darkphotonKN/fireplace/services/example-service/internal/example"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

// SetupServices wires the example-service dependencies and returns a configured
// gRPC server. example-service is decorative — there is no DB; the service
// exists as a reference template for future extractions.
func SetupServices(amqpChannel *amqp.Channel, _ discovery.Registry) *grpc.Server {
	publisher := commonbroker.NewAmqpPublisher(amqpChannel)

	service := example.NewService(publisher)

	handler := example.NewHandler(service)

	if err := example.SetupAMQPInfrastructure(amqpChannel); err != nil {
		slog.Error("Failed to setup AMQP infrastructure", "error", err)
	}
	consumer := example.NewConsumer(service, amqpChannel)
	consumer.Listen()

	grpcServer := grpc.NewServer()
	pb.RegisterExampleServiceServer(grpcServer, handler)

	slog.Info("example-service initialized successfully")
	return grpcServer
}
