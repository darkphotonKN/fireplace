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
	GetAllByPlanId(ctx context.Context, planId uuid.UUID, scope *string, itemType *string, upcoming *string) ([]*models.ChecklistItem, error)
	GetAllArchivedByPlanId(ctx context.Context, planId uuid.UUID, scope *string) ([]*models.ChecklistItem, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.ChecklistItem, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.ChecklistItem, error)
	HasChildren(ctx context.Context, id uuid.UUID) (bool, error)
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

func (s *service) GetAllByPlanId(ctx context.Context, planId uuid.UUID, scope *string, itemType *string, upcoming *string) ([]*models.ChecklistItem, error) {
	if scope != nil {
		if *scope != string(constants.ScopeLongterm) && *scope != string(constants.ScopeDaily) {
			return nil, fmt.Errorf("%w: scope must be either 'daily' or 'longterm'", constants.ErrInvalidInput)
		}
	}

	if itemType != nil {
		if *itemType != "task" && *itemType != "note" {
			return nil, fmt.Errorf("%w: type must be either 'task' or 'note'", constants.ErrInvalidInput)
		}
	}

	if upcoming != nil {
		if *upcoming != string(constants.UpcomingWeek) && *upcoming != string(constants.UpcomingMonth) {
			return nil, fmt.Errorf("%w: upcoming must be either 'week' or 'month'", constants.ErrInvalidInput)
		}
	}
	return s.repo.GetAllByPlanId(ctx, planId, scope, itemType, upcoming)
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
	if req.Scope != nil {
		if *req.Scope != string(constants.ScopeLongterm) && *req.Scope != string(constants.ScopeDaily) {
			return nil, fmt.Errorf("%w: scope must be 'daily' or 'longterm'", constants.ErrInvalidInput)
		}
		// NOTE: the PRD called for rejecting manual POST scope='daily' so dailies
		// could only enter via the AI suggestion accept flow. That rule is
		// unenforceable at this layer — the AI accept flow also POSTs here,
		// and there is no signal on the request to distinguish it from a
		// manual call. The FE's dailyAIOnly prop hides the manual add form;
		// that is the real UX gate. Re-add a guard here only if/when the AI
		// accept flow moves to a dedicated endpoint.
	}

	if req.Type != nil && *req.Type != "task" && *req.Type != "note" {
		return nil, fmt.Errorf("%w: type must be 'task' or 'note'", constants.ErrInvalidInput)
	}

	if req.ParentID != nil {
		if err := s.validateParentID(ctx, planID, *req.ParentID); err != nil {
			return nil, err
		}
	}

	count, err := s.repo.CountItems(ctx)
	if err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, req, planID, count+1)
}

// validateParentID verifies that parentID exists, belongs to the same plan,
// and is itself a top-level item (parent_id IS NULL).
func (s *service) validateParentID(ctx context.Context, planID, parentID uuid.UUID) error {
	parent, err := s.repo.GetByID(ctx, parentID)
	if err != nil {
		return fmt.Errorf("%w: parent not found", constants.ErrInvalidInput)
	}
	if parent == nil {
		return fmt.Errorf("%w: parent not found", constants.ErrInvalidInput)
	}
	if parent.PlanID != planID {
		return fmt.Errorf("%w: parent belongs to a different plan", constants.ErrInvalidInput)
	}
	if parent.ParentID != nil {
		return fmt.Errorf("%w: cannot nest under a child item (two-tier max)", constants.ErrInvalidInput)
	}
	return nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req UpdateReq) error {
	// Validate type value.
	if req.Type != nil && *req.Type != "task" && *req.Type != "note" {
		return fmt.Errorf("%w: type must be 'task' or 'note'", constants.ErrInvalidInput)
	}

	// A parent with children can't become a note — notes don't structurally
	// contain subtasks. Reject the transition before mutating anything.
	if req.Type != nil && *req.Type == "note" {
		hasKids, err := s.repo.HasChildren(ctx, id)
		if err != nil {
			return err
		}
		if hasKids {
			return fmt.Errorf("%w: cannot convert a parent with children into a note", constants.ErrInvalidInput)
		}
	}

	// On task→note transition, force done=false in the same UPDATE so the
	// chk_note_not_done CHECK isn't tripped. Only fetches the current row when
	// we actually need to know its done state.
	if req.Type != nil && *req.Type == "note" && req.Done == nil {
		current, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if current != nil && current.Done {
			f := false
			req.Done = &f
		}
	}

	// parent_id handling: validate target row exists in same plan and is
	// itself top-level; reject re-parenting a row that has children.
	if req.ParentID.Present && req.ParentID.Valid {
		current, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return constants.ErrNotFound
		}
		if err := s.validateParentID(ctx, current.PlanID, req.ParentID.Value); err != nil {
			return err
		}
		hasKids, err := s.repo.HasChildren(ctx, id)
		if err != nil {
			return err
		}
		if hasKids {
			return fmt.Errorf("%w: cannot re-parent a row that has children (would push them past tier 2)", constants.ErrInvalidInput)
		}
	}

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
	items, err := s.GetAllByPlanId(ctx, planId, nil, nil, &upcomingStr)

	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s *service) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.ChecklistItem, error) {
	return s.repo.GetByUserID(ctx, userID)
}
