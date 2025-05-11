package checklistitems

import (
	"context"
	"fmt"
	"time"

	"github.com/darkphotonKN/fireplace/internal/constants"
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
	GetAll(ctx context.Context, planId uuid.UUID, scope *string) ([]*models.ChecklistItem, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.ChecklistItem, error)
	CountItems(ctx context.Context) (int, error)
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) GetAll(ctx context.Context, planId uuid.UUID, scope *string) ([]*models.ChecklistItem, error) {
	// Validate scope if provided
	if scope != nil {
		if *scope != string(constants.ScopeLongterm) && *scope != string(constants.ScopeDaily) {
			return nil, fmt.Errorf("scope must be either 'daily' or 'longterm'")
		}
	}

	return s.repo.GetAll(ctx, planId, scope)
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*models.ChecklistItem, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req CreateReq, planID uuid.UUID) (*models.ChecklistItem, error) {
	// count number of current items in table
	count, err := s.repo.CountItems(ctx)

	if err != nil {
		return nil, err
	}

	// validate scope
	if req.Scope != nil {
		if *req.Scope != string(constants.ScopeLongterm) && *req.Scope != string(constants.ScopeDaily) {
			return nil, fmt.Errorf("Scope can only be either daily or longterm.")
		}
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
		t, err := time.Parse(time.RFC3339, *req.ScheduledTime)

		if err != nil {
			fmt.Printf("Error when parsing into time.RFC3339: %v\n", err)
			return err
		}

		fmt.Printf("Parsed time into time.RFC3339: %v\n", t)

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
