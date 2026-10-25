package guided

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// RefreshCap is the number of counted refreshes per (plan, user, date) (R17).
const RefreshCap = 3

// ErrRefreshLimitReached: RefreshCap refreshes are already spent for the key
// (R17). The refresh is refused and nothing stored is touched.
var ErrRefreshLimitReached = errors.New("guided: refresh limit reached")

// DayKey is one user's day on one plan: (plan, user, local date). Date is the
// user's local calendar date, held as midnight UTC. The day plan is stored
// per DayKey, and the refresh counter counts per DayKey, so anything drawn per
// day shares the same cap.
type DayKey struct {
	PlanID uuid.UUID
	UserID uuid.UUID
	Date   time.Time
}

// RefreshCounter is the refresh counter: how many counted refreshes a DayKey
// has spent, out of RefreshCap.
type RefreshCounter interface {
	// Spend takes one counted refresh for key and returns how many remain.
	// The check and the increment are one atomic step: with RefreshCap spent
	// it fails with ErrRefreshLimitReached and changes nothing.
	Spend(ctx context.Context, key DayKey) (remaining int, err error)
	// Return gives back one spent refresh, for a draw that failed after
	// spending (R54). With nothing spent it does nothing.
	Return(ctx context.Context, key DayKey) error
	// Remaining reports how many counted refreshes key has left (R20).
	Remaining(ctx context.Context, key DayKey) (int, error)
}

// RefreshCounters is the Postgres refresh counter (refresh_counters).
type RefreshCounters struct {
	db *sqlx.DB
}

func NewRefreshCounters(db *sqlx.DB) *RefreshCounters {
	return &RefreshCounters{db: db}
}

// Spend is one conditional upsert. Concurrent spends on one key serialize on
// its row; each re-checks spent < cap against the committed count, so at most
// RefreshCap ever succeed.
func (c *RefreshCounters) Spend(ctx context.Context, key DayKey) (int, error) {
	var spent int
	err := c.db.GetContext(ctx, &spent, `
		INSERT INTO refresh_counters (plan_id, user_id, plan_date, spent)
		VALUES ($1, $2, $3::date, 1)
		ON CONFLICT ON CONSTRAINT refresh_counters_pkey
		DO UPDATE SET spent = refresh_counters.spent + 1
		WHERE refresh_counters.spent < $4
		RETURNING spent`,
		key.PlanID, key.UserID, key.Date.Format(DateLayout), RefreshCap)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("%w: %d of %d spent for plan %s on %s",
			ErrRefreshLimitReached, RefreshCap, RefreshCap, key.PlanID, key.Date.Format(DateLayout))
	}
	if err != nil {
		return 0, commonhelpers.WrapDBErr("refresh_counters", "spend", err)
	}
	return RefreshCap - spent, nil
}

func (c *RefreshCounters) Return(ctx context.Context, key DayKey) error {
	_, err := c.db.ExecContext(ctx, `
		UPDATE refresh_counters SET spent = spent - 1
		WHERE plan_id = $1 AND user_id = $2 AND plan_date = $3::date AND spent > 0`,
		key.PlanID, key.UserID, key.Date.Format(DateLayout))
	if err != nil {
		return commonhelpers.WrapDBErr("refresh_counters", "return", err)
	}
	return nil
}

func (c *RefreshCounters) Remaining(ctx context.Context, key DayKey) (int, error) {
	var spent int
	err := c.db.GetContext(ctx, &spent, `
		SELECT spent FROM refresh_counters
		WHERE plan_id = $1 AND user_id = $2 AND plan_date = $3::date`,
		key.PlanID, key.UserID, key.Date.Format(DateLayout))
	if errors.Is(err, sql.ErrNoRows) {
		return RefreshCap, nil
	}
	if err != nil {
		return 0, commonhelpers.WrapDBErr("refresh_counters", "read", err)
	}
	return max(RefreshCap-spent, 0), nil
}
