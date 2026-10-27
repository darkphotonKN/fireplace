package dayplan

import (
	"context"
	"testing"
	"time"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// The revision ladder's storage against the insights test database. Skipped as
// openTestDB skips, and when revisions has not been migrated there.

func openRevisionDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db := openTickDB(t)
	var exists bool
	if err := db.Get(&exists, `SELECT to_regclass('public.revisions') IS NOT NULL`); err != nil || !exists {
		t.Skip("revisions not migrated in the test database")
	}
	return db
}

func onDate(key guided.DayKey, date time.Time) guided.DayKey {
	key.Date = date
	return key
}

func TestRevisionRepositoryIntegration_LadderRoundTrip(t *testing.T) {
	db := openRevisionDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	key := testKey(t, db)
	item := uuid.New()
	day1 := onDate(key, oct(20))
	day2 := onDate(key, oct(21))

	// The first tick ever: no tick on another date, no rungs.
	if _, err := repo.Tick(ctx, Tick{Key: day1, ItemID: item, EntryID: uuid.New()}); err != nil {
		t.Fatal(err)
	}
	st, err := repo.Ladder(ctx, day1, item)
	if err != nil {
		t.Fatal(err)
	}
	if !st.FirstTick || len(st.Rungs) != 0 || st.Off || st.Resume {
		t.Fatalf("state on the first tick = %+v, want a first tick and nothing else", st)
	}
	if err := repo.ApplyLadder(ctx, day1, item, onTick(st, day1.Date)); err != nil {
		t.Fatal(err)
	}

	// Revision 1 is due on day 2 and not before.
	if due, err := repo.DueRevisions(ctx, day1); err != nil || len(due) != 0 {
		t.Fatalf("due on day 1 = %+v, %v; want none", due, err)
	}
	due, err := repo.DueRevisions(ctx, day2)
	if err != nil || len(due) != 1 || due[0].ItemID != item || due[0].Rung != 1 || !due[0].DueOn.Equal(oct(21)) {
		t.Fatalf("due on day 2 = %+v, %v; want revision 1 of the item, due Oct 21", due, err)
	}

	// Ticking it on day 2: not a first tick any more; completes 1, schedules 2.
	if _, err := repo.Tick(ctx, Tick{Key: day2, ItemID: item, EntryID: uuid.New()}); err != nil {
		t.Fatal(err)
	}
	st, err = repo.Ladder(ctx, day2, item)
	if err != nil {
		t.Fatal(err)
	}
	if st.FirstTick || len(st.Rungs) != 1 || st.Rungs[0].Rung != 1 || !st.Rungs[0].ScheduledOn.Equal(oct(20)) {
		t.Fatalf("state on day 2 = %+v, want revision 1 scheduled Oct 20, not a first tick", st)
	}
	if err := repo.ApplyLadder(ctx, day2, item, onTick(st, day2.Date)); err != nil {
		t.Fatal(err)
	}
	if due, err := repo.DueRevisions(ctx, onDate(key, oct(24))); err != nil || len(due) != 1 || due[0].Rung != 2 {
		t.Fatalf("due on Oct 24 = %+v, %v; want revision 2", due, err)
	}

	// Untick day 2 undoes it: revision 1 reopened (due again), revision 2 gone.
	st, err = repo.Ladder(ctx, day2, item)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.ApplyLadder(ctx, day2, item, onUntick(st.Rungs, day2.Date)); err != nil {
		t.Fatal(err)
	}
	due, err = repo.DueRevisions(ctx, onDate(key, oct(24)))
	if err != nil || len(due) != 1 || due[0].Rung != 1 {
		t.Fatalf("due after the untick = %+v, %v; want revision 1 reopened and nothing else", due, err)
	}

	// Revisions are per user.
	other := onDate(key, oct(24))
	other.UserID = uuid.New()
	if due, err := repo.DueRevisions(ctx, other); err != nil || len(due) != 0 {
		t.Fatalf("another user's due = %+v, %v; want none", due, err)
	}
}

func TestRevisionRepositoryIntegration_SwitchCancelsTheOpenRungAndResumes(t *testing.T) {
	db := openRevisionDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	key := testKey(t, db)
	item, untouched := uuid.New(), uuid.New()
	day := onDate(key, oct(20))
	if err := repo.ApplyLadder(ctx, day, item, ladderChange{Schedule: schedule(1, oct(20))}); err != nil {
		t.Fatal(err)
	}

	// On for an item never off writes nothing: the next tick does not resume.
	if err := repo.SetRevisions(ctx, key.PlanID, key.UserID, untouched, true); err != nil {
		t.Fatal(err)
	}
	if st, err := repo.Ladder(ctx, day, untouched); err != nil || st.Resume || st.Off {
		t.Fatalf("never-off item = %+v, %v; want on, no resume", st, err)
	}

	if err := repo.SetRevisions(ctx, key.PlanID, key.UserID, item, false); err != nil {
		t.Fatal(err)
	}
	if due, err := repo.DueRevisions(ctx, onDate(key, oct(22))); err != nil || len(due) != 0 {
		t.Fatalf("due after off = %+v, %v; want the open rung cancelled", due, err)
	}
	off, err := repo.RevisionsOff(ctx, key.PlanID, key.UserID)
	if err != nil || !off[item] || off[untouched] || len(off) != 1 {
		t.Fatalf("switched off = %v, %v; want only the item", off, err)
	}
	if st, err := repo.Ladder(ctx, day, item); err != nil || !st.Off || len(st.Rungs) != 0 {
		t.Fatalf("state after off = %+v, %v; want off with no rungs", st, err)
	}

	if err := repo.SetRevisions(ctx, key.PlanID, key.UserID, item, true); err != nil {
		t.Fatal(err)
	}
	st, err := repo.Ladder(ctx, day, item)
	if err != nil || st.Off || !st.Resume {
		t.Fatalf("state after on = %+v, %v; want on and resuming", st, err)
	}
	if off, err := repo.RevisionsOff(ctx, key.PlanID, key.UserID); err != nil || len(off) != 0 {
		t.Fatalf("switched off after on = %v, %v; want none", off, err)
	}
}
