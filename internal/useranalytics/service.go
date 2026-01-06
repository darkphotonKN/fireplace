package useranalytics

import (
	"context"
	"fmt"
	"time"

	"github.com/darkphotonKN/fireplace/internal/models"
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
	data, err := s.checklistService.GetByUserID(ctx, userID)

	if err != nil {
		return nil, err
	}

	fmt.Printf("\nAll checklist data for user %s: %+v\n\n", userID, data)

	return s.repo.GetByUserAndDate(ctx, userID, date)
}
