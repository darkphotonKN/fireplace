package dayplan

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository stores day plans in the insights database (day_plans,
// day_plan_entries) and their day ticks (day_ticks).
type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

type planRow struct {
	ID            uuid.UUID `db:"id"`
	PlanID        uuid.UUID `db:"plan_id"`
	UserID        uuid.UUID `db:"user_id"`
	PlanDate      time.Time `db:"plan_date"`
	DrawID        uuid.UUID `db:"draw_id"`
	Source        string    `db:"source"`
	GeneratedAt   time.Time `db:"generated_at"`
	PlanStateHash string    `db:"plan_state_hash"`
	TickVersion   int64     `db:"tick_version"`
	FreeDrawUsed  bool      `db:"free_draw_used"`
}

type entryRow struct {
	ID            uuid.UUID  `db:"id"`
	GroupPosition int        `db:"group_position"`
	GroupLabel    string     `db:"group_label"`
	Kind          string     `db:"kind"`
	ItemID        *uuid.UUID `db:"item_id"`
	ParentItemID  *uuid.UUID `db:"parent_item_id"`
	Text          string     `db:"text"`
	Reason        string     `db:"reason"`
	Minutes       *int       `db:"minutes"`
	Revision      *int       `db:"revision"`
	Ticked        bool       `db:"ticked"`
}

// Find returns the stored plan for key: the key's row and the entries of the
// draw it serves, in drawn order, each with the day tick on its item for the
// key (R24). Entries are append-only per draw, so a replace committing between
// the two reads still leaves this draw's entries whole (R11).
func (r *Repository) Find(ctx context.Context, key guided.DayKey) (DayPlan, bool, error) {
	var row planRow
	err := r.db.GetContext(ctx, &row, `
		SELECT id, plan_id, user_id, plan_date, draw_id, source, generated_at, plan_state_hash, tick_version, free_draw_used
		FROM day_plans WHERE plan_id = $1 AND user_id = $2 AND plan_date = $3::date`,
		key.PlanID, key.UserID, key.Date.Format(guided.DateLayout))
	if errors.Is(err, sql.ErrNoRows) {
		return DayPlan{}, false, nil
	}
	if err != nil {
		return DayPlan{}, false, commonhelpers.WrapDBErr("day_plans", "find", err)
	}

	var entries []entryRow
	if err := r.db.SelectContext(ctx, &entries, `
		SELECT e.id, e.group_position, e.group_label, e.kind, e.item_id, e.parent_item_id, e.text, e.reason, e.minutes, e.revision,
			EXISTS (
				SELECT 1 FROM day_ticks t
				WHERE t.user_id = $2 AND t.plan_id = $3 AND t.plan_date = $4::date AND t.item_id = e.item_id
			) AS ticked
		FROM day_plan_entries e WHERE e.draw_id = $1 ORDER BY e.position`,
		row.DrawID, row.UserID, row.PlanID, key.Date.Format(guided.DateLayout)); err != nil {
		return DayPlan{}, false, commonhelpers.WrapDBErr("day_plan_entries", "list", err)
	}

	d := row.PlanDate
	return DayPlan{
		Key:           guided.DayKey{PlanID: row.PlanID, UserID: row.UserID, Date: time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)},
		Planned:       true,
		Source:        Source(row.Source),
		GeneratedAt:   row.GeneratedAt.UTC(),
		PlanStateHash: row.PlanStateHash,
		DrawID:        row.DrawID,
		TickVersion:   row.TickVersion,
		FreeDrawUsed:  row.FreeDrawUsed,
		Groups:        toGroups(entries),
	}, true, nil
}

// toGroups regroups entries read in drawn order: a new group starts wherever
// the group position changes.
func toGroups(rows []entryRow) []Group {
	groups := []Group{}
	last := -1
	for _, e := range rows {
		if e.GroupPosition != last {
			groups = append(groups, Group{Label: e.GroupLabel})
			last = e.GroupPosition
		}
		g := &groups[len(groups)-1]
		g.Entries = append(g.Entries, Entry{
			ID: e.ID, Kind: EntryKind(e.Kind), ItemID: e.ItemID, ParentItemID: e.ParentItemID,
			Text: e.Text, Reason: e.Reason, Minutes: e.Minutes, Revision: e.Revision, Ticked: e.Ticked,
		})
	}
	return groups
}

