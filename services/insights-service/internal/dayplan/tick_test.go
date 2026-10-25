package dayplan

import (
	"context"
	"errors"
	"testing"
	"time"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
)

// drawn is a stored day plan for f drawn by the model with groups, and its
// entries by item (new steps by text).
func drawn(t *testing.T, f *fixture, groups ...genGroup) DayPlan {
	t.Helper()
	f.gen.answer(modelOutput(groups...))
	p, err := f.planMyDay()
	if err != nil {
		t.Fatalf("PlanMyDay: %v", err)
	}
	return p
}

// entryFor is p's entry for item, or the new step with text.
func entryFor(t *testing.T, p DayPlan, item uuid.UUID, text string) Entry {
	t.Helper()
	for _, g := range p.Groups {
		for _, e := range g.Entries {
			if (e.ItemID != nil && *e.ItemID == item) || (e.Kind == EntryNewStep && e.Text == text) {
				return e
			}
		}
	}
	t.Fatalf("no entry for %s %q in %+v", item, text, p.Groups)
	return Entry{}
}

func (f *fixture) tick(entryID uuid.UUID, itemID *uuid.UUID) (Entry, error) {
	return f.svc.TickEntry(context.Background(), entryID, f.userID, itemID)
}

func TestTickEntry_ExistingItem_StoresOneTickAndTheReadShowsIt(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	p := drawn(t, f, genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30), existing(tf.graphs, "x", 20)}})
	bfs := entryFor(t, p, tf.bfs, "")

	got, err := f.tick(bfs.ID, nil)

	if err != nil {
		t.Fatalf("TickEntry: %v", err)
	}
	if !got.Ticked || got.ID != bfs.ID || got.Kind != EntryExistingItem || *got.ItemID != tf.bfs || got.Text != bfs.Text {
		t.Fatalf("ticked entry = %+v, want the entry, ticked", got)
	}
	if ticks := f.store.ticksFor(f.key()); len(ticks) != 1 || ticks[0] != tf.bfs {
		t.Fatalf("ticks = %v, want exactly one, on the entry's item", ticks)
	}
	read, err := f.getDayPlan()
	if err != nil {
		t.Fatal(err)
	}
	if e := entryFor(t, read, tf.bfs, ""); !e.Ticked {
		t.Fatalf("read entry = %+v, want ticked", e)
	}
	if e := entryFor(t, read, tf.graphs, ""); e.Ticked {
		t.Fatalf("other entry = %+v, want unticked", e)
	}
}

func TestTickEntry_TickingAgainChangesNothing(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	bfs := entryFor(t, drawn(t, f, genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}), tf.bfs, "")
	first, err := f.tick(bfs.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	again, err := f.tick(bfs.ID, &tf.bfs)

	if err != nil {
		t.Fatalf("re-tick: %v", err)
	}
	if again.ID != first.ID || !again.Ticked || *again.ItemID != *first.ItemID || again.Text != first.Text {
		t.Fatalf("re-tick = %+v, want %+v unchanged", again, first)
	}
	if ticks := f.store.ticksFor(f.key()); len(ticks) != 1 {
		t.Fatalf("ticks = %v, want still one", ticks)
	}
}

// withChild is the tree fixture plus a task created from a new step under
// parent (the user's Add), and returns its id.
func withChild(f *fixture, tf treeFixture, parent uuid.UUID, mutate func(*guided.PlanItem)) uuid.UUID {
	id := uuid.New()
	child := guided.PlanItem{ID: id, ParentID: &parent, Description: "Solve the climbing stairs problem", Type: "task", Scope: "longterm"}
	if mutate != nil {
		mutate(&child)
	}
	f.plans.setItems(append(tf.items(), child))
	return id
}

