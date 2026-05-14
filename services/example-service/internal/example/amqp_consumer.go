package example

import (
	"log/slog"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer scaffolds AMQP consumption for example-service. example-service is
// decorative, so the consumer is a no-op loop — its purpose is to demonstrate
// the consumer wiring (exchange + queue + bindings + per-message switch) that
// real services will replicate.
type Consumer struct {
	service ConsumerService
	channel *amqp.Channel
}

// ConsumerService is the slice of the service the consumer needs.
// Empty for now — real consumers extend this with their handler dependencies.
type ConsumerService interface{}

func NewConsumer(service ConsumerService, ch *amqp.Channel) *Consumer {
	return &Consumer{service: service, channel: ch}
}

func (c *Consumer) Listen() {
	go c.consumeEvents()
	slog.Info("example-service consumer listening for events...")
}

func (c *Consumer) consumeEvents() {
	msgs, err := c.channel.Consume(
		commonconstants.ExampleServiceEventsQueue,
		"",    // consumer
		false, // autoAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,
	)
	if err != nil {
		slog.Error("example-service: failed to register consumer", "error", err)
		return
	}

	for msg := range msgs {
		slog.Debug("example-service: received event (decorative, no handler)",
			"routing_key", msg.RoutingKey,
		)
		msg.Ack(false)
	}
}

// SetupAMQPInfrastructure declares example.events exchange + a per-service
// inbound queue. No bindings — example-service consumes nothing.
func SetupAMQPInfrastructure(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
		commonconstants.ExampleEventsExchange, "topic", true, false, false, false, nil,
	); err != nil {
		return err
	}

	if _, err := ch.QueueDeclare(
		commonconstants.ExampleServiceEventsQueue,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	); err != nil {
		return err
	}

	slog.Info("example-service AMQP infrastructure setup complete",
		"exchange", commonconstants.ExampleEventsExchange,
		"queue", commonconstants.ExampleServiceEventsQueue,
	)
	return nil
}
