package notesummary

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/ai"
	"github.com/google/uuid"
)

// fakePlans plays plan-service: the caller's access and one item.
type fakePlans struct {
	allowed   bool
	accessErr error
	item      Item
	itemErr   error

	itemReads int
}

func (f *fakePlans) IsOwnerOrMember(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return f.allowed, f.accessErr
}

func (f *fakePlans) GetItem(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (Item, error) {
	f.itemReads++
	return f.item, f.itemErr
}

// fakeModel plays the metered generator: it records what it was sent.
type fakeModel struct {
	out string
	err error
	// wait, when set, blocks until ctx is done and answers its error.
	wait bool

	calls  int
	prompt string
	call   ai.Call
}

func (f *fakeModel) Generate(ctx context.Context, call ai.Call, prompt string) (string, error) {
	f.calls++
	f.prompt, f.call = prompt, call
	if f.wait {
		<-ctx.Done()
		return "", ctx.Err()
	}
	return f.out, f.err
}

func note(body string) Item { return Item{Type: TypeNote, Description: body} }

func summarize(t *testing.T, plans *fakePlans, model *fakeModel) (string, error) {
	t.Helper()
	return NewService(plans, model).Summarize(context.Background(), uuid.New(), uuid.New(), uuid.New())
}

// R39: a caller with no relation to the plan is NotFound, and neither the
// item nor the model is touched.
func TestSummarize_NoRelation_NotFoundBeforeAnyRead(t *testing.T) {
	plans := &fakePlans{allowed: false, item: note("body")}
	model := &fakeModel{out: "summary"}

	_, err := summarize(t, plans, model)

	if !errors.Is(err, commonconstants.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if plans.itemReads != 0 || model.calls != 0 {
		t.Fatalf("item reads = %d, model calls = %d, want 0 and 0", plans.itemReads, model.calls)
	}
}

// R39: missing (plan-service NotFound) and archived items are NotFound; a
// task is ErrNotANote. None reaches the model.
func TestSummarize_ItemRefusals(t *testing.T) {
	cases := []struct {
		name    string
		item    Item
		itemErr error
		want    error
	}{
		{"missing or other plan", Item{}, commonconstants.ErrNotFound, commonconstants.ErrNotFound},
		{"archived", Item{Type: TypeNote, Description: "body", Archived: true}, nil, commonconstants.ErrNotFound},
		{"task", Item{Type: "task", Description: "do it"}, nil, ErrNotANote},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := &fakeModel{out: "summary"}
			_, err := summarize(t, &fakePlans{allowed: true, item: tc.item, itemErr: tc.itemErr}, model)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if model.calls != 0 {
				t.Fatalf("model calls = %d, want 0", model.calls)
			}
		})
	}
}

// R40: the model sees only the body, capped at 8000 runes. R41: the call is
// recorded under kind note_summary.
func TestSummarize_ModelSeesOnlyTheCappedBody(t *testing.T) {
	body := strings.Repeat("é", 9000)
	model := &fakeModel{out: "A summary."}

	if _, err := summarize(t, &fakePlans{allowed: true, item: note(body)}, model); err != nil {
		t.Fatalf("Summarize: %v", err)
	}

	if want := strings.Repeat("é", InputCap); model.prompt != want {
		t.Fatalf("prompt is %d runes, want exactly the first %d of the body", utf8.RuneCountInString(model.prompt), InputCap)
	}
	if model.call.Kind != ai.KindNoteSummary {
		t.Fatalf("kind = %q, want %q", model.call.Kind, ai.KindNoteSummary)
	}
}

// A short body goes to the model whole and unchanged.
func TestSummarize_ShortBodySentWhole(t *testing.T) {
	model := &fakeModel{out: "ok"}
	body := "line one\nline two"
	if _, err := summarize(t, &fakePlans{allowed: true, item: note(body)}, model); err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if model.prompt != body {
		t.Fatalf("prompt = %q, want %q", model.prompt, body)
	}
}

// R42: the output is trimmed and its whitespace collapsed; over 280 it is cut
// at the last word boundary within 279 and given "…".
func TestSummarize_Output(t *testing.T) {
	long := strings.TrimSpace(strings.Repeat("word ", 80))     // 399 chars
	ragged := strings.TrimSpace(strings.Repeat("abcde ", 100)) // a word straddles 279
	cases := []struct {
		name string
		out  string
		want string
	}{
		{"trimmed and collapsed", "  One\n\n paragraph\tof   text.  ", "One paragraph of text."},
		{"exactly 280 kept", strings.Repeat("a", 280), strings.Repeat("a", 280)},
		{"over 280, word ends at 279", long, strings.TrimSpace(strings.Repeat("word ", 56)) + "…"},
		{"over 280, word straddles 279", ragged, strings.TrimSpace(strings.Repeat("abcde ", 46)) + "…"},
		{"one unbroken word cut hard", strings.Repeat("x", 400), strings.Repeat("x", 279) + "…"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := summarize(t, &fakePlans{allowed: true, item: note("body")}, &fakeModel{out: tc.out})
			if err != nil {
				t.Fatalf("Summarize: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q (%d), want %q", got, utf8.RuneCountInString(got), tc.want)
			}
			if n := utf8.RuneCountInString(got); n < 1 || n > MaxLen {
				t.Fatalf("length %d outside 1-%d", n, MaxLen)
			}
		})
	}
}

// R42: empty output and a provider error are ErrFailed.
func TestSummarize_ModelFailures(t *testing.T) {
	cases := []struct {
		name  string
		model *fakeModel
	}{
		{"empty output", &fakeModel{out: ""}},
		{"whitespace output", &fakeModel{out: " \n\t "}},
		{"provider error", &fakeModel{err: errors.New("ai: chat completion failed: 500")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := summarize(t, &fakePlans{allowed: true, item: note("body")}, tc.model)
			if !errors.Is(err, ErrFailed) {
				t.Fatalf("err = %v, want ErrFailed", err)
			}
		})
	}
}

// R42: a call that outlives its timeout is ErrFailed.
func TestSummarize_Timeout_Failed(t *testing.T) {
	s := NewService(&fakePlans{allowed: true, item: note("body")}, &fakeModel{wait: true})
	s.timeout = 10 * time.Millisecond

	_, err := s.Summarize(context.Background(), uuid.New(), uuid.New(), uuid.New())

	if !errors.Is(err, ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
}

// R41: the budget refusal keeps its own identity, so it is not mistaken for a
// model failure.
func TestSummarize_BudgetExhausted_PassesThrough(t *testing.T) {
	_, err := summarize(t, &fakePlans{allowed: true, item: note("body")},
		&fakeModel{err: ai.ErrBudgetExhausted})

	if !errors.Is(err, ai.ErrBudgetExhausted) || errors.Is(err, ErrFailed) {
		t.Fatalf("err = %v, want ErrBudgetExhausted and not ErrFailed", err)
	}
}

// An access check that cannot be answered is a server fault, not a refusal.
func TestSummarize_AccessError_Propagates(t *testing.T) {
	boom := errors.New("plan-service down")
	_, err := summarize(t, &fakePlans{accessErr: boom}, &fakeModel{out: "x"})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the access error", err)
	}
}
