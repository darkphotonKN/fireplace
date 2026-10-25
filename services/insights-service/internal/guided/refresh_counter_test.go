package guided

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func openCounterDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db := openTestDB(t)
	var exists bool
	if err := db.Get(&exists, `SELECT to_regclass('public.refresh_counters') IS NOT NULL`); err != nil || !exists {
		t.Skip("refresh_counters not migrated in the test database")
	}
	return db
}

func testDayKey(t *testing.T, db *sqlx.DB) DayKey {
	t.Helper()
	key := DayKey{PlanID: uuid.New(), UserID: uuid.New(), Date: time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)}
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM refresh_counters WHERE plan_id = $1`, key.PlanID); err != nil {
			t.Errorf("cleanup refresh_counters: %v", err)
		}
	})
	return key
}

func TestRefreshCounters_ConcurrentSpendsAtTwoSpent_ExactlyOneSucceeds(t *testing.T) {
	db := openCounterDB(t)
	counters := NewRefreshCounters(db)
	ctx := context.Background()
	key := testDayKey(t, db)

	for i := 0; i < 2; i++ {
		if _, err := counters.Spend(ctx, key); err != nil {
			t.Fatalf("spend %d: %v", i+1, err)
		}
	}

	const racers = 8
	var wg sync.WaitGroup
	errs := make([]error, racers)
	start := make(chan struct{})
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = counters.Spend(ctx, key)
		}(i)
	}
	close(start)
	wg.Wait()

	won, refused := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			won++
		case errors.Is(err, ErrRefreshLimitReached):
			refused++
		default:
			t.Fatalf("spend: %v", err)
		}
	}
	if won != 1 || refused != racers-1 {
		t.Fatalf("won = %d, refused = %d; want exactly one of %d to win", won, refused, racers)
	}
	if left, err := counters.Remaining(ctx, key); err != nil || left != 0 {
		t.Fatalf("remaining = %d, %v; want 0", left, err)
	}
}

func TestRefreshCounters_SpendReturnRemaining_TrackTheCap(t *testing.T) {
	db := openCounterDB(t)
	counters := NewRefreshCounters(db)
	ctx := context.Background()
	key := testDayKey(t, db)

	if left, err := counters.Remaining(ctx, key); err != nil || left != RefreshCap {
		t.Fatalf("remaining with nothing spent = %d, %v; want %d", left, err, RefreshCap)
	}
	for want := RefreshCap - 1; want >= 0; want-- {
		left, err := counters.Spend(ctx, key)
		if err != nil || left != want {
			t.Fatalf("spend = %d, %v; want %d remaining", left, err, want)
		}
	}
	if _, err := counters.Spend(ctx, key); !errors.Is(err, ErrRefreshLimitReached) {
		t.Fatalf("spend past the cap = %v, want ErrRefreshLimitReached", err)
	}

	if err := counters.Return(ctx, key); err != nil {
		t.Fatalf("return: %v", err)
	}
	if left, err := counters.Remaining(ctx, key); err != nil || left != 1 {
		t.Fatalf("remaining after a return = %d, %v; want 1", left, err)
	}

	// Another date, and another user on the same plan, are separate keys.
	other := key
	other.Date = key.Date.AddDate(0, 0, 1)
	if left, err := counters.Remaining(ctx, other); err != nil || left != RefreshCap {
		t.Fatalf("remaining on another date = %d, %v; want %d", left, err, RefreshCap)
	}
	other = key
	other.UserID = uuid.New()
	if left, err := counters.Remaining(ctx, other); err != nil || left != RefreshCap {
		t.Fatalf("remaining for another user = %d, %v; want %d", left, err, RefreshCap)
	}
}

func TestRefreshCounters_ReturnWithNothingSpent_IsANoOp(t *testing.T) {
	db := openCounterDB(t)
	counters := NewRefreshCounters(db)
	ctx := context.Background()
	key := testDayKey(t, db)

	if err := counters.Return(ctx, key); err != nil {
		t.Fatalf("return: %v", err)
	}
	if left, err := counters.Remaining(ctx, key); err != nil || left != RefreshCap {
		t.Fatalf("remaining = %d, %v; want %d", left, err, RefreshCap)
	}
}
