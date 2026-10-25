package insights

import (
	"context"
	"fmt"
	"sync"

	planpb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/dayplan"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/status"
)

// PlanClient is insights-service's thin gRPC client to plan-service. Insights
// owns no plan or checklist data of its own — it asks plan-service for the focus
// and checklist context it needs to build LLM prompts.
//
// The ClientConn is cached and reused across calls (HTTP/2 multiplexing); see
// the gateway's internal/gateway/plan client for the same pattern.
type PlanClient struct {
	registry discovery.Registry
	mu       sync.Mutex
	conn     *grpc.ClientConn
}

func NewPlanClient(registry discovery.Registry) *PlanClient {
	return &PlanClient{registry: registry}
}

// fromPlanService converts a gRPC error returned by plan-service into a local
// insights domain error, starting a fresh error chain in this service. The wire
// status is the contract between services — we translate the code into one of
// our own domain sentinels rather than propagating plan-service's Go chain.
func fromPlanService(op string, err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		// Not a gRPC status (e.g. discovery / dial failure) — surface as-is with
		// context; the handler will treat it as a server fault.
		return fmt.Errorf("insights: plan client: %s: %w", op, err)
	}
	switch st.Code() {
	case codes.NotFound:
		return fmt.Errorf("insights: plan client: %s: %w", op, commonconstants.ErrNotFound)
	case codes.PermissionDenied:
		return fmt.Errorf("insights: plan client: %s: %w", op, commonconstants.ErrForbidden)
	case codes.InvalidArgument:
		return fmt.Errorf("insights: plan client: %s: %w", op, commonconstants.ErrInvalidInput)
	case codes.Unauthenticated:
		return fmt.Errorf("insights: plan client: %s: %w", op, commonconstants.ErrUnauthorized)
	default:
		// Unknown / Internal / Unavailable etc. — sever the downstream chain and
		// report a generic server fault (no domain sentinel ⇒ maps to Internal).
		return fmt.Errorf("insights: plan client: %s: %s", op, st.Message())
	}
}

func (c *PlanClient) ensureConn(ctx context.Context) (*grpc.ClientConn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && c.conn.GetState() != connectivity.Shutdown {
		return c.conn, nil
	}
	conn, err := discovery.ServiceConnection(ctx, "plan", c.registry)
	if err != nil {
		return nil, err
	}
	c.conn = conn
	return conn, nil
}

// AssertPlanOwnership returns nil if planID is owned by userID, error otherwise.
func (c *PlanClient) AssertPlanOwnership(ctx context.Context, planID, userID uuid.UUID) error {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		return fmt.Errorf("insights: plan client: connect: %w", err)
	}
	plans := planpb.NewPlanServiceClient(conn)
	_, err = plans.AssertPlanOwnership(ctx, &planpb.AssertPlanOwnershipRequest{
		PlanId: planID.String(),
		UserId: userID.String(),
	})
	return fromPlanService("assert plan ownership", err)
}

// GetPlanContext fetches the plan's focus and full checklist for prompt-building.
// plan-service's GetPlan enforces ownership (it takes the user_id), so this
// doubles as the ownership check for the generation flows.
func (c *PlanClient) GetPlanContext(ctx context.Context, planID, userID uuid.UUID) (*PlanContext, error) {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		return nil, fmt.Errorf("insights: plan client: connect: %w", err)
	}

	plans := planpb.NewPlanServiceClient(conn)
	plan, err := plans.GetPlan(ctx, &planpb.GetPlanRequest{
		Id:     planID.String(),
		UserId: userID.String(),
	})
	if err != nil {
		return nil, fromPlanService("get plan", err)
	}

	items := planpb.NewChecklistServiceClient(conn)
	resp, err := items.ListItems(ctx, &planpb.ListItemsRequest{
		PlanId: planID.String(),
		UserId: userID.String(),
	})
	if err != nil {
		return nil, fromPlanService("list items", err)
	}

	checklist := make([]ChecklistItem, 0, len(resp.Items))
	for _, it := range resp.Items {
		checklist = append(checklist, ChecklistItem{
			Description: it.Description,
			Scope:       it.Scope,
		})
	}

	return &PlanContext{
		PlanID:         planID,
		Focus:          plan.Focus,
		ChecklistItems: checklist,
	}, nil
}

// GetPlanProfile returns the plan's type and focus, which the day plan's
// prompt and rule gate read beside its items (FS-02JBZ R7, R17). plan-service
// answers NotFound for a plan it does not have.
func (c *PlanClient) GetPlanProfile(ctx context.Context, planID, userID uuid.UUID) (dayplan.PlanProfile, error) {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		return dayplan.PlanProfile{}, fmt.Errorf("insights: plan client: connect: %w", err)
	}
	plan, err := planpb.NewPlanServiceClient(conn).GetPlan(ctx, &planpb.GetPlanRequest{
		Id:     planID.String(),
		UserId: userID.String(),
	})
	if err != nil {
		return dayplan.PlanProfile{}, fromPlanService("get plan", err)
	}
	return toPlanProfile(plan), nil
}

func toPlanProfile(p *planpb.Plan) dayplan.PlanProfile {
	return dayplan.PlanProfile{Type: p.GetPlanType(), Focus: p.GetFocus()}
}

// ListPlanItems returns every item plan-service lists for the plan, with the
// fields guided suggestions and the day plan need (id, parent id, type, scope,
// done, archived, due date, last update). It does not filter: which items form
// the plan tree is the guided flow's rule (FS-DF4ZB R1), not this client's.
func (c *PlanClient) ListPlanItems(ctx context.Context, planID, userID uuid.UUID) ([]guided.PlanItem, error) {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		return nil, fmt.Errorf("insights: plan client: connect: %w", err)
	}
	resp, err := planpb.NewChecklistServiceClient(conn).ListItems(ctx, &planpb.ListItemsRequest{
		PlanId: planID.String(),
		UserId: userID.String(),
	})
	if err != nil {
		return nil, fromPlanService("list items", err)
	}
	return toPlanItems(resp.GetItems())
}

// toPlanItems maps plan-service items. A malformed id from plan-service is a
// server fault, not something to drop silently.
func toPlanItems(items []*planpb.ChecklistItem) ([]guided.PlanItem, error) {
	out := make([]guided.PlanItem, 0, len(items))
	for _, it := range items {
		id, err := uuid.Parse(it.GetId())
		if err != nil {
			return nil, fmt.Errorf("insights: plan client: item id %q: %w", it.GetId(), err)
		}
		item := guided.PlanItem{
			ID:          id,
			Description: it.GetDescription(),
			Type:        it.GetType(),
			Scope:       it.GetScope(),
			Done:        it.GetDone(),
			Archived:    it.GetArchived(),
			UpdatedAt:   it.GetUpdatedAt().AsTime(),
		}
		if it.GetParentId() != "" {
			parentID, err := uuid.Parse(it.GetParentId())
			if err != nil {
				return nil, fmt.Errorf("insights: plan client: parent id %q: %w", it.GetParentId(), err)
			}
			item.ParentID = &parentID
		}
		if it.DueDate != nil {
			due := it.GetDueDate().AsTime()
			item.DueDate = &due
		}
		out = append(out, item)
	}
	return out, nil
}

// Close releases the underlying gRPC connection.
func (c *PlanClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}
