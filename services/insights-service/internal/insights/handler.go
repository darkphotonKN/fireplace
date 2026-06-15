package insights

import (
	"context"
	"fmt"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/insights"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"github.com/google/uuid"
)

// InsightsService is the internal service contract the gRPC handler depends on.
type InsightsService interface {
	GenerateSuggestion(ctx context.Context, planID, userID uuid.UUID) (string, error)
	GenerateDailySuggestions(ctx context.Context, planID, userID uuid.UUID) ([]string, error)
	SuggestVideos(ctx context.Context, planID, userID uuid.UUID) ([]Video, error)
}

type Handler struct {
	pb.UnimplementedInsightsServiceServer
	service InsightsService
}

func NewHandler(s InsightsService) *Handler {
	return &Handler{service: s}
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
		return nil, commongrpc.Fail(ctx, "insights: generate suggestion", err)
	}
	return &pb.GenerateSuggestionResponse{Suggestion: suggestion}, nil
}

func (h *Handler) GenerateDailySuggestions(ctx context.Context, req *pb.GenerateDailySuggestionsRequest) (*pb.GenerateDailySuggestionsResponse, error) {
	planID, userID, err := parseIDs(req.PlanId, req.UserId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "insights: generate daily suggestions", err)
	}

	suggestions, err := h.service.GenerateDailySuggestions(ctx, planID, userID)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "insights: generate daily suggestions", err)
	}
	return &pb.GenerateDailySuggestionsResponse{Suggestions: suggestions}, nil
}

func (h *Handler) SuggestVideos(ctx context.Context, req *pb.SuggestVideosRequest) (*pb.SuggestVideosResponse, error) {
	planID, userID, err := parseIDs(req.PlanId, req.UserId)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "insights: suggest videos", err)
	}

	videos, err := h.service.SuggestVideos(ctx, planID, userID)
	if err != nil {
		return nil, commongrpc.Fail(ctx, "insights: suggest videos", err)
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
