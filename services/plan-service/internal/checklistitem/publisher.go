package checklistitem

import (
	"context"
	"log/slog"

	eventspb "github.com/darkphotonKN/fireplace/common/api/proto/events"
	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *service) PublishItemCompleted(ctx context.Context, item *Item) {
	body, err := proto.Marshal(&eventspb.ChecklistItemCompletedEvent{
		Id:          item.ID.String(),
		PlanId:      item.PlanID.String(),
		CompletedAt: timestamppb.Now(),
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to marshal checklist_item.completed", "err", err, "item_id", item.ID)
		return
	}
	if err := s.publishCh.PublishWithContext(ctx,
		commonconstants.PlanEventsExchange,
		commonconstants.ChecklistItemCompleted,
		commonbroker.Message{
			ContentType:  "application/protobuf",
			Body:         body,
			DeliveryMode: commonbroker.Persistent,
		},
	); err != nil {
		slog.ErrorContext(ctx, "failed to publish checklist_item.completed", "err", err, "item_id", item.ID)
	}
}

func (s *service) PublishItemUncompleted(ctx context.Context, item *Item) {
	body, err := proto.Marshal(&eventspb.ChecklistItemUncompletedEvent{
		Id:            item.ID.String(),
		PlanId:        item.PlanID.String(),
		UncompletedAt: timestamppb.Now(),
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to marshal checklist_item.uncompleted", "err", err, "item_id", item.ID)
		return
	}
	if err := s.publishCh.PublishWithContext(ctx,
		commonconstants.PlanEventsExchange,
		commonconstants.ChecklistItemUncompleted,
		commonbroker.Message{
			ContentType:  "application/protobuf",
			Body:         body,
			DeliveryMode: commonbroker.Persistent,
		},
	); err != nil {
		slog.ErrorContext(ctx, "failed to publish checklist_item.uncompleted", "err", err, "item_id", item.ID)
	}
}
