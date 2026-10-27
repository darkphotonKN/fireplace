package dayplan

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/ai"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
)

// PlanProfile is what the day plan reads about the plan itself, beside its
// items: the plan's type and focus (R7).
type PlanProfile struct {
	Type  string
	Focus string
}

// PlanReader is the slice of plan-service the day plan reads. GetPlanProfile
// answers NotFound for a plan the caller cannot access (R23).
type PlanReader interface {
	GetPlanProfile(ctx context.Context, planID, userID uuid.UUID) (PlanProfile, error)
	ListPlanItems(ctx context.Context, planID, userID uuid.UUID) ([]guided.PlanItem, error)
}

// Generator makes one metered structured-output call (ai.StructuredGenerator):
// it checks the global token budget, refusing with ai.ErrBudgetExhausted, and
// records every attempt in the ledger under the request's Call.
type Generator interface {
	Generate(ctx context.Context, req ai.StructuredRequest) (ai.Completion, error)
}

// Store persists day plans, one per key.
type Store interface {
	// Find returns the stored plan for key; ok is false when none exists.
	Find(ctx context.Context, key guided.DayKey) (p DayPlan, ok bool, err error)
	// Save stores p as its key's plan, in one step, if the key is still at
	// prev: the same draw with no tick or untick since (the zero DrawRef:
	// nothing stored yet). It returns the plan the key serves afterwards and
	// whether that is p; when a concurrent draw, tick or untick got there
	// first it is the stored plan, and p is not stored (R11, R21).
	Save(ctx context.Context, p DayPlan, prev DrawRef) (stored DayPlan, won bool, err error)

	// FindByEntry returns the user's stored plan whose current draw holds
	// entry entryID; ok is false when there is none (unknown, another user's,
	// or only in a replaced draw).
	FindByEntry(ctx context.Context, entryID, userID uuid.UUID) (p DayPlan, ok bool, err error)
	// Tick records t in one step, binding the entry first when t.Bind is set,
	// and moves the key's tick version (DrawRef). created is false when the
	// item was already ticked for the date (R27). A bind that finds the entry
	// no longer an unbound new step of the key's draw, or the item already
	// bound to another entry of it, fails with ErrEntryNotTickable.
	Tick(ctx context.Context, t Tick) (created bool, err error)
	// Untick removes the day tick on item for key, in one step with moving
	// the key's tick version; removed is false when there was none (R29).
	Untick(ctx context.Context, key guided.DayKey, itemID uuid.UUID) (removed bool, err error)
}

// Service serves day plans: GetDayPlan only reads; PlanMyDay is the one
// operation that draws (D7).
type Service struct {
	plans     PlanReader
	gen       Generator
	snippets  guided.SnippetSource
	store     Store
	revisions RevisionStore
	refreshes guided.RefreshCounter
	now       func() time.Time
}

func NewService(plans PlanReader, gen Generator, snippets guided.SnippetSource, store Store, revisions RevisionStore, refreshes guided.RefreshCounter) *Service {
	return &Service{plans: plans, gen: gen, snippets: snippets, store: store, revisions: revisions, refreshes: refreshes, now: time.Now}
}

// GetDayPlan returns the stored day plan for (plan, user, date), or the
// not_planned state when there is none (R14). It never calls the model, never
// draws by rules and never copies another date's plan forward (ADR-0019).
// date is YYYY-MM-DD, within one day of the server's UTC date (R22).
//
// A stored plan with existing_item entries also reads the plan's type, since
// only a learning plan's entries carry the revisions switch (R49).
func (s *Service) GetDayPlan(ctx context.Context, planID, userID uuid.UUID, date string) (DayPlan, error) {
	key, err := guided.NewDayKey(planID, userID, date, s.now())
	if err != nil {
		return DayPlan{}, err
	}
	p, ok, err := s.store.Find(ctx, key)
	if err != nil {
		return DayPlan{}, fmt.Errorf("dayplan: load: %w", err)
	}
	if !ok {
		return s.present(ctx, DayPlan{Key: key}, false)
	}
	learning := false
	if hasExistingItems(p) {
		profile, err := s.plans.GetPlanProfile(ctx, planID, userID)
		if err != nil {
			return DayPlan{}, fmt.Errorf("dayplan: read plan: %w", err)
		}
		learning = profile.learning()
	}
	return s.present(ctx, p, learning)
}

