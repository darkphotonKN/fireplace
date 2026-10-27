package dayplan

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/ai"
	"github.com/google/uuid"
)

func TestGetDayPlan_NothingStored_IsNotPlannedAndNeverGenerates(t *testing.T) {
	f := newFixture(nil)

	p, err := f.getDayPlan()

	if err != nil {
		t.Fatalf("GetDayPlan: %v", err)
	}
	if p.Planned || p.Groups == nil || len(p.Groups) != 0 || p.Source != "" || !p.GeneratedAt.IsZero() {
		t.Fatalf("plan = %+v, want not_planned with empty, non-nil groups", p)
	}
	if p.Key != f.key() || p.RefreshesRemaining != 3 {
		t.Fatalf("key %+v remaining %d, want %+v and 3", p.Key, p.RefreshesRemaining, f.key())
	}
	if f.gen.calls() != 0 || f.store.saves != 0 {
		t.Fatalf("generator calls %d, saves %d: a read must not draw", f.gen.calls(), f.store.saves)
	}
}

func TestPlanMyDay_DrawsOnceStoresTheValidatedGroupsAndARereadDoesNotGenerate(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	f.gen.answer(modelOutput(
		genGroup{Label: "Graphs", Entries: []genEntry{
			existing(tf.bfs, "model's wording", 45),
			existing(tf.graphs, "Graphs", 20),
		}},
		genGroup{Label: "Dynamic programming", Entries: []genEntry{
			newStep(tf.dp, "Solve the climbing stairs problem", 30),
		}},
	))

	p, err := f.planMyDay()

	if err != nil {
		t.Fatalf("PlanMyDay: %v", err)
	}
	if f.gen.calls() != 1 {
		t.Fatalf("generator calls = %d, want exactly 1", f.gen.calls())
	}
	call := f.gen.requests[0].Call
	if call.Kind != ai.KindDayPlan || call.PlanID != f.planID || call.UserID != f.userID {
		t.Fatalf("call = %+v, want kind day_plan for this plan and user", call)
	}
	if !p.Planned || p.Source != SourceAI || !p.GeneratedAt.Equal(today) || p.PlanStateHash == "" || p.RefreshesRemaining != 3 {
		t.Fatalf("plan = %+v, want a planned ai plan drawn now with a hash and 3 refreshes left", p)
	}
	if len(p.Groups) != 2 || p.Groups[0].Label != "Graphs" || p.Groups[1].Label != "Dynamic programming" {
		t.Fatalf("groups = %+v", p.Groups)
	}
	bfs, graphs, step := p.Groups[0].Entries[0], p.Groups[0].Entries[1], p.Groups[1].Entries[0]
	if bfs.Kind != EntryExistingItem || *bfs.ItemID != tf.bfs || *bfs.ParentItemID != tf.graphs ||
		bfs.Text != "Solve 3 BFS problems" || bfs.Reason != "due soon" || *bfs.Minutes != 45 || bfs.Revision != nil {
		t.Fatalf("child entry = %+v, want the item's own text and its parent", bfs)
	}
	if graphs.ParentItemID != nil || *graphs.ItemID != tf.graphs {
		t.Fatalf("top-level entry = %+v, want no parent", graphs)
	}
	if step.Kind != EntryNewStep || step.ItemID != nil || *step.ParentItemID != tf.dp || step.Text != "Solve the climbing stairs problem" {
		t.Fatalf("new step = %+v", step)
	}
	ids := map[uuid.UUID]bool{bfs.ID: true, graphs.ID: true, step.ID: true}
	if len(ids) != 3 || ids[uuid.Nil] {
		t.Fatalf("entry ids %v, want three distinct ids", ids)
	}

	again, err := f.getDayPlan()
	if err != nil {
		t.Fatalf("GetDayPlan: %v", err)
	}
	if f.gen.calls() != 1 {
		t.Fatalf("a read after Plan my day generated (calls %d)", f.gen.calls())
	}
	if !again.Planned || len(again.Groups) != 2 || again.Groups[1].Entries[0].ID != step.ID {
		t.Fatalf("reread = %+v, want the stored plan", again)
	}
}

// editPlan changes the plan so the next Plan my day is not gated.
func editPlan(f *fixture, tf treeFixture, n int) {
	items := tf.items()
	items[3].Description = fmt.Sprintf("Dynamic programming, edit %d", n)
	f.plans.setItems(items)
}

