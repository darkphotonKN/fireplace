package config

import (
	"fmt"
	"log/slog"
	"os"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/insights"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/ai"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/dayplan"
	videodiscovery "github.com/darkphotonKN/fireplace/services/insights-service/internal/discovery"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/insights"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/rag"
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

	// One meter for every generator: the global daily token budget over the
	// usage ledger (ADR-0019 clauses 3–4, FS-02JBZ R31–R35). Every generator
	// below is built on it, so every model call is checked and recorded.
	ledger := guided.NewRepository(db)
	budget := ai.DailyTokenBudget()
	meter := ai.NewMeter(ledger, budget)
	slog.Info("insights: LLM daily token budget", "tokens", budget)

	checklistGen := ai.NewChecklistGen(meter)
	searchTermGen := ai.NewSearchTermGen(meter)

	youtubeFinder, err := videodiscovery.NewYoutubeVideoFinder()
	if err != nil {
		return nil, fmt.Errorf("insights: init youtube video finder: %w", err)
	}
	videoFinder := insights.NewDiscoveryVideoFinder(youtubeFinder)

	repo := insights.NewRepository(db)
	service := insights.NewService(planClient, checklistGen, searchTermGen, videoFinder, redisClient, repo, db)

	// The day plan (FS-02JBZ) has its own structured generator and config
	// (R6). Its redraws are capped by the per (plan, user, date) refresh
	// counter (R17, R20). Retrieval runs on the stub until the human-owned
	// Retriever lands (I-DF4ZB-14); the rag repository already tells the
	// snippet source whether a user has personal memory. One repository holds
	// day plans, day ticks and the revision ladder.
	memory := rag.NewRepository(db)
	refreshes := guided.NewRefreshCounters(db)
	dayPlanGen := ai.NewStructuredGenerator(ai.DayPlanConfig(), os.Getenv("OPENAI_API_KEY"), meter)
	dayPlanRepo := dayplan.NewRepository(db)
	dayPlanService := dayplan.NewService(planClient, dayPlanGen,
		guided.NewSnippetSource(guided.NoRetrieval{}, memory), dayPlanRepo, dayPlanRepo, refreshes)

	handler := insights.NewHandler(service, dayPlanService)

	grpcServer := grpc.NewServer()
	pb.RegisterInsightsServiceServer(grpcServer, handler)

	slog.Info("insights-service initialized successfully")
	return grpcServer, nil
}