// PlanMyDay draws or redraws the day plan for (plan, user, date) (R13) and
// returns what the key then serves. In order:
//
//   - The rule gate (R17): a stored ai plan drawn from an equal plan-state
//     hash is returned as is, with no generation and no count.
//   - No candidates (R18): a plan with no open long-term task, or with every
//     slot taken by an entry ticked today, is stored as a planned ai plan of
//     just its ticked entries, with no generation and no count, and the free
//     draw stays unused.
//   - Otherwise one structured generation (R6), validated (R9) and stored in
//     one step (R11). The key's first successful AI draw is free; every later
//     one first spends from the per (plan, user, date) refresh counter (R15,
//     A3), refused with guided.ErrRefreshLimitReached at the cap and the
//     stored plan untouched. A draw that fails, or loses a race to a
//     concurrent draw or to a tick that landed while it generated, gives
//     its spend back (R21).
//   - A failed generation (the global token budget, ai.ErrBudgetExhausted;
//     the provider down or timing out; schema-invalid output) falls back to a
//     rules plan, stored with source rules (R36): the caller gets a plan, not
//     an error. See drawByRules.
//
// Every draw, AI or rules, keeps the entries ticked today in place with their
// group, text, id and tick, and adds only what fits beside them (R12, R39).
func (s *Service) PlanMyDay(ctx context.Context, planID, userID uuid.UUID, date string) (DayPlan, error) {
	key, err := guided.NewDayKey(planID, userID, date, s.now())
	if err != nil {
		return DayPlan{}, err
	}
	d, err := s.startDraw(ctx, key)
	if err != nil {
		return DayPlan{}, err
	}
	if d.gateHolds() {
		return s.present(ctx, d.stored, d.profile.learning())
	}
	if !d.hasCandidates() {
		stored, _, err := s.save(ctx, d, d.plan(SourceAI, d.pinned, s.now(), d.stored.FreeDrawUsed))
		if err != nil {
			return DayPlan{}, err
		}
		return s.present(ctx, stored, d.profile.learning())
	}

	// Accepted behaviour (P4): the spend is taken before generating and given
	// back only once the draw is known to have lost. Two concurrent counted
	// redraws of one key with one refresh left therefore do not both reach
	// the model: the first spends the last refresh, and the second is
	// refused with guided.ErrRefreshLimitReached (429) at once, rather than
	// waiting to receive the first's plan. A retry after the first finishes
	// is served by the rule gate, so the cost is one refused click.
	counted := d.stored.FreeDrawUsed
	if counted {
		if _, err := s.refreshes.Spend(ctx, key); err != nil {
			return DayPlan{}, fmt.Errorf("dayplan: spend refresh: %w", err)
		}
	}
	groups, err := s.generate(ctx, d)
	if err != nil {
		if counted {
			s.returnRefresh(ctx, key)
		}
		return s.drawByRules(ctx, d, err)
	}
	stored, won, err := s.save(ctx, d, d.plan(SourceAI, groups, s.now(), true))
	if counted && (err != nil || !won) {
		s.returnRefresh(ctx, key)
	}
	if err != nil {
		return DayPlan{}, err
	}
	return s.present(ctx, stored, d.profile.learning())
}

// draft is a Plan my day past its reads: the key, what the plan-service said
// and the plan the key serves now (the zero DayPlan, with draw uuid.Nil, when
// none is stored). updatedAt is each item's last update, which only the rules
// draw reads. pinned are the stored plan's entries ticked today, in their
// groups and order (R12). due are the revisions due on the plan date of items
// still in the tree, oldest due first; none on a project plan (R41, R44).
type draft struct {
	key       guided.DayKey
	profile   PlanProfile
	tree      guided.PlanTree
	updatedAt map[uuid.UUID]time.Time
	hash      string
	stored    DayPlan
	pinned    []Group
	due       []dueRevision
}

