package dayplan

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/ai"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
)

// The revision ladder through the service (FS-02JBZ R41–R49). The fixture's
// clock is Oct 25, so a revision scheduled by a tick today is due on Oct 26,
// still inside the date window.

const tomorrowDate = "2026-10-26"

func (f *fixture) planMyDayOn(date string) (DayPlan, error) {
	return f.svc.PlanMyDay(context.Background(), f.planID, f.userID, date)
}

func (f *fixture) setRevisions(item uuid.UUID, enabled bool) error {
	return f.svc.SetItemRevisions(context.Background(), f.planID, f.userID, item, enabled)
}

// tickedToday draws today's plan with an entry for item and ticks it.
func tickedToday(t *testing.T, f *fixture, item uuid.UUID) Entry {
	t.Helper()
	e := entryFor(t, drawn(t, f, genGroup{Label: "Graphs", Entries: []genEntry{existing(item, "x", 30)}}), item, "")
	ticked, err := f.tick(e.ID, nil)
	if err != nil {
		t.Fatalf("TickEntry: %v", err)
	}
	return ticked
}

func (f *fixture) dueOn(date time.Time) []dueRevision {
	key := f.key()
	key.Date = date
	due, _ := f.store.DueRevisions(context.Background(), key)
	return due
}

func TestRevisions_FirstTickOnALearningPlanSchedulesRevisionOneForTomorrow(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())

	got := tickedToday(t, f, tf.bfs)

	due := f.dueOn(oct(26))
	if len(due) != 1 || due[0].ItemID != tf.bfs || due[0].Rung != 1 || !due[0].DueOn.Equal(oct(26)) {
		t.Fatalf("due on Oct 26 = %+v, want revision 1 of the item", due)
	}
	if len(f.dueOn(oct(25))) != 0 {
		t.Fatal("revision 1 must not be due on the tick's own date")
	}
	if got.RevisionsEnabled == nil || !*got.RevisionsEnabled {
		t.Fatalf("ticked entry revisionsEnabled = %v, want true on a learning plan", got.RevisionsEnabled)
	}
}

func TestRevisions_ProjectPlanSchedulesNothingAndCarriesNoSwitch(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	f.plans.profile.Type = "project"

	got := tickedToday(t, f, tf.bfs)

	if due := f.dueOn(oct(26)); len(due) != 0 {
		t.Fatalf("due on a project plan = %+v, want none", due)
	}
	if got.RevisionsEnabled != nil || got.Revision != nil {
		t.Fatalf("project entry = %+v, want no revision and no revisionsEnabled", got)
	}
	read, err := f.getDayPlan()
	if err != nil {
		t.Fatal(err)
	}
	if e := entryFor(t, read, tf.bfs, ""); e.RevisionsEnabled != nil {
		t.Fatalf("read entry revisionsEnabled = %v, want absent on a project plan", *e.RevisionsEnabled)
	}
}

func TestRevisions_DueRevisionIsStampedPromptedAndHashed(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	tickedToday(t, f, tf.bfs)

	f.gen.answer(modelOutput(genGroup{Label: "Review", Entries: []genEntry{existing(tf.bfs, "x", 20), existing(tf.dp, "x", 30)}}))
	p, err := f.planMyDayOn(tomorrowDate)
	if err != nil {
		t.Fatal(err)
	}

	bfs := entryFor(t, p, tf.bfs, "")
	if bfs.Revision == nil || *bfs.Revision != 1 {
		t.Fatalf("bfs entry revision = %v, want the server's stamp 1", bfs.Revision)
	}
	if dp := entryFor(t, p, tf.dp, ""); dp.Revision != nil {
		t.Fatalf("dp entry revision = %v, want none", *dp.Revision)
	}
	prompt := f.gen.requests[len(f.gen.requests)-1].User
	if !strings.Contains(prompt, `"revision":"revision 1"`) || !strings.Contains(prompt, tf.bfs.String()) || strings.Contains(prompt, "DUE REVISIONS: none") {
		t.Fatalf("prompt does not list the due revision as \"revision 1\":\n%s", prompt)
	}

	// The due revision is part of the rule gate's hash: completing it changes
	// the question, so the next draw is not served from the stored plan.
	key := f.key()
	key.Date = oct(26)
	stored, _ := f.store.stored(key)
	withDue := stored.PlanStateHash
	if withDue == stateHash(guided.NewPlanTree(tf.items()), f.plans.profile, oct(26), nil, nil) {
		t.Fatal("hash ignores the due revisions")
	}
}

