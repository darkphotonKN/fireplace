package checklistitem

import (
	"context"
	"errors"
	"time"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service interface {
	Create(ctx context.Context, in CreateItemInput) (*Item, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Item, error)
	ListByPlanID(ctx context.Context, in ListItemsInput) ([]*Item, error)
	ListArchivedByPlanID(ctx context.Context, planID uuid.UUID, scope *string) ([]*Item, error)
	ListUpcoming(ctx context.Context, planID uuid.UUID) ([]*Item, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Item, error)
	ListInDateWindow(ctx context.Context, planID uuid.UUID, windowStart, windowEnd time.Time) ([]*Item, error)
	Update(ctx context.Context, in UpdateItemInput) (*Item, error)
	UpdateDates(ctx context.Context, in UpdateDatesInput) (*Item, error)
	Archive(ctx context.Context, id uuid.UUID, archived bool) (*Item, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DailyReset(ctx context.Context) (int64, error)
}

type Handler struct {
	pb.UnimplementedChecklistServiceServer
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) CreateItem(ctx context.Context, req *pb.CreateItemRequest) (*pb.ChecklistItem, error) {
	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid plan_id: %v", err)
	}

	var parentID *uuid.UUID
	if req.ParentId != nil {
		pid, err := uuid.Parse(*req.ParentId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid parent_id: %v", err)
		}
		parentID = &pid
	}

	var scopePtr *string
	if req.Scope != "" {
		s := req.Scope
		scopePtr = &s
	}

	item, err := h.service.Create(ctx, CreateItemInput{
		PlanID:      planID,
		Description: req.Description,
		Scope:       scopePtr,
		Type:        req.Type,
		ParentID:    parentID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return itemToProto(item), nil
}

func (h *Handler) GetItem(ctx context.Context, req *pb.GetItemRequest) (*pb.ChecklistItem, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	item, err := h.service.GetByID(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return itemToProto(item), nil
}

func (h *Handler) ListItems(ctx context.Context, req *pb.ListItemsRequest) (*pb.ListItemsResponse, error) {
	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid plan_id: %v", err)
	}
	items, err := h.service.ListByPlanID(ctx, ListItemsInput{
		PlanID: planID,
		Scope:  req.Scope,
		Type:   req.Type,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return itemListToProto(items), nil
}

func (h *Handler) ListArchivedItems(ctx context.Context, req *pb.ListArchivedItemsRequest) (*pb.ListItemsResponse, error) {
	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid plan_id: %v", err)
	}
	items, err := h.service.ListArchivedByPlanID(ctx, planID, nil)
	if err != nil {
		return nil, mapError(err)
	}
	return itemListToProto(items), nil
}

func (h *Handler) ListUpcomingItems(ctx context.Context, req *pb.ListUpcomingItemsRequest) (*pb.ListItemsResponse, error) {
	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid plan_id: %v", err)
	}
	items, err := h.service.ListUpcoming(ctx, planID)
	if err != nil {
		return nil, mapError(err)
	}
	return itemListToProto(items), nil
}

func (h *Handler) ListItemsByUser(ctx context.Context, req *pb.ListItemsByUserRequest) (*pb.ListItemsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}
	items, err := h.service.GetByUserID(ctx, userID)
	if err != nil {
		return nil, mapError(err)
	}
	return itemListToProto(items), nil
}

func (h *Handler) ListItemsInDateWindow(ctx context.Context, req *pb.ListItemsInDateWindowRequest) (*pb.ListItemsResponse, error) {
	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid plan_id: %v", err)
	}
	if req.WindowStart == nil || req.WindowEnd == nil {
		return nil, status.Error(codes.InvalidArgument, "window_start and window_end are required")
	}
	items, err := h.service.ListInDateWindow(ctx, planID, req.WindowStart.AsTime(), req.WindowEnd.AsTime())
	if err != nil {
		return nil, mapError(err)
	}
	return itemListToProto(items), nil
}

func (h *Handler) UpdateItem(ctx context.Context, req *pb.UpdateItemRequest) (*pb.ChecklistItem, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}

	in := UpdateItemInput{
		ID:          id,
		Description: req.Description,
		Done:        req.Done,
		Type:        req.Type,
		Scope:       req.Scope,
	}

	// Parent_id three-state: clear_parent_id wins (clears column); else if
	// parent_id is set, re-parent; else leave alone.
	if req.ClearParentId {
		in.SetParent = true
		in.ParentID = nil
	} else if req.ParentId != nil {
		pid, err := uuid.Parse(*req.ParentId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid parent_id: %v", err)
		}
		in.SetParent = true
		in.ParentID = &pid
	}

	item, err := h.service.Update(ctx, in)
	if err != nil {
		return nil, mapError(err)
	}
	return itemToProto(item), nil
}

