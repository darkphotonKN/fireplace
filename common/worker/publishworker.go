package commonworker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/darkphotonKN/fireplace/common/broker"
	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	commonmodel "github.com/darkphotonKN/fireplace/common/model"
	"github.com/google/uuid"
)

// Polls local outbox table and attempts to
// publish at a set interval
type PublishWorker struct {
	eventDrainer EventDrainer
	publisher    broker.Publisher
	interval     time.Duration
}

type EventDrainer interface {
	GetUnpublished(ctx context.Context) ([]*commonmodel.OutboxEvent, error)
	MarkUnpublished(ctx context.Context, ids []uuid.UUID) error
}

func NewPublishWorker(eventDrainer EventDrainer, publisher broker.Publisher, interval time.Duration) *PublishWorker {
	return &PublishWorker{
		eventDrainer: eventDrainer,
		publisher:    publisher,
		interval:     interval,
	}
}

// synchornous worker loop, let caller decide on concurrency at the call site
// by default this should be initiated in main.go however, after the event drain
// injections and publishers have been initialized
func (w *PublishWorker) Run(ctx context.Context) error {
	timer := time.NewTicker(w.interval)

	for {
		select {
		case <-timer.C:
			return nil

		case <-ctx.Done():
			// TODO: wip

		}
	}
}

// drains the outbox events, pulls out all the unpublished
// ones and attempts to publish them, skipping ones that
// error and marking the ones that complete. retry marking completion
// if errored, need to make sure published ones are not processed again
// if possible (though downstream idempotency design will dedup, this is
// to improve effeciency)
func (w *PublishWorker) Drain(ctx context.Context) error {
	// drain unpublished events
	events, err := w.eventDrainer.GetUnpublished(ctx)

	if err != nil {
		return fmt.Errorf("worker draining unpublished events: %w", err)
	}

	// attempt to batch publish
	successfulIds := make([]uuid.UUID, 0, len(events)) // max length as many as events fetched

	for _, event := range events {
		err := w.publisher.PublishWithContext(ctx, event.Exchange, event.RoutingKey,
			commonbroker.Message{
				ContentType:  "application/protobuf",
				Body:         event.Payload,
				DeliveryMode: commonbroker.Persistent,
			})

		if err != nil {
			// couldn't publish, leave for next worker, just slog for tracking
			slog.Warn("worker publish attempt failed for event %s : %w", event.ID, err)
		}
	}

	// mark finished ones as published
	if len(successfulIds) == 0 {
		// this worker's job is done, no successful publishes, wait for next interval
		return nil
	}

	err = w.eventDrainer.MarkUnpublished(ctx, successfulIds)

	if err != nil {
		return fmt.Errorf("worker couldn't mark published: %w", err)
	}
	return nil
}