// Save stores p in one transaction if the key is still at prev: with the zero
// DrawRef, an insert that does nothing if a plan for the key exists;
// otherwise an update of the key's row conditional on its draw_id and
// tick_version. Concurrent saves of one key, and a save racing a tick or
// untick (which update the row too), serialize on its unique key or its row,
// and every save after the first change finds the condition false (R21). The
// loser rolls back and reads the plan that is stored. The winner keeps the
// row's tick_version and returns p with it.
func (r *Repository) Save(ctx context.Context, p DayPlan, prev DrawRef) (DayPlan, bool, error) {
	version, won, err := r.save(ctx, p, prev)
	if err != nil {
		return DayPlan{}, false, err
	}
	if won {
		p.TickVersion = version
		return p, true, nil
	}
	stored, ok, err := r.Find(ctx, p.Key)
	if err != nil {
		return DayPlan{}, false, err
	}
	if !ok {
		return DayPlan{}, false, fmt.Errorf("dayplan: plan for %s on %s conflicted but is not stored",
			p.Key.PlanID, p.Key.Date.Format(guided.DateLayout))
	}
	return stored, false, nil
}

func (r *Repository) save(ctx context.Context, p DayPlan, prev DrawRef) (int64, bool, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, false, commonhelpers.WrapDBErr("day_plans", "begin", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit

	date := p.Key.Date.Format(guided.DateLayout)
	var row struct {
		ID          uuid.UUID `db:"id"`
		TickVersion int64     `db:"tick_version"`
	}
	if prev == (DrawRef{}) {
		err = tx.GetContext(ctx, &row, `
			INSERT INTO day_plans (plan_id, user_id, plan_date, draw_id, source, generated_at, plan_state_hash, free_draw_used)
			VALUES ($1, $2, $3::date, $4, $5, $6, $7, $8)
			ON CONFLICT ON CONSTRAINT day_plans_key DO NOTHING
			RETURNING id, tick_version`,
			p.Key.PlanID, p.Key.UserID, date, p.DrawID, string(p.Source), p.GeneratedAt, p.PlanStateHash, p.FreeDrawUsed)
	} else {
		err = tx.GetContext(ctx, &row, `
			UPDATE day_plans
			SET draw_id = $5, source = $6, generated_at = $7, plan_state_hash = $8, free_draw_used = $9
			WHERE plan_id = $1 AND user_id = $2 AND plan_date = $3::date AND draw_id = $4 AND tick_version = $10
			RETURNING id, tick_version`,
			p.Key.PlanID, p.Key.UserID, date, prev.DrawID, p.DrawID, string(p.Source), p.GeneratedAt, p.PlanStateHash, p.FreeDrawUsed, prev.TickVersion)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, commonhelpers.WrapDBErr("day_plans", "save", err)
	}
	id := row.ID

	position := 0
	for gi, g := range p.Groups {
		for _, e := range g.Entries {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO day_plan_entries
					(id, day_plan_id, draw_id, position, group_position, group_label, kind, item_id, parent_item_id, text, reason, minutes, revision)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
				e.ID, id, p.DrawID, position, gi, g.Label, string(e.Kind), e.ItemID, e.ParentItemID, e.Text, e.Reason, e.Minutes, e.Revision,
			); err != nil {
				return 0, false, commonhelpers.WrapDBErr("day_plan_entries", "insert", err)
			}
			position++
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, false, commonhelpers.WrapDBErr("day_plans", "commit", err)
	}
	return row.TickVersion, true, nil
}

// FindByEntry returns the user's stored plan whose current draw holds entry
// entryID, as Find reads it. An entry only in a replaced draw, or another
// user's, is not found.
func (r *Repository) FindByEntry(ctx context.Context, entryID, userID uuid.UUID) (DayPlan, bool, error) {
	var row struct {
		PlanID   uuid.UUID `db:"plan_id"`
		PlanDate time.Time `db:"plan_date"`
	}
	err := r.db.GetContext(ctx, &row, `
		SELECT p.plan_id, p.plan_date
		FROM day_plan_entries e JOIN day_plans p ON p.draw_id = e.draw_id
		WHERE e.id = $1 AND p.user_id = $2`, entryID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return DayPlan{}, false, nil
	}
	if err != nil {
		return DayPlan{}, false, commonhelpers.WrapDBErr("day_plan_entries", "find by id", err)
	}
	d := row.PlanDate
	return r.Find(ctx, guided.DayKey{PlanID: row.PlanID, UserID: userID, Date: time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)})
}

