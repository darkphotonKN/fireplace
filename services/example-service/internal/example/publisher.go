package example

import (
	"context"
	"log/slog"
)

// PublishExampleEvent is a STUB demonstrating the publish pattern. Real impl:
// marshal a protobuf event, then call s.publishCh.PublishWithContext with
// ExampleEventsExchange + a routing key. Kept here as a template so the shape
// of "what a publish method looks like" is visible alongside the handler/service.
func (s *service) PublishExampleEvent(_ context.Context, payload string) {
	// TODO: real implementation — see ember/user-service/internal/user/publisher.go
	slog.Debug("publish example.event stubbed", "payload", payload)
}
