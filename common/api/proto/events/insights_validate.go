package events

import (
	"fmt"

	"github.com/google/uuid"
)

// Validate checks an InsightGeneratedEvent before anything is written from it —
// the identifiers it is addressed to, and the shape of the item graph it
// carries. It lives here, beside the generated code rather than inside either
// service, so the producer and the consumer cannot disagree about what a legal
// payload is.
//
// Generated .pb.go files are never hand-edited; this is a separate file in the
// same package for exactly that reason.
//
// What it deliberately does NOT check: scope and type values. Those are
// plan-service's rules, enforced by its own CHECK constraints and service
// layer. Restating a downstream rule here would give it two homes that can
// drift apart (ADR-0005).
func (e *InsightGeneratedEvent) Validate() error {
	if _, err := uuid.Parse(e.PlanId); err != nil {
		return fmt.Errorf("events: insight.generated has invalid plan_id %q: %w", e.PlanId, err)
	}
	if _, err := uuid.Parse(e.UserId); err != nil {
		return fmt.Errorf("events: insight.generated has invalid user_id %q: %w", e.UserId, err)
	}
	return e.validateItems()
}

// validateItems enforces the two rules the wire format cannot express itself:
// every parent_index addresses a real item, and nesting stops at two tiers.
func (e *InsightGeneratedEvent) validateItems() error {
	for i, it := range e.Items {
		if it.ParentIndex == nil {
			continue
		}
		p := *it.ParentIndex
		if p < 0 || int(p) >= len(e.Items) {
			return fmt.Errorf("events: item %d has parent_index %d, out of range for %d items", i, p, len(e.Items))
		}
		if int(p) == i {
			return fmt.Errorf("events: item %d is its own parent", i)
		}
		if e.Items[p].ParentIndex != nil {
			return fmt.Errorf("events: item %d nests under item %d, which is itself nested: two-tier maximum", i, p)
		}
	}
	return nil
}

// Validate checks the failure event's identifiers. Its failure_class is an enum,
// so the wire format already constrains it — but an UNSPECIFIED class is
// rejected here rather than accepted and silently rendered as generic copy: a
// producer that forgot to set it is a bug, and swallowing it would hide the bug
// behind a plausible-looking error message shown to a user.
func (e *InsightGenerationFailedEvent) Validate() error {
	if _, err := uuid.Parse(e.PlanId); err != nil {
		return fmt.Errorf("events: insight.generation_failed has invalid plan_id %q: %w", e.PlanId, err)
	}
	if _, err := uuid.Parse(e.UserId); err != nil {
		return fmt.Errorf("events: insight.generation_failed has invalid user_id %q: %w", e.UserId, err)
	}
	if e.FailureClass == FailureClass_FAILURE_CLASS_UNSPECIFIED {
		return fmt.Errorf("events: insight.generation_failed for plan %s carries no failure_class", e.PlanId)
	}
	return nil
}

// Retryable reports whether a plan in this failure class should be offered a
// retry action. FS-0006 R37 makes failure_class drive exactly two things — the
// copy the user sees, and whether retry is offered at all — so the second one
// is decided here, once, rather than by each surface re-deriving it from a
// switch it wrote itself.
//
// UNSPECIFIED is not retryable: an unrecognised class must fail toward the
// conservative answer, never toward "offer a button that spends money".
func (f FailureClass) Retryable() bool {
	switch f {
	case FailureClass_FAILURE_CLASS_GENERATION_EXHAUSTED, FailureClass_FAILURE_CLASS_STALLED:
		return true
	default:
		return false
	}
}
