package dayplan

import (
	"context"
	"errors"
	"fmt"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
)

var (
	// ErrEntryNotFound: no entry of the caller's stored day plans has the id
	// (R26). It is a NotFound, so the shared mapper answers 404.
	ErrEntryNotFound = fmt.Errorf("%w: day plan entry", commonconstants.ErrNotFound)
	// ErrEntryNotTickable: the entry cannot carry a day tick (R26, D3): an
	// unbound new step without a valid item to bind, an item now archived or
	// deleted, or an item id that contradicts the entry's own.
	ErrEntryNotTickable = errors.New("dayplan: entry not tickable")
)

// itemTypeTask is plan-service's task item type: only a task binds (R25).
const itemTypeTask = "task"

// Tick is one day tick: the entry's item done for the plan date (R24), per
// (user, plan, item, date) of Key, with the entry it was ticked through.
// Bind says the entry is an unbound new step that becomes an existing item
// for ItemID first (R25).
type Tick struct {
	Key     guided.DayKey
	ItemID  uuid.UUID
	EntryID uuid.UUID
	DrawID  uuid.UUID
	Bind    bool
}

// TickEntry marks entry entryID done for its plan date (R25) and returns it,
// ticked. An existing item's entry takes no itemID, or its own; an unbound new
// step needs the id of the item the user created from it, which plan-service
// must report as a non-archived task under the entry's parent in the same plan
// (the plan the entry's day plan is for). The entry is then bound: it becomes
// an existing item for that item, keeping its text and reason.
//
// On a learning plan a new tick also moves the item's revision ladder (R42,
// R43): it completes the rung due, or starts the ladder on the item's first
// tick ever, and schedules the next rung. A repeated tick moves nothing (R27)
// on a ladder its first tick already moved. The tick and the ladder are
// separate writes: a ladder write that fails after the tick is stored is
// returned as an error, the tick stays, and ticking again repairs the ladder.
//
// It refuses with ErrEntryNotFound for an entry that is not the caller's,
// guided.ErrDateOutOfRange outside the date window (R30) and
// ErrEntryNotTickable (R26). Ticking a ticked entry returns it unchanged (R27).
// It never touches the item's done (R28): the plan reader has no write.
func (s *Service) TickEntry(ctx context.Context, entryID, userID uuid.UUID, itemID *uuid.UUID) (Entry, error) {
	p, e, err := s.findEntry(ctx, entryID, userID)
	if err != nil {
		return Entry{}, err
	}
	profile, err := s.plans.GetPlanProfile(ctx, p.Key.PlanID, userID)
	if err != nil {
		return Entry{}, fmt.Errorf("dayplan: read plan: %w", err)
	}
	items, err := s.plans.ListPlanItems(ctx, p.Key.PlanID, userID)
	if err != nil {
		return Entry{}, fmt.Errorf("dayplan: read plan items: %w", err)
	}
	target, bind, err := tickTarget(p, e, itemID, items)
	if err != nil {
		return Entry{}, err
	}
	if _, err := s.store.Tick(ctx, Tick{Key: p.Key, ItemID: target, EntryID: e.ID, DrawID: p.DrawID, Bind: bind}); err != nil {
		return Entry{}, fmt.Errorf("dayplan: tick: %w", err)
	}
	// The ladder moves on a repeated tick too: a retry after a ladder write
	// failed must still repair it. onTick on a ladder the tick already moved
	// changes nothing, and ApplyLadder's writes are conditional.
	if profile.learning() {
		if err := s.climbLadder(ctx, p.Key, target); err != nil {
			return Entry{}, err
		}
	}
	e.Kind, e.ItemID, e.Ticked = EntryExistingItem, &target, true
	if profile.learning() {
		off, err := s.revisions.RevisionsOff(ctx, p.Key.PlanID, userID)
		if err != nil {
			return Entry{}, fmt.Errorf("dayplan: revisions switches: %w", err)
		}
		enabled := !off[target]
		e.RevisionsEnabled = &enabled
	}
	return e, nil
}

