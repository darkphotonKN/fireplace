package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeLedger is an in-memory usage ledger: attempts with their record time.
type fakeLedger struct {
	mu       sync.Mutex
	attempts []Attempt
	at       []time.Time
	clock    func() time.Time
	sumErr   error
}

func newFakeLedger(clock func() time.Time) *fakeLedger { return &fakeLedger{clock: clock} }

// RecordAttempt fails on a done context, writing nothing, as the database
// round trip would.
func (l *fakeLedger) RecordAttempt(ctx context.Context, a Attempt) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.attempts = append(l.attempts, a)
	l.at = append(l.at, l.clock())
	return nil
}

func (l *fakeLedger) TokensSince(ctx context.Context, since time.Time) (int64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.sumErr != nil {
		return 0, l.sumErr
	}
	var sum int64
	for i, a := range l.attempts {
		if !l.at[i].Before(since) {
			sum += int64(a.Usage.InputTokens + a.Usage.OutputTokens)
		}
	}
	return sum, nil
}

// spend records tokens as already used at the ledger's current clock.
func (l *fakeLedger) spend(tokens int) {
	l.RecordAttempt(context.Background(), Attempt{Call: Call{Kind: KindDayPlan}, Usage: Usage{InputTokens: tokens}})
}

func (l *fakeLedger) recorded() []Attempt {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]Attempt(nil), l.attempts...)
}

var noon = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

func testCall() Call {
	return Call{Kind: KindChecklistSuggestion, PlanID: uuid.New(), UserID: uuid.New()}
}

func TestStructuredGenerator_AtBudgetMakesNoCallAndFailsExhausted(t *testing.T) {
	var calls atomic.Int32
	client := fakeOpenAI(t, func(w http.ResponseWriter, _ map[string]any) {
		calls.Add(1)
		io.WriteString(w, okResponse)
	})
	ledger := newFakeLedger(fixedClock(noon))
	ledger.spend(1000)
	meter := newMeter(ledger, 1000, fixedClock(noon))
	gen := newStructuredGenerator(GeneratorConfig{Model: "m"}, client, meter, 0)

	_, err := gen.Generate(context.Background(), StructuredRequest{Call: testCall(), Schema: json.RawMessage(`{}`)})

	if !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("err = %v, want ErrBudgetExhausted", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("provider calls = %d, want 0", calls.Load())
	}
	if n := len(ledger.recorded()); n != 1 {
		t.Fatalf("ledger rows = %d, want only the prior spend (a refused call records nothing)", n)
	}
}

func TestGenerator_AtBudgetMakesNoCallAndFailsExhausted(t *testing.T) {
	var calls atomic.Int32
	client := fakeOpenAI(t, func(w http.ResponseWriter, _ map[string]any) {
		calls.Add(1)
		io.WriteString(w, textResponse)
	})
	ledger := newFakeLedger(fixedClock(noon))
	ledger.spend(1500) // over, not just at
	gen := newGenerator("sys", client, newMeter(ledger, 1000, fixedClock(noon)), 0)

	_, err := gen.Generate(context.Background(), testCall(), "prompt")

	if !errors.Is(err, ErrBudgetExhausted) || calls.Load() != 0 {
		t.Fatalf("err = %v, calls = %d; want ErrBudgetExhausted and no call", err, calls.Load())
	}
}

const textResponse = `{
  "id": "x", "object": "chat.completion", "model": "gpt-5-mini-2025-08-07",
  "choices": [{"index": 0, "finish_reason": "stop", "message": {"role": "assistant", "content": "Write the outline"}}],
  "usage": {"prompt_tokens": 300, "completion_tokens": 20, "total_tokens": 320, "prompt_tokens_details": {"cached_tokens": 128}}
}`

func TestGenerator_UnderBudgetCallsAndRecordsItsKind(t *testing.T) {
	var calls atomic.Int32
	client := fakeOpenAI(t, func(w http.ResponseWriter, _ map[string]any) {
		calls.Add(1)
		io.WriteString(w, textResponse)
	})
	ledger := newFakeLedger(fixedClock(noon))
	ledger.spend(999)
	gen := newGenerator("sys", client, newMeter(ledger, 1000, fixedClock(noon)), 0)
	call := testCall()

	out, err := gen.Generate(context.Background(), call, "prompt")

	if err != nil || out != "Write the outline" || calls.Load() != 1 {
		t.Fatalf("out = %q, err = %v, calls = %d", out, err, calls.Load())
	}
	rows := ledger.recorded()
	want := Attempt{
		Call: call, Model: "gpt-5-mini-2025-08-07",
		Usage:     Usage{InputTokens: 300, CachedInputTokens: 128, OutputTokens: 20},
		Succeeded: true,
	}
	if len(rows) != 2 || rows[1] != want {
		t.Fatalf("rows = %+v, want the prior spend then %+v", rows, want)
	}
}

