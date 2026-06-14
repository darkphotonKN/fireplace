package commonmodel

import (
	"time"

	"github.com/google/uuid"
)

// Outbox events for publishing, same shape for all
// micorservices.
type OutboxEvent struct {
	ID          uuid.UUID
	RoutingKey  string
	Exchange    string
	Payload     []byte
	CreatedAt   time.Time
	PublishedAt *time.Time
}
