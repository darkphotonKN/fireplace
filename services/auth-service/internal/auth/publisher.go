package auth

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// Publishers are STUBS. Real impl: marshal a protobuf event from
// common/api/proto/events, then call s.publishCh.PublishWithContext with
// AuthEventsExchange + the appropriate routing key from commonconstants.
// See ember/user-service/internal/user/publisher.go for the reference shape.

func (s *service) PublishUserCreated(_ context.Context, u *User) {
	// TODO: implement in Phase 3 when SignUp lands.
	slog.Debug("publish user.created stubbed", "user_id", u.ID)
}

func (s *service) PublishUserDeleted(_ context.Context, id uuid.UUID) {
	// TODO: implement in Phase 3 when DeleteUser lands.
	slog.Debug("publish user.deleted stubbed", "user_id", id)
}