func TestRevisions_TheLadderClimbsThroughTicksOfStampedEntries(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	tickedToday(t, f, tf.bfs)

	f.gen.answer(modelOutput(genGroup{Label: "Review", Entries: []genEntry{existing(tf.bfs, "x", 20)}}))
	p, err := f.planMyDayOn(tomorrowDate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.tick(entryFor(t, p, tf.bfs, "").ID, nil); err != nil {
		t.Fatal(err)
	}

	if due := f.dueOn(oct(28)); len(due) != 0 {
		t.Fatalf("due on Oct 28 = %+v, want none: revision 2 is due Oct 29", due)
	}
	due := f.dueOn(oct(29))
	if len(due) != 1 || due[0].Rung != 2 {
		t.Fatalf("due on Oct 29 = %+v, want revision 2 (Oct 26 + 3)", due)
	}
}

func TestRevisions_DoneItemStillRevisesArchivedOrDeletedDoesNot(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	tickedToday(t, f, tf.bfs)
	// The item is finished: its revision is still offered and accepted.
	items := tf.items()
	items[1].Done = true
	f.plans.setItems(items)

	f.gen.answer(modelOutput(genGroup{Label: "Review", Entries: []genEntry{existing(tf.bfs, "x", 20)}}))
	p, err := f.planMyDayOn(tomorrowDate)
	if err != nil {
		t.Fatal(err)
	}
	if e := entryFor(t, p, tf.bfs, ""); e.Revision == nil || *e.Revision != 1 {
		t.Fatalf("done item's entry = %+v, want kept with revision 1", e)
	}

	// Archived: the revision is filtered out against the plan tree.
	items[1].Archived = true
	f.plans.setItems(items)
	f.gen.answer(modelOutput(genGroup{Label: "Review", Entries: []genEntry{existing(tf.bfs, "x", 20), existing(tf.dp, "x", 20)}}))
	p, err = f.planMyDayOn(tomorrowDate)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range p.Groups {
		for _, e := range g.Entries {
			if e.ItemID != nil && *e.ItemID == tf.bfs {
				t.Fatalf("archived item's revision resurfaced: %+v", e)
			}
		}
	}
	if prompt := f.gen.requests[len(f.gen.requests)-1].User; !strings.Contains(prompt, "DUE REVISIONS: none") {
		t.Fatalf("archived item's revision reached the prompt:\n%s", prompt)
	}
}

func TestRevisions_RulesPlanPutsDueRevisionsFirstInARevisionsGroup(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	tickedToday(t, f, tf.bfs)

	f.gen.fail(ai.ErrBudgetExhausted)
	p, err := f.planMyDayOn(tomorrowDate)
	if err != nil {
		t.Fatal(err)
	}

	if p.Source != SourceRules || len(p.Groups) == 0 || p.Groups[0].Label != "Revisions" {
		t.Fatalf("rules plan = %+v, want a first Revisions group", p.Groups)
	}
	e := p.Groups[0].Entries[0]
	if *e.ItemID != tf.bfs || e.Revision == nil || *e.Revision != 1 || e.Reason != "Revision 1 due" {
		t.Fatalf("first rules entry = %+v, want bfs stamped revision 1", e)
	}
}

func TestRevisions_UntickingTheFirstTickRemovesRevisionOne(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	e := tickedToday(t, f, tf.bfs)

	if err := f.svc.UntickEntry(context.Background(), e.ID, f.userID); err != nil {
		t.Fatal(err)
	}

	if due := f.dueOn(oct(26)); len(due) != 0 {
		t.Fatalf("due after the untick = %+v, want revision 1 removed", due)
	}
	// The item has no first tick any more: ticking again starts the ladder.
	if _, err := f.tick(e.ID, nil); err != nil {
		t.Fatal(err)
	}
	if due := f.dueOn(oct(26)); len(due) != 1 || due[0].Rung != 1 {
		t.Fatalf("due after ticking again = %+v, want revision 1", due)
	}
}

func TestRevisions_UntickingARevisionTickReopensItAndRemovesTheNextRung(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	tickedToday(t, f, tf.bfs)
	f.gen.answer(modelOutput(genGroup{Label: "Review", Entries: []genEntry{existing(tf.bfs, "x", 20)}}))
	p, err := f.planMyDayOn(tomorrowDate)
	if err != nil {
		t.Fatal(err)
	}
	rev := entryFor(t, p, tf.bfs, "")
	if _, err := f.tick(rev.ID, nil); err != nil {
		t.Fatal(err)
	}

	if err := f.svc.UntickEntry(context.Background(), rev.ID, f.userID); err != nil {
		t.Fatal(err)
	}

	due := f.dueOn(oct(29))
	if len(due) != 1 || due[0].Rung != 1 {
		t.Fatalf("due after the untick = %+v, want revision 1 reopened and revision 2 gone", due)
	}
}

func TestRevisions_SwitchOffCancelsTheOpenRungAndOnResumesAtTheNextTick(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	tickedToday(t, f, tf.bfs)

	if err := f.setRevisions(tf.bfs, false); err != nil {
		t.Fatal(err)
	}
	if due := f.dueOn(oct(26)); len(due) != 0 {
		t.Fatalf("due after off = %+v, want the open rung cancelled", due)
	}
	read, err := f.getDayPlan()
	if err != nil {
		t.Fatal(err)
	}
	if e := entryFor(t, read, tf.bfs, ""); e.RevisionsEnabled == nil || *e.RevisionsEnabled {
		t.Fatalf("entry revisionsEnabled after off = %v, want false", e.RevisionsEnabled)
	}

	// On schedules nothing by itself.
	if err := f.setRevisions(tf.bfs, true); err != nil {
		t.Fatal(err)
	}
	if due := f.dueOn(oct(26)); len(due) != 0 {
		t.Fatalf("due right after on = %+v, want none", due)
	}
	// The next tick resumes at revision 1 (nothing completed yet).
	f.gen.answer(modelOutput(genGroup{Label: "Review", Entries: []genEntry{existing(tf.bfs, "x", 20)}}))
	p, err := f.planMyDayOn(tomorrowDate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.tick(entryFor(t, p, tf.bfs, "").ID, nil); err != nil {
		t.Fatal(err)
	}
	if due := f.dueOn(oct(27)); len(due) != 1 || due[0].Rung != 1 {
		t.Fatalf("due after the resuming tick = %+v, want revision 1 on Oct 27", due)
	}
}

func TestSetItemRevisions_ItemNotInThePlanIsNotFound(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())

	err := f.setRevisions(uuid.New(), false)

	if !errors.Is(err, commonconstants.ErrNotFound) {
		t.Fatalf("err = %v, want NotFound", err)
	}
}

func TestSetItemRevisions_InaccessiblePlanIsNotFound(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	f.plans.err = commonconstants.ErrNotFound

	if err := f.setRevisions(tf.bfs, false); !errors.Is(err, commonconstants.ErrNotFound) {
		t.Fatalf("err = %v, want NotFound", err)
	}
}

func TestSetItemRevisions_ProjectPlanIsAcceptedWithNoEffect(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	f.plans.profile.Type = "project"

	if err := f.setRevisions(tf.bfs, false); err != nil {
		t.Fatalf("err = %v, want accepted", err)
	}
	if off, _ := f.store.RevisionsOff(context.Background(), f.planID, f.userID); len(off) != 0 {
		t.Fatalf("switched off on a project plan = %v, want no effect", off)
	}
}

// With three groups pinned by today's ticks, the rules draw cannot open its
// Revisions group, so a due revision's item comes in under another rule, here
// as its top-level item's next step joining a pinned group. It still carries
// its revision stamp (R10, R45): stamping does not depend on which rule placed
// the entry.
func TestRevisions_RulesPlanStampsADueRevisionThatEntersThroughAnotherRule(t *testing.T) {
	graphs, bfs, dp, trees := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	items := []guided.PlanItem{
		{ID: graphs, Description: "Graphs", Type: "task", Scope: "longterm"},
		{ID: bfs, ParentID: &graphs, Description: "Solve 3 BFS problems", Type: "task", Scope: "longterm"},
		{ID: dp, Description: "Dynamic programming", Type: "task", Scope: "longterm"},
		{ID: trees, Description: "Trees", Type: "task", Scope: "longterm"},
	}
	f := newFixture(items)
	tickedToday(t, f, bfs) // revision 1 of bfs falls due on Oct 26

	f.gen.answer(modelOutput(
		genGroup{Label: "Graphs", Entries: []genEntry{existing(graphs, "x", 30)}},
		genGroup{Label: "Dynamic programming", Entries: []genEntry{existing(dp, "x", 30)}},
		genGroup{Label: "Trees", Entries: []genEntry{existing(trees, "x", 30)}},
	))
	tomorrow, err := f.planMyDayOn(tomorrowDate)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []uuid.UUID{graphs, dp, trees} {
		if _, err := f.tick(entryFor(t, tomorrow, item, "").ID, nil); err != nil {
			t.Fatalf("tick %s: %v", item, err)
		}
	}

	f.gen.fail(errors.New("provider down"))
	p, err := f.planMyDayOn(tomorrowDate)
	if err != nil {
		t.Fatal(err)
	}

	if p.Source != SourceRules || len(p.Groups) != MaxGroups || p.Groups[0].Label != "Graphs" {
		t.Fatalf("rules plan = %+v, want the three pinned groups and no Revisions group", p.Groups)
	}
	e := entryFor(t, p, bfs, "")
	if e.Reason != "Next step" || e.Revision == nil || *e.Revision != 1 {
		t.Fatalf("bfs entry = %+v, want it placed as a next step and stamped revision 1", e)
	}
}
