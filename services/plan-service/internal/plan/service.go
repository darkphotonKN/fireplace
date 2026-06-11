package plan

import (
	"context"
	"fmt"

	pbevents "github.com/darkphotonKN/fireplace/common/api/proto/events"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/darkphotonKN/fireplace/services/plan-service/internal/outbox"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	// Transaction methods
	CreateTx(ctx context.Context, tx *sqlx.Tx, p *Plan) (*Plan, error)
}

type service struct {
	repo          Repository
	outboxService OutboxService
	db            *sqlx.DB
}

type OutboxService interface {
	CreateTx(ctx context.Context, tx *sqlx.Tx, params outbox.CreateOutboxParams) error
}

func NewService(repo Repository, outboxService OutboxService, db *sqlx.DB) *service {
	return &service{repo: repo, outboxService: outboxService, db: db}
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

	// transction with write and outbox to tie write with event publish and guarantee it
	// through our go workers publishing the outbox events

	var newPlan *Plan

	err := commonhelpers.ExecTx(ctx, s.db, func(tx *sqlx.Tx) error {
		// write plan
		plan, err := s.repo.CreateTx(ctx, tx, p)

		if err != nil {
			return err
		}

		payload, err := proto.Marshal(&pbevents.PlanCreatedEvent{
			Id:        plan.ID.String(),
			UserId:    plan.UserID.String(),
			Name:      plan.Name,
			Focus:     plan.Focus,
			PlanType:  plan.PlanType,
			CreatedAt: timestamppb.New(plan.CreatedAt),
		})

		if err != nil {
			return err
		}

		// write outbox
		err = s.outboxService.CreateTx(ctx, tx, outbox.CreateOutboxParams{
			RoutingKey: commonconstants.PlanCreated,
			Exchange:   commonconstants.PlanEventsExchange,
			Payload:    payload,
		})

		if err != nil {
			return err
		}

		newPlan = plan

		return nil
	})

	if err != nil {
		return nil, err
	}

	return newPlan, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*Plan, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("plan: get by id: %w", err)
	}
	return p, nil
}

// AssertOwnership returns ErrNotFound if the plan doesn't exist and ErrForbidden
// if it belongs to a different user. Used by sibling services that need a cheap
// ownership check before reading plan-scoped data.
func (s *service) AssertOwnership(ctx context.Context, planID, userID uuid.UUID) error {
	p, err := s.repo.GetByID(ctx, planID)
	if err != nil {
		return fmt.Errorf("plan: assert ownership: %w", err) // preserves ErrNotFound
	}
	if p.UserID != userID {
		// Business decision: an existing plan owned by someone else is forbidden.
		return commonconstants.ErrForbidden
	}
	return nil
}

func (s *service) Update(ctx context.Context, in *UpdatePlanInput) (*Plan, error) {
	if err := s.repo.Update(ctx, in); err != nil {
		return nil, fmt.Errorf("plan: update: %w", err)
	}
	p, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		return nil, fmt.Errorf("plan: update: reload: %w", err)
	}
	return p, nil
}

func (s *service) ToggleDailyReset(ctx context.Context, id, userID uuid.UUID) (*Plan, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("plan: toggle daily reset: %w", err)
	}
	flipped := !p.DailyReset
	if err := s.repo.Update(ctx, &UpdatePlanInput{
		ID:         id,
		UserID:     userID,
		DailyReset: &flipped,
	}); err != nil {
		return nil, fmt.Errorf("plan: toggle daily reset: %w", err)
	}
	updated, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("plan: toggle daily reset: reload: %w", err)
	}
	return updated, nil
}

func (s *service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return fmt.Errorf("plan: delete: %w", err)
	}
	s.PublishPlanDeleted(ctx, id, userID)
	return nil
}

func (s *service) ListByUser(ctx context.Context, userID uuid.UUID) ([]*Plan, error) {
	plans, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("plan: list by user: %w", err)
	}
	return plans, nil
}

func (s *service) ListShared(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Plan, error) {
	plans, err := s.repo.ListShared(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("plan: list shared: %w", err)
	}
	return plans, nil
}

func (s *service) Search(ctx context.Context, in SearchInput) ([]*SearchResult, error) {
	out, err := s.repo.Search(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("plan: search: %w", err)
	}
	return out, nil
}

// CascadeDeleteForUser removes all plans (and their cascading children) owned
// by the given user. Called by the AMQP consumer when auth-service emits
// user.deleted. Errors are best-effort — we log and continue so a single
// poison message doesn't block the queue.
func (s *service) CascadeDeleteForUser(ctx context.Context, userID uuid.UUID) error {
	plans, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("plan: cascade delete: list plans for user %s: %w", userID, err)
	}
	var firstErr error
	for _, p := range plans {
		if err := s.repo.Delete(ctx, p.ID, userID); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return fmt.Errorf("plan: cascade delete for user %s: %w", userID, firstErr)
	}
	return nil
}