func TestPlanMyDay_FreeDrawThenThreeCountedThenTheFourthIsRefusedAndThePlanKept(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	f.gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}))

	first, err := f.planMyDay()
	if err != nil || first.RefreshesRemaining != 3 {
		t.Fatalf("free draw: remaining %d err %v, want 3", first.RefreshesRemaining, err)
	}
	for i, want := range []int{2, 1, 0} {
		editPlan(f, tf, i)
		p, err := f.planMyDay()
		if err != nil || p.RefreshesRemaining != want {
			t.Fatalf("counted draw %d: remaining %d err %v, want %d", i+1, p.RefreshesRemaining, err, want)
		}
	}
	before, _ := f.store.stored(f.key())
	calls := f.gen.calls()

	editPlan(f, tf, 99)
	_, err = f.planMyDay()

	if !errors.Is(err, guided.ErrRefreshLimitReached) {
		t.Fatalf("fourth counted draw err = %v, want ErrRefreshLimitReached", err)
	}
	if f.gen.calls() != calls {
		t.Fatal("a refused draw called the model")
	}
	if after, _ := f.store.stored(f.key()); after.DrawID != before.DrawID {
		t.Fatal("a refused draw touched the stored plan")
	}
}

func TestPlanMyDay_CountedDrawsShareTheCounterWithTheBriefsRefresh(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	f.gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}))
	if _, err := f.planMyDay(); err != nil {
		t.Fatal(err)
	}
	// Two counted brief refreshes on the same key, through the same counter.
	for range 2 {
		if _, err := f.counter.Spend(context.Background(), f.key()); err != nil {
			t.Fatal(err)
		}
	}

	editPlan(f, tf, 1)
	p, err := f.planMyDay()
	if err != nil || p.RefreshesRemaining != 0 {
		t.Fatalf("third counted draw: remaining %d err %v, want 0", p.RefreshesRemaining, err)
	}
	editPlan(f, tf, 2)
	if _, err := f.planMyDay(); !errors.Is(err, guided.ErrRefreshLimitReached) {
		t.Fatalf("err = %v, want the shared cap reached", err)
	}
}

func TestPlanMyDay_UnchangedPlanReturnsTheStoredPlanWithNoCallAndNoCount(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	f.gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}))
	first, err := f.planMyDay()
	if err != nil {
		t.Fatal(err)
	}
	// The free draw is used; a second draw would count if it reached the model.
	f.svc.now = func() time.Time { return today.Add(time.Hour) }

	again, err := f.planMyDay()

	if err != nil {
		t.Fatal(err)
	}
	if f.gen.calls() != 1 || f.counter.spentFor(f.key()) != 0 {
		t.Fatalf("calls %d spent %d, want the gate to serve with no call and no count", f.gen.calls(), f.counter.spentFor(f.key()))
	}
	if again.DrawID != first.DrawID || !again.GeneratedAt.Equal(first.GeneratedAt) || again.RefreshesRemaining != 3 {
		t.Fatalf("gated plan = %+v, want the stored one with its draw time", again)
	}
}

func TestPlanMyDay_GateCoversPlanTypeFocusAndTheTree(t *testing.T) {
	tf := newTreeFixture()
	for name, change := range map[string]func(f *fixture){
		"plan type": func(f *fixture) { f.plans.profile.Type = "project" },
		"focus":     func(f *fixture) { f.plans.profile.Focus = "Something else" },
		"tree":      func(f *fixture) { editPlan(f, tf, 1) },
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(tf.items())
			f.gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}))
			if _, err := f.planMyDay(); err != nil {
				t.Fatal(err)
			}
			change(f)

			if _, err := f.planMyDay(); err != nil {
				t.Fatal(err)
			}
			if f.gen.calls() != 2 || f.counter.spentFor(f.key()) != 1 {
				t.Fatalf("calls %d spent %d, want a counted redraw", f.gen.calls(), f.counter.spentFor(f.key()))
			}
		})
	}
}

func TestPlanMyDay_AStoredRulesPlanNeverSatisfiesTheGate(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	f.gen.fail(ai.ErrBudgetExhausted)
	if p, err := f.planMyDay(); err != nil || p.Source != SourceRules {
		t.Fatalf("over budget: %+v err %v, want a rules plan", p, err)
	}

	// The budget is back and the plan is unchanged: the model is still tried,
	// and as the key's first successful AI draw it is free.
	f.gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}))
	p, err := f.planMyDay()

	if err != nil {
		t.Fatal(err)
	}
	if f.gen.calls() != 2 || p.Source != SourceAI || !p.FreeDrawUsed || f.counter.spentFor(f.key()) != 0 || p.RefreshesRemaining != 3 {
		t.Fatalf("calls %d plan %+v spent %d, want the model tried and the draw free", f.gen.calls(), p, f.counter.spentFor(f.key()))
	}

	// The next AI draw is a counted one.
	editPlan(f, tf, 1)
	if p, err := f.planMyDay(); err != nil || p.RefreshesRemaining != 2 {
		t.Fatalf("redraw after the free AI draw: remaining %d err %v, want 2", p.RefreshesRemaining, err)
	}
}

