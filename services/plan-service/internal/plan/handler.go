package plan

import (
	"context"
	"errors"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service interface {
	Create(ctx context.Context, in *CreatePlanInput) (*Plan, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Plan, error)
	Update(ctx context.Context, in *UpdatePlanInput) (*Plan, error)
	ToggleDailyReset(ctx context.Context, id, userID uuid.UUID) (*Plan, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*Plan, error)
	ListShared(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Plan, error)
	Search(ctx context.Context, in SearchInput) ([]*SearchResult, error)
	AssertOwnership(ctx context.Context, planID, userID uuid.UUID) error
}

type Handler struct {
	pb.UnimplementedPlanServiceServer
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) CreatePlan(ctx context.Context, req *pb.CreatePlanRequest) (*pb.Plan, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	// Caller can override the per-plan-type daily-reset default by setting
	// daily_reset in the request. The proto field is a plain bool so we always
	// pass it through; the service treats explicit-true differently from
	// "not provided" only when in.DailyReset is nil. To keep the contract
	// simple, treat every CreatePlan call as explicit.
	dr := req.DailyReset
	p, err := h.service.Create(ctx, &CreatePlanInput{
		UserID:      userID,
		Name:        req.Name,
		Focus:       req.Focus,
		Description: req.Description,
		PlanType:    req.PlanType,
		DailyReset:  &dr,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return planToProto(p), nil
}

func (h *Handler) GetPlan(ctx context.Context, req *pb.GetPlanRequest) (*pb.Plan, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	p, err := h.service.GetByID(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return planToProto(p), nil
}

func (h *Handler) ListPlans(ctx context.Context, req *pb.ListPlansRequest) (*pb.ListPlansResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}
	plans, err := h.service.ListByUser(ctx, userID)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*pb.Plan, 0, len(plans))
	for _, p := range plans {
		out = append(out, planToProto(p))
	}
	return &pb.ListPlansResponse{Plans: out}, nil
}

func (h *Handler) ListSharedPlans(ctx context.Context, req *pb.ListSharedPlansRequest) (*pb.ListPlansResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}
	plans, err := h.service.ListShared(ctx, userID, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*pb.Plan, 0, len(plans))
	for _, p := range plans {
		out = append(out, planToProto(p))
	}
	return &pb.ListPlansResponse{Plans: out}, nil
}

func (h *Handler) SearchPlans(ctx context.Context, req *pb.SearchPlansRequest) (*pb.SearchPlansResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}
	results, err := h.service.Search(ctx, SearchInput{
		UserID: userID,
		Term:   req.Query,
		Limit:  int(req.Limit),
		Offset: int(req.Offset),
	})
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*pb.SearchPlanResult, 0, len(results))
	for _, r := range results {
		out = append(out, &pb.SearchPlanResult{
			Id:          r.ID.String(),
			Name:        r.Name,
			Description: r.Description,
		})
	}
	return &pb.SearchPlansResponse{Results: out}, nil
}

func (h *Handler) UpdatePlan(ctx context.Context, req *pb.UpdatePlanRequest) (*pb.Plan, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}
	p, err := h.service.Update(ctx, &UpdatePlanInput{
		ID:          id,
		UserID:      userID,
		Name:        req.Name,
		Focus:       req.Focus,
		Description: req.Description,
		PlanType:    req.PlanType,
		DailyReset:  req.DailyReset,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return planToProto(p), nil
}

func (h *Handler) ToggleDailyReset(ctx context.Context, req *pb.ToggleDailyResetRequest) (*pb.Plan, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}
	p, err := h.service.ToggleDailyReset(ctx, id, userID)
	if err != nil {
		return nil, mapError(err)
	}
	return planToProto(p), nil
}

func (h *Handler) DeletePlan(ctx context.Context, req *pb.DeletePlanRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}
	if err := h.service.Delete(ctx, id, userID); err != nil {
		return nil, mapError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) AssertPlanOwnership(ctx context.Context, req *pb.AssertPlanOwnershipRequest) (*emptypb.Empty, error) {
	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid plan_id: %v", err)
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}
	if err := h.service.AssertOwnership(ctx, planID, userID); err != nil {
		return nil, mapError(err)
	}
	return &emptypb.Empty{}, nil
}

func planToProto(p *Plan) *pb.Plan {
	return &pb.Plan{
		Id:          p.ID.String(),
		UserId:      p.UserID.String(),
		Name:        p.Name,
		Focus:       p.Focus,
		Description: p.Description,
		PlanType:    p.PlanType,
		DailyReset:  p.DailyReset,
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}
}

func mapError(err error) error {
	switch {
	case errors.Is(err, commonconstants.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, commonconstants.ErrDuplicateResource):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, commonconstants.ErrInvalidInput),
		errors.Is(err, commonconstants.ErrConstraintViolation):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, commonconstants.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, commonconstants.ErrUnauthorized):
		return status.Error(codes.Unauthenticated, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
