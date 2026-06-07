package orchestrator

import (
	"context"
	"log/slog"
)

// PublishOrchestratorEvent is a STUB demonstrating the publish pattern. Real
// impl: marshal a protobuf event, then call s.publishCh.PublishWithContext with
// OrchestratorEventsExchange + a routing key. Kept here as a template so the
// shape of "what a publish method looks like" is visible alongside the
// handler/service.
func (s *service) PublishOrchestratorEvent(_ context.Context, payload string) {
	// TODO: real implementation — marshal a proto event and publish it.
	slog.Debug("publish orchestrator.event stubbed", "payload", payload)
}
