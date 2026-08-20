// Package insightsgw is the gateway's gRPC client for insights-service.
//
// It replaces the in-process insights implementation that lived in
// internal/insights — the last piece of the AI-suggestion domain still
// physically inside the gateway after the strangler split. The HTTP surface
// these calls sit behind was serialized first (FS-0004 I-0019), so the contract
// was established before this rewrite landed underneath it (ADR-0002 §6).
package insightsgw

import (
	"context"
	"sync"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/insights"
	"github.com/darkphotonKN/fireplace/common/discovery"
	insightstransport "github.com/darkphotonKN/fireplace/services/api-gateway/internal/insights"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

const targetService = "insights"

// Client wraps a long-lived gRPC connection to insights-service. Cached
// ClientConn, lazy redial on Shutdown — same shape as gateway/auth,
// gateway/plan and gateway/calendar.
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

// GenerateSuggestions returns one actionable next checklist item.
//
// userID is now carried where the in-process implementation took only a plan
// id: insights-service fetches plan context through plan-service, which
// enforces ownership. The gateway asserted nothing here before.
func (c *Client) GenerateSuggestions(ctx context.Context, planID, userID uuid.UUID) (string, error) {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		return "", err
	}
	resp, err := pb.NewInsightsServiceClient(conn).GenerateSuggestion(ctx, &pb.GenerateSuggestionRequest{
		PlanId: planID.String(),
		UserId: userID.String(),
	})
	if err != nil {
		return "", err
	}
	return resp.GetSuggestion(), nil
}

// GenerateDailySuggestions returns the daily set derived from long-term items.
func (c *Client) GenerateDailySuggestions(ctx context.Context, planID, userID uuid.UUID) ([]string, error) {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := pb.NewInsightsServiceClient(conn).GenerateDailySuggestions(ctx, &pb.GenerateDailySuggestionsRequest{
		PlanId: planID.String(),
		UserId: userID.String(),
	})
	if err != nil {
		return nil, err
	}
	return resp.GetSuggestions(), nil
}

// SuggestVideos returns recommended learning videos for the plan.
//
// The proto Video is translated into the gateway's transport mirror here rather
// than leaking a protobuf message into the serialized surface (ADR-0003 §3).
func (c *Client) SuggestVideos(ctx context.Context, planID, userID uuid.UUID) ([]insightstransport.VideoSuggestionResponse, error) {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := pb.NewInsightsServiceClient(conn).SuggestVideos(ctx, &pb.SuggestVideosRequest{
		PlanId: planID.String(),
		UserId: userID.String(),
	})
	if err != nil {
		return nil, err
	}

	out := make([]insightstransport.VideoSuggestionResponse, 0, len(resp.GetVideos()))
	for _, v := range resp.GetVideos() {
		out = append(out, insightstransport.VideoSuggestionResponse{
			Title:       v.GetTitle(),
			URL:         v.GetUrl(),
			Source:      v.GetSource(),
			Type:        v.GetType(),
			Description: v.GetDescription(),
		})
	}
	return out, nil
}
