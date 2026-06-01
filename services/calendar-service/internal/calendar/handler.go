package calendar

import (
	"context"
	"fmt"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/calendar"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"github.com/google/uuid"
)

type CalendarService interface {
	GetCalendar(ctx context.Context, planID, userID uuid.UUID, view, date string) (*GetCalendarOutput, error)
}

type Handler struct {
	pb.UnimplementedCalendarServiceServer
	service CalendarService
}

func NewHandler(s CalendarService) *Handler {
	return &Handler{service: s}
}

// badUUID builds an InvalidArgument-class domain error for a malformed id so it
// flows through the shared mapper like any other domain error.
func badUUID(field, value string) error {
	return fmt.Errorf("%w: %s %q", commonconstants.ErrUUIDCouldNotBeParsed, field, value)
}

func (h *Handler) GetCalendar(ctx context.Context, req *pb.GetCalendarRequest) (*pb.GetCalendarResponse, error) {
	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "calendar: get calendar", badUUID("plan_id", req.PlanId))
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "calendar: get calendar", badUUID("user_id", req.UserId))
	}

	out, err := h.service.GetCalendar(ctx, planID, userID, req.View, req.Date)
	if err != nil {
		// commongrpc.Status preserves any gRPC status propagated from
		// plan-service so the gateway still maps it correctly.
		return nil, commongrpc.Fail(ctx, "calendar: get calendar", err)
	}

	items := make([]*pb.CalendarItem, 0, len(out.Items))
	for _, it := range out.Items {
		items = append(items, &pb.CalendarItem{
			Id:          it.ID.String(),
			Description: it.Description,
			Scope:       it.Scope,
			Done:        it.Done,
			StartDate:   it.StartDate,
			DueDate:     it.DueDate,
		})
	}
	return &pb.GetCalendarResponse{
		PlanId:      out.PlanID.String(),
		View:        out.View,
		WindowStart: out.WindowStart,
		WindowEnd:   out.WindowEnd,
		Items:       items,
	}, nil
}