func TestTickEntry_NewStep_BindsToAValidChildOrIsNotTickable(t *testing.T) {
	tf := newTreeFixture()
	const stepText = "Solve the climbing stairs problem"
	tests := []struct {
		name     string
		item     func(f *fixture) *uuid.UUID
		bindable bool
	}{
		{"valid task child of the step's parent", func(f *fixture) *uuid.UUID {
			return ptr(withChild(f, tf, tf.dp, nil))
		}, true},
		{"no item id", func(f *fixture) *uuid.UUID { return nil }, false},
		{"child of another parent", func(f *fixture) *uuid.UUID {
			return ptr(withChild(f, tf, tf.graphs, nil))
		}, false},
		{"top-level item", func(f *fixture) *uuid.UUID {
			return ptr(withChild(f, tf, tf.dp, func(it *guided.PlanItem) { it.ParentID = nil }))
		}, false},
		{"item of another plan (unknown to this plan)", func(f *fixture) *uuid.UUID { return ptr(uuid.New()) }, false},
		{"note", func(f *fixture) *uuid.UUID {
			return ptr(withChild(f, tf, tf.dp, func(it *guided.PlanItem) { it.Type = "note" }))
		}, false},
		{"archived child", func(f *fixture) *uuid.UUID {
			return ptr(withChild(f, tf, tf.dp, func(it *guided.PlanItem) { it.Archived = true }))
		}, false},
		{"item another entry already points at", func(f *fixture) *uuid.UUID { return &tf.bfs }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(tf.items())
			p := drawn(t, f,
				genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}},
				genGroup{Label: "Dynamic programming", Entries: []genEntry{newStep(tf.dp, stepText, 30)}},
			)
			step := entryFor(t, p, uuid.Nil, stepText)
			item := tt.item(f)

			got, err := f.tick(step.ID, item)

			if !tt.bindable {
				if !errors.Is(err, ErrEntryNotTickable) {
					t.Fatalf("err = %v, want ErrEntryNotTickable", err)
				}
				if ticks := f.store.ticksFor(f.key()); len(ticks) != 0 {
					t.Fatalf("ticks = %v, want none", ticks)
				}
				if e := entryFor(t, mustRead(t, f), uuid.Nil, stepText); e.Kind != EntryNewStep {
					t.Fatalf("refused entry = %+v, want still an unbound new step", e)
				}
				return
			}
			if err != nil {
				t.Fatalf("TickEntry: %v", err)
			}
			if got.Kind != EntryExistingItem || *got.ItemID != *item || !got.Ticked || got.Text != stepText ||
				got.Reason != step.Reason || *got.ParentItemID != tf.dp || got.ID != step.ID {
				t.Fatalf("bound entry = %+v, want an existing item for %s keeping text, reason and parent", got, *item)
			}
			read := entryFor(t, mustRead(t, f), *item, "")
			if read.ID != step.ID || read.Kind != EntryExistingItem || !read.Ticked {
				t.Fatalf("read entry = %+v, want it stored bound and ticked", read)
			}
		})
	}
}

func mustRead(t *testing.T, f *fixture) DayPlan {
	t.Helper()
	p, err := f.getDayPlan()
	if err != nil {
		t.Fatalf("GetDayPlan: %v", err)
	}
	return p
}

func TestTickEntry_RefusesAnArchivedOrDeletedItemAndAContradictingItemID(t *testing.T) {
	tf := newTreeFixture()
	tests := []struct {
		name   string
		change func(f *fixture)
		itemID func() *uuid.UUID
	}{
		{"item archived", func(f *fixture) {
			items := tf.items()
			items[1].Archived = true
			f.plans.setItems(items)
		}, func() *uuid.UUID { return nil }},
		{"item deleted", func(f *fixture) {
			items := tf.items()
			f.plans.setItems(append(items[:1:1], items[2:]...))
		}, func() *uuid.UUID { return nil }},
		{"item id contradicts the entry's", func(f *fixture) {}, func() *uuid.UUID { return &tf.graphs }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(tf.items())
			bfs := entryFor(t, drawn(t, f, genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}), tf.bfs, "")
			tt.change(f)

			_, err := f.tick(bfs.ID, tt.itemID())

			if !errors.Is(err, ErrEntryNotTickable) {
				t.Fatalf("err = %v, want ErrEntryNotTickable", err)
			}
			if ticks := f.store.ticksFor(f.key()); len(ticks) != 0 {
				t.Fatalf("ticks = %v, want none", ticks)
			}
		})
	}
}

func TestTickEntry_UnknownOrAnotherUsersEntryIsNotFound(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	bfs := entryFor(t, drawn(t, f, genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}), tf.bfs, "")

	for name, call := range map[string]func() error{
		"unknown tick":        func() error { _, err := f.tick(uuid.New(), nil); return err },
		"another user tick":   func() error { _, err := f.svc.TickEntry(context.Background(), bfs.ID, uuid.New(), nil); return err },
		"unknown untick":      func() error { return f.svc.UntickEntry(context.Background(), uuid.New(), f.userID) },
		"another user untick": func() error { return f.svc.UntickEntry(context.Background(), bfs.ID, uuid.New()) },
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, ErrEntryNotFound) || !errors.Is(err, commonconstants.ErrNotFound) {
				t.Fatalf("err = %v, want ErrEntryNotFound (a NotFound)", err)
			}
		})
	}
	if ticks := f.store.ticksFor(f.key()); len(ticks) != 0 {
		t.Fatalf("ticks = %v, want none", ticks)
	}
}

func TestUntickEntry_RemovesTheTickAndUntickingAnUntickedEntrySucceeds(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	const stepText = "Solve the climbing stairs problem"
	p := drawn(t, f, genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30), existing(tf.graphs, "x", 30)}},
		genGroup{Label: "Dynamic programming", Entries: []genEntry{newStep(tf.dp, stepText, 30)}})
	bfs := entryFor(t, p, tf.bfs, "")
	if _, err := f.tick(bfs.ID, nil); err != nil {
		t.Fatal(err)
	}

	if err := f.svc.UntickEntry(context.Background(), bfs.ID, f.userID); err != nil {
		t.Fatalf("UntickEntry: %v", err)
	}
	if ticks := f.store.ticksFor(f.key()); len(ticks) != 0 {
		t.Fatalf("ticks = %v, want none after untick", ticks)
	}
	if e := entryFor(t, mustRead(t, f), tf.bfs, ""); e.Ticked {
		t.Fatalf("read entry = %+v, want unticked", e)
	}

	for name, id := range map[string]uuid.UUID{
		"unticked again":   bfs.ID,
		"never ticked":     entryFor(t, p, tf.graphs, "").ID,
		"unbound new step": entryFor(t, p, uuid.Nil, stepText).ID,
	} {
		if err := f.svc.UntickEntry(context.Background(), id, f.userID); err != nil {
			t.Fatalf("%s: UntickEntry = %v, want success", name, err)
		}
	}
}

