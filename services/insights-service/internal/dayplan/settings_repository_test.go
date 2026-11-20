package dayplan

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// The day plan setting's storage against the insights test database (R95).
// Skipped as openTestDB skips, and when day_plan_settings has not been
// migrated there.

func openSettingsDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db := openTestDB(t)
	var exists bool
	if err := db.Get(&exists, `SELECT to_regclass('public.day_plan_settings') IS NOT NULL`); err != nil || !exists {
		t.Skip("day_plan_settings not migrated in the test database")
	}
	return db
}

// settingsPlan is a fresh plan id whose settings rows are removed after the test.
func settingsPlan(t *testing.T, db *sqlx.DB) uuid.UUID {
	t.Helper()
	planID := uuid.New()
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM day_plan_settings WHERE plan_id = $1`, planID); err != nil {
			t.Errorf("cleanup day_plan_settings: %v", err)
		}
	})
	return planID
}

func TestSettingsRepositoryIntegration_NeverSavedThenSavedReplacedAndCleared(t *testing.T) {
	db := openSettingsDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	planID, userID := settingsPlan(t, db), uuid.New()

	got, err := repo.DayPlanSettings(ctx, planID, userID)
	if err != nil || got.Asked || got.DailyMinutes != nil {
		t.Fatalf("never saved = %+v, %v; want not asked, no minutes", got, err)
	}

	for _, want := range []*int{ptr(120), ptr(720), nil, ptr(30)} {
		saved, err := repo.SaveDayPlanSettings(ctx, planID, userID, want)
		if err != nil {
			t.Fatalf("SaveDayPlanSettings(%v): %v", want, err)
		}
		read, err := repo.DayPlanSettings(ctx, planID, userID)
		if err != nil {
			t.Fatal(err)
		}
		for name, s := range map[string]Settings{"saved": saved, "read": read} {
			if !s.Asked || !equalMinutes(s.DailyMinutes, want) {
				t.Fatalf("%s after saving %v = %+v; want asked with those minutes", name, deref(want), s)
			}
		}
	}
}

func TestSettingsRepositoryIntegration_EachMemberOfASharedPlanHasTheirOwn(t *testing.T) {
	db := openSettingsDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	planID, owner, member := settingsPlan(t, db), uuid.New(), uuid.New()

	if _, err := repo.SaveDayPlanSettings(ctx, planID, owner, ptr(180)); err != nil {
		t.Fatal(err)
	}

	got, err := repo.DayPlanSettings(ctx, planID, member)
	if err != nil || got.Asked || got.DailyMinutes != nil {
		t.Fatalf("member's setting = %+v, %v; want untouched by the owner's save", got, err)
	}
}

// The migration's CHECK refuses what R94 does, whoever writes it.
func TestSettingsMigrationIntegration_CheckRefusesOffStepAndOutOfRange(t *testing.T) {
	db := openSettingsDB(t)
	planID := settingsPlan(t, db)

	for _, minutes := range []int{0, 29, 45, 721, 750} {
		if _, err := db.Exec(`INSERT INTO day_plan_settings (plan_id, user_id, daily_minutes) VALUES ($1, $2, $3)`,
			planID, uuid.New(), minutes); err == nil {
			t.Errorf("daily_minutes %d was stored, want the CHECK to refuse it", minutes)
		}
	}
	for _, minutes := range []any{30, 90, 720, nil} {
		if _, err := db.Exec(`INSERT INTO day_plan_settings (plan_id, user_id, daily_minutes) VALUES ($1, $2, $3)`,
			planID, uuid.New(), minutes); err != nil {
			t.Errorf("daily_minutes %v refused: %v", minutes, err)
		}
	}
}

func equalMinutes(a, b *int) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func deref(m *int) any {
	if m == nil {
		return nil
	}
	return *m
}
