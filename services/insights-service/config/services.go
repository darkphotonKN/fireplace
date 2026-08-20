package config

import (
	"fmt"
	"log/slog"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/insights"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/ai"
	videodiscovery "github.com/darkphotonKN/fireplace/services/insights-service/internal/discovery"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/insights"
	"github.com/jmoiron/sqlx"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

// SetupServices wires insights-service dependencies and returns a configured
// gRPC server. The generation flow reads plan focus + checklist from
// plan-service over gRPC and produces suggestions through the OpenAI-backed
// ContentGenerator.
//
// Two generators are injected rather than one: a generator IS its system prompt,
// and the checklist and video-search prompts are different. This mirrors the
// api-gateway, which constructed a separate insights service around each.
//
// redisClient is injected as the narrow insights.Cache interface (DIP: the
// service depends on the abstraction, the composition root supplies the
// concrete). It is wired but not yet used — cache-backed logic comes later.
func SetupServices(db *sqlx.DB, _ *amqp.Channel, registry discovery.Registry, redisClient *redis.Client) (*grpc.Server, error) {
	planClient := insights.NewPlanClient(registry)

	checklistGen := ai.NewChecklistGen()
	searchTermGen := ai.NewSearchTermGen()

	youtubeFinder, err := videodiscovery.NewYoutubeVideoFinder()
	if err != nil {
		return nil, fmt.Errorf("insights: init youtube video finder: %w", err)
	}
	videoFinder := insights.NewDiscoveryVideoFinder(youtubeFinder)

	repo := insights.NewRepository(db)
	service := insights.NewService(planClient, checklistGen, searchTermGen, videoFinder, redisClient, repo, db)
	handler := insights.NewHandler(service)

	grpcServer := grpc.NewServer()
	pb.RegisterInsightsServiceServer(grpcServer, handler)

	slog.Info("insights-service initialized successfully")
	return grpcServer, nil
}