// startDraw reads the plan's profile and items from plan-service (which
// answers NotFound for a plan the caller cannot access, R23), the key's stored
// plan and, on a learning plan, the revisions due on the plan date.
func (s *Service) startDraw(ctx context.Context, key guided.DayKey) (draft, error) {
	profile, err := s.plans.GetPlanProfile(ctx, key.PlanID, key.UserID)
	if err != nil {
		return draft{}, fmt.Errorf("dayplan: read plan: %w", err)
	}
	items, err := s.plans.ListPlanItems(ctx, key.PlanID, key.UserID)
	if err != nil {
		return draft{}, fmt.Errorf("dayplan: read plan items: %w", err)
	}
	stored, _, err := s.store.Find(ctx, key)
	if err != nil {
		return draft{}, fmt.Errorf("dayplan: load: %w", err)
	}
	tree := guided.NewPlanTree(items)
	updatedAt := make(map[uuid.UUID]time.Time, len(items))
	for _, it := range items {
		updatedAt[it.ID] = it.UpdatedAt
	}
	due, err := s.dueRevisions(ctx, key, profile, tree)
	if err != nil {
		return draft{}, err
	}
	pinned, ticked := tickedEntries(stored)
	return draft{
		key: key, profile: profile, tree: tree, updatedAt: updatedAt,
		hash: stateHash(tree, profile, key.Date, ticked, due), stored: stored, pinned: pinned, due: due,
	}, nil
}

// tickedEntries is p's entries ticked today, in their groups and order, and
// their items.
func tickedEntries(p DayPlan) ([]Group, []uuid.UUID) {
	var groups []Group
	var items []uuid.UUID
	for _, g := range p.Groups {
		var entries []Entry
		for _, e := range g.Entries {
			if e.Ticked && e.ItemID != nil {
				entries = append(entries, e)
				items = append(items, *e.ItemID)
			}
		}
		if len(entries) > 0 {
			groups = append(groups, Group{Label: g.Label, Entries: entries})
		}
	}
	return groups, items
}

// gateHolds reports whether the stored plan may be served instead of drawing:
// it is an ai plan drawn from an equal hash. A rules plan never holds (R17).
func (d draft) gateHolds() bool {
	return d.stored.Source == SourceAI && d.stored.PlanStateHash == d.hash
}

// hasCandidates reports whether anything can be planned: an open long-term
// task, top-level or child, or a due revision (R18), and a slot not taken by
// an entry ticked today (R12).
func (d draft) hasCandidates() bool {
	pinned := 0
	for _, g := range d.pinned {
		pinned += len(g.Entries)
	}
	if pinned >= MaxEntries {
		return false
	}
	if len(d.due) > 0 {
		return true
	}
	for _, it := range d.tree.Items {
		if !it.Done {
			return true
		}
	}
	return false
}

// plan is the plan d draws from source with groups at now, each entry without
// an id given one (an entry carried over keeps its own).
func (d draft) plan(source Source, groups []Group, now time.Time, freeDrawUsed bool) DayPlan {
	for gi := range groups {
		for ei := range groups[gi].Entries {
			if groups[gi].Entries[ei].ID == uuid.Nil {
				groups[gi].Entries[ei].ID = uuid.New()
			}
		}
	}
	return DayPlan{
		Key: d.key, Planned: true, Source: source, GeneratedAt: now, PlanStateHash: d.hash,
		DrawID: uuid.New(), FreeDrawUsed: freeDrawUsed, Groups: groups,
	}
}

// save stores p over the draw d started from, returning what the key serves.
func (s *Service) save(ctx context.Context, d draft, p DayPlan) (DayPlan, bool, error) {
	stored, won, err := s.store.Save(ctx, p, d.stored.Ref())
	if err != nil {
		return DayPlan{}, false, fmt.Errorf("dayplan: store: %w", err)
	}
	return stored, won, nil
}

