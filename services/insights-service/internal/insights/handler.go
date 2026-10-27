package insights

import (
	"context"
	"errors"
	"fmt"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/insights"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/common/errcode"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/ai"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/dayplan"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// InsightsService is the internal service contract the gRPC handler depends on.
type InsightsService interface {
	GenerateSuggestion(ctx context.Context, planID, userID uuid.UUID) (string, error)
	SuggestVideos(ctx context.Context, planID, userID uuid.UUID) ([]Video, error)
}

// DayPlanService is the day plan (FS-02JBZ) the handler serves: a read that
// never generates, Plan my day, the day tick on an entry, and the per-item
// revisions switch.
type DayPlanService interface {
	GetDayPlan(ctx context.Context, planID, userID uuid.UUID, date string) (dayplan.DayPlan, error)
	PlanMyDay(ctx context.Context, planID, userID uuid.UUID, date string) (dayplan.DayPlan, error)
	TickEntry(ctx context.Context, entryID, userID uuid.UUID, itemID *uuid.UUID) (dayplan.Entry, error)
	UntickEntry(ctx context.Context, entryID, userID uuid.UUID) error
	SetItemRevisions(ctx context.Context, planID, userID, itemID uuid.UUID, enabled bool) error
}

type Handler struct {
	pb.UnimplementedInsightsServiceServer
	service  InsightsService
	dayPlans DayPlanService
}

func NewHandler(s InsightsService, dayPlans DayPlanService) *Handler {
	return &Handler{service: s, dayPlans: dayPlans}
}

// badUUID builds an InvalidArgument-class domain error for a malformed id so it
// flows through the shared mapper like any other domain error.
func badUUID(field, value string) error {
	return fmt.Errorf("%w: %s %q", commonconstants.ErrUUIDCouldNotBeParsed, field, value)
}

// parseIDs parses the (plan_id, user_id) pair common to every request.
func parseIDs(planIDStr, userIDStr string) (uuid.UUID, uuid.UUID, error) {
	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, badUUID("plan_id", planIDStr)
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, badUUID("user_id", userIDStr)
	}
	return planID, userID, nil
}

func (h *Handler) GenerateSuggestion(ctx context.Context, req *pb.GenerateSuggestionRequest) (*pb.GenerateSuggestionResponse, error) {
	planID, userID, err := parseIDs(req.PlanId, req.UserId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "insights: generate suggestion", err)
	}

	suggestion, err := h.service.GenerateSuggestion(ctx, planID, userID)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "insights: generate suggestion", generationStatus(err))
	}
	return &pb.GenerateSuggestionResponse{Suggestion: suggestion}, nil
}

func (h *Handler) SuggestVideos(ctx context.Context, req *pb.SuggestVideosRequest) (*pb.SuggestVideosResponse, error) {
	planID, userID, err := parseIDs(req.PlanId, req.UserId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "insights: suggest videos", err)
	}

	videos, err := h.service.SuggestVideos(ctx, planID, userID)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "insights: suggest videos", generationStatus(err))
	}

	out := make([]*pb.Video, 0, len(videos))
	for _, v := range videos {
		out = append(out, &pb.Video{
			Title:       v.Title,
			Url:         v.URL,
			Source:      v.Source,
			Type:        v.Type,
			Description: v.Description,
		})
	}
	return &pb.SuggestVideosResponse{Videos: out}, nil
}

// GetDayPlan serves FS-02JBZ's read: the stored day plan or not_planned,
// never a generation (R14).
func (h *Handler) GetDayPlan(ctx context.Context, req *pb.GetDayPlanRequest) (*pb.GetDayPlanResponse, error) {
	const op = "insights: get day plan"
	planID, userID, err := parseIDs(req.GetPlanId(), req.GetUserId())
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	p, err := h.dayPlans.GetDayPlan(ctx, planID, userID, req.GetDate())
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, dayPlanStatus(err))
	}
	return &pb.GetDayPlanResponse{DayPlan: toPBDayPlan(p)}, nil
}

// PlanMyDay serves FS-02JBZ's draw (R13): its refusals are the date rule and
// the refresh cap (R15, R22). A failed generation is not one of them: the day
// plan falls back to a rules plan (R36).
func (h *Handler) PlanMyDay(ctx context.Context, req *pb.PlanMyDayRequest) (*pb.PlanMyDayResponse, error) {
	const op = "insights: plan my day"
	planID, userID, err := parseIDs(req.GetPlanId(), req.GetUserId())
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	p, err := h.dayPlans.PlanMyDay(ctx, planID, userID, req.GetDate())
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, dayPlanStatus(err))
	}
	return &pb.PlanMyDayResponse{DayPlan: toPBDayPlan(p)}, nil
}

// TickDayPlanEntry serves FS-02JBZ's day tick (R25-R28, R30).
func (h *Handler) TickDayPlanEntry(ctx context.Context, req *pb.TickDayPlanEntryRequest) (*pb.TickDayPlanEntryResponse, error) {
	const op = "insights: tick day plan entry"
	entryID, userID, err := parseEntryIDs(req.GetEntryId(), req.GetUserId())
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	var itemID *uuid.UUID
	if req.ItemId != nil {
		id, err := uuid.Parse(req.GetItemId())
		if err != nil {
			return nil, commongrpc.Fail(ctx, op, badUUID("item_id", req.GetItemId()))
		}
		itemID = &id
	}
	e, err := h.dayPlans.TickEntry(ctx, entryID, userID, itemID)
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, tickStatus(err))
	}
	return &pb.TickDayPlanEntryResponse{Entry: toPBEntry(e)}, nil
}

