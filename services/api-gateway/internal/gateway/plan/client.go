package plangw

import (
	"context"
	"strconv"
	"sync"
	"time"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const targetService = "plan"

// Client wraps a long-lived gRPC connection to plan-service. The connection is
// established lazily on first use and reused across all calls — gRPC's HTTP/2
// layer multiplexes streams over a single conn, so concurrent RPCs are cheap.
// Opening a new conn per RPC (the previous shape) serialized badly under
// concurrent load.
type Client struct {
	registry discovery.Registry
	mu       sync.Mutex
	conn     *grpc.ClientConn
}

func NewClient(registry discovery.Registry) *Client {
	return &Client{registry: registry}
}

// ensureConn returns the cached ClientConn, redialing if it's missing or
// shut down. All RPC entry points go through here.
func (c *Client) ensureConn(ctx context.Context) (*grpc.ClientConn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && c.conn.GetState() != connectivity.Shutdown {
		return c.conn, nil
	}
	conn, err := discovery.ServiceConnection(ctx, targetService, c.registry)
	if err != nil {
		return nil, err
	}
	c.conn = conn
	return conn, nil
}

func (c *Client) planClient(ctx context.Context) (pb.PlanServiceClient, error) {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		return nil, err
	}
	return pb.NewPlanServiceClient(conn), nil
}

func (c *Client) checklistClient(ctx context.Context) (pb.ChecklistServiceClient, error) {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		return nil, err
	}
	return pb.NewChecklistServiceClient(conn), nil
}

// Close releases the underlying gRPC connection. Wire from gateway shutdown.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

// --- Plan methods ---

func (c *Client) CreatePlan(ctx context.Context, userID uuid.UUID, req CreatePlanReq) (*PlanResp, error) {
	plans, err := c.planClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := plans.CreatePlan(ctx, &pb.CreatePlanRequest{
		UserId:      userID.String(),
		Name:        req.Name,
		Focus:       req.Focus,
		Description: req.Description,
		PlanType:    req.PlanType,
	})
	if err != nil {
		return nil, err
	}
	return planFromProto(resp), nil
}

func (c *Client) GetPlan(ctx context.Context, id, userID uuid.UUID) (*PlanResp, error) {
	plans, err := c.planClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := plans.GetPlan(ctx, &pb.GetPlanRequest{Id: id.String(), UserId: userID.String()})
	if err != nil {
		return nil, err
	}
	return planFromProto(resp), nil
}

// ListPlans is every plan userID owns or is a member of, newest first, unpaged
// (FS-12ZMF R22).
func (c *Client) ListPlans(ctx context.Context, userID uuid.UUID) ([]*PlanResp, error) {
	plans, err := c.planClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := plans.ListPlans(ctx, &pb.ListPlansRequest{UserId: userID.String(), IncludeShared: true})
	if err != nil {
		return nil, err
	}
	return planSliceFromProto(resp.Plans), nil
}