func TestPlanMyDay_NoCandidatesStoresAnEmptyPlannedPlanWithNoCallAndNoCount(t *testing.T) {
	tf := newTreeFixture()
	items := tf.items()
	for i := range items {
		items[i].Done = true
	}
	f := newFixture(items)

	p, err := f.planMyDay()

	if err != nil {
		t.Fatal(err)
	}
	if !p.Planned || p.Source != SourceAI || len(p.Groups) != 0 || p.Groups == nil || p.FreeDrawUsed {
		t.Fatalf("plan = %+v, want an empty planned ai plan with the free draw unused", p)
	}
	if f.gen.calls() != 0 || f.counter.spentFor(f.key()) != 0 {
		t.Fatalf("calls %d spent %d, want neither", f.gen.calls(), f.counter.spentFor(f.key()))
	}

	// An item opens: the next draw reaches the model and is still the free one.
	f.plans.setItems(tf.items())
	f.gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}))
	if p, err = f.planMyDay(); err != nil || f.counter.spentFor(f.key()) != 0 || len(p.Groups) != 1 {
		t.Fatalf("after an item opened: plan %+v spent %d err %v, want a free AI draw", p, f.counter.spentFor(f.key()), err)
	}
}

func TestPlanMyDay_OnlyInvalidEntriesStoreAPlannedPlanWithZeroEntries(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	f.gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.dfs, "done already", 30)}}))

	p, err := f.planMyDay()

	if err != nil || !p.Planned || len(p.Groups) != 0 || f.gen.calls() != 1 {
		t.Fatalf("plan %+v err %v calls %d, want a planned plan with no groups after one call", p, err, f.gen.calls())
	}
	if !p.FreeDrawUsed {
		t.Fatal("a successful generation with nothing valid still used the free draw")
	}
}

// generationFailures are the causes R36 falls back on: the global token
// budget, the provider down or timing out after transport retries, and output
// invalid against the schema.
var generationFailures = map[string]func(*fakeGenerator){
	"budget exhausted": func(g *fakeGenerator) { g.fail(fmt.Errorf("%w: 200000 of 200000", ai.ErrBudgetExhausted)) },
	"provider down":    func(g *fakeGenerator) { g.fail(errors.New("ai: chat completion failed after 3 attempts")) },
	"timeout":          func(g *fakeGenerator) { g.fail(fmt.Errorf("ai: chat completion: %w", context.DeadlineExceeded)) },
	"schema-invalid":   func(g *fakeGenerator) { g.answer(`{"moves": []}`) },
}

func TestPlanMyDay_FailedFirstDrawStoresARulesPlanAndKeepsTheFreeDraw(t *testing.T) {
	for name, failWith := range generationFailures {
		t.Run(name, func(t *testing.T) {
			tf := newTreeFixture()
			f := newFixture(tf.items())
			failWith(f.gen)

			p, err := f.planMyDay()

			if err != nil {
				t.Fatalf("err = %v, want a rules plan", err)
			}
			if !p.Planned || p.Source != SourceRules || p.FreeDrawUsed || p.RefreshesRemaining != 3 {
				t.Fatalf("plan %+v, want a planned rules plan with the free draw unused and 3 remaining", p)
			}
			if stored, ok := f.store.stored(f.key()); !ok || stored.DrawID != p.DrawID || stored.Source != SourceRules {
				t.Fatal("the rules plan was not stored")
			}
			if f.gen.calls() != 1 || f.counter.spentFor(f.key()) != 0 {
				t.Fatalf("calls %d spent %d, want only the failed generation and nothing spent", f.gen.calls(), f.counter.spentFor(f.key()))
			}
		})
	}
}