func TestGenerator_EveryTransportAttemptIsRecorded(t *testing.T) {
	var calls atomic.Int32
	client := fakeOpenAI(t, func(w http.ResponseWriter, _ map[string]any) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			io.WriteString(w, `{"error":{"message":"down"}}`)
			return
		}
		io.WriteString(w, textResponse)
	})
	ledger := newFakeLedger(fixedClock(noon))
	gen := newGenerator("sys", client, newMeter(ledger, 1000, fixedClock(noon)), 0)
	call := Call{Kind: KindVideoTerms, PlanID: uuid.New(), UserID: uuid.New()}

	if _, err := gen.Generate(context.Background(), call, "prompt"); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	rows := ledger.recorded()
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want one per attempt (3)", len(rows))
	}
	for i, r := range rows[:2] {
		if r.Succeeded || r.Usage != (Usage{}) || r.Call != call || r.Model != defaultModel {
			t.Fatalf("retry %d = %+v, want a failed row with what the provider reported (nothing)", i, r)
		}
	}
	if !rows[2].Succeeded || rows[2].Usage.InputTokens != 300 {
		t.Fatalf("final attempt = %+v", rows[2])
	}
}

func TestStructuredGenerator_RetryIsRefusedOnceTheBudgetIsSpent(t *testing.T) {
	var calls atomic.Int32
	client := fakeOpenAI(t, func(w http.ResponseWriter, _ map[string]any) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, `{"error":{"message":"down"}}`)
	})
	ledger := newFakeLedger(fixedClock(noon))
	meter := newMeter(ledger, 1000, fixedClock(noon))
	gen := newStructuredGenerator(GeneratorConfig{Model: "m"}, client, meter, 0)
	// Another caller spends the budget while the first attempt is in flight.
	gen.caller.afterAttempt = func() { ledger.spend(1000) }

	_, err := gen.Generate(context.Background(), StructuredRequest{Call: testCall(), Schema: json.RawMessage(`{}`)})

	if !errors.Is(err, ErrBudgetExhausted) || calls.Load() != 1 {
		t.Fatalf("err = %v, calls = %d; want the retry refused before the second call", err, calls.Load())
	}
}

func TestStructuredGenerator_RejectedContentIsRecordedAsFailed(t *testing.T) {
	client := fakeOpenAI(t, func(w http.ResponseWriter, _ map[string]any) { io.WriteString(w, okResponse) })
	ledger := newFakeLedger(fixedClock(noon))
	gen := newStructuredGenerator(GeneratorConfig{Model: "m"}, client, newMeter(ledger, 1_000_000, fixedClock(noon)), 0)
	rejected := errors.New("not a brief")

	_, err := gen.Generate(context.Background(), StructuredRequest{
		Call: testCall(), Schema: json.RawMessage(`{}`),
		Accept: func(content string) error { return rejected },
	})

	if !errors.Is(err, rejected) {
		t.Fatalf("err = %v, want the Accept error", err)
	}
	rows := ledger.recorded()
	if len(rows) != 1 || rows[0].Succeeded || rows[0].Usage.OutputTokens != 90 {
		t.Fatalf("rows = %+v, want one failed row with its usage", rows)
	}
}

func TestMeter_YesterdaysUTCUsageDoesNotCount(t *testing.T) {
	now := noon
	clock := func() time.Time { return now }
	ledger := newFakeLedger(clock)
	meter := newMeter(ledger, 1000, clock)

	now = time.Date(2026, 10, 24, 23, 59, 59, 0, time.UTC)
	ledger.spend(5000)
	if err := meter.Allow(context.Background()); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("same UTC day: err = %v, want exhausted", err)
	}

	now = time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	if err := meter.Allow(context.Background()); err != nil {
		t.Fatalf("next UTC day: err = %v, want allowed", err)
	}
}

