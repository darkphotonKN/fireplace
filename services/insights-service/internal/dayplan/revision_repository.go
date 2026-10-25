package dayplan

import (
	"context"
	"time"

	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// The revision ladder's storage (revisions, revision_switches), per user,
// plan and item. The rules live in ladder.go; this only reads and applies.

type rungRow struct {
	Rung        int        `db:"rung"`
	ScheduledOn time.Time  `db:"scheduled_on"`
	DueOn       time.Time  `db:"due_on"`
	CompletedOn *time.Time `db:"completed_on"`
}

// Ladder returns the item's ladder as a day tick on key's date sees it: its
// rungs, whether it has a day tick on any other date (if not, this is its
// first tick ever, R42), and its switch.
func (r *Repository) Ladder(ctx context.Context, key guided.DayKey, itemID uuid.UUID) (ladderState, error) {
	var rows []rungRow
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT rung, scheduled_on, due_on, completed_on
		FROM revisions WHERE user_id = $1 AND plan_id = $2 AND item_id = $3 ORDER BY rung`,
		key.UserID, key.PlanID, itemID); err != nil {
		return ladderState{}, commonhelpers.WrapDBErr("revisions", "list", err)
	}
	var flags struct {
		TickedBefore bool  `db:"ticked_before"`
		Enabled      *bool `db:"enabled"`
	}
	if err := r.db.GetContext(ctx, &flags, `
		SELECT
			EXISTS (
				SELECT 1 FROM day_ticks
				WHERE user_id = $1 AND plan_id = $2 AND item_id = $3 AND plan_date <> $4::date
			) AS ticked_before,
			(SELECT enabled FROM revision_switches WHERE user_id = $1 AND plan_id = $2 AND item_id = $3) AS enabled`,
		key.UserID, key.PlanID, itemID, key.Date.Format(guided.DateLayout)); err != nil {
		return ladderState{}, commonhelpers.WrapDBErr("revisions", "ladder", err)
	}

	st := ladderState{
		FirstTick: !flags.TickedBefore,
		Off:       flags.Enabled != nil && !*flags.Enabled,
		Resume:    flags.Enabled != nil && *flags.Enabled,
	}
	for _, row := range rows {
		rg := rung{Rung: row.Rung, ScheduledOn: utcDay(row.ScheduledOn), DueOn: utcDay(row.DueOn)}
		if row.CompletedOn != nil {
			c := utcDay(*row.CompletedOn)
			rg.CompletedOn = &c
		}
		st.Rungs = append(st.Rungs, rg)
	}
	return st, nil
}

// ApplyLadder applies c to the item's ladder for key's date in one
// transaction. Each write is conditional, so applying a change twice leaves
// the ladder as applying it once: a rung is completed only while open,
// reopened only if completed on the date, and scheduled only if absent.
func (r *Repository) ApplyLadder(ctx context.Context, key guided.DayKey, itemID uuid.UUID, c ladderChange) error {
	if c.Complete == nil && c.Schedule == nil && c.Reopen == nil && len(c.Remove) == 0 {
		return nil
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return commonhelpers.WrapDBErr("revisions", "begin", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit

	date := key.Date.Format(guided.DateLayout)
	if len(c.Remove) > 0 {
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM revisions WHERE user_id = $1 AND plan_id = $2 AND item_id = $3 AND rung = ANY($4)`,
			key.UserID, key.PlanID, itemID, pq.Array(c.Remove)); err != nil {
			return commonhelpers.WrapDBErr("revisions", "remove", err)
		}
	}
	if c.Reopen != nil {
		if _, err := tx.ExecContext(ctx, `
			UPDATE revisions SET completed_on = NULL
			WHERE user_id = $1 AND plan_id = $2 AND item_id = $3 AND rung = $4 AND completed_on = $5::date`,
			key.UserID, key.PlanID, itemID, *c.Reopen, date); err != nil {
			return commonhelpers.WrapDBErr("revisions", "reopen", err)
		}
	}
	if c.Complete != nil {
		if _, err := tx.ExecContext(ctx, `
			UPDATE revisions SET completed_on = $5::date
			WHERE user_id = $1 AND plan_id = $2 AND item_id = $3 AND rung = $4 AND completed_on IS NULL`,
			key.UserID, key.PlanID, itemID, *c.Complete, date); err != nil {
			return commonhelpers.WrapDBErr("revisions", "complete", err)
		}
	}
	if s := c.Schedule; s != nil {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO revisions (user_id, plan_id, item_id, rung, scheduled_on, due_on)
			VALUES ($1, $2, $3, $4, $5::date, $6::date)
			ON CONFLICT ON CONSTRAINT revisions_key DO NOTHING`,
			key.UserID, key.PlanID, itemID, s.Rung,
			s.ScheduledOn.Format(guided.DateLayout), s.DueOn.Format(guided.DateLayout)); err != nil {
			return commonhelpers.WrapDBErr("revisions", "schedule", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return commonhelpers.WrapDBErr("revisions", "commit", err)
	}
	return nil
}

// DueRevisions returns the user's revisions due on key's date on key's plan
// (R44): open, due on or before the date, of items whose revisions are not
// switched off; oldest due first. Whether the item is still in the plan is
// the caller's check against the plan tree.
func (r *Repository) DueRevisions(ctx context.Context, key guided.DayKey) ([]dueRevision, error) {
	var rows []struct {
		ItemID uuid.UUID `db:"item_id"`
		Rung   int       `db:"rung"`
		DueOn  time.Time `db:"due_on"`
	}
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT v.item_id, v.rung, v.due_on
		FROM revisions v
		WHERE v.user_id = $1 AND v.plan_id = $2 AND v.completed_on IS NULL AND v.due_on <= $3::date
		AND NOT EXISTS (
			SELECT 1 FROM revision_switches s
			WHERE s.user_id = v.user_id AND s.plan_id = v.plan_id AND s.item_id = v.item_id AND NOT s.enabled
		)
		ORDER BY v.due_on, v.item_id`,
		key.UserID, key.PlanID, key.Date.Format(guided.DateLayout)); err != nil {
		return nil, commonhelpers.WrapDBErr("revisions", "due", err)
	}
	out := make([]dueRevision, 0, len(rows))
	for _, row := range rows {
		out = append(out, dueRevision{ItemID: row.ItemID, Rung: row.Rung, DueOn: utcDay(row.DueOn)})
	}
	return out, nil
}

