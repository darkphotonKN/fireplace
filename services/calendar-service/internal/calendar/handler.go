package calendar

import (
	"context"
	"errors"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/calendar"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (h *Handler) GetCalendar(ctx context.Context, req *pb.GetCalendarRequest) (*pb.GetCalendarResponse, error) {
	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid plan_id: %v", err)
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	out, err := h.service.GetCalendar(ctx, planID, userID, req.View, req.Date)
	if err != nil {
		return nil, mapError(err)
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

func mapError(err error) error {
	switch {
	case errors.Is(err, commonconstants.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, commonconstants.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, commonconstants.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		// Propagate any underlying gRPC status from plan-service through
		// untouched so the gateway can map it correctly.
		if _, ok := status.FromError(err); ok {
			return err
		}
		return status.Error(codes.Internal, err.Error())
	}
}
