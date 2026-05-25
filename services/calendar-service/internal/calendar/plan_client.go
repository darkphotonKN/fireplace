package calendar

import (
	"context"
	"sync"
	"time"

	planpb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PlanClient is calendar-service's thin gRPC client to plan-service. Calendar
// owns no checklist data of its own — it asks plan-service for both ownership
// checks and date-window item reads.
//
// The ClientConn is cached and reused across calls (HTTP/2 multiplexing).
// Opening a fresh conn per call serialised badly under load — see gateway
// gateway/plan/client.go for the same pattern.
type PlanClient struct {
	registry discovery.Registry
	mu       sync.Mutex
	conn     *grpc.ClientConn
}

func NewPlanClient(registry discovery.Registry) *PlanClient {
	return &PlanClient{registry: registry}
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
		return err
	}
	plans := planpb.NewPlanServiceClient(conn)
	_, err = plans.AssertPlanOwnership(ctx, &planpb.AssertPlanOwnershipRequest{
		PlanId: planID.String(),
		UserId: userID.String(),
	})
	return err
}

// ListItemsInDateWindow returns checklist items for the plan whose date range
// intersects [windowStart, windowEnd].
func (c *PlanClient) ListItemsInDateWindow(ctx context.Context, planID, userID uuid.UUID, windowStart, windowEnd time.Time) ([]*planpb.ChecklistItem, error) {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		return nil, err
	}
	items := planpb.NewChecklistServiceClient(conn)
	resp, err := items.ListItemsInDateWindow(ctx, &planpb.ListItemsInDateWindowRequest{
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