// generate makes the one generation of a draw, validates its output and
// stamps the due revisions on it (R10). The generator records its usage,
// failed or not, as kind day_plan; the parse is its Accept, so output outside
// the schema is a failed generation.
func (s *Service) generate(ctx context.Context, d draft) ([]Group, error) {
	prompt := buildPrompt(promptInput{
		Tree: d.tree, Profile: d.profile, Date: d.key.Date, Snippets: s.snippets.Find(ctx, d.key, d.topLevelTexts()),
		Ticked: d.pinned, Due: d.due,
	})
	var generated []generatedGroup
	_, err := s.gen.Generate(ctx, ai.StructuredRequest{
		Call:       ai.Call{Kind: ai.KindDayPlan, PlanID: d.key.PlanID, UserID: d.key.UserID},
		System:     prompt.System,
		User:       prompt.User,
		SchemaName: schemaName,
		Schema:     schema,
		Accept: func(content string) (err error) {
			generated, err = parseGenerated(content)
			return err
		},
	})
	if err != nil {
		return nil, err
	}
	groups := validate(d.tree, d.pinned, d.due, generated)
	stampRevisions(groups, d.due)
	return groups, nil
}

// topLevelTexts is the retrieval query: the open top-level items' texts.
func (d draft) topLevelTexts() []string {
	parents := d.tree.ParentItems()
	texts := make([]string, 0, len(parents))
	for _, p := range parents {
		texts = append(texts, p.Description)
	}
	return texts
}

// drawByRules answers a draw whose generation failed with cause: a rules plan
// (R36–R38), stored over the draw d started from, as an AI plan would be. It
// makes no model call and records nothing in the ledger (R40).
//
// It is free (R16): the caller has already given back any refresh the failed
// draw spent, and the free draw stays as it was, so the key's first
// successful AI draw is still free. A rules plan never satisfies the rule
// gate (R17), so the next Plan my day tries the model again.
func (s *Service) drawByRules(ctx context.Context, d draft, cause error) (DayPlan, error) {
	slog.WarnContext(ctx, "dayplan: generation failed, drawing by rules",
		"err", cause, "plan_id", d.key.PlanID, "budget_exhausted", errors.Is(cause, ai.ErrBudgetExhausted))
	groups := drawRules(d.rulesInput())
	// The ranker stamps what its revisions rule places; a due revision whose
	// Revisions group could not open (three groups pinned) can still enter
	// under another rule, and is stamped here like any AI draw's (R10, R45).
	stampRevisions(groups, d.due)
	stored, _, err := s.save(ctx, d, d.plan(SourceRules, groups, s.now(), d.stored.FreeDrawUsed))
	if err != nil {
		return DayPlan{}, err
	}
	return s.present(ctx, stored, d.profile.learning())
}

// rulesInput is what d gives a rules draw: the due revisions, ranked first
// (R37), and the entries ticked today, pinned (R39).
func (d draft) rulesInput() rulesInput {
	return rulesInput{Tree: d.tree, UpdatedAt: d.updatedAt, Date: d.key.Date, DueRevisions: d.due, Pinned: d.pinned}
}

// refundTimeout bounds a refund that outlives its request.
const refundTimeout = 3 * time.Second

// returnRefresh gives back a refresh spent on a draw that did not store. A
// failure to return it is logged, not returned: the caller already has the
// draw's outcome, and the cost is one refresh the user did not get.
//
// The refund is detached from ctx's cancellation, with its own short timeout:
// a caller that hung up mid-draw (the usual way a draw fails late) has still
// spent, and the spend must still come back.
func (s *Service) returnRefresh(ctx context.Context, key guided.DayKey) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), refundTimeout)
	defer cancel()
	if err := s.refreshes.Return(ctx, key); err != nil {
		slog.ErrorContext(ctx, "dayplan: return refresh failed", "err", err, "plan_id", key.PlanID)
	}
}

// present fills what is derived rather than stored: refreshesRemaining comes
// from the refresh counter (R19), revisionsEnabled from the revisions
// switches when the plan is a learning plan (R49), and groups are never nil.
func (s *Service) present(ctx context.Context, p DayPlan, learning bool) (DayPlan, error) {
	if p.Groups == nil {
		p.Groups = []Group{}
	}
	p, err := s.withRevisionSwitches(ctx, p, learning)
	if err != nil {
		return DayPlan{}, err
	}
	left, err := s.refreshes.Remaining(ctx, p.Key)
	if err != nil {
		return DayPlan{}, fmt.Errorf("dayplan: refreshes remaining: %w", err)
	}
	p.RefreshesRemaining = left
	return p, nil
}
