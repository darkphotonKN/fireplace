package commonworker

import (
	"context"
	"time"

	"github.com/darkphotonKN/fireplace/common/broker"
	commonmodel "github.com/darkphotonKN/fireplace/common/model"
)

// Polls local outbox table and attempts to
// publish at a set interval
type PublishWorker struct {
	eventDrainer EventDrainer
	publisher    broker.Publisher
	interval     time.Duration
}

type EventDrainer interface {
	DrainUnpublishedEvents(ctx context.Context) ([]*commonmodel.OutboxEvent, error)
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
