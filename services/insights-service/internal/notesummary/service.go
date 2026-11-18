// Package notesummary writes a short summary of a note's body on request
// (FS-PGG2N R38-R44). It returns the text and stores nothing: saving it is the
// client's item update, so this package writes nothing to plan-service and
// publishes no event (R43). It is reached only from the SummarizeNote RPC,
// which the gateway calls only on a user's press (R44, ADR-0019).
package notesummary

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/ai"
	"github.com/google/uuid"
)

const (
	// InputCap is NOTE_SUMMARY_INPUT_CAP: how many characters (runes) of the
	// body the model sees (R40). The rest of a longer body is not sent.
	InputCap = 8000
	// MaxLen is the longest summary returned, in characters (R42).
	MaxLen = 280

	// TypeNote is plan-service's item type for a note.
	TypeNote = "note"

	// callTimeout bounds the model call, retries included (R42).
	callTimeout = 30 * time.Second
)

var (
	// ErrNotANote: the item is a task, which has no body to summarize (R39).
	ErrNotANote = errors.New("notesummary: item is not a note")
	// ErrFailed: the model failed, timed out or answered nothing usable (R42).
	// Retrying is correct.
	ErrFailed = errors.New("notesummary: summary failed")
)

// Item is the part of a plan-service item a summary needs.
type Item struct {
	Type        string
	Description string
	Archived    bool
}

// Plans is the slice of plan-service this package reads. plan-service answers
// the caller's relation; enforcing it is this package's job (ADR-0015).
type Plans interface {
	// IsOwnerOrMember reports whether userID owns or is a member of planID.
	IsOwnerOrMember(ctx context.Context, planID, userID uuid.UUID) (bool, error)
	// GetItem reads itemID in planID. A missing item, or one in another plan,
	// is commonconstants.ErrNotFound.
	GetItem(ctx context.Context, planID, userID, itemID uuid.UUID) (Item, error)
}

// Generator is the metered model seam (ai.Generator): it refuses with
// ai.ErrBudgetExhausted over the global token budget and records every
// attempt in the generations ledger under call (R41).
type Generator interface {
	Generate(ctx context.Context, call ai.Call, prompt string) (string, error)
}

type Service struct {
	plans   Plans
	gen     Generator
	timeout time.Duration
}

func NewService(plans Plans, gen Generator) *Service {
	return &Service{plans: plans, gen: gen, timeout: callTimeout}
}

// Summarize returns a summary of the note itemID in planID for userID, 1 to
// MaxLen characters. Refusals: no relation, or a missing, archived or other
// plan's item, is commonconstants.ErrNotFound; a task is ErrNotANote; the
// budget spent is ai.ErrBudgetExhausted; any model failure is ErrFailed.
func (s *Service) Summarize(ctx context.Context, planID, userID, itemID uuid.UUID) (string, error) {
	allowed, err := s.plans.IsOwnerOrMember(ctx, planID, userID)
	if err != nil {
		return "", fmt.Errorf("notesummary: plan access: %w", err)
	}
	if !allowed {
		return "", fmt.Errorf("notesummary: plan %s: %w", planID, commonconstants.ErrNotFound)
	}

	item, err := s.plans.GetItem(ctx, planID, userID, itemID)
	if err != nil {
		return "", fmt.Errorf("notesummary: get item: %w", err)
	}
	if item.Archived {
		return "", fmt.Errorf("notesummary: item %s is archived: %w", itemID, commonconstants.ErrNotFound)
	}
	if item.Type != TypeNote {
		return "", fmt.Errorf("notesummary: item %s: %w", itemID, ErrNotANote)
	}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	call := ai.Call{Kind: ai.KindNoteSummary, PlanID: planID, UserID: userID}
	raw, err := s.gen.Generate(ctx, call, capRunes(item.Description, InputCap))
	if errors.Is(err, ai.ErrBudgetExhausted) {
		return "", fmt.Errorf("notesummary: %w", err)
	}
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFailed, err)
	}

	summary := fit(strings.Join(strings.Fields(raw), " "))
	if summary == "" {
		return "", fmt.Errorf("%w: empty model output", ErrFailed)
	}
	return summary, nil
}

// capRunes is s's first n runes.
func capRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// fit shortens s to MaxLen runes: over it, s is cut at the last word boundary
// within MaxLen-1 runes and given "…" (R42, G13). A single word longer than
// that is cut where it stands.
func fit(s string) string {
	r := []rune(s)
	if len(r) <= MaxLen {
		return s
	}
	keep := MaxLen - 1
	if !unicode.IsSpace(r[keep]) {
		for i := keep - 1; i > 0; i-- {
			if unicode.IsSpace(r[i]) {
				keep = i
				break
			}
		}
	}
	return strings.TrimRightFunc(string(r[:keep]), unicode.IsSpace) + "…"
}