// Tick records t in one transaction. First it moves the key's tick_version,
// which takes the key's row lock: a redraw in flight then loses its save
// (R21), and ticks of one key, and ticks against saves, run one after the
// other. When t.Bind is set, the entry's row in draw t.DrawID becomes an
// existing item for t.ItemID, provided it is still an unbound new step of the
// draw the key serves and no other entry of the draw holds the item
// (ErrEntryNotTickable otherwise; under the row lock, a concurrent bind of the
// item has committed before this statement reads). Then the tick is
// inserted, doing nothing if the item is already ticked for the date (created
// false, R27).
func (r *Repository) Tick(ctx context.Context, t Tick) (bool, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, commonhelpers.WrapDBErr("day_ticks", "begin", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit

	date := t.Key.Date.Format(guided.DateLayout)
	if err := bumpTickVersion(ctx, tx, t.Key); err != nil {
		return false, err
	}
	if t.Bind {
		res, err := tx.ExecContext(ctx, `
			UPDATE day_plan_entries e SET kind = 'existing_item', item_id = $3
			WHERE e.draw_id = $1 AND e.id = $2 AND e.kind = 'new_step'
			AND EXISTS (
				SELECT 1 FROM day_plans p
				WHERE p.draw_id = $1 AND p.plan_id = $4 AND p.user_id = $5 AND p.plan_date = $6::date
			)
			AND NOT EXISTS (
				SELECT 1 FROM day_plan_entries o WHERE o.draw_id = $1 AND o.item_id = $3
			)`, t.DrawID, t.EntryID, t.ItemID, t.Key.PlanID, t.Key.UserID, date)
		if err != nil {
			return false, commonhelpers.WrapDBErr("day_plan_entries", "bind", err)
		}
		if n, err := res.RowsAffected(); err != nil || n != 1 {
			return false, fmt.Errorf("%w: entry %s is no longer an unbound new step of the plan, or item %s has an entry", ErrEntryNotTickable, t.EntryID, t.ItemID)
		}
	}

	var id uuid.UUID
	err = tx.GetContext(ctx, &id, `
		INSERT INTO day_ticks (user_id, plan_id, item_id, plan_date, entry_id)
		VALUES ($1, $2, $3, $4::date, $5)
		ON CONFLICT ON CONSTRAINT day_ticks_key DO NOTHING
		RETURNING id`, t.Key.UserID, t.Key.PlanID, t.ItemID, date, t.EntryID)
	created := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, commonhelpers.WrapDBErr("day_ticks", "insert", err)
	}
	if err := tx.Commit(); err != nil {
		return false, commonhelpers.WrapDBErr("day_ticks", "commit", err)
	}
	return created, nil
}

// Untick deletes the day tick on itemID for key, in one transaction with
// moving the key's tick_version (as Tick does); removed is false when there
// was none.
func (r *Repository) Untick(ctx context.Context, key guided.DayKey, itemID uuid.UUID) (bool, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, commonhelpers.WrapDBErr("day_ticks", "begin", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit

	if err := bumpTickVersion(ctx, tx, key); err != nil {
		return false, err
	}
	res, err := tx.ExecContext(ctx, `
		DELETE FROM day_ticks
		WHERE user_id = $1 AND plan_id = $2 AND plan_date = $3::date AND item_id = $4`,
		key.UserID, key.PlanID, key.Date.Format(guided.DateLayout), itemID)
	if err != nil {
		return false, commonhelpers.WrapDBErr("day_ticks", "delete", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, commonhelpers.WrapDBErr("day_ticks", "delete", err)
	}
	if err := tx.Commit(); err != nil {
		return false, commonhelpers.WrapDBErr("day_ticks", "commit", err)
	}
	return n > 0, nil
}

// bumpTickVersion moves key's tick_version inside tx, taking the key's row
// lock until tx ends. A key with no stored plan has no row and nothing moves.
func bumpTickVersion(ctx context.Context, tx *sqlx.Tx, key guided.DayKey) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE day_plans SET tick_version = tick_version + 1
		WHERE plan_id = $1 AND user_id = $2 AND plan_date = $3::date`,
		key.PlanID, key.UserID, key.Date.Format(guided.DateLayout)); err != nil {
		return commonhelpers.WrapDBErr("day_plans", "tick version", err)
	}
	return nil
}
