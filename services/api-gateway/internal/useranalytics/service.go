package useranalytics

import (
	"context"
	"fmt"
	"time"

	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/models"
	"github.com/google/uuid"
)

type service struct {
	repo             Repository
	checklistService UserAnalyticsCheckListService
}

type UserAnalyticsCheckListService interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.ChecklistItem, error)
}

func NewService(repo Repository, checklistService UserAnalyticsCheckListService) *service {
	return &service{repo: repo, checklistService: checklistService}
}

func (s *service) GetUserAnalytics(ctx context.Context, userID uuid.UUID, date time.Time) (*UserAnalytics, error) {

	// grab all the checklist data for the user
	if _, err := s.checklistService.GetByUserID(ctx, userID); err != nil {
		return nil, fmt.Errorf("useranalytics: get analytics: load checklist data: %w", err)
	}

	analytics, err := s.repo.GetByUserAndDate(ctx, userID, date)
	if err != nil {
		return nil, fmt.Errorf("useranalytics: get analytics: %w", err)
	}
	return analytics, nil
}
