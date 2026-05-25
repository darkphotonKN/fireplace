package config

import (
	"log/slog"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/calendar"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/darkphotonKN/fireplace/services/calendar-service/internal/calendar"
	"github.com/jmoiron/sqlx"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

// SetupServices wires calendar-service dependencies and returns a configured
// gRPC server. The DB is wired through for the future calendar_entries
// feature; the current GetCalendar flow only reads via plan-service over gRPC.
func SetupServices(_ *sqlx.DB, _ *amqp.Channel, registry discovery.Registry) *grpc.Server {
	planClient := calendar.NewPlanClient(registry)
	service := calendar.NewService(planClient)
	handler := calendar.NewHandler(service)

	grpcServer := grpc.NewServer()
	pb.RegisterCalendarServiceServer(grpcServer, handler)

	slog.Info("calendar-service initialized successfully")
	return grpcServer
}
