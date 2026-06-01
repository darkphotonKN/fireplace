package plan

import (
	"context"
	"log/slog"

	eventspb "github.com/darkphotonKN/fireplace/common/api/proto/events"
	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *service) PublishPlanCreated(ctx context.Context, p *Plan) {
	body, err := proto.Marshal(&eventspb.PlanCreatedEvent{
		Id:        p.ID.String(),
		UserId:    p.UserID.String(),
		Name:      p.Name,
		CreatedAt: timestamppb.New(p.CreatedAt),
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to marshal plan.created event", "err", err, "plan_id", p.ID)
		return
	}
	if err := s.publishCh.PublishWithContext(ctx,
		commonconstants.PlanEventsExchange,
		commonconstants.PlanCreated,
		commonbroker.Message{
			ContentType:  "application/protobuf",
			Body:         body,
			DeliveryMode: commonbroker.Persistent,
		},
	); err != nil {
		slog.ErrorContext(ctx, "failed to publish plan.created", "err", err, "plan_id", p.ID)
	}
}

func (s *service) PublishPlanDeleted(ctx context.Context, planID, userID uuid.UUID) {
	body, err := proto.Marshal(&eventspb.PlanDeletedEvent{
		Id:        planID.String(),
		UserId:    userID.String(),
		DeletedAt: timestamppb.Now(),
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to marshal plan.deleted event", "err", err, "plan_id", planID)
		return
	}
	if err := s.publishCh.PublishWithContext(ctx,
		commonconstants.PlanEventsExchange,
		commonconstants.PlanDeleted,
		commonbroker.Message{
			ContentType:  "application/protobuf",
			Body:         body,
			DeliveryMode: commonbroker.Persistent,
		},
	); err != nil {
		slog.ErrorContext(ctx, "failed to publish plan.deleted", "err", err, "plan_id", planID)
	}
}