func TestPlanMyDay_FailedCountedDrawReplacesTheAIPlanWithARulesPlanAndGivesTheSpendBack(t *testing.T) {
	for name, failWith := range generationFailures {
		t.Run(name, func(t *testing.T) {
			tf := newTreeFixture()
			f := newFixture(tf.items())
			f.gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}))
			first, err := f.planMyDay()
			if err != nil {
				t.Fatal(err)
			}
			editPlan(f, tf, 1)
			failWith(f.gen)

			p, err := f.planMyDay()

			if err != nil {
				t.Fatalf("err = %v, want a rules plan", err)
			}
			if p.Source != SourceRules || p.DrawID == first.DrawID || !p.FreeDrawUsed || p.RefreshesRemaining != 3 {
				t.Fatalf("plan %+v, want a new rules plan, the free draw still used, 3 remaining", p)
			}
			if f.gen.calls() != 2 || f.counter.spentFor(f.key()) != 0 {
				t.Fatalf("calls %d spent %d, want one generation per draw and the counted spend given back", f.gen.calls(), f.counter.spentFor(f.key()))
			}
		})
	}
}

func TestPlanMyDay_RulesPlanRanksTheTreeWithNoModelOutput(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	f.gen.fail(ai.ErrBudgetExhausted)

	p, err := f.planMyDay()
	if err != nil {
		t.Fatal(err)
	}

	// Graphs is due tomorrow; then each top-level item's next open step, in
	// plan order since none has been updated more recently than another.
	want := "Graphs: Graphs (Due Oct 26) | Solve 3 BFS problems (Next step); Dynamic programming: Dynamic programming (Next step)"
	if render(p.Groups) != want {
		t.Fatalf("rules plan\n got %s\nwant %s", render(p.Groups), want)
	}
	for _, g := range p.Groups {
		for _, e := range g.Entries {
			if e.ID == uuid.Nil || e.Minutes != nil || e.Kind != EntryExistingItem {
				t.Fatalf("entry %+v, want an id, no minutes, an existing item", e)
			}
		}
	}
	if reread, err := f.getDayPlan(); err != nil || render(reread.Groups) != want || reread.Source != SourceRules {
		t.Fatalf("read after the rules draw: %+v err %v", reread, err)
	}
}

func TestPlanMyDay_NextStepFollowsTheMostRecentlyUpdatedTopLevelItem(t *testing.T) {
	tf := newTreeFixture()
	items := tf.items()
	items[0].UpdatedAt = today.Add(-48 * time.Hour) // Graphs
	items[3].UpdatedAt = today.Add(-time.Hour)      // Dynamic programming
	items[0].DueDate = nil
	f := newFixture(items)
	f.gen.fail(ai.ErrBudgetExhausted)

	p, err := f.planMyDay()

	if err != nil {
		t.Fatal(err)
	}
	want := "Dynamic programming: Dynamic programming (Next step); Graphs: Solve 3 BFS problems (Next step)"
	if render(p.Groups) != want {
		t.Fatalf("rules plan\n got %s\nwant %s", render(p.Groups), want)
	}
}

func TestPlanMyDay_AtTheCapAFailingGeneratorIsStillRefusedBeforeAnyDraw(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	f.gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}))
	if _, err := f.planMyDay(); err != nil {
		t.Fatal(err)
	}
	for i := range guided.RefreshCap {
		editPlan(f, tf, i)
		if _, err := f.planMyDay(); err != nil {
			t.Fatal(err)
		}
	}
	before, _ := f.store.stored(f.key())
	calls := f.gen.calls()
	f.gen.fail(ai.ErrBudgetExhausted)
	editPlan(f, tf, 99)

	_, err := f.planMyDay()

	if !errors.Is(err, guided.ErrRefreshLimitReached) {
		t.Fatalf("err = %v, want ErrRefreshLimitReached", err)
	}
	if after, _ := f.store.stored(f.key()); after.DrawID != before.DrawID || f.gen.calls() != calls {
		t.Fatal("a refused draw generated or stored")
	}
}

func TestPlanMyDay_ConcurrentFirstDrawsStoreOnePlanAndBothReceiveIt(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	f.gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}))
	f.gen.barrier = &sync.WaitGroup{}
	f.gen.barrier.Add(2)

	plans := make([]DayPlan, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); plans[i], errs[i] = f.planMyDay() }()
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("draw %d: %v", i, err)
		}
	}
	stored, _ := f.store.stored(f.key())
	if plans[0].DrawID != stored.DrawID || plans[1].DrawID != stored.DrawID {
		t.Fatalf("draws got %s and %s, want both the stored %s", plans[0].DrawID, plans[1].DrawID, stored.DrawID)
	}
	if f.counter.spentFor(f.key()) != 0 || len(f.gen.attempts) != 2 {
		t.Fatalf("spent %d attempts %d, want nothing spent and the lost generation still recorded", f.counter.spentFor(f.key()), len(f.gen.attempts))
	}
}

