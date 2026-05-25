package calendargw

import (
	"context"
	"sync"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/calendar"
	"github.com/darkphotonKN/fireplace/common/discovery"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

const targetService = "calendar"

// Client wraps a long-lived gRPC connection to calendar-service. Cached
// ClientConn, lazy redial on Shutdown. Same shape as gateway/auth and
// gateway/plan to keep the gateway-side patterns uniform.
type Client struct {
	registry discovery.Registry
	mu       sync.Mutex
	conn     *grpc.ClientConn
}

func NewClient(registry discovery.Registry) *Client {
	return &Client{registry: registry}
}

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

func (c *Client) GetCalendar(ctx context.Context, planID, userID uuid.UUID, view, date string) (*GetCalendarResp, error) {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		return nil, err
	}
	cal := pb.NewCalendarServiceClient(conn)
	resp, err := cal.GetCalendar(ctx, &pb.GetCalendarRequest{
		PlanId: planID.String(),
		UserId: userID.String(),
		View:   view,
		Date:   date,
	})
	if err != nil {
		return nil, err
	}

	items := make([]CalendarItem, 0, len(resp.Items))
	for _, it := range resp.Items {
		id, _ := uuid.Parse(it.Id)
		items = append(items, CalendarItem{
			ID:          id,
			Description: it.Description,
			Scope:       it.Scope,
			Done:        it.Done,
			StartDate:   it.StartDate,
			DueDate:     it.DueDate,
		})
	}
	return &GetCalendarResp{
		PlanID:      resp.PlanId,
		View:        resp.View,
		WindowStart: resp.WindowStart,
		WindowEnd:   resp.WindowEnd,
		Items:       items,
	}, nil
}
