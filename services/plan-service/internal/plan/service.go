package plan

import (
	"context"
	"errors"

	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, p *Plan) (*Plan, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Plan, error)
	Update(ctx context.Context, in *UpdatePlanInput) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*Plan, error)
	ListShared(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Plan, error)
	Search(ctx context.Context, in SearchInput) ([]*SearchResult, error)
	CreateShare(ctx context.Context, planID, userID uuid.UUID) error
}

type service struct {
	repo      Repository
	publishCh commonbroker.Publisher
}

func NewService(repo Repository, publishCh commonbroker.Publisher) *service {
	return &service{repo: repo, publishCh: publishCh}
}

func (s *service) Create(ctx context.Context, in *CreatePlanInput) (*Plan, error) {
	// Default daily_reset by plan type if not explicitly provided.
	dailyReset := true
	if in.DailyReset != nil {
		dailyReset = *in.DailyReset
	} else if in.PlanType == PlanTypeProject {
		dailyReset = false
	}

	p := &Plan{
		UserID:      in.UserID,
		Name:        in.Name,
		Focus:       in.Focus,
		Description: in.Description,
		PlanType:    in.PlanType,
		DailyReset:  dailyReset,
	}

	created, err := s.repo.Create(ctx, p)
	if err != nil {
		return nil, err
	}

	s.PublishPlanCreated(ctx, created)
	return created, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*Plan, error) {
	return s.repo.GetByID(ctx, id)
}

// AssertOwnership returns ErrNotFound if the plan doesn't exist and ErrForbidden
// if it belongs to a different user. Used by sibling services that need a cheap
// ownership check before reading plan-scoped data.
func (s *service) AssertOwnership(ctx context.Context, planID, userID uuid.UUID) error {
	p, err := s.repo.GetByID(ctx, planID)
	if err != nil {
		return err // already commonconstants.ErrNotFound when applicable
	}
	if p.UserID != userID {
		return commonconstants.ErrForbidden
	}
	return nil
}

func (s *service) Update(ctx context.Context, in *UpdatePlanInput) (*Plan, error) {
	if err := s.repo.Update(ctx, in); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, in.ID)
}

func (s *service) ToggleDailyReset(ctx context.Context, id, userID uuid.UUID) (*Plan, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	flipped := !p.DailyReset
	if err := s.repo.Update(ctx, &UpdatePlanInput{
		ID:         id,
		UserID:     userID,
		DailyReset: &flipped,
	}); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return err
	}
	s.PublishPlanDeleted(ctx, id, userID)
	return nil
}

func (s *service) ListByUser(ctx context.Context, userID uuid.UUID) ([]*Plan, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *service) ListShared(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Plan, error) {
	return s.repo.ListShared(ctx, userID, limit, offset)
}

func (s *service) Search(ctx context.Context, in SearchInput) ([]*SearchResult, error) {
	return s.repo.Search(ctx, in)
}

// CascadeDeleteForUser removes all plans (and their cascading children) owned
// by the given user. Called by the AMQP consumer when auth-service emits
// user.deleted. Errors are best-effort — we log and continue so a single
// poison message doesn't block the queue.
func (s *service) CascadeDeleteForUser(ctx context.Context, userID uuid.UUID) error {
	plans, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return err
	}
	var firstErr error
	for _, p := range plans {
		if err := s.repo.Delete(ctx, p.ID, userID); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return errors.New("cascade delete partial failure: " + firstErr.Error())
	}
	return nil
}
