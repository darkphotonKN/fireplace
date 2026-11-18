package config

import (
	"context"
	"log/slog"
	"sync"
	"time"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	"github.com/darkphotonKN/fireplace/common/discovery"
	commonworker "github.com/darkphotonKN/fireplace/common/worker"
	"github.com/darkphotonKN/fireplace/services/plan-service/internal/checklistitem"
	"github.com/darkphotonKN/fireplace/services/plan-service/internal/notecomment"
	"github.com/darkphotonKN/fireplace/services/plan-service/internal/outbox"
	"github.com/darkphotonKN/fireplace/services/plan-service/internal/plan"
	"github.com/jmoiron/sqlx"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

// SetupServices wires plan-service dependencies and returns a configured gRPC
// server with both PlanService and ChecklistService registered.
func SetupServices(workerCtx context.Context, db *sqlx.DB, amqpChannel *amqp.Channel, _ discovery.Registry, wg *sync.WaitGroup) *grpc.Server {
	publisher := commonbroker.NewAmqpPublisher(amqpChannel)

	outboxRepo := outbox.NewRepository(db)
	outboxService := outbox.NewService(outboxRepo)

	// --- plans ---
	planRepo := plan.NewRepository(db)
	planService := plan.NewService(planRepo, outboxService, db)
	planHandler := plan.NewHandler(planService)

	// --- checklist items ---
	checklistRepo := checklistitem.NewRepository(db)
	checklistService := checklistitem.NewService(checklistRepo, publisher)
	checklistHandler := checklistitem.NewHandler(checklistService)

	// --- note comments (FS-65B3N), served on ChecklistService beside items ---
	commentRepo := notecomment.NewRepository(db)
	comments := notecomment.NewHandler(notecomment.NewService(commentRepo))

	// setup publish worker
	worker := commonworker.NewPublishWorker(outboxService, publisher, db, time.Minute*2)

	// for blocking cleanup
	wg.Add(1)
	go func() {
		defer wg.Done()

		err := worker.Run(workerCtx)
		if err != nil {
			slog.ErrorContext(workerCtx, "error from worker during SetupService call",
				"error", err,
			)
		}
	}()

	// AMQP infra: own plan.events exchange + scaffold consumer for auth.events.
	if err := plan.SetupAMQPInfrastructure(amqpChannel); err != nil {
		slog.Error("Failed to setup AMQP infrastructure", "error", err)
	}
	consumer := plan.NewConsumer(planService, amqpChannel)
	consumer.Listen()

	grpcServer := grpc.NewServer()
	pb.RegisterPlanServiceServer(grpcServer, planHandler)
	pb.RegisterChecklistServiceServer(grpcServer, checklistServer{checklistHandler, comments})

	slog.Info("plan-service initialized successfully")
	return grpcServer
}

// checklistServer is ChecklistService: item RPCs from checklistitem, comment
// RPCs from notecomment. checklistitem's handler embeds the Unimplemented
// server and notecomment's does not, so each comment method resolves to
// notecomment's (the shallower one) and nothing else is ambiguous.
type checklistServer struct {
	*itemHandler
	*commentHandler
}

// Aliases give the two embedded handlers distinct field names.
type (
	itemHandler    = checklistitem.Handler
	commentHandler = notecomment.Handler
)
