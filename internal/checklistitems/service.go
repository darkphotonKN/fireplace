package checklistitems

import (
	"context"
	"fmt"
	"time"

	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/google/uuid"
)

type service struct {
	repo Repository
}

type Repository interface {
	Create(ctx context.Context, req CreateReq, planID uuid.UUID, sequenceNo int) (*models.ChecklistItem, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateReq) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetAll(ctx context.Context, planId uuid.UUID) ([]*models.ChecklistItem, error)
	CountItems(ctx context.Context) (int, error)
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) GetAll(ctx context.Context, planId uuid.UUID) ([]*models.ChecklistItem, error) {
	return s.repo.GetAll(ctx, planId)
}

func (s *service) Create(ctx context.Context, req CreateReq, planID uuid.UUID) (*models.ChecklistItem, error) {
	// count number of current items in table
	count, err := s.repo.CountItems(ctx)

	if err != nil {
		return nil, err
	}

	// add 1 to make new sequence
	return s.repo.Create(ctx, req, planID, count+1)
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req UpdateReq) error {
	// TODO: additional business logic for scheduled time
	// if req.ScheduledTime
	return s.repo.Update(ctx, id, req)
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) SetSchedule(ctx context.Context, id uuid.UUID, req SetScheduleReq) error {
	var updateData UpdateReq

	if req.ScheduledTime != nil {

		t, err := time.Parse("2006-01-02T15:04:05Z07:00", *req.ScheduledTime)

		if err != nil {
			return err
		}

		// format struct for updating scheduled time in database
		updateData = UpdateReq{
			ScheduledTime: &t,
		}

		// 2. validate the time, ensure it's in the future
		if t.Before(time.Now()) {
			return fmt.Errorf("scheduled time must be a datetime in the future")
		}
	}

	// 3. if time validation checks out, update the time
	return s.repo.Update(ctx, id, updateData)
}
