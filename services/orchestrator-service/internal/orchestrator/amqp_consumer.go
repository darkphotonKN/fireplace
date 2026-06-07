package orchestrator

import (
	"log/slog"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer scaffolds AMQP consumption for orchestrator-service. The service is
// injected (ConsumerService) so real event handlers can call into business
// logic. For now it is a no-op loop demonstrating the consumer wiring
// (exchange + queue + bindings + per-message switch) that real orchestration
// flows will replicate.
type Consumer struct {
	service ConsumerService
	channel *amqp.Channel
}

// ConsumerService is the slice of the service the consumer needs. Empty for now
// — extend it with the methods event handlers must call, and they are satisfied
// by the *service injected in config.SetupServices.
type ConsumerService interface{}

func NewConsumer(service ConsumerService, ch *amqp.Channel) *Consumer {
	return &Consumer{service: service, channel: ch}
}

func (c *Consumer) Listen() {
	go c.consumeEvents()
	slog.Info("orchestrator-service consumer listening for events...")
}

func (c *Consumer) consumeEvents() {
	msgs, err := c.channel.Consume(
		commonconstants.OrchestratorServiceEventsQueue,
		"",    // consumer
		false, // autoAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,
	)
	if err != nil {
		slog.Error("orchestrator-service: failed to register consumer", "error", err)
		return
	}

	for msg := range msgs {
		slog.Debug("orchestrator-service: received event (scaffold, no handler)",
			"routing_key", msg.RoutingKey,
		)
		msg.Ack(false)
	}
}

// SetupAMQPInfrastructure declares orchestrator.events exchange + a per-service
// inbound queue. No bindings yet — orchestrator-service consumes nothing until
// real cross-service flows are wired (bind the queue to the exchanges whose
// events it must react to).
func SetupAMQPInfrastructure(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
		commonconstants.OrchestratorEventsExchange, "topic", true, false, false, false, nil,
	); err != nil {
		return err
	}

	if _, err := ch.QueueDeclare(
		commonconstants.OrchestratorServiceEventsQueue,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	); err != nil {
		return err
	}

	slog.Info("orchestrator-service AMQP infrastructure setup complete",
		"exchange", commonconstants.OrchestratorEventsExchange,
		"queue", commonconstants.OrchestratorServiceEventsQueue,
	)
	return nil
}
