package commonmodel

import (
	"time"

	"github.com/google/uuid"
)

// Outbox events for publishing, same shape for all
// micorservices.
type OutboxEvent struct {
	ID          uuid.UUID  `db:"id"`
	RoutingKey  string     `db:"routing_key"`
	Exchange    string     `db:"exchange"`
	Payload     []byte     `db:"payload"`
	CreatedAt   time.Time  `db:"created_at"`
	PublishedAt *time.Time `db:"published_at"`
}
