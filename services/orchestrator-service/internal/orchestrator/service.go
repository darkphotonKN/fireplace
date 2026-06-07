package orchestrator

import (
	"context"
	"fmt"

	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
)

// service holds the orchestrator's dependencies. The AMQP publisher is injected
// so the service can emit orchestrator.events; more collaborators (e.g. gRPC
// clients to downstream services) get added here as real orchestration lands.
type service struct {
	publishCh commonbroker.Publisher
}

func NewService(publishCh commonbroker.Publisher) *service {
	return &service{publishCh: publishCh}
}

// Ping is the placeholder business method behind the Ping RPC.
func (s *service) Ping(_ context.Context, msg string) string {
	return fmt.Sprintf("pong: %s", msg)
}
