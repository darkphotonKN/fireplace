package insights

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"

	"google.golang.org/protobuf/proto"
)

type Consumer struct {
	service ConsumerService
	channel *amqp.Channel
}

// ConsumerService is the slice of the service the consumer needs.
type ConsumerService interface {
	Create(ctx context.Context, param PlanCreatedParam) error
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
			// parse into protobuf to attempt to match contract
			var planCreated pb.CreatePlanRequest
			err := proto.Unmarshal(msg.Body, &planCreated)
			if err != nil {
				// unmarshal error, could be local bug or schema evolution, deploy mismatch, send to DLQ
				// for manual inspection and replay later
				slog.ErrorContext(ctx, "Unmarshal error", "error", err)
				msg.Nack(false, false)
				continue
			}

			userIdUUID, err := uuid.Parse(planCreated.UserId)
			if err != nil {
				msg.Nack(false, false)
				slog.ErrorContext(ctx, "UUID parse error", "error", err)
				continue
			}

			planIDUUID, err := uuid.Parse(planCreated.UserId) // TODO: need to update tomorrow, missing field
			if err != nil {
				msg.Nack(false, false)
				slog.ErrorContext(ctx, "UUID parse error", "error", err)
				continue
			}

			err = c.service.Create(ctx, PlanCreatedParam{
				EventID: userIdUUID, // NOTE: WIP need to use the right value
				UserID:  userIdUUID, // NOTE: WIP need to use the right value
				PlanID:  planIDUUID, // NOTE: WIP need to use the right value
			})
			if err != nil {
				c.errorHandler(ctx, err, msg)
				continue
			}

		default:
			slog.Debug("plan-service: unhandled routing key", "routing_key", msg.RoutingKey)
			msg.Ack(false)
		}
	}
}

func (c *Consumer) errorHandler(ctx context.Context, err error, msg amqp.Delivery) {
	// we handle our own senintel erorr here to differentiate between a db plain duplicate
	// error with when we actually consider that duplicate a duplicate attempt of an
	// already processed message
	if errors.Is(err, ErrEventAlreadyProcessed) {
		// duplicate, ack success and drop event
		msg.Ack(false)

		// includes all the wrapped context from the layers
		slog.DebugContext(ctx, "duplicate error, acking to remove", "error", err)
		return
	}

	// transient error, not malformed or duplicate, NACK requeue
	if errors.Is(err, commonconstants.ErrTransient) {
		msg.Nack(false, true)

		slog.ErrorContext(ctx, "transient error, event NACKED and requeuing", "error", err)
		return
	}

	// unexpected error, log and DLQ to be safe
	if errors.Is(err, ErrUnexpectedError) {
		msg.Nack(false, false) // dlq

		slog.ErrorContext(ctx, "Unexpected error, requeuing", "error", err)
		return
	}

	// default, send to DLQ and requeue for unexpected situation
	msg.Nack(false, false)
	slog.ErrorContext(ctx, "Unexpected error, requeuing", "error", err)
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
