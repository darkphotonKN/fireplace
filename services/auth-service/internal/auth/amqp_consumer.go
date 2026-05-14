package auth

import (
	"log/slog"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer scaffolds the AMQP consumption for auth-service. auth-service is
// currently upstream of all domain events (only publishes), so the consumer
// is a no-op stub — the queue + exchange are declared so other services can
// bind into auth.events, and this consumer is ready to handle inbound events
// if/when cross-service flows demand it.
type Consumer struct {
	service ConsumerService
	channel *amqp.Channel
}

// ConsumerService is the slice of the service the consumer needs.
// Empty for now — extend when wiring real handlers.
type ConsumerService interface{}

func NewConsumer(service ConsumerService, ch *amqp.Channel) *Consumer {
	return &Consumer{service: service, channel: ch}
}

func (c *Consumer) Listen() {
	go c.consumeEvents()
	slog.Info("auth-service consumer listening for events...")
}

func (c *Consumer) consumeEvents() {
	msgs, err := c.channel.Consume(
		commonconstants.AuthServiceEventsQueue,
		"",    // consumer
		false, // autoAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,
	)
	if err != nil {
		slog.Error("auth-service: failed to register consumer", "error", err)
		return
	}

	for msg := range msgs {
		slog.Debug("auth-service: received event (no handler bound)",
			"routing_key", msg.RoutingKey,
		)
		msg.Ack(false)
	}
}

// SetupAMQPInfrastructure declares the auth.events exchange + a per-service
// inbound queue. Bindings are empty for now — auth-service consumes nothing.
func SetupAMQPInfrastructure(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
		commonconstants.AuthEventsExchange, "topic", true, false, false, false, nil,
	); err != nil {
		return err
	}

	if _, err := ch.QueueDeclare(
		commonconstants.AuthServiceEventsQueue,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	); err != nil {
		slog.Error("auth-service: failed to declare queue", "error", err)
		return err
	}

	slog.Info("auth-service AMQP infrastructure setup complete",
		"exchange", commonconstants.AuthEventsExchange,
		"queue", commonconstants.AuthServiceEventsQueue,
	)
	return nil
}