func TestPlanMyDay_ConcurrentCountedRedrawsSpendAtMostOneDraw(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	f.gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}))
	if _, err := f.planMyDay(); err != nil {
		t.Fatal(err)
	}
	editPlan(f, tf, 1)
	f.gen.barrier = &sync.WaitGroup{}
	f.gen.barrier.Add(2)

	plans := make([]DayPlan, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); plans[i], errs[i] = f.planMyDay() }()
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("redraw %d: %v", i, err)
		}
	}
	stored, _ := f.store.stored(f.key())
	if plans[0].DrawID != stored.DrawID || plans[1].DrawID != stored.DrawID {
		t.Fatal("both callers must receive the one stored plan")
	}
	if spent := f.counter.spentFor(f.key()); spent != 1 {
		t.Fatalf("spent = %d, want exactly one draw's allowance", spent)
	}
	if plans[0].RefreshesRemaining != 2 && plans[1].RefreshesRemaining != 2 {
		t.Fatalf("remaining = %d and %d, want the settled count 2", plans[0].RefreshesRemaining, plans[1].RefreshesRemaining)
	}
}

func TestDayPlan_DateMustBeWithinOneDayOfServerUTC(t *testing.T) {
	tf := newTreeFixture()
	tests := []struct {
		date    string
		wantErr error
	}{
		{"2026-10-24", nil},
		{"2026-10-26", nil},
		{"2026-10-23", guided.ErrDateOutOfRange},
		{"2026-10-27", guided.ErrDateOutOfRange},
		{"25-10-2026", commonconstants.ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			f := newFixture(tf.items())
			f.gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}))

			_, readErr := f.svc.GetDayPlan(context.Background(), f.planID, f.userID, tt.date)
			_, drawErr := f.svc.PlanMyDay(context.Background(), f.planID, f.userID, tt.date)

			for op, err := range map[string]error{"read": readErr, "plan my day": drawErr} {
				if tt.wantErr == nil && err != nil {
					t.Fatalf("%s: %v", op, err)
				}
				if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Fatalf("%s err = %v, want %v", op, err, tt.wantErr)
				}
			}
			if tt.wantErr != nil && (f.gen.calls() != 0 || f.plans.reads != 0) {
				t.Fatal("an out-of-window date reached plan-service or the model")
			}
		})
	}
}

func TestPlanMyDay_PlanServiceNotFoundIsReturnedAndNothingDrawn(t *testing.T) {
	f := newFixture(nil)
	f.plans.err = fmt.Errorf("insights: plan client: get plan: %w", commonconstants.ErrNotFound)

	_, err := f.planMyDay()

	if !errors.Is(err, commonconstants.ErrNotFound) {
		t.Fatalf("err = %v, want NotFound", err)
	}
	if f.gen.calls() != 0 || f.store.saves != 0 {
		t.Fatal("an inaccessible plan was drawn")
	}
}

func TestDayPlan_EachUserHasTheirOwnPlanForASharedPlan(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	f.gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}))
	if _, err := f.planMyDay(); err != nil {
		t.Fatal(err)
	}

	other, err := f.svc.GetDayPlan(context.Background(), f.planID, uuid.New(), todayDate)

	if err != nil || other.Planned {
		t.Fatalf("another member's read = %+v err %v, want not_planned", other, err)
	}
}

func TestPlanMyDay_RequestCarriesPlanTypeFocusAndDateUnderIdenticalRules(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	f.gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}))
	if _, err := f.planMyDay(); err != nil {
		t.Fatal(err)
	}
	editPlan(f, tf, 1)
	if _, err := f.planMyDay(); err != nil {
		t.Fatal(err)
	}

	first, second := f.gen.requests[0], f.gen.requests[1]
	for _, want := range []string{"PLAN TYPE: learning", "PLAN FOCUS: Get through the graph chapter", "Today is 2026-10-25"} {
		if !strings.Contains(first.User, want) {
			t.Fatalf("user message lacks %q:\n%s", want, first.User)
		}
	}
	if first.System != second.System || first.SchemaName != "day_plan" || string(first.Schema) != string(schema) {
		t.Fatal("rules or schema differ between calls")
	}
}

// A caller that goes away mid-draw (its context cancelled) still gets its
// spend back: the refund is not tied to the request's lifetime.
func TestPlanMyDay_ACancelledRequestStillGivesItsSpendBack(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	drawn(t, f, genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}})
	editPlan(f, tf, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.gen.during = cancel
	f.gen.fail(context.Canceled)

	_, _ = f.svc.PlanMyDay(ctx, f.planID, f.userID, todayDate)

	if spent := f.counter.spentFor(f.key()); spent != 0 {
		t.Fatalf("spent = %d after a cancelled counted draw, want the spend given back", spent)
	}
}
