package dayplan

import (
	"context"
	"database/sql"
	"errors"

	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/google/uuid"
)

// The day plan setting's storage (day_plan_settings), per plan and user
// (R95): a row is "asked", and its daily_minutes may be NULL.

// DayPlanSettings returns the user's setting for the plan; the zero
// Settings, not asked, when there is no row.
func (r *Repository) DayPlanSettings(ctx context.Context, planID, userID uuid.UUID) (Settings, error) {
	var minutes sql.NullInt32
	err := r.db.GetContext(ctx, &minutes, `
		SELECT daily_minutes FROM day_plan_settings WHERE plan_id = $1 AND user_id = $2`,
		planID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return Settings{}, nil
	}
	if err != nil {
		return Settings{}, commonhelpers.WrapDBErr("day_plan_settings", "get", err)
	}
	return settingsFrom(minutes), nil
}

// SaveDayPlanSettings upserts the user's setting for the plan, replacing any
// earlier one, and returns what is stored.
func (r *Repository) SaveDayPlanSettings(ctx context.Context, planID, userID uuid.UUID, dailyMinutes *int) (Settings, error) {
	var minutes sql.NullInt32
	if err := r.db.GetContext(ctx, &minutes, `
		INSERT INTO day_plan_settings (plan_id, user_id, daily_minutes) VALUES ($1, $2, $3)
		ON CONFLICT (plan_id, user_id) DO UPDATE SET daily_minutes = EXCLUDED.daily_minutes
		RETURNING daily_minutes`,
		planID, userID, dailyMinutes); err != nil {
		return Settings{}, commonhelpers.WrapDBErr("day_plan_settings", "save", err)
	}
	return settingsFrom(minutes), nil
}

// settingsFrom is a stored row's setting: asked, with its minutes if any.
func settingsFrom(minutes sql.NullInt32) Settings {
	s := Settings{Asked: true}
	if minutes.Valid {
		m := int(minutes.Int32)
		s.DailyMinutes = &m
	}
	return s
}