// RevisionsOff returns the user's items on the plan whose revisions are
// switched off.
func (r *Repository) RevisionsOff(ctx context.Context, planID, userID uuid.UUID) (map[uuid.UUID]bool, error) {
	var ids []uuid.UUID
	if err := r.db.SelectContext(ctx, &ids, `
		SELECT item_id FROM revision_switches WHERE user_id = $1 AND plan_id = $2 AND NOT enabled`,
		userID, planID); err != nil {
		return nil, commonhelpers.WrapDBErr("revision_switches", "list", err)
	}
	off := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		off[id] = true
	}
	return off, nil
}

// SetRevisions switches the item's revisions (R47). Off records the switch
// and cancels the item's open rung in one transaction. On flips a switched-off
// item back on, which the next tick reads as a resume; an item never switched
// off is already on, and nothing is written.
func (r *Repository) SetRevisions(ctx context.Context, planID, userID, itemID uuid.UUID, enabled bool) error {
	if enabled {
		if _, err := r.db.ExecContext(ctx, `
			UPDATE revision_switches SET enabled = TRUE, updated_at = NOW()
			WHERE user_id = $1 AND plan_id = $2 AND item_id = $3 AND NOT enabled`,
			userID, planID, itemID); err != nil {
			return commonhelpers.WrapDBErr("revision_switches", "on", err)
		}
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return commonhelpers.WrapDBErr("revision_switches", "begin", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO revision_switches (user_id, plan_id, item_id, enabled) VALUES ($1, $2, $3, FALSE)
		ON CONFLICT (user_id, plan_id, item_id) DO UPDATE SET enabled = FALSE, updated_at = NOW()`,
		userID, planID, itemID); err != nil {
		return commonhelpers.WrapDBErr("revision_switches", "off", err)
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM revisions WHERE user_id = $1 AND plan_id = $2 AND item_id = $3 AND completed_on IS NULL`,
		userID, planID, itemID); err != nil {
		return commonhelpers.WrapDBErr("revisions", "cancel", err)
	}
	if err := tx.Commit(); err != nil {
		return commonhelpers.WrapDBErr("revision_switches", "commit", err)
	}
	return nil
}

// utcDay is a DATE column's value as the plan date it names.
func utcDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
