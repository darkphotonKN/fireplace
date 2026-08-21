package broker

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

/**
* Adapter for rabbitmq version of our publisher.
**/
type AmqpPublisher struct {
	ch *amqp.Channel
}

func NewAmqpPublisher(ch *amqp.Channel) *AmqpPublisher {
	return &AmqpPublisher{ch: ch}
}

// toPublishing maps our transport-neutral Message onto amqp's own. Split out of
// PublishWithContext so the mapping is testable without a live broker — the
// channel is a concrete *amqp.Channel, so anything left inline here can only be
// exercised against a running RabbitMQ.
func toPublishing(msg Message) amqp.Publishing {
	return amqp.Publishing{
		MessageId:     msg.MessageId,
		ContentType:   msg.ContentType,
		Body:          msg.Body,
		DeliveryMode:  msg.DeliveryMode,
		CorrelationId: msg.CorrelationId,
		Headers:       msg.Headers,
	}
}

func (p *AmqpPublisher) PublishWithContext(_ context.Context, exchange, key string, msg Message) error {
	return p.ch.Publish(exchange, key, false, false, toPublishing(msg))
}
