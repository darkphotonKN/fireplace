package auth

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

func (s *service) PublishUserCreated(ctx context.Context, u *User) {
	body, err := proto.Marshal(&eventspb.UserCreatedEvent{
		Id:        u.ID.String(),
		Email:     u.Email,
		Name:      u.Name,
		CreatedAt: timestamppb.New(u.CreatedAt),
	})
	if err != nil {
		slog.Error("failed to marshal user.created event", "error", err, "user_id", u.ID)
		return
	}
	if err := s.publishCh.PublishWithContext(ctx,
		commonconstants.AuthEventsExchange,
		commonconstants.UserCreated,
		commonbroker.Message{
			ContentType:  "application/protobuf",
			Body:         body,
			DeliveryMode: commonbroker.Persistent,
		},
	); err != nil {
		slog.Error("failed to publish user.created", "error", err, "user_id", u.ID)
	}
}

func (s *service) PublishUserUpdated(_ context.Context, u *User) {
	// No proto event defined yet; placeholder until consumers actually need it.
	slog.Debug("publish user.updated (no consumers wired)", "user_id", u.ID)
}

func (s *service) PublishUserDeleted(ctx context.Context, id uuid.UUID) {
	body, err := proto.Marshal(&eventspb.UserDeletedEvent{
		Id:        id.String(),
		DeletedAt: timestamppb.Now(),
	})
	if err != nil {
		slog.Error("failed to marshal user.deleted event", "error", err, "user_id", id)
		return
	}
	if err := s.publishCh.PublishWithContext(ctx,
		commonconstants.AuthEventsExchange,
		commonconstants.UserDeleted,
		commonbroker.Message{
			ContentType:  "application/protobuf",
			Body:         body,
			DeliveryMode: commonbroker.Persistent,
		},
	); err != nil {
		slog.Error("failed to publish user.deleted", "error", err, "user_id", id)
	}
}