// UntickDayPlanEntry serves FS-02JBZ's untick (R29, R30).
func (h *Handler) UntickDayPlanEntry(ctx context.Context, req *pb.UntickDayPlanEntryRequest) (*pb.UntickDayPlanEntryResponse, error) {
	const op = "insights: untick day plan entry"
	entryID, userID, err := parseEntryIDs(req.GetEntryId(), req.GetUserId())
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	if err := h.dayPlans.UntickEntry(ctx, entryID, userID); err != nil {
		return nil, commongrpc.Fail(ctx, op, tickStatus(err))
	}
	return &pb.UntickDayPlanEntryResponse{}, nil
}

// SetItemRevisions serves FS-02JBZ's revisions switch (R47, R48). A plan the
// caller cannot access and an item not in the plan are both NotFound.
func (h *Handler) SetItemRevisions(ctx context.Context, req *pb.SetItemRevisionsRequest) (*pb.SetItemRevisionsResponse, error) {
	const op = "insights: set item revisions"
	planID, userID, err := parseIDs(req.GetPlanId(), req.GetUserId())
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	itemID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, commongrpc.Fail(ctx, op, badUUID("item_id", req.GetItemId()))
	}
	if err := h.dayPlans.SetItemRevisions(ctx, planID, userID, itemID, req.GetEnabled()); err != nil {
		return nil, commongrpc.Fail(ctx, op, err)
	}
	return &pb.SetItemRevisionsResponse{}, nil
}

// parseEntryIDs parses the (entry_id, user_id) pair of a day tick request.
func parseEntryIDs(entryIDStr, userIDStr string) (uuid.UUID, uuid.UUID, error) {
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, badUUID("entry_id", entryIDStr)
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, badUUID("user_id", userIDStr)
	}
	return entryID, userID, nil
}

// tickStatus gives a day tick's refusals their wire form: an entry that cannot
// be ticked is FailedPrecondition with DAY_PLAN_ENTRY_NOT_TICKABLE (R26, 400
// at the gateway); the date rule is the day plan's (R30). An unknown entry is
// a NotFound for the shared mapper.
func tickStatus(err error) error {
	if errors.Is(err, dayplan.ErrEntryNotTickable) {
		return commongrpc.WithCode(codes.FailedPrecondition, errcode.DayPlanEntryNotTickable, err.Error())
	}
	return dayPlanStatus(err)
}

// generationStatus gives the legacy generators' budget refusal its wire form:
// ResourceExhausted with LLM_BUDGET_EXHAUSTED as ErrorInfo, which the gateway
// maps to 429 (FS-02JBZ R33). Anything else goes to the shared mapper as is.
func generationStatus(err error) error {
	if errors.Is(err, ai.ErrBudgetExhausted) {
		return commongrpc.WithCode(codes.ResourceExhausted, errcode.LLMBudgetExhausted, err.Error())
	}
	return err
}

// dayPlanStatus gives the day plan's own refusals their wire form: the date
// and refresh-cap refusals carry their domain codes as ErrorInfo (FS-DF4ZB R17,
// R21, R52). Anything else goes to the shared mapper as is. A failed
// generation is never one of them: the day plan falls back to rules (R36).
func dayPlanStatus(err error) error {
	switch {
	case errors.Is(err, guided.ErrDateOutOfRange):
		return commongrpc.WithCode(codes.InvalidArgument, errcode.BriefDateOutOfRange, err.Error())
	case errors.Is(err, guided.ErrRefreshLimitReached):
		return commongrpc.WithCode(codes.ResourceExhausted, errcode.BriefRefreshLimitReached, err.Error())
	default:
		return err
	}
}

// Day plan status values on the wire (FS-02JBZ §API surface).
const (
	dayPlanPlanned    = "planned"
	dayPlanNotPlanned = "not_planned"
)

// toPBDayPlan maps a day plan. Source and generated_at are set iff it is
// planned; groups are never nil.
func toPBDayPlan(p dayplan.DayPlan) *pb.DayPlan {
	out := &pb.DayPlan{
		PlanId:             p.Key.PlanID.String(),
		Date:               p.Key.Date.Format(guided.DateLayout),
		Status:             dayPlanNotPlanned,
		RefreshesRemaining: int32(p.RefreshesRemaining),
		Groups:             make([]*pb.DayPlanGroup, 0, len(p.Groups)),
	}
	if p.Planned {
		source := string(p.Source)
		out.Status, out.Source, out.GeneratedAt = dayPlanPlanned, &source, timestamppb.New(p.GeneratedAt)
	}
	for _, g := range p.Groups {
		group := &pb.DayPlanGroup{Label: g.Label, Entries: make([]*pb.DayPlanEntry, 0, len(g.Entries))}
		for _, e := range g.Entries {
			group.Entries = append(group.Entries, toPBEntry(e))
		}
		out.Groups = append(out.Groups, group)
	}
	return out
}

func toPBEntry(e dayplan.Entry) *pb.DayPlanEntry {
	out := &pb.DayPlanEntry{Id: e.ID.String(), Kind: string(e.Kind), Text: e.Text, Reason: e.Reason, Ticked: e.Ticked}
	if e.ItemID != nil {
		id := e.ItemID.String()
		out.ItemId = &id
	}
	if e.ParentItemID != nil {
		id := e.ParentItemID.String()
		out.ParentItemId = &id
	}
	if e.Minutes != nil {
		m := int32(*e.Minutes)
		out.Minutes = &m
	}
	if e.Revision != nil {
		r := int32(*e.Revision)
		out.Revision = &r
	}
	if e.RevisionsEnabled != nil {
		enabled := *e.RevisionsEnabled
		out.RevisionsEnabled = &enabled
	}
	return out
}
