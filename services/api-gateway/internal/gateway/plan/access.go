package plangw

import (
	"context"
	"fmt"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
)

// The gateway's one plan-access seam (ADR-0015 §3–4, FS-12ZMF R4–R8).
//
// plan-service answers the data question, "what is this user's relation to this
// plan". This file answers the policy question, "does that relation allow this
// operation", and it is the only place in the gateway that does. Every
// plan-scoped route asks here before it does anything else.

// Need is what an operation requires of the caller's relation.
type Need int

const (
	// NeedMember is owner-or-member: every plan-scoped read and every checklist,
	// notes, calendar and insights operation (R7).
	NeedMember Need = iota
	// NeedOwner is owner only: update plan, toggle daily reset, delete plan, add
	// member and remove member (R6).
	NeedOwner
)

// RelationSource is plan-service's relation answer. *Client satisfies it. It
// carries the relation alone: plan-service also sends the plan's owner, but the
// client drops it, so nothing behind this seam can show it to a stranger.
type RelationSource interface {
	GetPlanAccess(ctx context.Context, planID, userID uuid.UUID) (Relation, error)
}

// Access decides whether a caller may perform an operation on a plan.
//
// A nil *Access, or one built without a source, refuses everything with 403
// (R8): the failure mode of an authorization seam is never "allow".
type Access struct {
	src RelationSource
}

func NewAccess(src RelationSource) *Access {
	return &Access{src: src}
}

// Require answers the caller's relation when it satisfies need, and
// RelationNone with an error otherwise:
//   - no relation (a stranger, or no such plan) → ErrNotFound, so a guessed id
//     is indistinguishable from a missing plan (R5);
//   - a member on an owner-only operation → ErrForbidden (R6);
//   - no answer from plan-service → that error, unchanged, so it maps to its
//     own status (503 when unreachable) and the operation never runs (R8).
//
// None of these errors is an InvalidArgument, so none can be misread by
// apierr.CodeFor's "name" heuristic.
func (a *Access) Require(ctx context.Context, planID, userID uuid.UUID, need Need) (Relation, error) {
	if a == nil || a.src == nil {
		return RelationNone, fmt.Errorf("%w: plan access checker not configured", commonconstants.ErrForbidden)
	}
	rel, err := a.src.GetPlanAccess(ctx, planID, userID)
	if err != nil {
		return RelationNone, fmt.Errorf("plan access: %w", err)
	}
	switch {
	case rel == RelationOwner:
		return rel, nil
	case rel == RelationMember && need == NeedMember:
		return rel, nil
	case rel == RelationMember:
		return RelationNone, fmt.Errorf("%w: plan %s is owner-only for this operation", commonconstants.ErrForbidden, planID)
	default:
		return RelationNone, fmt.Errorf("%w: plan %s", commonconstants.ErrNotFound, planID)
	}
}

// RequireMember is Require(…, NeedMember) for the gateway's other domains
// (notes, calendar, insights), which only ever need owner-or-member and
// declare this one method at the consumer rather than importing Need.
func (a *Access) RequireMember(ctx context.Context, planID, userID uuid.UUID) error {
	_, err := a.Require(ctx, planID, userID, NeedMember)
	return err
}
