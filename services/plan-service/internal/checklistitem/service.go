package checklistitem

import (
	"context"
	"fmt"
	"time"

	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, in CreateItemInput, sequenceNo int) (*Item, error)
	Update(ctx context.Context, in UpdateItemInput) error
	UpdateDates(ctx context.Context, id uuid.UUID, startDate, dueDate *time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByPlanID(ctx context.Context, in ListItemsInput) ([]*Item, error)
	ListArchivedByPlanID(ctx context.Context, planID uuid.UUID, scope *string) ([]*Item, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Item, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Item, error)
	HasChildren(ctx context.Context, id uuid.UUID) (bool, error)
	CountItems(ctx context.Context) (int, error)
	BulkResetDailyItems(ctx context.Context) (int64, error)
}

type service struct {
	repo      Repository
	publishCh commonbroker.Publisher
}

func NewService(repo Repository, publishCh commonbroker.Publisher) *service {
	return &service{repo: repo, publishCh: publishCh}
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*Item, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) ListByPlanID(ctx context.Context, in ListItemsInput) ([]*Item, error) {
	if in.Scope != nil && *in.Scope != ScopeDaily && *in.Scope != ScopeLongterm {
		return nil, fmt.Errorf("%w: scope must be 'daily' or 'longterm'", commonconstants.ErrInvalidInput)
	}
	if in.Type != nil && *in.Type != TypeTask && *in.Type != TypeNote {
		return nil, fmt.Errorf("%w: type must be 'task' or 'note'", commonconstants.ErrInvalidInput)
	}
	if in.Upcoming != nil && *in.Upcoming != UpcomingWeek && *in.Upcoming != UpcomingMonth {
		return nil, fmt.Errorf("%w: upcoming must be 'week' or 'month'", commonconstants.ErrInvalidInput)
	}
	return s.repo.ListByPlanID(ctx, in)
}

func (s *service) ListArchivedByPlanID(ctx context.Context, planID uuid.UUID, scope *string) ([]*Item, error) {
	if scope != nil && *scope != ScopeDaily && *scope != ScopeLongterm {
		return nil, fmt.Errorf("%w: scope must be 'daily' or 'longterm'", commonconstants.ErrInvalidInput)
	}
	return s.repo.ListArchivedByPlanID(ctx, planID, scope)
}

func (s *service) ListUpcoming(ctx context.Context, planID uuid.UUID) ([]*Item, error) {
	week := UpcomingWeek
	return s.ListByPlanID(ctx, ListItemsInput{PlanID: planID, Upcoming: &week})
}

func (s *service) Create(ctx context.Context, in CreateItemInput) (*Item, error) {
	if in.Scope != nil && *in.Scope != ScopeDaily && *in.Scope != ScopeLongterm {
		return nil, fmt.Errorf("%w: scope must be 'daily' or 'longterm'", commonconstants.ErrInvalidInput)
	}
	if in.Type != nil && *in.Type != TypeTask && *in.Type != TypeNote {
		return nil, fmt.Errorf("%w: type must be 'task' or 'note'", commonconstants.ErrInvalidInput)
	}
	if in.ParentID != nil {
		if err := s.validateParent(ctx, in.PlanID, *in.ParentID); err != nil {
			return nil, err
		}
	}

	count, err := s.repo.CountItems(ctx)
	if err != nil {
		return nil, err
	}
	item, err := s.repo.Create(ctx, in, count+1)
	if err != nil {
		return nil, err
	}
	return item, nil
}

// validateParent enforces the two-tier rule: parent must exist, belong to the
// same plan, and itself have no parent.
func (s *service) validateParent(ctx context.Context, planID, parentID uuid.UUID) error {
	parent, err := s.repo.GetByID(ctx, parentID)
	if err != nil {
		return fmt.Errorf("%w: parent not found", commonconstants.ErrInvalidInput)
	}
	if parent.PlanID != planID {
		return fmt.Errorf("%w: parent belongs to a different plan", commonconstants.ErrInvalidInput)
	}
	if parent.ParentID != nil {
		return fmt.Errorf("%w: cannot nest under a child item (two-tier max)", commonconstants.ErrInvalidInput)
	}
	return nil
}

func (s *service) Update(ctx context.Context, in UpdateItemInput) (*Item, error) {
	if in.Type != nil && *in.Type != TypeTask && *in.Type != TypeNote {
		return nil, fmt.Errorf("%w: type must be 'task' or 'note'", commonconstants.ErrInvalidInput)
	}

	// task→note transition: reject if it has children (notes can't be parents).
	if in.Type != nil && *in.Type == TypeNote {
		hasKids, err := s.repo.HasChildren(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		if hasKids {
			return nil, fmt.Errorf("%w: cannot convert a parent with children into a note", commonconstants.ErrInvalidInput)
		}
	}

	// task→note: force done=false to satisfy chk_note_not_done.
	if in.Type != nil && *in.Type == TypeNote && in.Done == nil {
		current, err := s.repo.GetByID(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		if current.Done {
			f := false
			in.Done = &f
		}
	}

	// Re-parent (indent): validate target is in same plan, is top-level, and
	// this row has no children of its own (would push past tier 2).
	if in.SetParent && in.ParentID != nil {
		current, err := s.repo.GetByID(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		if err := s.validateParent(ctx, current.PlanID, *in.ParentID); err != nil {
			return nil, err
		}
		hasKids, err := s.repo.HasChildren(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		if hasKids {
			return nil, fmt.Errorf("%w: cannot re-parent a row with children", commonconstants.ErrInvalidInput)
		}
	}

	if err := s.repo.Update(ctx, in); err != nil {
		return nil, err
	}

	updated, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		return nil, err
	}

	// Emit completion event when an Update sets done=true.
	if in.Done != nil && *in.Done {
		s.PublishItemCompleted(ctx, updated)
	}
	if in.Done != nil && !*in.Done {
		s.PublishItemUncompleted(ctx, updated)
	}
	return updated, nil
}

func (s *service) UpdateDates(ctx context.Context, in UpdateDatesInput) (*Item, error) {
	if !in.SetStart && !in.SetDue {
		return s.repo.GetByID(ctx, in.ID)
	}

	current, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		return nil, err
	}

	finalStart := current.StartDate
	finalDue := current.DueDate
	if in.SetStart {
		finalStart = in.StartDate
	}
	if in.SetDue {
		finalDue = in.DueDate
	}
	if finalStart != nil && finalDue != nil && finalStart.After(*finalDue) {
		return nil, fmt.Errorf("%w: start_date must be on or before due_date", commonconstants.ErrInvalidInput)
	}

	if err := s.repo.UpdateDates(ctx, in.ID, finalStart, finalDue); err != nil {
		return nil, err
	}
	current.StartDate = finalStart
	current.DueDate = finalDue
	return current, nil
}

func (s *service) Archive(ctx context.Context, id uuid.UUID, archived bool) (*Item, error) {
	if err := s.repo.Update(ctx, UpdateItemInput{ID: id, Archived: &archived}); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) DailyReset(ctx context.Context) (int64, error) {
	return s.repo.BulkResetDailyItems(ctx)
}

// GetByUserID returns non-archived items across all the user's plans.
// Used by useranalytics in 4c via gRPC.
func (s *service) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Item, error) {
	return s.repo.GetByUserID(ctx, userID)
}
