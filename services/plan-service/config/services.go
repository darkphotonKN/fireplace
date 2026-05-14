package config

import (
	"log/slog"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/darkphotonKN/fireplace/services/plan-service/internal/checklistitem"
	"github.com/darkphotonKN/fireplace/services/plan-service/internal/plan"
	"github.com/jmoiron/sqlx"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

// SetupServices wires plan-service dependencies and returns a configured gRPC
// server with both PlanService and ChecklistService registered.
func SetupServices(db *sqlx.DB, amqpChannel *amqp.Channel, _ discovery.Registry) *grpc.Server {
	publisher := commonbroker.NewAmqpPublisher(amqpChannel)

	// --- plans ---
	planRepo := plan.NewRepository(db)
	planService := plan.NewService(planRepo, publisher)
	planHandler := plan.NewHandler(planService)

	// --- checklist items ---
	checklistRepo := checklistitem.NewRepository(db)
	checklistService := checklistitem.NewService(checklistRepo, publisher)
	checklistHandler := checklistitem.NewHandler(checklistService)

	// AMQP infra: own plan.events exchange + scaffold consumer for auth.events.
	if err := plan.SetupAMQPInfrastructure(amqpChannel); err != nil {
		slog.Error("Failed to setup AMQP infrastructure", "error", err)
	}
	consumer := plan.NewConsumer(planService, amqpChannel)
	consumer.Listen()

	grpcServer := grpc.NewServer()
	pb.RegisterPlanServiceServer(grpcServer, planHandler)
	pb.RegisterChecklistServiceServer(grpcServer, checklistHandler)

	slog.Info("plan-service initialized successfully")
	return grpcServer
}
