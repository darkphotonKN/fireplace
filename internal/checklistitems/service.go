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
	UpdateDates(ctx context.Context, id uuid.UUID, startDate, dueDate *time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetAll(ctx context.Context, scope *string) ([]*models.ChecklistItem, error)
	GetAllByPlanId(ctx context.Context, planId uuid.UUID, scope *string, upcoming *string) ([]*models.ChecklistItem, error)
	GetAllArchivedByPlanId(ctx context.Context, planId uuid.UUID, scope *string) ([]*models.ChecklistItem, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.ChecklistItem, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.ChecklistItem, error)
	CountItems(ctx context.Context) (int, error)
	BulkResetDailyItems(ctx context.Context) error
}

func NewService(repo Repository) *service {
	return &service{
		repo: repo,
	}
}

func (s *service) GetAll(ctx context.Context, scope *string) ([]*models.ChecklistItem, error) {
	return s.repo.GetAll(ctx, scope)
}

func (s *service) GetAllByPlanId(ctx context.Context, planId uuid.UUID, scope *string, upcoming *string) ([]*models.ChecklistItem, error) {
	if scope != nil {
		if *scope != string(constants.ScopeLongterm) && *scope != string(constants.ScopeDaily) {
			return nil, fmt.Errorf("scope must be either 'daily' or 'longterm'")
		}
	}

	if upcoming != nil {
		if *upcoming != string(constants.UpcomingWeek) && *upcoming != string(constants.UpcomingMonth) {
			return nil, fmt.Errorf("Upcoming needs to be either 'week' or 'month'")
		}
	}
	return s.repo.GetAllByPlanId(ctx, planId, scope, upcoming)
}

func (s *service) GetAllArchivedByPlanId(ctx context.Context, planId uuid.UUID, scope *string) ([]*models.ChecklistItem, error) {
	// Validate scope if provided
	if scope != nil {
		if *scope != string(constants.ScopeLongterm) && *scope != string(constants.ScopeDaily) {
			return nil, fmt.Errorf("scope must be either 'daily' or 'longterm'")
		}
	}

	return s.repo.GetAllArchivedByPlanId(ctx, planId, scope)
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
	return s.repo.Update(ctx, id, req)
}

// UpdateDates merges the request with the current row, validates
// start_date <= due_date when both resolve to non-null, and persists.
// Returns the updated item.
func (s *service) UpdateDates(ctx context.Context, id uuid.UUID, req UpdateDatesReq) (*models.ChecklistItem, error) {
	if !req.StartDate.Present && !req.DueDate.Present {
		// No-op — return current item as-is.
		return s.repo.GetByID(ctx, id)
	}

	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Resolve final values: present overrides current; absent preserves current.
	var finalStart, finalDue *time.Time
	switch {
	case req.StartDate.Present && req.StartDate.Valid:
		v := req.StartDate.Value
		finalStart = &v
	case req.StartDate.Present && !req.StartDate.Valid:
		finalStart = nil
	default:
		finalStart = current.StartDate
	}
	switch {
	case req.DueDate.Present && req.DueDate.Valid:
		v := req.DueDate.Value
		finalDue = &v
	case req.DueDate.Present && !req.DueDate.Valid:
		finalDue = nil
	default:
		finalDue = current.DueDate
	}

	if finalStart != nil && finalDue != nil && finalStart.After(*finalDue) {
		return nil, fmt.Errorf("%w: start_date must be on or before due_date", constants.ErrInvalidInput)
	}

	if err := s.repo.UpdateDates(ctx, id, finalStart, finalDue); err != nil {
		return nil, err
	}

	current.StartDate = finalStart
	current.DueDate = finalDue
	return current, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

/**
* Resets the daily items from done true to false so that they can be repeated.
**/
func (s *service) ResetDailyItems(ctx context.Context) error {
	// NOTE: old implementation
	// daily := string(constants.ScopeDaily)
	//
	// items, err := s.GetAll(ctx, &daily)
	// if err != nil {
	// 	return err
	// }
	//
	// for _, item := range items {
	// 	// update done to false, if already completed
	// 	if item.Done {
	// 		notDone := false
	// 		err := s.Update(ctx, item.ID, UpdateReq{
	// 			Done: &notDone,
	// 		})
	//
	// 		if err != nil {
	// 			return err
	// 		}
	// 	}
	// }
	//
	// return nil

	return s.repo.BulkResetDailyItems(ctx)
}

func (s *service) Archive(ctx context.Context, id uuid.UUID) error {
	archived := true
	return s.repo.Update(ctx, id, UpdateReq{
		Archived: &archived,
	})
}

func (s *service) GetUpcoming(ctx context.Context, planId uuid.UUID) ([]*models.ChecklistItem, error) {
	upcomingStr := string(constants.UpcomingWeek)
	items, err := s.GetAllByPlanId(ctx, planId, nil, &upcomingStr)

	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s *service) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.ChecklistItem, error) {
	return s.repo.GetByUserID(ctx, userID)
}
