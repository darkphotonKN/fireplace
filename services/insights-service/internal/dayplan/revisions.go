package dayplan

import (
	"context"
	"fmt"
	"slices"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
)

// ErrItemNotFound: the item is not in the plan (R48). It is a NotFound, so
// the shared mapper answers 404.
var ErrItemNotFound = fmt.Errorf("%w: plan item", commonconstants.ErrNotFound)

// planTypeLearning is plan-service's learning plan type: the only plans that
// revise (R41).
const planTypeLearning = "learning"

// learning reports whether the plan revises (R41).
func (p PlanProfile) learning() bool { return p.Type == planTypeLearning }

// RevisionStore persists the revision ladder per (user, plan, item). The
// rules are ladder.go's; the store only reads and applies them.
type RevisionStore interface {
	// Ladder returns the item's ladder as a day tick on key's date sees it.
	Ladder(ctx context.Context, key guided.DayKey, itemID uuid.UUID) (ladderState, error)
	// ApplyLadder applies c to the item's ladder for key's date, in one step.
	ApplyLadder(ctx context.Context, key guided.DayKey, itemID uuid.UUID, c ladderChange) error
	// DueRevisions returns the open revisions due on or before key's date of
	// items not switched off, oldest due first (R44). Items no longer in the
	// plan are not filtered here.
	DueRevisions(ctx context.Context, key guided.DayKey) ([]dueRevision, error)
	// RevisionsOff returns the user's items on the plan switched off.
	RevisionsOff(ctx context.Context, planID, userID uuid.UUID) (map[uuid.UUID]bool, error)
	// SetRevisions switches the item's revisions: off cancels its open rung;
	// on resumes at its next tick (R47).
	SetRevisions(ctx context.Context, planID, userID, itemID uuid.UUID, enabled bool) error
}

// SetItemRevisions switches revisions for (user, plan, item) on or off
// (R47, R48). Off cancels the item's open revision and schedules no more; on
// schedules nothing by itself, and the item's next day tick schedules the
// rung after the last completed, never past MaxRevision.
//
// A plan the caller cannot access is NotFound from plan-service; an item not
// in the plan is ErrItemNotFound. On a project plan it is accepted and changes
// nothing (R48).
func (s *Service) SetItemRevisions(ctx context.Context, planID, userID, itemID uuid.UUID, enabled bool) error {
	profile, err := s.plans.GetPlanProfile(ctx, planID, userID)
	if err != nil {
		return fmt.Errorf("dayplan: read plan: %w", err)
	}
	items, err := s.plans.ListPlanItems(ctx, planID, userID)
	if err != nil {
		return fmt.Errorf("dayplan: read plan items: %w", err)
	}
	found := false
	for _, it := range items {
		found = found || it.ID == itemID
	}
	if !found {
		return fmt.Errorf("%w: %s", ErrItemNotFound, itemID)
	}
	if !profile.learning() {
		return nil
	}
	if err := s.revisions.SetRevisions(ctx, planID, userID, itemID, enabled); err != nil {
		return fmt.Errorf("dayplan: set revisions: %w", err)
	}
	return nil
}

// climbLadder is a new day tick's effect on the item's ladder (R42, R43):
// the rung it completes and the one it schedules, by the ladder's rule.
func (s *Service) climbLadder(ctx context.Context, key guided.DayKey, itemID uuid.UUID) error {
	st, err := s.revisions.Ladder(ctx, key, itemID)
	if err != nil {
		return fmt.Errorf("dayplan: read ladder: %w", err)
	}
	if err := s.revisions.ApplyLadder(ctx, key, itemID, onTick(st, key.Date)); err != nil {
		return fmt.Errorf("dayplan: climb ladder: %w", err)
	}
	return nil
}

// undoLadder takes back what the item's tick on key's date did to its ladder
// (R46, D5). Run again, or with no such tick, it changes nothing.
func (s *Service) undoLadder(ctx context.Context, key guided.DayKey, itemID uuid.UUID) error {
	st, err := s.revisions.Ladder(ctx, key, itemID)
	if err != nil {
		return fmt.Errorf("dayplan: read ladder: %w", err)
	}
	if err := s.revisions.ApplyLadder(ctx, key, itemID, onUntick(st.Rungs, key.Date)); err != nil {
		return fmt.Errorf("dayplan: undo ladder: %w", err)
	}
	return nil
}

// dueRevisions is the revisions due on key's date of a learning plan whose
// items are still in tree (R44): done items still revise (S7), archived or
// deleted ones are dropped here, against the plan as it is now. A project
// plan has none (R41).
func (s *Service) dueRevisions(ctx context.Context, key guided.DayKey, profile PlanProfile, tree guided.PlanTree) ([]dueRevision, error) {
	if !profile.learning() {
		return nil, nil
	}
	due, err := s.revisions.DueRevisions(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("dayplan: due revisions: %w", err)
	}
	inTree := make(map[uuid.UUID]bool, len(tree.Items))
	for _, it := range tree.Items {
		inTree[it.ID] = true
	}
	kept := due[:0]
	for _, r := range due {
		if inTree[r.ItemID] {
			kept = append(kept, r)
		}
	}
	return kept, nil
}

// stampRevisions sets revision = N on every existing_item entry whose item
// has revision N due (R10, R45). The model never sets it; entries of items
// with nothing due keep what they carry (a pinned entry keeps its stamp).
func stampRevisions(groups []Group, due []dueRevision) {
	rungOf := make(map[uuid.UUID]int, len(due))
	for _, r := range due {
		rungOf[r.ItemID] = r.Rung
	}
	for gi := range groups {
		for ei := range groups[gi].Entries {
			e := &groups[gi].Entries[ei]
			if e.Kind != EntryExistingItem {
				continue
			}
			if n, ok := rungOf[*e.ItemID]; ok {
				e.Revision = &n
			}
		}
	}
}

// withRevisionSwitches sets revisionsEnabled on p's existing_item entries
// when the plan is a learning plan (R49); on a project plan, and on new
// steps, it stays unset.
func (s *Service) withRevisionSwitches(ctx context.Context, p DayPlan, learning bool) (DayPlan, error) {
	if !learning || !hasExistingItems(p) {
		return p, nil
	}
	off, err := s.revisions.RevisionsOff(ctx, p.Key.PlanID, p.Key.UserID)
	if err != nil {
		return DayPlan{}, fmt.Errorf("dayplan: revisions switches: %w", err)
	}
	// A copy, so the plan the store handed over is not written through.
	groups := make([]Group, len(p.Groups))
	for gi, g := range p.Groups {
		groups[gi] = Group{Label: g.Label, Entries: slices.Clone(g.Entries)}
		for ei := range groups[gi].Entries {
			e := &groups[gi].Entries[ei]
			if e.Kind == EntryExistingItem {
				enabled := !off[*e.ItemID]
				e.RevisionsEnabled = &enabled
			}
		}
	}
	p.Groups = groups
	return p, nil
}

func hasExistingItems(p DayPlan) bool {
	for _, g := range p.Groups {
		for _, e := range g.Entries {
			if e.Kind == EntryExistingItem {
				return true
			}
		}
	}
	return false
}

// revisionLabel is how the prompt names rung n: "revision N" (R7, D6).
func revisionLabel(n int) string {
	return fmt.Sprintf("revision %d", n)
}
