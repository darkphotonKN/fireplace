package insights

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
)

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

func (c *Consumer) consumePlanEvents() {
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

	ctx := context.Background()

	for msg := range msgs {
		switch msg.RoutingKey {
		case commonconstants.PlanCreated:
		// generate default insights

		default:
			slog.Debug("plan-service: unhandled routing key", "routing_key", msg.RoutingKey)
			msg.Ack(false)
		}
	}
}

// SetupAMQPInfrastructure declares plan.events (this service's own exchange)
// + a queue bound to auth.events for user.deleted events.
func SetupAMQPInfrastructure(ch *amqp.Channel) error {
	for _, ex := range []string{commonconstants.PlanEventsExchange} {
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
		commonconstants.InsightsPlanEventsQueue,
		commonconstants.PlanCreated,
		commonconstants.PlanEventsExchange,
		false, nil,
	); err != nil {
		return err
	}

	slog.Info("plan-service AMQP infrastructure setup complete",
		"queue", commonconstants.InsightsPlanEventsQueue,
	)
	return nil
}
