package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// Kind is what a generation produces: the usage ledger's kind column.
type Kind string

const (
	KindStepSuggestions     Kind = "step_suggestions"
	KindItemTree            Kind = "item_tree"
	KindChecklistSuggestion Kind = "checklist_suggestion"
	KindVideoTerms          Kind = "video_terms"
	KindDayPlan             Kind = "day_plan"
	// "daily_suggestions" was the legacy daily set, removed with its endpoint
	// (FS-DF4ZB R55), and "daily_brief" the retired daily brief (FS-02JBZ
	// R92). The generations CHECK still admits both because ledger rows
	// written before the removals keep them; nothing writes them any more.
)

// Call says what one model call is for and on whose behalf, so the ledger can
// attribute its usage.
type Call struct {
	Kind   Kind
	PlanID uuid.UUID
	UserID uuid.UUID
}

// Attempt is the usage of one provider request: one ledger row. Usage is what
// the provider reported, which is nothing for a request that failed in
// transport. Succeeded is false for a transport failure and for a response
// the caller could not use.
type Attempt struct {
	Call
	Model     string
	Usage     Usage
	Succeeded bool
}

// Ledger is the usage ledger (FS-DF4ZB R9, FS-02JBZ R31): the meter writes
// every attempt to it and reads today's spend from it.
type Ledger interface {
	RecordAttempt(ctx context.Context, a Attempt) error
	// TokensSince sums input plus output tokens (cached input is part of
	// input) over the attempts recorded at or after since.
	TokensSince(ctx context.Context, since time.Time) (int64, error)
}

// ErrBudgetExhausted: today's ledger sum is at or over the global daily token
// budget, so no model call was made (FS-02JBZ R33). Retrying after the UTC day
// rolls over is correct; retrying now is not.
var ErrBudgetExhausted = errors.New("ai: daily token budget exhausted")

// DefaultDailyTokenBudget is the budget when LLM_DAILY_TOKEN_BUDGET is unset
// (FS-02JBZ R34, D8).
const DefaultDailyTokenBudget int64 = 200000

// DailyTokenBudget reads LLM_DAILY_TOKEN_BUDGET. Unset, unparsable or not
// positive, it is DefaultDailyTokenBudget; a bad value is logged so a typo in
// production config is visible rather than silently lowering the cap.
func DailyTokenBudget() int64 {
	v := os.Getenv("LLM_DAILY_TOKEN_BUDGET")
	if v == "" {
		return DefaultDailyTokenBudget
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		slog.Warn("ai: LLM_DAILY_TOKEN_BUDGET is not a positive integer, using the default",
			"value", v, "default", DefaultDailyTokenBudget)
		return DefaultDailyTokenBudget
	}
	return n
}

// Meter is the global daily token budget of ADR-0019 clauses 3–4: a circuit
// breaker across every generator, and the recorder that makes the ledger it
// reads complete. Every model call this service makes goes through one Meter,
// via the attempt loop both generators share (chatCaller), so any generator
// built on Generator or StructuredGenerator, including ones added later (day
// plan, step and item-tree suggestions), is guarded and recorded with no
// wiring of its own.
//
// The budget day is the server's UTC calendar day, and spend is input plus
// output tokens (FS-02JBZ R32, S8). Embedding calls do not go through the
// Meter: they are not generation and are not counted (S8).
//
// The check is soft at the edge (R35, accepted by ADR-0019): spend is read
// before a call and its cost is known only after, so calls that pass the
// check concurrently near the limit can overshoot it by up to the sum of their
// costs, each bounded by its input plus its output ceiling. There is
// deliberately no lock or reservation: it is a circuit breaker, not an
// invoice.
type Meter struct {
	ledger Ledger
	budget int64
	now    func() time.Time
}

// NewMeter builds the meter for budget tokens per UTC day over ledger.
func NewMeter(ledger Ledger, budget int64) *Meter {
	return newMeter(ledger, budget, time.Now)
}

func newMeter(ledger Ledger, budget int64, now func() time.Time) *Meter {
	return &Meter{ledger: ledger, budget: budget, now: now}
}

// Allow reports whether a model call may be made now: nil under budget,
// ErrBudgetExhausted at or over it. A ledger that cannot be read refuses the
// call with that error: a breaker that cannot see spend does not let it
// through.
func (m *Meter) Allow(ctx context.Context) error {
	spent, err := m.ledger.TokensSince(ctx, budgetDayStart(m.now()))
	if err != nil {
		return fmt.Errorf("ai: read today's token spend: %w", err)
	}
	if spent >= m.budget {
		return fmt.Errorf("%w: %d of %d tokens spent today (UTC)", ErrBudgetExhausted, spent, m.budget)
	}
	return nil
}

// recordTimeout bounds a ledger write that outlives its request.
const recordTimeout = 3 * time.Second

// Record writes one attempt to the ledger. A failure to record is logged, not
// returned: the call has already been paid for, and failing it would only make
// the caller pay for another.
//
// The write is detached from ctx's cancellation, with its own short timeout:
// an attempt whose caller hung up was still paid for, and a ledger missing it
// would let the budget overshoot (R31).
func (m *Meter) Record(ctx context.Context, a Attempt) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), recordTimeout)
	defer cancel()
	if err := m.ledger.RecordAttempt(ctx, a); err != nil {
		slog.ErrorContext(ctx, "ai: record generation usage failed",
			"err", err, "kind", a.Kind, "plan_id", a.PlanID)
	}
}

// budgetDayStart is the UTC midnight that opens t's budget day.
func budgetDayStart(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}