func TestMeter_TheBudgetDayIsUTCWhateverTheServerZone(t *testing.T) {
	tokyo := time.FixedZone("JST", 9*3600)
	// 08:00 in Tokyo on the 25th is 23:00 UTC on the 24th.
	local := time.Date(2026, 10, 25, 8, 0, 0, 0, tokyo)
	var since time.Time
	meter := newMeter(sinceSpy{&since}, 1000, fixedClock(local))

	meter.Allow(context.Background())

	if want := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC); !since.Equal(want) {
		t.Fatalf("summed since %v, want UTC midnight %v", since, want)
	}
}

type sinceSpy struct{ got *time.Time }

func (s sinceSpy) TokensSince(ctx context.Context, since time.Time) (int64, error) {
	*s.got = since
	return 0, nil
}
func (sinceSpy) RecordAttempt(context.Context, Attempt) error { return nil }

func TestMeter_LedgerReadFailureRefusesTheCall(t *testing.T) {
	ledger := newFakeLedger(fixedClock(noon))
	ledger.sumErr = errors.New("db down")
	meter := newMeter(ledger, 1000, fixedClock(noon))

	err := meter.Allow(context.Background())

	if err == nil || errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("err = %v, want a non-budget failure (the breaker cannot see spend)", err)
	}
}

func TestDailyTokenBudget_DefaultAndOverride(t *testing.T) {
	for _, tc := range []struct {
		env  string
		want int64
	}{
		{"", 200000},
		{"2000000", 2000000},
		{"not-a-number", 200000},
		{"0", 200000},
		{"-5", 200000},
	} {
		t.Setenv("LLM_DAILY_TOKEN_BUDGET", tc.env)
		if got := DailyTokenBudget(); got != tc.want {
			t.Errorf("LLM_DAILY_TOKEN_BUDGET=%q: budget = %d, want %d", tc.env, got, tc.want)
		}
	}
}

// openMeter is a meter with room to spare, for tests about something else.
func openMeter() *Meter {
	return newMeter(newFakeLedger(fixedClock(noon)), 1_000_000_000, fixedClock(noon))
}

// A request whose caller hung up mid-call has still been paid for: the failed
// attempt lands in the ledger even though the context it ran under is done.
func TestGenerator_ACancelledCallIsStillRecorded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := fakeOpenAI(t, func(w http.ResponseWriter, _ map[string]any) {
		cancel()
		time.Sleep(50 * time.Millisecond) // the client gives up first
		io.WriteString(w, textResponse)
	})
	ledger := newFakeLedger(fixedClock(noon))
	gen := newGenerator("sys", client, newMeter(ledger, 1000, fixedClock(noon)), 0)

	if _, err := gen.Generate(ctx, testCall(), "prompt"); err == nil {
		t.Fatal("want the cancelled call's error")
	}

	if rows := ledger.recorded(); len(rows) != 1 || rows[0].Succeeded {
		t.Fatalf("rows = %+v, want one failed attempt recorded", rows)
	}
}

// A provider answer that no retry can change (a bad request, a bad key, an
// unknown model override) fails after one attempt, recorded as failed. A
// throttle or a server error is retried.
func TestGenerator_OnlyRetryableStatusesAreRetried(t *testing.T) {
	tests := []struct {
		status int
		calls  int32
	}{
		{http.StatusBadRequest, 1},
		{http.StatusUnauthorized, 1},
		{http.StatusNotFound, 1},
		{http.StatusTooManyRequests, 3},
		{http.StatusInternalServerError, 3},
		{http.StatusServiceUnavailable, 3},
	}
	for _, tc := range tests {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			var calls atomic.Int32
			client := fakeOpenAI(t, func(w http.ResponseWriter, _ map[string]any) {
				calls.Add(1)
				w.WriteHeader(tc.status)
				io.WriteString(w, `{"error":{"message":"no"}}`)
			})
			ledger := newFakeLedger(fixedClock(noon))
			gen := newStructuredGenerator(GeneratorConfig{Model: "m"}, client, newMeter(ledger, 1000, fixedClock(noon)), 0)

			_, err := gen.Generate(context.Background(), StructuredRequest{Call: testCall(), Schema: json.RawMessage(`{}`)})

			if err == nil {
				t.Fatal("want an error")
			}
			if calls.Load() != tc.calls {
				t.Fatalf("calls = %d, want %d", calls.Load(), tc.calls)
			}
			rows := ledger.recorded()
			if int32(len(rows)) != tc.calls {
				t.Fatalf("rows = %d, want one per attempt (%d)", len(rows), tc.calls)
			}
			for _, r := range rows {
				if r.Succeeded {
					t.Fatalf("row %+v, want failed", r)
				}
			}
		})
	}
}
