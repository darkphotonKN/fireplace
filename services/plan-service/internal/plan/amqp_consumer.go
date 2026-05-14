package plan

import (
	"context"
	"log/slog"

	eventspb "github.com/darkphotonKN/fireplace/common/api/proto/events"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
)

// Consumer wires plan-service's AMQP consumption. The queue is bound to the
// auth.events exchange so user.deleted events drive cascade-delete of plans
// + plan-scoped data (resources, checklist_items via FK CASCADE).
type Consumer struct {
	service ConsumerService
	channel *amqp.Channel
}

// ConsumerService is the slice of the service the consumer needs.
type ConsumerService interface {
	CascadeDeleteForUser(ctx context.Context, userID uuid.UUID) error
}

func NewConsumer(service ConsumerService, ch *amqp.Channel) *Consumer {
	return &Consumer{service: service, channel: ch}
}

func (c *Consumer) Listen() {
	go c.consumeEvents()
	slog.Info("plan-service consumer listening for events...")
}

func (c *Consumer) consumeEvents() {
	msgs, err := c.channel.Consume(
		commonconstants.PlanServiceEventsQueue,
		"",    // consumer
		false, // autoAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,
	)
	if err != nil {
		slog.Error("plan-service: failed to register consumer", "error", err)
		return
	}

	for msg := range msgs {
		ctx := context.Background()
		switch msg.RoutingKey {
		case commonconstants.UserDeleted:
			c.handleUserDeleted(ctx, msg)
		default:
			slog.Debug("plan-service: unhandled routing key", "routing_key", msg.RoutingKey)
			msg.Ack(false)
		}
	}
}

// handleUserDeleted unmarshals the auth event and cascades the delete to
// every plan owned by the user. Failures requeue once via Nack(false, true);
// duplicate processing is acceptable here because plan deletion is idempotent
// at the "row already gone" boundary.
func (c *Consumer) handleUserDeleted(ctx context.Context, msg amqp.Delivery) {
	var event eventspb.UserDeletedEvent
	if err := proto.Unmarshal(msg.Body, &event); err != nil {
		slog.Error("plan-service: failed to unmarshal user.deleted", "error", err)
		msg.Ack(false) // unparseable — don't requeue, just drop
		return
	}
	userID, err := uuid.Parse(event.Id)
	if err != nil {
		slog.Error("plan-service: invalid user_id in user.deleted", "error", err, "raw_id", event.Id)
		msg.Ack(false)
		return
	}
	if err := c.service.CascadeDeleteForUser(ctx, userID); err != nil {
		slog.Error("plan-service: cascade delete failed", "error", err, "user_id", userID)
		_ = msg.Nack(false, true) // requeue once
		return
	}
	slog.Info("plan-service: cascade-deleted plans for user", "user_id", userID)
	msg.Ack(false)
}

// SetupAMQPInfrastructure declares plan.events (this service's own exchange)
// + a queue bound to auth.events for user.deleted events.
func SetupAMQPInfrastructure(ch *amqp.Channel) error {
	for _, ex := range []string{commonconstants.PlanEventsExchange, commonconstants.AuthEventsExchange} {
		if err := ch.ExchangeDeclare(ex, "topic", true, false, false, false, nil); err != nil {
			return err
		}
	}

	if _, err := ch.QueueDeclare(
		commonconstants.PlanServiceEventsQueue,
		true, false, false, false, nil,
	); err != nil {
		return err
	}

	if err := ch.QueueBind(
		commonconstants.PlanServiceEventsQueue,
		commonconstants.UserDeleted,
		commonconstants.AuthEventsExchange,
		false, nil,
	); err != nil {
		return err
	}

	slog.Info("plan-service AMQP infrastructure setup complete",
		"queue", commonconstants.PlanServiceEventsQueue,
	)
	return nil
}
