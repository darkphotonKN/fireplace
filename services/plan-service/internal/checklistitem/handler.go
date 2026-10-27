package checklistitem

import (
	"context"
	"fmt"
	"time"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service interface {
	Create(ctx context.Context, in CreateItemInput) (*Item, error)
	GetInPlan(ctx context.Context, planID, id uuid.UUID) (*Item, error)
	ListByPlanID(ctx context.Context, in ListItemsInput) ([]*Item, error)
	ListArchivedByPlanID(ctx context.Context, planID uuid.UUID, scope *string) ([]*Item, error)
	ListUpcoming(ctx context.Context, planID uuid.UUID) ([]*Item, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Item, error)
	ListInDateWindow(ctx context.Context, planID uuid.UUID, windowStart, windowEnd time.Time) ([]*Item, error)
	Update(ctx context.Context, in UpdateItemInput) (*Item, error)
	Reorder(ctx context.Context, in ReorderInput) ([]*Item, error)
	Duplicate(ctx context.Context, planID, id uuid.UUID) ([]*Item, error)
	UpdateDates(ctx context.Context, in UpdateDatesInput) (*Item, error)
	Archive(ctx context.Context, planID, id uuid.UUID, archived bool) (*Item, error)
	Delete(ctx context.Context, planID, id uuid.UUID) error
}

type Handler struct {
	pb.UnimplementedChecklistServiceServer
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

// itemRef parses an item-by-id request's item id and the plan it is addressed
// under. The plan id is required: a request without one is a caller bug and is
// refused, never run unscoped (FS-12ZMF R12).
func itemRef(rawID, rawPlanID string) (id, planID uuid.UUID, err error) {
	if id, err = uuid.Parse(rawID); err != nil {
		return uuid.Nil, uuid.Nil, badUUID("id", rawID)
	}
	if rawPlanID == "" {
		return uuid.Nil, uuid.Nil, fmt.Errorf("%w: plan_id is required", commonconstants.ErrInvalidInput)
	}
	if planID, err = uuid.Parse(rawPlanID); err != nil {
		return uuid.Nil, uuid.Nil, badUUID("plan_id", rawPlanID)
	}
	return id, planID, nil
}

func (h *Handler) CreateItem(ctx context.Context, req *pb.CreateItemRequest) (*pb.ChecklistItem, error) {
	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: create", badUUID("plan_id", req.PlanId))
	}

	var parentID *uuid.UUID
	if req.ParentId != nil {
		pid, err := uuid.Parse(*req.ParentId)
		if err != nil {
			return nil, commongrpc.Fail(ctx, "checklistitem: create", badUUID("parent_id", *req.ParentId))
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
		return nil, commongrpc.Fail(ctx, "checklistitem: create", err)
	}
	return itemToProto(item), nil
}

func (h *Handler) GetItem(ctx context.Context, req *pb.GetItemRequest) (*pb.ChecklistItem, error) {
	id, planID, err := itemRef(req.Id, req.PlanId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: get", err)
	}
	item, err := h.service.GetInPlan(ctx, planID, id)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: get", err)
	}
	return itemToProto(item), nil
}

func (h *Handler) ListItems(ctx context.Context, req *pb.ListItemsRequest) (*pb.ListItemsResponse, error) {
	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: list", badUUID("plan_id", req.PlanId))
	}
	items, err := h.service.ListByPlanID(ctx, ListItemsInput{
		PlanID: planID,
		Scope:  req.Scope,
		Type:   req.Type,
	})
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: list", err)
	}
	return itemListToProto(items), nil
}

func (h *Handler) ListArchivedItems(ctx context.Context, req *pb.ListArchivedItemsRequest) (*pb.ListItemsResponse, error) {
	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: list archived", badUUID("plan_id", req.PlanId))
	}
	items, err := h.service.ListArchivedByPlanID(ctx, planID, nil)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: list archived", err)
	}
	return itemListToProto(items), nil
}

func (h *Handler) ListUpcomingItems(ctx context.Context, req *pb.ListUpcomingItemsRequest) (*pb.ListItemsResponse, error) {
	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: list upcoming", badUUID("plan_id", req.PlanId))
	}
	items, err := h.service.ListUpcoming(ctx, planID)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: list upcoming", err)
	}
	return itemListToProto(items), nil
}

func (h *Handler) ListItemsByUser(ctx context.Context, req *pb.ListItemsByUserRequest) (*pb.ListItemsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: list by user", badUUID("user_id", req.UserId))
	}
	items, err := h.service.GetByUserID(ctx, userID)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: list by user", err)
	}
	return itemListToProto(items), nil
}