func (c *Client) ListSharedPlans(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*PlanResp, error) {
	plans, err := c.planClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := plans.ListSharedPlans(ctx, &pb.ListSharedPlansRequest{
		UserId: userID.String(),
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return planSliceFromProto(resp.Plans), nil
}

func (c *Client) SearchPlans(ctx context.Context, userID uuid.UUID, params SearchParam) ([]*pb.SearchPlanResult, error) {
	plans, err := c.planClient(ctx)
	if err != nil {
		return nil, err
	}
	limit, _ := strconv.Atoi(params.Limit)
	if limit <= 0 {
		limit = 20
	}
	offset, _ := strconv.Atoi(params.Offset)
	resp, err := plans.SearchPlans(ctx, &pb.SearchPlansRequest{
		UserId: userID.String(),
		Query:  params.Term,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return resp.Results, nil
}

func (c *Client) UpdatePlan(ctx context.Context, id, userID uuid.UUID, req UpdatePlanReq) (*PlanResp, error) {
	plans, err := c.planClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := plans.UpdatePlan(ctx, &pb.UpdatePlanRequest{
		Id:          id.String(),
		UserId:      userID.String(),
		Name:        req.Name,
		Focus:       req.Focus,
		Description: req.Description,
		DailyReset:  req.DailyReset,
	})
	if err != nil {
		return nil, err
	}
	return planFromProto(resp), nil
}

func (c *Client) DeletePlan(ctx context.Context, id, userID uuid.UUID) error {
	plans, err := c.planClient(ctx)
	if err != nil {
		return err
	}
	_, err = plans.DeletePlan(ctx, &pb.DeletePlanRequest{Id: id.String(), UserId: userID.String()})
	return err
}

// GetPlanAccess asks plan-service for userID's relation to planID. A missing
// plan is RelationNone, not an error; an error means no answer, and callers
// must fail closed on it (ADR-0015 §3).
//
// plan-service's answer also names the plan's owner, even for a stranger. It is
// dropped here, at the boundary, so no gateway code can carry it any further.
func (c *Client) GetPlanAccess(ctx context.Context, planID, userID uuid.UUID) (Relation, error) {
	plans, err := c.planClient(ctx)
	if err != nil {
		return RelationNone, err
	}
	resp, err := plans.GetPlanAccess(ctx, &pb.GetPlanAccessRequest{
		PlanId: planID.String(),
		UserId: userID.String(),
	})
	if err != nil {
		return RelationNone, err
	}
	return relationFromProto(resp.Relation), nil
}

// --- Plan member methods (FS-12ZMF R15–R21) ---

func (c *Client) ListPlanMembers(ctx context.Context, planID, userID uuid.UUID) (*PlanMembers, error) {
	plans, err := c.planClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := plans.ListPlanMembers(ctx, &pb.ListPlanMembersRequest{
		PlanId: planID.String(),
		UserId: userID.String(),
	})
	if err != nil {
		return nil, err
	}
	return planMembersFromProto(resp), nil
}

func (c *Client) AddPlanMember(ctx context.Context, planID, userID, memberID uuid.UUID) (*Membership, error) {
	plans, err := c.planClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := plans.AddPlanMember(ctx, &pb.AddPlanMemberRequest{
		PlanId:       planID.String(),
		UserId:       userID.String(),
		MemberUserId: memberID.String(),
	})
	if err != nil {
		return nil, err
	}
	m := membershipFromProto(resp)
	return &m, nil
}

func (c *Client) RemovePlanMember(ctx context.Context, planID, userID, memberID uuid.UUID) error {
	plans, err := c.planClient(ctx)
	if err != nil {
		return err
	}
	_, err = plans.RemovePlanMember(ctx, &pb.RemovePlanMemberRequest{
		PlanId:       planID.String(),
		UserId:       userID.String(),
		MemberUserId: memberID.String(),
	})
	return err
}

func planMembersFromProto(r *pb.ListPlanMembersResponse) *PlanMembers {
	ownerID, _ := uuid.Parse(r.OwnerId)
	out := &PlanMembers{
		OwnerID:       ownerID,
		PlanCreatedAt: r.PlanCreatedAt.AsTime(),
		Members:       make([]Membership, 0, len(r.Members)),
	}
	for _, m := range r.Members {
		out.Members = append(out.Members, membershipFromProto(m))
	}
	return out
}

func membershipFromProto(m *pb.PlanMember) Membership {
	id, _ := uuid.Parse(m.UserId)
	return Membership{UserID: id, Since: m.Since.AsTime()}
}

// relationFromProto maps plan-service's answer. Anything unrecognised is
// RelationNone, so a check built on it fails closed.
func relationFromProto(r pb.PlanRelation) Relation {
	switch r {
	case pb.PlanRelation_PLAN_RELATION_OWNER:
		return RelationOwner
	case pb.PlanRelation_PLAN_RELATION_MEMBER:
		return RelationMember
	default:
		return RelationNone
	}
}

// --- Checklist methods ---

func (c *Client) CreateChecklist(ctx context.Context, planID, userID uuid.UUID, req CreateChecklistReq) (*ChecklistResp, error) {
	items, err := c.checklistClient(ctx)
	if err != nil {
		return nil, err
	}

	scopeStr := ""
	if req.Scope != nil {
		scopeStr = *req.Scope
	}
	var parentIDStr *string
	if req.ParentID != nil {
		s := req.ParentID.String()
		parentIDStr = &s
	}

	resp, err := items.CreateItem(ctx, &pb.CreateItemRequest{
		PlanId:      planID.String(),
		UserId:      userID.String(),
		Description: req.Description,
		Scope:       scopeStr,
		Type:        req.Type,
		ParentId:    parentIDStr,
	})
	if err != nil {
		return nil, err
	}
	return checklistFromProto(resp), nil
}

func (c *Client) GetChecklist(ctx context.Context, planID, id, userID uuid.UUID) (*ChecklistResp, error) {
	items, err := c.checklistClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := items.GetItem(ctx, &pb.GetItemRequest{Id: id.String(), PlanId: planID.String(), UserId: userID.String()})
	if err != nil {
		return nil, err
	}
	return checklistFromProto(resp), nil
}

func (c *Client) ListChecklists(ctx context.Context, planID, userID uuid.UUID, scope, itemType *string) ([]*ChecklistResp, error) {
	items, err := c.checklistClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := items.ListItems(ctx, &pb.ListItemsRequest{
		PlanId: planID.String(),
		UserId: userID.String(),
		Scope:  scope,
		Type:   itemType,
	})
	if err != nil {
		return nil, err
	}
	return checklistSliceFromProto(resp.Items), nil
}

func (c *Client) ListArchivedChecklists(ctx context.Context, planID, userID uuid.UUID) ([]*ChecklistResp, error) {
	items, err := c.checklistClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := items.ListArchivedItems(ctx, &pb.ListArchivedItemsRequest{
		PlanId: planID.String(),
		UserId: userID.String(),
	})
	if err != nil {
		return nil, err
	}
	return checklistSliceFromProto(resp.Items), nil
}

func (c *Client) ListUpcomingChecklists(ctx context.Context, planID, userID uuid.UUID) ([]*ChecklistResp, error) {
	items, err := c.checklistClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := items.ListUpcomingItems(ctx, &pb.ListUpcomingItemsRequest{
		PlanId: planID.String(),
		UserId: userID.String(),
	})
	if err != nil {
		return nil, err
	}
	return checklistSliceFromProto(resp.Items), nil
}

// ListChecklistsByUser returns all non-archived items across the user's
// plans. Used by the useranalytics consumer adapter.
func (c *Client) ListChecklistsByUser(ctx context.Context, userID uuid.UUID) ([]*ChecklistResp, error) {
	items, err := c.checklistClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := items.ListItemsByUser(ctx, &pb.ListItemsByUserRequest{UserId: userID.String()})
	if err != nil {
		return nil, err
	}
	return checklistSliceFromProto(resp.Items), nil
}

func (c *Client) UpdateChecklist(ctx context.Context, planID, id, userID uuid.UUID, req UpdateChecklistReq) (*ChecklistResp, error) {
	items, err := c.checklistClient(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := items.UpdateItem(ctx, updateItemRequest(planID, id, userID, req))
	if err != nil {
		return nil, err
	}
	return checklistFromProto(resp), nil
}

// updateItemRequest maps the PATCH body onto the proto. parentId is three-state
// (omit, null, value); intention is two-state (omit leaves it, any string sets
// it, and plan-service trims and clears an empty result).
func updateItemRequest(planID, id, userID uuid.UUID, req UpdateChecklistReq) *pb.UpdateItemRequest {
	pbReq := &pb.UpdateItemRequest{
		Id:          id.String(),
		PlanId:      planID.String(),
		UserId:      userID.String(),
		Description: req.Description,
		Done:        req.Done,
		Type:        req.Type,
		Scope:       req.Scope,
		Intention:   req.Intention,
	}
	if req.ParentID.Present {
		if req.ParentID.Valid {
			s := req.ParentID.Value.String()
			pbReq.ParentId = &s
		} else {
			pbReq.ClearParentId = true
		}
	}
	return pbReq
}

// ReorderChecklists hands one sibling set's complete order to plan-service,
// which validates it and writes the sequences in a single transaction. The
// reordered siblings come back in their new order (FS-0009 R2.5).
func (c *Client) ReorderChecklists(ctx context.Context, planID, userID uuid.UUID, req ReorderChecklistReq) ([]*ChecklistResp, error) {
	items, err := c.checklistClient(ctx)
	if err != nil {
		return nil, err
	}

	var parentIDStr *string
	if req.ParentID != nil {
		s := req.ParentID.String()
		parentIDStr = &s
	}
	ids := make([]string, 0, len(req.IDs))
	for _, id := range req.IDs {
		ids = append(ids, id.String())
	}

	resp, err := items.ReorderItems(ctx, &pb.ReorderItemsRequest{
		PlanId:   planID.String(),
		UserId:   userID.String(),
		Scope:    req.Scope,
		ParentId: parentIDStr,
		Ids:      ids,
	})
	if err != nil {
		return nil, err
	}
	return checklistSliceFromProto(resp.Items), nil
}

// DuplicateChecklist asks plan-service to copy one item (and a top-level
// item's live children) in a single transaction. The created items come back
// as the copy first, then its copied children in order (FS-F6AEE R7).
func (c *Client) DuplicateChecklist(ctx context.Context, planID, id, userID uuid.UUID) ([]*ChecklistResp, error) {
	items, err := c.checklistClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := items.DuplicateItem(ctx, &pb.DuplicateItemRequest{
		Id: id.String(), PlanId: planID.String(), UserId: userID.String(),
	})
	if err != nil {
		return nil, err
	}
	return checklistSliceFromProto(resp.Items), nil
}

func (c *Client) UpdateChecklistDates(ctx context.Context, planID, id, userID uuid.UUID, req UpdateDatesReq) (*ChecklistResp, error) {
	items, err := c.checklistClient(ctx)
	if err != nil {
		return nil, err
	}

	pbReq := &pb.UpdateItemDatesRequest{
		Id:     id.String(),
		PlanId: planID.String(),
		UserId: userID.String(),
	}
	if req.StartDate.Present && req.StartDate.Valid {
		pbReq.StartDate = timestamppb.New(req.StartDate.Value)
	}
	if req.DueDate.Present && req.DueDate.Valid {
		pbReq.DueDate = timestamppb.New(req.DueDate.Value)
	}
	resp, err := items.UpdateItemDates(ctx, pbReq)
	if err != nil {
		return nil, err
	}
	return checklistFromProto(resp), nil
}

func (c *Client) ArchiveChecklist(ctx context.Context, planID, id, userID uuid.UUID, archived bool) (*ChecklistResp, error) {
	items, err := c.checklistClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := items.ArchiveItem(ctx, &pb.ArchiveItemRequest{
		Id: id.String(), PlanId: planID.String(), UserId: userID.String(), Archived: archived,
	})
	if err != nil {
		return nil, err
	}
	return checklistFromProto(resp), nil
}

func (c *Client) DeleteChecklist(ctx context.Context, planID, id, userID uuid.UUID) error {
	items, err := c.checklistClient(ctx)
	if err != nil {
		return err
	}
	_, err = items.DeleteItem(ctx, &pb.DeleteItemRequest{Id: id.String(), PlanId: planID.String(), UserId: userID.String()})
	return err
}

// ListItemsInDateWindow returns checklist items for the plan whose date range
// intersects [windowStart, windowEnd].
//
// Added when calendar-service folded into the gateway (ADR-0009 §1): calendar
// carried its own gRPC client to plan-service, and keeping it would have meant
// two connections to the same service from one process. It reuses this
// client's cached conn instead.
func (c *Client) ListItemsInDateWindow(ctx context.Context, planID, userID uuid.UUID, windowStart, windowEnd time.Time) ([]*pb.ChecklistItem, error) {
	items, err := c.checklistClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := items.ListItemsInDateWindow(ctx, &pb.ListItemsInDateWindowRequest{
		PlanId:      planID.String(),
		UserId:      userID.String(),
		WindowStart: timestamppb.New(windowStart),
		WindowEnd:   timestamppb.New(windowEnd),
	})
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// --- proto → HTTP response converters ---

func planFromProto(p *pb.Plan) *PlanResp {
	id, _ := uuid.Parse(p.Id)
	userID, _ := uuid.Parse(p.UserId)
	return &PlanResp{
		ID: id, UserID: userID,
		Name: p.Name, Focus: p.Focus, Description: p.Description,
		PlanType: p.PlanType, DailyReset: p.DailyReset,
		CreatedAt: p.CreatedAt.AsTime(), UpdatedAt: p.UpdatedAt.AsTime(),
	}
}

func planSliceFromProto(plans []*pb.Plan) []*PlanResp {
	out := make([]*PlanResp, 0, len(plans))
	for _, p := range plans {
		out = append(out, planFromProto(p))
	}
	return out
}

func checklistFromProto(c *pb.ChecklistItem) *ChecklistResp {
	id, _ := uuid.Parse(c.Id)
	planID, _ := uuid.Parse(c.PlanId)
	out := &ChecklistResp{
		ID: id, PlanID: planID,
		Description: c.Description, Done: c.Done, Sequence: c.Sequence,
		Type: c.Type, Scope: c.Scope, Archived: c.Archived,
		CreatedAt: c.CreatedAt.AsTime(), UpdatedAt: c.UpdatedAt.AsTime(),
		Intention: c.Intention,
	}
	if c.ParentId != nil {
		pid, err := uuid.Parse(*c.ParentId)
		if err == nil {
			out.ParentID = &pid
		}
	}
	if c.StartDate != nil {
		t := c.StartDate.AsTime()
		out.StartDate = &t
	}
	if c.DueDate != nil {
		t := c.DueDate.AsTime()
		out.DueDate = &t
	}
	_ = time.Time{}
	return out
}

func checklistSliceFromProto(items []*pb.ChecklistItem) []*ChecklistResp {
	out := make([]*ChecklistResp, 0, len(items))
	for _, c := range items {
		out = append(out, checklistFromProto(c))
	}
	return out
}