func (h *Handler) UpdateItemDates(ctx context.Context, req *pb.UpdateItemDatesRequest) (*pb.ChecklistItem, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}

	in := UpdateDatesInput{ID: id}
	if req.StartDate != nil {
		in.SetStart = true
		t := req.StartDate.AsTime()
		in.StartDate = &t
	}
	if req.DueDate != nil {
		in.SetDue = true
		t := req.DueDate.AsTime()
		in.DueDate = &t
	}

	item, err := h.service.UpdateDates(ctx, in)
	if err != nil {
		return nil, mapError(err)
	}
	return itemToProto(item), nil
}

func (h *Handler) ArchiveItem(ctx context.Context, req *pb.ArchiveItemRequest) (*pb.ChecklistItem, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	item, err := h.service.Archive(ctx, id, req.Archived)
	if err != nil {
		return nil, mapError(err)
	}
	return itemToProto(item), nil
}

func (h *Handler) DeleteItem(ctx context.Context, req *pb.DeleteItemRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	if err := h.service.Delete(ctx, id); err != nil {
		return nil, mapError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) DailyReset(ctx context.Context, _ *pb.DailyResetRequest) (*pb.DailyResetResponse, error) {
	n, err := h.service.DailyReset(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return &pb.DailyResetResponse{ResetCount: int32(n)}, nil
}

// --- helpers ---

func itemToProto(i *Item) *pb.ChecklistItem {
	out := &pb.ChecklistItem{
		Id:          i.ID.String(),
		PlanId:      i.PlanID.String(),
		Description: i.Description,
		Done:        i.Done,
		Sequence:    fmtSequence(i.Sequence),
		Type:        i.Type,
		Scope:       i.Scope,
		Archived:    i.Archived,
		CreatedAt:   timestamppb.New(i.CreatedAt),
		UpdatedAt:   timestamppb.New(i.UpdatedAt),
	}
	if i.ParentID != nil {
		s := i.ParentID.String()
		out.ParentId = &s
	}
	if i.StartDate != nil {
		out.StartDate = timestamppb.New(*i.StartDate)
	}
	if i.DueDate != nil {
		out.DueDate = timestamppb.New(*i.DueDate)
	}
	return out
}

func itemListToProto(items []*Item) *pb.ListItemsResponse {
	out := make([]*pb.ChecklistItem, 0, len(items))
	for _, i := range items {
		out = append(out, itemToProto(i))
	}
	return &pb.ListItemsResponse{Items: out}
}

// fmtSequence converts the int sequence to a string for the proto field. The
// monolith stored sequences as ints in the DB; the proto is a string so the
// client can carry alphanumeric ordering keys in the future without breaking.
func fmtSequence(seq int) string {
	if seq == 0 {
		return ""
	}
	return itoa(seq)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var b [20]byte
	idx := len(b)
	for i > 0 {
		idx--
		b[idx] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		idx--
		b[idx] = '-'
	}
	return string(b[idx:])
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
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