func (h *Handler) ListItemsInDateWindow(ctx context.Context, req *pb.ListItemsInDateWindowRequest) (*pb.ListItemsResponse, error) {
	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: list in date window", badUUID("plan_id", req.PlanId))
	}
	if req.WindowStart == nil || req.WindowEnd == nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: list in date window",
			fmt.Errorf("%w: window_start and window_end are required", commonconstants.ErrInvalidInput))
	}
	items, err := h.service.ListInDateWindow(ctx, planID, req.WindowStart.AsTime(), req.WindowEnd.AsTime())
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: list in date window", err)
	}
	return itemListToProto(items), nil
}

func (h *Handler) UpdateItem(ctx context.Context, req *pb.UpdateItemRequest) (*pb.ChecklistItem, error) {
	id, planID, err := itemRef(req.Id, req.PlanId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: update", err)
	}

	in := UpdateItemInput{
		ID:          id,
		PlanID:      planID,
		Description: req.Description,
		Done:        req.Done,
		Type:        req.Type,
		Scope:       req.Scope,
	}

	// Intention: unset leaves the column alone; any string, empty included, is
	// a set the service trims (an empty result clears it).
	if req.Intention != nil {
		in.SetIntention = true
		in.Intention = req.Intention
	}

	// Parent_id three-state: clear_parent_id wins (clears column); else if
	// parent_id is set, re-parent; else leave alone.
	if req.ClearParentId {
		in.SetParent = true
		in.ParentID = nil
	} else if req.ParentId != nil {
		pid, err := uuid.Parse(*req.ParentId)
		if err != nil {
			return nil, commongrpc.Fail(ctx, "checklistitem: update", badUUID("parent_id", *req.ParentId))
		}
		in.SetParent = true
		in.ParentID = &pid
	}

	item, err := h.service.Update(ctx, in)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: update", err)
	}
	return itemToProto(item), nil
}

func (h *Handler) ReorderItems(ctx context.Context, req *pb.ReorderItemsRequest) (*pb.ListItemsResponse, error) {
	const op = "checklistitem: reorder"

	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, badUUID("plan_id", req.PlanId))
	}

	var parentID *uuid.UUID
	if req.ParentId != nil {
		pid, err := uuid.Parse(*req.ParentId)
		if err != nil {
			return nil, commongrpc.Fail(ctx, op, badUUID("parent_id", *req.ParentId))
		}
		parentID = &pid
	}

	ids := make([]uuid.UUID, 0, len(req.Ids))
	for _, raw := range req.Ids {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, commongrpc.Fail(ctx, op, badUUID("ids", raw))
		}
		ids = append(ids, id)
	}

	items, err := h.service.Reorder(ctx, ReorderInput{
		PlanID:   planID,
		Scope:    req.Scope,
		ParentID: parentID,
		IDs:      ids,
	})
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	return itemListToProto(items), nil
}

// DuplicateItem returns the created items: the copy first, then its copied
// children in order (FS-F6AEE R7).
func (h *Handler) DuplicateItem(ctx context.Context, req *pb.DuplicateItemRequest) (*pb.ListItemsResponse, error) {
	const op = "checklistitem: duplicate"

	id, planID, err := itemRef(req.Id, req.PlanId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	items, err := h.service.Duplicate(ctx, planID, id)
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	return itemListToProto(items), nil
}

func (h *Handler) UpdateItemDates(ctx context.Context, req *pb.UpdateItemDatesRequest) (*pb.ChecklistItem, error) {
	id, planID, err := itemRef(req.Id, req.PlanId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: update dates", err)
	}

	in := UpdateDatesInput{ID: id, PlanID: planID}
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
		return nil, commongrpc.Fail(ctx, "checklistitem: update dates", err)
	}
	return itemToProto(item), nil
}

func (h *Handler) ArchiveItem(ctx context.Context, req *pb.ArchiveItemRequest) (*pb.ChecklistItem, error) {
	id, planID, err := itemRef(req.Id, req.PlanId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: archive", err)
	}
	item, err := h.service.Archive(ctx, planID, id, req.Archived)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: archive", err)
	}
	return itemToProto(item), nil
}

func (h *Handler) DeleteItem(ctx context.Context, req *pb.DeleteItemRequest) (*emptypb.Empty, error) {
	id, planID, err := itemRef(req.Id, req.PlanId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: delete", err)
	}
	if err := h.service.Delete(ctx, planID, id); err != nil {
		return nil, commongrpc.Fail(ctx, "checklistitem: delete", err)
	}
	return &emptypb.Empty{}, nil
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
		Intention:   i.Intention,
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
