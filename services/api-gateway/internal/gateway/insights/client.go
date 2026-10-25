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
	"fmt"
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

// GetDayPlan returns the caller's day plan for the plan and date, or its
// not_planned state (FS-02JBZ R14). Domain refusals arrive as gRPC status with
// an ErrorInfo code and are returned untouched for apierr to map.
func (c *Client) GetDayPlan(ctx context.Context, planID, userID uuid.UUID, date string) (insightstransport.DayPlanResponse, error) {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		return insightstransport.DayPlanResponse{}, err
	}
	resp, err := pb.NewInsightsServiceClient(conn).GetDayPlan(ctx, &pb.GetDayPlanRequest{
		PlanId: planID.String(),
		UserId: userID.String(),
		Date:   date,
	})
	if err != nil {
		return insightstransport.DayPlanResponse{}, err
	}
	return toDayPlan(resp.GetDayPlan())
}

// PlanMyDay draws or redraws the caller's day plan for the plan and date
// (FS-02JBZ R13). The cap refusal arrives as ResourceExhausted with an
// ErrorInfo code and is returned untouched for apierr to map (429).
func (c *Client) PlanMyDay(ctx context.Context, planID, userID uuid.UUID, date string) (insightstransport.DayPlanResponse, error) {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		return insightstransport.DayPlanResponse{}, err
	}
	resp, err := pb.NewInsightsServiceClient(conn).PlanMyDay(ctx, &pb.PlanMyDayRequest{
		PlanId: planID.String(),
		UserId: userID.String(),
		Date:   date,
	})
	if err != nil {
		return insightstransport.DayPlanResponse{}, err
	}
	return toDayPlan(resp.GetDayPlan())
}

// TickDayPlanEntry ticks an entry of the caller's day plan done for its date
// (FS-02JBZ R25), binding an unbound new step to itemID. Refusals arrive as
// gRPC status (DAY_PLAN_ENTRY_NOT_TICKABLE as ErrorInfo on
// FailedPrecondition) and are returned untouched for apierr to map.
func (c *Client) TickDayPlanEntry(ctx context.Context, entryID, userID uuid.UUID, itemID *uuid.UUID) (insightstransport.DayPlanEntryResponse, error) {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		return insightstransport.DayPlanEntryResponse{}, err
	}
	req := &pb.TickDayPlanEntryRequest{EntryId: entryID.String(), UserId: userID.String()}
	if itemID != nil {
		id := itemID.String()
		req.ItemId = &id
	}
	resp, err := pb.NewInsightsServiceClient(conn).TickDayPlanEntry(ctx, req)
	if err != nil {
		return insightstransport.DayPlanEntryResponse{}, err
	}
	return toDayPlanEntry(resp.GetEntry())
}

// UntickDayPlanEntry removes the day tick of an entry of the caller's day plan
// (FS-02JBZ R29).
func (c *Client) UntickDayPlanEntry(ctx context.Context, entryID, userID uuid.UUID) error {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		return err
	}
	_, err = pb.NewInsightsServiceClient(conn).UntickDayPlanEntry(ctx, &pb.UntickDayPlanEntryRequest{
		EntryId: entryID.String(), UserId: userID.String(),
	})
	return err
}

// SetItemRevisions switches the revision ladder for one item of the caller's
// plan (FS-02JBZ R47, R48). NotFound (plan or item) is returned untouched for
// apierr to map.
func (c *Client) SetItemRevisions(ctx context.Context, planID, userID, itemID uuid.UUID, enabled bool) error {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		return err
	}
	_, err = pb.NewInsightsServiceClient(conn).SetItemRevisions(ctx, &pb.SetItemRevisionsRequest{
		PlanId: planID.String(), UserId: userID.String(), ItemId: itemID.String(), Enabled: enabled,
	})
	return err
}

// toDayPlan translates the proto day plan into the gateway's transport mirror
// (ADR-0003 §3). A malformed id from insights-service is a server fault.
func toDayPlan(p *pb.DayPlan) (insightstransport.DayPlanResponse, error) {
	planID, err := uuid.Parse(p.GetPlanId())
	if err != nil {
		return insightstransport.DayPlanResponse{}, fmt.Errorf("insights gateway: day plan id %q: %w", p.GetPlanId(), err)
	}
	out := insightstransport.DayPlanResponse{
		PlanID:             planID,
		Date:               p.GetDate(),
		Status:             p.GetStatus(),
		Source:             p.GetSource(),
		RefreshesRemaining: int(p.GetRefreshesRemaining()),
		Groups:             make([]insightstransport.DayPlanGroupResponse, 0, len(p.GetGroups())),
	}
	if p.GeneratedAt != nil {
		at := p.GetGeneratedAt().AsTime()
		out.GeneratedAt = &at
	}
	for _, g := range p.GetGroups() {
		group := insightstransport.DayPlanGroupResponse{
			Label: g.GetLabel(), Entries: make([]insightstransport.DayPlanEntryResponse, 0, len(g.GetEntries())),
		}
		for _, e := range g.GetEntries() {
			entry, err := toDayPlanEntry(e)
			if err != nil {
				return insightstransport.DayPlanResponse{}, err
			}
			group.Entries = append(group.Entries, entry)
		}
		out.Groups = append(out.Groups, group)
	}
	return out, nil
}

func toDayPlanEntry(e *pb.DayPlanEntry) (insightstransport.DayPlanEntryResponse, error) {
	id, err := uuid.Parse(e.GetId())
	if err != nil {
		return insightstransport.DayPlanEntryResponse{}, fmt.Errorf("insights gateway: day plan entry id %q: %w", e.GetId(), err)
	}
	out := insightstransport.DayPlanEntryResponse{
		ID: id, Kind: e.GetKind(), Text: e.GetText(), Reason: e.GetReason(), Ticked: e.GetTicked(),
	}
	if out.ItemID, err = optionalUUID("item id", e.ItemId); err != nil {
		return insightstransport.DayPlanEntryResponse{}, err
	}
	if out.ParentItemID, err = optionalUUID("parent item id", e.ParentItemId); err != nil {
		return insightstransport.DayPlanEntryResponse{}, err
	}
	if e.Minutes != nil {
		m := int(e.GetMinutes())
		out.Minutes = &m
	}
	if e.Revision != nil {
		r := int(e.GetRevision())
		out.Revision = &r
	}
	if e.RevisionsEnabled != nil {
		enabled := e.GetRevisionsEnabled()
		out.RevisionsEnabled = &enabled
	}
	return out, nil
}

// optionalUUID parses an optional proto id; unset stays nil.
func optionalUUID(field string, s *string) (*uuid.UUID, error) {
	if s == nil {
		return nil, nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil, fmt.Errorf("insights gateway: day plan %s %q: %w", field, *s, err)
	}
	return &id, nil
}
