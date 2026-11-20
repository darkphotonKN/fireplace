package dayplan

import (
	"context"
	"errors"
	"fmt"
	"testing"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
)

// FS-02JBZ §Change: Hours a day, R93–R95, R101–R103: the day plan setting
// per (user, plan), returned on every day plan and saved without drawing.

func (f *fixture) setDailyMinutes(minutes *int) (Settings, error) {
	return f.svc.SetDayPlanSettings(context.Background(), f.planID, f.userID, minutes)
}

func TestSetDayPlanSettings_AFreshKeyIsNotAskedAndASaveIsReadBack(t *testing.T) {
	f := newFixture(nil)

	p, err := f.getDayPlan()
	if err != nil {
		t.Fatalf("GetDayPlan: %v", err)
	}
	if p.Settings.Asked || p.Settings.DailyMinutes != nil {
		t.Fatalf("settings = %+v, want not asked and no daily minutes", p.Settings)
	}

	saved, err := f.setDailyMinutes(ptr(120))
	if err != nil {
		t.Fatalf("SetDayPlanSettings: %v", err)
	}
	if !saved.Asked || saved.DailyMinutes == nil || *saved.DailyMinutes != 120 {
		t.Fatalf("saved = %+v, want 120 and asked", saved)
	}
	p, err = f.getDayPlan()
	if err != nil {
		t.Fatalf("GetDayPlan: %v", err)
	}
	if !p.Settings.Asked || p.Settings.DailyMinutes == nil || *p.Settings.DailyMinutes != 120 {
		t.Fatalf("settings after the save = %+v, want 120 and asked", p.Settings)
	}
}

func TestSetDayPlanSettings_NilClearsTheMinutesAndStaysAsked(t *testing.T) {
	f := newFixture(nil)
	if _, err := f.setDailyMinutes(ptr(120)); err != nil {
		t.Fatal(err)
	}

	saved, err := f.setDailyMinutes(nil)
	if err != nil {
		t.Fatalf("SetDayPlanSettings(nil): %v", err)
	}
	p, err := f.getDayPlan()
	if err != nil {
		t.Fatal(err)
	}
	for name, s := range map[string]Settings{"saved": saved, "read": p.Settings} {
		if !s.Asked || s.DailyMinutes != nil {
			t.Fatalf("%s = %+v, want asked with no daily minutes", name, s)
		}
	}
}

func TestSetDayPlanSettings_OffRangeOrOffStepIsInvalidInputAndNothingIsRead(t *testing.T) {
	for _, minutes := range []int{-30, 0, 29, 45, 721, 750} {
		f := newFixture(nil)

		_, err := f.setDailyMinutes(ptr(minutes))

		if !errors.Is(err, ErrDailyMinutesInvalid) || !errors.Is(err, commonconstants.ErrInvalidInput) {
			t.Fatalf("%d: err = %v, want ErrDailyMinutesInvalid (an InvalidInput)", minutes, err)
		}
		if f.plans.reads != 0 || len(f.store.settings) != 0 {
			t.Fatalf("%d: plan reads %d, settings %v; a refused value reached plan-service or the store", minutes, f.plans.reads, f.store.settings)
		}
	}
	for _, minutes := range []int{30, 60, 90, 720} {
		if _, err := newFixture(nil).setDailyMinutes(ptr(minutes)); err != nil {
			t.Fatalf("%d refused: %v", minutes, err)
		}
	}
}

func TestSetDayPlanSettings_APlanTheCallerCannotAccessIsNotFoundAndNothingIsSaved(t *testing.T) {
	f := newFixture(nil)
	f.plans.err = fmt.Errorf("insights: plan client: get plan: %w", commonconstants.ErrNotFound)

	_, err := f.setDailyMinutes(ptr(120))

	if !errors.Is(err, commonconstants.ErrNotFound) {
		t.Fatalf("err = %v, want NotFound", err)
	}
	if len(f.store.settings) != 0 {
		t.Fatalf("settings %v stored for an inaccessible plan", f.store.settings)
	}
}

// R103: a save makes no model call and no rules draw, leaves the stored day
// plan as it is, spends nothing and records nothing in the ledger.
func TestSetDayPlanSettings_NeverDrawsAndLeavesThePlanAndTheCounterAlone(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	drawn(t, f, genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}})
	editPlan(f, tf, 1)
	before, _ := f.store.stored(f.key())
	calls, saves, spent := f.gen.calls(), f.store.saves, f.counter.spentFor(f.key())

	if _, err := f.setDailyMinutes(ptr(90)); err != nil {
		t.Fatal(err)
	}

	after, _ := f.store.stored(f.key())
	if f.gen.calls() != calls || len(f.gen.attempts) != calls {
		t.Fatalf("generator calls %d attempts %d, want %d: a save called the model", f.gen.calls(), len(f.gen.attempts), calls)
	}
	if f.store.saves != saves || after.DrawID != before.DrawID || !after.GeneratedAt.Equal(before.GeneratedAt) {
		t.Fatalf("saves %d (was %d), draw %s (was %s): a save drew", f.store.saves, saves, after.DrawID, before.DrawID)
	}
	if got := f.counter.spentFor(f.key()); got != spent {
		t.Fatalf("spent = %d, want %d: a save touched the counter", got, spent)
	}
}

// R101: the setting rides on every day plan PlanMyDay returns, planned or not.
func TestPlanMyDay_CarriesTheCallersSettingInEveryState(t *testing.T) {
	tf := newTreeFixture()
	f := newFixture(tf.items())
	f.gen.answer(modelOutput(genGroup{Label: "Graphs", Entries: []genEntry{existing(tf.bfs, "x", 30)}}))

	fresh, err := f.planMyDay()
	if err != nil {
		t.Fatal(err)
	}
	if !fresh.Planned || fresh.Settings.Asked || fresh.Settings.DailyMinutes != nil {
		t.Fatalf("plan %v settings %+v, want planned and not asked", fresh.Planned, fresh.Settings)
	}

	if _, err := f.setDailyMinutes(ptr(150)); err != nil {
		t.Fatal(err)
	}
	again, err := f.planMyDay() // planned again after the save
	if err != nil {
		t.Fatal(err)
	}
	f.gen.fail(errors.New("provider down"))
	editPlan(f, tf, 1)
	rules, err := f.planMyDay() // a failed draw falls back to rules
	if err != nil {
		t.Fatal(err)
	}
	for name, p := range map[string]DayPlan{"again": again, "rules": rules} {
		if !p.Settings.Asked || p.Settings.DailyMinutes == nil || *p.Settings.DailyMinutes != 150 {
			t.Fatalf("%s settings = %+v, want 150 and asked", name, p.Settings)
		}
	}
	if rules.Source != SourceRules {
		t.Fatalf("source = %q, want rules", rules.Source)
	}
}

// R93: on a shared plan each member has their own setting.
func TestSetDayPlanSettings_OneMembersSaveDoesNotChangeAnothers(t *testing.T) {
	f := newFixture(nil)
	if _, err := f.setDailyMinutes(ptr(240)); err != nil {
		t.Fatal(err)
	}

	other, err := f.svc.GetDayPlan(context.Background(), f.planID, uuid.New(), todayDate)

	if err != nil || other.Settings.Asked || other.Settings.DailyMinutes != nil {
		t.Fatalf("another member's settings = %+v err %v, want not asked", other.Settings, err)
	}
}
