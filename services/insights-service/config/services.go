package config

import (
	"log/slog"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/insights"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/insights"
	"github.com/jmoiron/sqlx"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

// SetupServices wires insights-service dependencies and returns a configured
// gRPC server. The DB is wired through for the future generated_insights cache;
// the current generation flow reads plan focus + checklist from plan-service
// over gRPC and produces suggestions through an injected ContentGenerator.
//
// NOTE: the ContentGenerator is currently insights.StubContentGenerator — the
// real OpenAI-backed generator still lives in the api-gateway and is the next
// piece to migrate. Until then the Generate* / SuggestVideos RPCs surface
// Unimplemented. The plan-service read path is fully wired and live.
//
// redisClient is injected as the narrow insights.Cache interface (DIP: the
// service depends on the abstraction, the composition root supplies the
// concrete). It is wired but not yet used — cache-backed logic comes later.
func SetupServices(db *sqlx.DB, _ *amqp.Channel, registry discovery.Registry, redisClient *redis.Client) *grpc.Server {
	planClient := insights.NewPlanClient(registry)
	generator := insights.StubContentGenerator{}
	repo := insights.NewRepository(db)
	service := insights.NewService(planClient, generator, redisClient, repo)
	handler := insights.NewHandler(service)

	grpcServer := grpc.NewServer()
	pb.RegisterInsightsServiceServer(grpcServer, handler)

	slog.Info("insights-service initialized successfully")
	return grpcServer
}
