package plan

import (
	"context"
	"errors"
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
	ListAccessible(ctx context.Context, userID uuid.UUID) ([]*Plan, error)
	ListShared(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Plan, error)
	Search(ctx context.Context, in SearchInput) ([]*SearchResult, error)
	CreateShare(ctx context.Context, planID, userID uuid.UUID) error
	GetShare(ctx context.Context, planID, userID uuid.UUID) (*Member, error)
	ListShares(ctx context.Context, planID uuid.UUID) ([]*Member, error)
	DeleteShare(ctx context.Context, planID, userID uuid.UUID) error
	GetAccess(ctx context.Context, planID, userID uuid.UUID) (*AccessRow, error)
	DeleteSharesForUser(ctx context.Context, userID uuid.UUID) error

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

	// transaction with write and outbox to tie write with event publish and guarantee it
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
			// wrap error, db and transaction has no context of this
			return fmt.Errorf("create plan error when marshalling to proto plan id %s: %w", plan.ID, err)
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

// GetAccess answers userID's relation to planID (ADR-0015 §1–2). It decides
// nothing about what the relation allows; the gateway does. A missing plan is
// RelationNone, never ErrNotFound, so a stranger cannot tell a guessed id from
// a missing plan. Ownership wins over a share row.
func (s *service) GetAccess(ctx context.Context, planID, userID uuid.UUID) (*Access, error) {
	row, err := s.repo.GetAccess(ctx, planID, userID)
	if errors.Is(err, commonconstants.ErrNotFound) {
		return &Access{Relation: RelationNone}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("plan: get access: %w", err)
	}
	access := &Access{Relation: RelationNone, OwnerID: row.OwnerID}
	switch {
	case row.OwnerID == userID:
		access.Relation = RelationOwner
	case row.Shared:
		access.Relation = RelationMember
	}
	return access, nil
}

// ListMembers answers planID's owner, its creation time and its members in add
// order (FS-12ZMF R15). Who may ask is the gateway's decision (ADR-0015 §3).
func (s *service) ListMembers(ctx context.Context, planID uuid.UUID) (*Members, error) {
	p, err := s.repo.GetByID(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("plan: list members: %w", err)
	}
	shares, err := s.repo.ListShares(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("plan: list members: %w", err)
	}
	return &Members{OwnerID: p.UserID, PlanCreatedAt: p.CreatedAt, Members: shares}, nil
}

// AddMember shares planID, which callerID must own, with memberID (FS-12ZMF
// R17). The share is written with role 'edit' and a re-add writes nothing, so
// the member read back keeps its original since. The owner cannot be a member:
// that is this service's rule, refused as ErrInvalidInput (MEMBER_IS_OWNER at
// the gateway). A plan the caller does not own is ErrNotFound; the gateway has
// already refused that, so this is defence in depth.
func (s *service) AddMember(ctx context.Context, planID, callerID, memberID uuid.UUID) (*Member, error) {
	if err := s.requireOwner(ctx, planID, callerID); err != nil {
		return nil, fmt.Errorf("plan: add member: %w", err)
	}
	if memberID == callerID {
		return nil, fmt.Errorf("plan: add member: %w: the plan's owner cannot be its member", commonconstants.ErrInvalidInput)
	}
	if err := s.repo.CreateShare(ctx, planID, memberID); err != nil {
		return nil, fmt.Errorf("plan: add member: %w", err)
	}
	m, err := s.repo.GetShare(ctx, planID, memberID)
	if err != nil {
		return nil, fmt.Errorf("plan: add member: read back: %w", err)
	}
	return m, nil
}

// RemoveMember deletes memberID's share on planID, which callerID must own
// (FS-12ZMF R19). No such share is ErrNotFound, and so is the owner's own id
// even if a stray share row names them: the owner is never removable.
func (s *service) RemoveMember(ctx context.Context, planID, callerID, memberID uuid.UUID) error {
	if err := s.requireOwner(ctx, planID, callerID); err != nil {
		return fmt.Errorf("plan: remove member: %w", err)
	}
	if memberID == callerID {
		return fmt.Errorf("plan: remove member: %w: the owner is not a member", commonconstants.ErrNotFound)
	}
	if err := s.repo.DeleteShare(ctx, planID, memberID); err != nil {
		return fmt.Errorf("plan: remove member: %w", err)
	}
	return nil
}

// requireOwner is ErrNotFound unless planID exists and userID owns it. A plan
// can't change owner, so the check holds for the write that follows.
func (s *service) requireOwner(ctx context.Context, planID, userID uuid.UUID) error {
	p, err := s.repo.GetByID(ctx, planID)
	if err != nil {
		return err
	}
	if p.UserID != userID {
		return fmt.Errorf("%w: plan %s is not owned by the caller", commonconstants.ErrNotFound, planID)
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

func (s *service) ListAccessible(ctx context.Context, userID uuid.UUID) ([]*Plan, error) {
	plans, err := s.repo.ListAccessible(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("plan: list accessible: %w", err)
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
// by the given user, and every share row naming them on someone else's plan
// (FS-12ZMF R25: users live in the gateway's database, so no FK does it).
// Called by the AMQP consumer when auth-service emits user.deleted. Both halves
// are idempotent, so a redelivery is a no-op. Errors are returned for the
// consumer's ack/requeue policy to decide.
func (s *service) CascadeDeleteForUser(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.DeleteSharesForUser(ctx, userID); err != nil {
		return fmt.Errorf("plan: cascade delete: shares for user %s: %w", userID, err)
	}
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
