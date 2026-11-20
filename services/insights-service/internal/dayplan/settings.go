package dayplan

import (
	"context"
	"fmt"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
)

// Bounds of a day plan setting's daily time (FS-02JBZ R94): whole minutes,
// 30 to 720 in steps of 30.
const (
	MinDailyMinutes  = 30
	MaxDailyMinutes  = 720
	DailyMinutesStep = 30
)

// ErrDailyMinutesInvalid: daily minutes outside 30–720 or off the 30-minute
// step (R105). It is an InvalidInput, so the shared mapper answers
// InvalidArgument.
var ErrDailyMinutesInvalid = fmt.Errorf("%w: daily minutes must be %d-%d in steps of %d",
	commonconstants.ErrInvalidInput, MinDailyMinutes, MaxDailyMinutes, DailyMinutesStep)

// Settings is one user's day plan setting for one plan (R93, R94).
// DailyMinutes is the user's daily time on the plan, nil when unset. Asked is
// true once the user has saved anything for the plan, a nil DailyMinutes
// included; the zero Settings is a user who never has.
type Settings struct {
	DailyMinutes *int
	Asked        bool
}

// SettingsStore persists day plan settings per (plan, user) (R95).
type SettingsStore interface {
	// DayPlanSettings returns the user's setting for the plan; the zero
	// Settings when they never saved one.
	DayPlanSettings(ctx context.Context, planID, userID uuid.UUID) (Settings, error)
	// SaveDayPlanSettings stores dailyMinutes as the user's setting for the
	// plan, replacing any earlier one, and returns what is stored.
	SaveDayPlanSettings(ctx context.Context, planID, userID uuid.UUID, dailyMinutes *int) (Settings, error)
}

// SetDayPlanSettings saves the caller's day plan setting for the plan (R102):
// dailyMinutes, or nil to go back to the default, and returns the saved
// setting, asked from then on. A later save replaces it.
//
// It refuses daily minutes off the range or step with ErrDailyMinutesInvalid
// (R105), and a plan the caller cannot access is NotFound from plan-service,
// asked as the caller (R106). It never draws: no model call, no rules draw,
// nothing written to the stored day plan, the refresh counter or the ledger
// (R103). The value applies the next time the user plans their day.
func (s *Service) SetDayPlanSettings(ctx context.Context, planID, userID uuid.UUID, dailyMinutes *int) (Settings, error) {
	if !validDailyMinutes(dailyMinutes) {
		return Settings{}, fmt.Errorf("%w: got %d", ErrDailyMinutesInvalid, *dailyMinutes)
	}
	if _, err := s.plans.GetPlanProfile(ctx, planID, userID); err != nil {
		return Settings{}, fmt.Errorf("dayplan: read plan: %w", err)
	}
	saved, err := s.settings.SaveDayPlanSettings(ctx, planID, userID, dailyMinutes)
	if err != nil {
		return Settings{}, fmt.Errorf("dayplan: save settings: %w", err)
	}
	return saved, nil
}

// validDailyMinutes reports whether m is unset or within R94's range and step.
func validDailyMinutes(m *int) bool {
	return m == nil || (*m >= MinDailyMinutes && *m <= MaxDailyMinutes && *m%DailyMinutesStep == 0)
}
