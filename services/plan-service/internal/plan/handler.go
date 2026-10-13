package plan

import (
	"context"
	"fmt"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"github.com/google/uuid"
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
	ListAccessible(ctx context.Context, userID uuid.UUID) ([]*Plan, error)
	ListShared(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Plan, error)
	Search(ctx context.Context, in SearchInput) ([]*SearchResult, error)
	AssertOwnership(ctx context.Context, planID, userID uuid.UUID) error
	GetAccess(ctx context.Context, planID, userID uuid.UUID) (*Access, error)
	ListMembers(ctx context.Context, planID uuid.UUID) (*Members, error)
	AddMember(ctx context.Context, planID, callerID, memberID uuid.UUID) (*Member, error)
	RemoveMember(ctx context.Context, planID, callerID, memberID uuid.UUID) error
}

type Handler struct {
	pb.UnimplementedPlanServiceServer
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

// badUUID builds an InvalidArgument-class domain error for a malformed id so it
// flows through the shared mapper like any other domain error.
func badUUID(field, value string) error {
	return fmt.Errorf("%w: %s %q", commonconstants.ErrUUIDCouldNotBeParsed, field, value)
}

func (h *Handler) CreatePlan(ctx context.Context, req *pb.CreatePlanRequest) (*pb.Plan, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "plan: create plan", badUUID("user_id", req.UserId))
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
		return nil, commongrpc.Fail(ctx, "plan: create plan", err)
	}
	return planToProto(p), nil
}

func (h *Handler) GetPlan(ctx context.Context, req *pb.GetPlanRequest) (*pb.Plan, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "plan: get plan", badUUID("id", req.Id))
	}
	p, err := h.service.GetByID(ctx, id)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "plan: get plan", err)
	}
	return planToProto(p), nil
}

func (h *Handler) ListPlans(ctx context.Context, req *pb.ListPlansRequest) (*pb.ListPlansResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "plan: list plans", badUUID("user_id", req.UserId))
	}
	list := h.service.ListByUser
	if req.IncludeShared {
		list = h.service.ListAccessible
	}
	plans, err := list(ctx, userID)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "plan: list plans", err)
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
		return nil, commongrpc.Fail(ctx, "plan: list shared plans", badUUID("user_id", req.UserId))
	}
	plans, err := h.service.ListShared(ctx, userID, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, commongrpc.Fail(ctx, "plan: list shared plans", err)
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
		return nil, commongrpc.Fail(ctx, "plan: search plans", badUUID("user_id", req.UserId))
	}
	results, err := h.service.Search(ctx, SearchInput{
		UserID: userID,
		Term:   req.Query,
		Limit:  int(req.Limit),
		Offset: int(req.Offset),
	})
	if err != nil {
		return nil, commongrpc.Fail(ctx, "plan: search plans", err)
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
		return nil, commongrpc.Fail(ctx, "plan: update plan", badUUID("id", req.Id))
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "plan: update plan", badUUID("user_id", req.UserId))
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
		return nil, commongrpc.Fail(ctx, "plan: update plan", err)
	}
	return planToProto(p), nil
}

func (h *Handler) ToggleDailyReset(ctx context.Context, req *pb.ToggleDailyResetRequest) (*pb.Plan, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "plan: toggle daily reset", badUUID("id", req.Id))
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "plan: toggle daily reset", badUUID("user_id", req.UserId))
	}
	p, err := h.service.ToggleDailyReset(ctx, id, userID)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "plan: toggle daily reset", err)
	}
	return planToProto(p), nil
}

func (h *Handler) DeletePlan(ctx context.Context, req *pb.DeletePlanRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "plan: delete plan", badUUID("id", req.Id))
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "plan: delete plan", badUUID("user_id", req.UserId))
	}
	if err := h.service.Delete(ctx, id, userID); err != nil {
		return nil, commongrpc.Fail(ctx, "plan: delete plan", err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) AssertPlanOwnership(ctx context.Context, req *pb.AssertPlanOwnershipRequest) (*emptypb.Empty, error) {
	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "plan: assert ownership", badUUID("plan_id", req.PlanId))
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "plan: assert ownership", badUUID("user_id", req.UserId))
	}
	if err := h.service.AssertOwnership(ctx, planID, userID); err != nil {
		return nil, commongrpc.Fail(ctx, "plan: assert ownership", err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) GetPlanAccess(ctx context.Context, req *pb.GetPlanAccessRequest) (*pb.GetPlanAccessResponse, error) {
	const op = "plan: get plan access"
	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, badUUID("plan_id", req.PlanId))
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, badUUID("user_id", req.UserId))
	}
	access, err := h.service.GetAccess(ctx, planID, userID)
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	resp := &pb.GetPlanAccessResponse{Relation: relationToProto(access.Relation)}
	if access.OwnerID != uuid.Nil {
		resp.OwnerId = access.OwnerID.String()
	}
	return resp, nil
}

func (h *Handler) ListPlanMembers(ctx context.Context, req *pb.ListPlanMembersRequest) (*pb.ListPlanMembersResponse, error) {
	const op = "plan: list plan members"
	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, badUUID("plan_id", req.PlanId))
	}
	if _, err := uuid.Parse(req.UserId); err != nil {
		return nil, commongrpc.Fail(ctx, op, badUUID("user_id", req.UserId))
	}
	members, err := h.service.ListMembers(ctx, planID)
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	out := make([]*pb.PlanMember, 0, len(members.Members))
	for _, m := range members.Members {
		out = append(out, memberToProto(m))
	}
	return &pb.ListPlanMembersResponse{
		OwnerId:       members.OwnerID.String(),
		PlanCreatedAt: timestamppb.New(members.PlanCreatedAt),
		Members:       out,
	}, nil
}

func (h *Handler) AddPlanMember(ctx context.Context, req *pb.AddPlanMemberRequest) (*pb.PlanMember, error) {
	const op = "plan: add plan member"
	planID, callerID, memberID, err := parseMemberRequest(req.PlanId, req.UserId, req.MemberUserId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	m, err := h.service.AddMember(ctx, planID, callerID, memberID)
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	return memberToProto(m), nil
}

func (h *Handler) RemovePlanMember(ctx context.Context, req *pb.RemovePlanMemberRequest) (*emptypb.Empty, error) {
	const op = "plan: remove plan member"
	planID, callerID, memberID, err := parseMemberRequest(req.PlanId, req.UserId, req.MemberUserId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	if err := h.service.RemoveMember(ctx, planID, callerID, memberID); err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	return &emptypb.Empty{}, nil
}

// parseMemberRequest parses the three ids every member write carries.
func parseMemberRequest(planID, userID, memberUserID string) (plan, caller, member uuid.UUID, err error) {
	if plan, err = uuid.Parse(planID); err != nil {
		return plan, caller, member, badUUID("plan_id", planID)
	}
	if caller, err = uuid.Parse(userID); err != nil {
		return plan, caller, member, badUUID("user_id", userID)
	}
	if member, err = uuid.Parse(memberUserID); err != nil {
		return plan, caller, member, badUUID("member_user_id", memberUserID)
	}
	return plan, caller, member, nil
}

func memberToProto(m *Member) *pb.PlanMember {
	return &pb.PlanMember{UserId: m.UserID.String(), Since: timestamppb.New(m.Since)}
}

func relationToProto(r Relation) pb.PlanRelation {
	switch r {
	case RelationOwner:
		return pb.PlanRelation_PLAN_RELATION_OWNER
	case RelationMember:
		return pb.PlanRelation_PLAN_RELATION_MEMBER
	default:
		return pb.PlanRelation_PLAN_RELATION_NONE
	}
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