// UntickEntry removes the day tick of entry entryID for its plan date (R29)
// and undoes what that tick did to the item's revision ladder (R46, D5): the
// rung it completed is reopened and the rung it scheduled removed. An entry
// with no tick, including an unbound new step, succeeds and changes nothing,
// except that a ladder an earlier untick failed to undo is undone now.
// Refusals are TickEntry's ErrEntryNotFound and date window.
//
// The undo runs whatever the plan's type is now: a ladder the tick moved
// while the plan was a learning plan is still that tick's to take back.
func (s *Service) UntickEntry(ctx context.Context, entryID, userID uuid.UUID) error {
	p, e, err := s.findEntry(ctx, entryID, userID)
	if err != nil {
		return err
	}
	if e.ItemID == nil {
		return nil
	}
	if _, err := s.store.Untick(ctx, p.Key, *e.ItemID); err != nil {
		return fmt.Errorf("dayplan: untick: %w", err)
	}
	// Undone whether or not a tick was removed just now: a retry after the
	// undo failed finds the tick gone and must still take the rungs back.
	// Only a tick on the date completes or schedules a rung on it, so with no
	// tick ever there is nothing to undo, and a clean rerun changes nothing.
	return s.undoLadder(ctx, p.Key, *e.ItemID)
}

// findEntry is the caller's stored plan holding entry entryID in its current
// draw, and that entry, while the plan date is within the window (R30).
func (s *Service) findEntry(ctx context.Context, entryID, userID uuid.UUID) (DayPlan, Entry, error) {
	p, ok, err := s.store.FindByEntry(ctx, entryID, userID)
	if err != nil {
		return DayPlan{}, Entry{}, fmt.Errorf("dayplan: load entry: %w", err)
	}
	e, found := p.entry(entryID)
	if !ok || !found {
		return DayPlan{}, Entry{}, fmt.Errorf("%w: %s", ErrEntryNotFound, entryID)
	}
	if _, err := guided.NewDayKey(p.Key.PlanID, userID, p.Key.Date.Format(guided.DateLayout), s.now()); err != nil {
		return DayPlan{}, Entry{}, err
	}
	return p, e, nil
}

// entry is p's entry with id.
func (p DayPlan) entry(id uuid.UUID) (Entry, bool) {
	for _, g := range p.Groups {
		for _, e := range g.Entries {
			if e.ID == id {
				return e, true
			}
		}
	}
	return Entry{}, false
}

// tickTarget is the item a tick on e lands on, and whether e must be bound to
// it first (R25, R26). items is plan-service's answer for the plan, archived
// items included; no client-sent id is trusted without it.
func tickTarget(p DayPlan, e Entry, itemID *uuid.UUID, items []guided.PlanItem) (uuid.UUID, bool, error) {
	byID := make(map[uuid.UUID]guided.PlanItem, len(items))
	for _, it := range items {
		byID[it.ID] = it
	}

	if e.Kind == EntryExistingItem {
		if itemID != nil && *itemID != *e.ItemID {
			return uuid.Nil, false, fmt.Errorf("%w: item %s contradicts the entry's item %s", ErrEntryNotTickable, *itemID, *e.ItemID)
		}
		if it, ok := byID[*e.ItemID]; !ok || it.Archived {
			return uuid.Nil, false, fmt.Errorf("%w: item %s is archived or deleted", ErrEntryNotTickable, *e.ItemID)
		}
		return *e.ItemID, false, nil
	}

	if itemID == nil {
		return uuid.Nil, false, fmt.Errorf("%w: a new step needs the item created from it", ErrEntryNotTickable)
	}
	it, ok := byID[*itemID]
	if !ok || it.Archived || it.Type != itemTypeTask || it.ParentID == nil || e.ParentItemID == nil || *it.ParentID != *e.ParentItemID {
		return uuid.Nil, false, fmt.Errorf("%w: item %s is not a task under the step's parent in this plan", ErrEntryNotTickable, *itemID)
	}
	if other, taken := p.entryForItem(*itemID); taken && other.ID != e.ID {
		return uuid.Nil, false, fmt.Errorf("%w: item %s already has entry %s", ErrEntryNotTickable, *itemID, other.ID)
	}
	return *itemID, true, nil
}

// entryForItem is p's entry pointing at item.
func (p DayPlan) entryForItem(item uuid.UUID) (Entry, bool) {
	for _, g := range p.Groups {
		for _, e := range g.Entries {
			if e.ItemID != nil && *e.ItemID == item {
				return e, true
			}
		}
	}
	return Entry{}, false
}
