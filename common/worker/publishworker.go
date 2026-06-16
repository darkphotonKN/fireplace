package commonworker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/darkphotonKN/fireplace/common/broker"
	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	commonmodel "github.com/darkphotonKN/fireplace/common/model"
	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Polls local outbox table and attempts to
// publish at a set interval
type PublishWorker struct {
	eventDrainer EventDrainer
	publisher    broker.Publisher
	db           *sqlx.DB
	interval     time.Duration
}

type EventDrainer interface {
	GetUnpublished(ctx context.Context, tx *sqlx.Tx) ([]*commonmodel.OutboxEvent, error)
	MarkPublished(ctx context.Context, tx *sqlx.Tx, ids []uuid.UUID) error
}

func NewPublishWorker(eventDrainer EventDrainer, publisher broker.Publisher, db *sqlx.DB, interval time.Duration) *PublishWorker {
	return &PublishWorker{
		eventDrainer: eventDrainer,
		publisher:    publisher,
		db:           db,
		interval:     interval,
	}
}

// synchornous worker loop, let caller decide on concurrency at the call site
// by default this should be initiated in main.go however, after the event drain
// injections and publishers have been initialized
func (w *PublishWorker) Run(ctx context.Context) error {
	timer := time.NewTicker(w.interval)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			// dont return on err, just log and continue (next worker or cycle will pick it up)
			err := w.Drain(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "worker attempted to drain outbox events",
					"error", err,
				)
			}

		case <-ctx.Done():
			// start our timer to allow for final drain (previous tick might be
			// processing)
			// don't tie this to parent or it cancels with sigterm
			cleanUpCtx, cleanUpCancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cleanUpCancel()

			// attempt one last drain, automatically cancel propogated after 5 seconds to
			// not hold up all process after 5 seconds
			err := w.Drain(cleanUpCtx)
			if err != nil {
				slog.ErrorContext(ctx, "worker attempted to drain outbox events",
					"error", err,
				)
			}

			return nil
			// no default, that runs when no case matches
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
	return commonhelpers.ExecTx(ctx, w.db, func(tx *sqlx.Tx) error {
		// drain unpublished events
		events, err := w.eventDrainer.GetUnpublished(ctx, tx)

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
				slog.Warn("worker publish attempt failed for event",
					"event_id", event.ID,
					"error", err,
				)
				continue
			}

			successfulIds = append(successfulIds, event.ID)
		}

		// mark finished ones as published
		if len(successfulIds) == 0 {
			// this worker's job is done, no successful publishes, wait for next interval
			return nil
		}

		err = w.eventDrainer.MarkPublished(ctx, tx, successfulIds)

		if err != nil {
			return fmt.Errorf("worker couldn't mark published: %w", err)
		}
		return nil
	})
}