func TestTickAndUntick_OutsideTheDateWindowAreRefused(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	bfs := entryFor(t, drawn(t, f, genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}), tf.bfs, "")
	if _, err := f.tick(bfs.ID, nil); err != nil {
		t.Fatal(err)
	}

	// Yesterday's plan is still in the window (S17).
	f.svc.now = func() time.Time { return today.AddDate(0, 0, 1) }
	if err := f.svc.UntickEntry(context.Background(), bfs.ID, f.userID); err != nil {
		t.Fatalf("untick on yesterday's plan: %v", err)
	}

	f.svc.now = func() time.Time { return today.AddDate(0, 0, 2) }
	if _, err := f.tick(bfs.ID, nil); !errors.Is(err, guided.ErrDateOutOfRange) {
		t.Fatalf("tick err = %v, want ErrDateOutOfRange", err)
	}
	if err := f.svc.UntickEntry(context.Background(), bfs.ID, f.userID); !errors.Is(err, guided.ErrDateOutOfRange) {
		t.Fatalf("untick err = %v, want ErrDateOutOfRange", err)
	}
	if ticks := f.store.ticksFor(f.key()); len(ticks) != 0 {
		t.Fatalf("ticks = %v: the out-of-window tick must not be stored", ticks)
	}
}

// A ladder write that fails after the tick is stored leaves the tick and
// returns the error; the client's retry finds the tick already there and still
// repairs the ladder. A clean run after that changes nothing.
func TestTickEntry_RetryAfterALadderFailureSchedulesRevisionOne(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	e := entryFor(t, drawn(t, f, genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}), tf.bfs, "")
	f.store.failApplies(1)

	if _, err := f.tick(e.ID, nil); err == nil {
		t.Fatal("tick with a failing ladder write: want the error")
	}
	if ticks := f.store.ticksFor(f.key()); len(ticks) != 1 {
		t.Fatalf("ticks = %v, want the tick kept", ticks)
	}
	if due := f.dueOn(oct(26)); len(due) != 0 {
		t.Fatalf("due = %+v, want nothing scheduled yet", due)
	}

	if _, err := f.tick(e.ID, nil); err != nil {
		t.Fatalf("retry: %v", err)
	}
	due := f.dueOn(oct(26))
	if len(due) != 1 || due[0].ItemID != tf.bfs || due[0].Rung != 1 {
		t.Fatalf("due after the retry = %+v, want revision 1 scheduled", due)
	}

	before := f.store.rungsOf(f.planID, f.userID, tf.bfs)
	if _, err := f.tick(e.ID, nil); err != nil {
		t.Fatalf("clean re-tick: %v", err)
	}
	if after := f.store.rungsOf(f.planID, f.userID, tf.bfs); len(after) != len(before) || after[0] != before[0] {
		t.Fatalf("ladder after a clean re-tick = %+v, want %+v unchanged", after, before)
	}
}

// The untick's twin: the untick is stored, the undo fails, and the retry,
// finding no tick to remove, still takes the rung back.
func TestUntickEntry_RetryAfterALadderFailureRemovesTheRung(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	e := tickedToday(t, f, tf.bfs)
	f.store.failApplies(1)

	if err := f.svc.UntickEntry(context.Background(), e.ID, f.userID); err == nil {
		t.Fatal("untick with a failing ladder write: want the error")
	}
	if ticks := f.store.ticksFor(f.key()); len(ticks) != 0 {
		t.Fatalf("ticks = %v, want the untick kept", ticks)
	}
	if due := f.dueOn(oct(26)); len(due) != 1 {
		t.Fatalf("due = %+v, want revision 1 still there", due)
	}

	if err := f.svc.UntickEntry(context.Background(), e.ID, f.userID); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if rungs := f.store.rungsOf(f.planID, f.userID, tf.bfs); len(rungs) != 0 {
		t.Fatalf("ladder after the retry = %+v, want revision 1 removed", rungs)
	}

	if err := f.svc.UntickEntry(context.Background(), e.ID, f.userID); err != nil {
		t.Fatalf("clean re-untick: %v", err)
	}
	if rungs := f.store.rungsOf(f.planID, f.userID, tf.bfs); len(rungs) != 0 {
		t.Fatalf("ladder after a clean re-untick = %+v, want still empty", rungs)
	}
}
