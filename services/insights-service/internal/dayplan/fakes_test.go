package dayplan

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/ai"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
)

// today is the fixed server clock of every service test.
var today = time.Date(2026, 10, 25, 15, 0, 0, 0, time.UTC)

const todayDate = "2026-10-25"

type fakePlans struct {
	mu      sync.Mutex
	profile PlanProfile
	items   []guided.PlanItem
	err     error
	reads   int
}

func (f *fakePlans) GetPlanProfile(ctx context.Context, planID, userID uuid.UUID) (PlanProfile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	return f.profile, f.err
}

func (f *fakePlans) ListPlanItems(ctx context.Context, planID, userID uuid.UUID) ([]guided.PlanItem, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	return f.items, f.err
}

func (f *fakePlans) setItems(items []guided.PlanItem) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.items = items
}

// fakeGenerator answers every call with content, judged by the request's
// Accept as the metered generator does, and records one attempt per call.
type fakeGenerator struct {
	mu       sync.Mutex
	content  string
	err      error
	requests []ai.StructuredRequest
	attempts []ai.Attempt
	// barrier, when set, holds each call until that many calls have arrived,
	// so concurrent draws are forced to race past the store.
	barrier *sync.WaitGroup
	// during, when set, runs inside each call, after the store was read and
	// before the draw is saved: what happens while a generation is in flight.
	during func()
}

func (f *fakeGenerator) Generate(ctx context.Context, req ai.StructuredRequest) (ai.Completion, error) {
	f.mu.Lock()
	f.requests = append(f.requests, req)
	content, err := f.content, f.err
	f.mu.Unlock()
	if f.barrier != nil {
		f.barrier.Done()
		f.barrier.Wait()
	}
	if f.during != nil {
		f.during()
	}
	if err == nil && req.Accept != nil {
		err = req.Accept(content)
	}
	f.mu.Lock()
	f.attempts = append(f.attempts, ai.Attempt{Call: req.Call, Model: "gpt-5-mini", Succeeded: err == nil})
	f.mu.Unlock()
	return ai.Completion{Content: content, Model: "gpt-5-mini"}, err
}

func (f *fakeGenerator) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func (f *fakeGenerator) answer(content string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.content, f.err = content, nil
}

func (f *fakeGenerator) fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

// memStore keeps one plan per key and applies Save's draw condition under a
// lock, as the Postgres conditional update does. Ticks are kept per key and
// item, and read onto entries as the repository's join does.
type memStore struct {
	mu    sync.Mutex
	plans map[guided.DayKey]DayPlan
	ticks map[guided.DayKey]map[uuid.UUID]uuid.UUID // item -> entry
	saves int
	// rungs and switches are the revision ladder per (plan, user, item), as
	// the revisions and revision_switches tables hold it.
	rungs    map[ladderKey][]rung
	switches map[ladderKey]bool
	// applyFailures is how many of the next ApplyLadder calls fail, writing
	// nothing.
	applyFailures int
}

var errLadderWrite = errors.New("ladder write failed")

// failApplies makes the next n ladder writes fail.
func (m *memStore) failApplies(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.applyFailures = n
}

// rungsOf is a copy of the item's ladder.
func (m *memStore) rungsOf(planID, userID, itemID uuid.UUID) []rung {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.rungs[ladderKey{planID, userID, itemID}])
}

type ladderKey struct{ planID, userID, itemID uuid.UUID }

func newMemStore() *memStore {
	return &memStore{
		plans: map[guided.DayKey]DayPlan{}, ticks: map[guided.DayKey]map[uuid.UUID]uuid.UUID{},
		rungs: map[ladderKey][]rung{}, switches: map[ladderKey]bool{},
	}
}

func (m *memStore) Find(ctx context.Context, key guided.DayKey) (DayPlan, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.plans[key]
	return m.withTicks(p), ok, nil
}

// withTicks is a copy of p with each entry's tick read in.
func (m *memStore) withTicks(p DayPlan) DayPlan {
	groups := make([]Group, len(p.Groups))
	for gi, g := range p.Groups {
		groups[gi] = Group{Label: g.Label, Entries: make([]Entry, len(g.Entries))}
		for ei, e := range g.Entries {
			_, e.Ticked = m.ticks[p.Key][derefID(e.ItemID)]
			groups[gi].Entries[ei] = e
		}
	}
	if p.Groups == nil {
		groups = nil
	}
	p.Groups = groups
	return p
}

func derefID(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}
	return *id
}

func (m *memStore) Save(ctx context.Context, p DayPlan, prev DrawRef) (DayPlan, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saves++
	current, ok := m.plans[p.Key]
	if (ok && current.Ref() != prev) || (!ok && prev != DrawRef{}) {
		return m.withTicks(current), false, nil
	}
	p.TickVersion = current.TickVersion
	m.plans[p.Key] = p
	return p, true, nil
}

func (m *memStore) FindByEntry(ctx context.Context, entryID, userID uuid.UUID) (DayPlan, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key, p := range m.plans {
		if key.UserID != userID {
			continue
		}
		if _, ok := p.entry(entryID); ok {
			return m.withTicks(p), true, nil
		}
	}
	return DayPlan{}, false, nil
}

// Tick moves the key's tick version first, as the repository does, then
// binds and ticks.
func (m *memStore) Tick(ctx context.Context, t Tick) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.plans[t.Key]; ok {
		p.TickVersion++
		m.plans[t.Key] = p
	}
	if t.Bind {
		if p, ok := m.plans[t.Key]; ok {
			if other, taken := p.entryForItem(t.ItemID); taken && other.ID != t.EntryID {
				return false, ErrEntryNotTickable
			}
		}
		p, ok := m.plans[t.Key]
		bound := false
		for gi := range p.Groups {
			for ei := range p.Groups[gi].Entries {
				e := &p.Groups[gi].Entries[ei]
				if ok && p.DrawID == t.DrawID && e.ID == t.EntryID && e.Kind == EntryNewStep {
					item := t.ItemID
					e.Kind, e.ItemID, bound = EntryExistingItem, &item, true
				}
			}
		}
		if !bound {
			return false, ErrEntryNotTickable
		}
	}
	if m.ticks[t.Key] == nil {
		m.ticks[t.Key] = map[uuid.UUID]uuid.UUID{}
	}
	if _, ok := m.ticks[t.Key][t.ItemID]; ok {
		return false, nil
	}
	m.ticks[t.Key][t.ItemID] = t.EntryID
	return true, nil
}

func (m *memStore) Untick(ctx context.Context, key guided.DayKey, itemID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.plans[key]; ok {
		p.TickVersion++
		m.plans[key] = p
	}
	if _, ok := m.ticks[key][itemID]; !ok {
		return false, nil
	}
	delete(m.ticks[key], itemID)
	return true, nil
}

func (m *memStore) Ladder(ctx context.Context, key guided.DayKey, itemID uuid.UUID) (ladderState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	lk := ladderKey{key.PlanID, key.UserID, itemID}
	st := ladderState{Rungs: slices.Clone(m.rungs[lk]), FirstTick: true}
	for k, items := range m.ticks {
		if _, ok := items[itemID]; ok && k.PlanID == key.PlanID && k.UserID == key.UserID && !k.Date.Equal(key.Date) {
			st.FirstTick = false
		}
	}
	if enabled, ok := m.switches[lk]; ok {
		st.Off, st.Resume = !enabled, enabled
	}
	return st, nil
}

func (m *memStore) ApplyLadder(ctx context.Context, key guided.DayKey, itemID uuid.UUID, c ladderChange) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.applyFailures > 0 {
		m.applyFailures--
		return errLadderWrite
	}
	lk := ladderKey{key.PlanID, key.UserID, itemID}
	rungs := slices.DeleteFunc(m.rungs[lk], func(r rung) bool { return slices.Contains(c.Remove, r.Rung) })
	for i := range rungs {
		r := &rungs[i]
		if c.Reopen != nil && r.Rung == *c.Reopen && r.CompletedOn != nil && r.CompletedOn.Equal(key.Date) {
			r.CompletedOn = nil
		}
		if c.Complete != nil && r.Rung == *c.Complete && r.CompletedOn == nil {
			date := key.Date
			r.CompletedOn = &date
		}
	}
	if c.Schedule != nil && !slices.ContainsFunc(rungs, func(r rung) bool { return r.Rung == c.Schedule.Rung }) {
		rungs = append(rungs, *c.Schedule)
	}
	m.rungs[lk] = rungs
	return nil
}

func (m *memStore) DueRevisions(ctx context.Context, key guided.DayKey) ([]dueRevision, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []dueRevision
	for lk, rungs := range m.rungs {
		if lk.planID != key.PlanID || lk.userID != key.UserID {
			continue
		}
		if enabled, ok := m.switches[lk]; ok && !enabled {
			continue
		}
		for _, r := range rungs {
			if r.CompletedOn == nil && !r.DueOn.After(key.Date) {
				out = append(out, dueRevision{ItemID: lk.itemID, Rung: r.Rung, DueOn: r.DueOn})
			}
		}
	}
	slices.SortFunc(out, func(a, b dueRevision) int {
		if c := a.DueOn.Compare(b.DueOn); c != 0 {
			return c
		}
		return strings.Compare(a.ItemID.String(), b.ItemID.String())
	})
	return out, nil
}

func (m *memStore) RevisionsOff(ctx context.Context, planID, userID uuid.UUID) (map[uuid.UUID]bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	off := map[uuid.UUID]bool{}
	for lk, enabled := range m.switches {
		if lk.planID == planID && lk.userID == userID && !enabled {
			off[lk.itemID] = true
		}
	}
	return off, nil
}

func (m *memStore) SetRevisions(ctx context.Context, planID, userID, itemID uuid.UUID, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	lk := ladderKey{planID, userID, itemID}
	if enabled {
		if _, ok := m.switches[lk]; ok {
			m.switches[lk] = true
		}
		return nil
	}
	m.switches[lk] = false
	m.rungs[lk] = slices.DeleteFunc(m.rungs[lk], func(r rung) bool { return r.CompletedOn == nil })
	return nil
}

// ticksFor is the ticked items of key.
func (m *memStore) ticksFor(key guided.DayKey) []uuid.UUID {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []uuid.UUID
	for item := range m.ticks[key] {
		out = append(out, item)
	}
	return out
}

func (m *memStore) stored(key guided.DayKey) (DayPlan, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.plans[key]
	return m.withTicks(p), ok
}

// memCounter is the refresh counter with the cap, atomic under a lock.
type memCounter struct {
	mu    sync.Mutex
	spent map[guided.DayKey]int
}

func newMemCounter() *memCounter { return &memCounter{spent: map[guided.DayKey]int{}} }

func (c *memCounter) Spend(ctx context.Context, key guided.DayKey) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.spent[key] >= guided.RefreshCap {
		return 0, guided.ErrRefreshLimitReached
	}
	c.spent[key]++
	return guided.RefreshCap - c.spent[key], nil
}

// Return fails on a done context, as the database round trip would.
func (c *memCounter) Return(ctx context.Context, key guided.DayKey) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.spent[key] > 0 {
		c.spent[key]--
	}
	return nil
}

func (c *memCounter) Remaining(ctx context.Context, key guided.DayKey) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return guided.RefreshCap - c.spent[key], nil
}

func (c *memCounter) spentFor(key guided.DayKey) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.spent[key]
}

type fixture struct {
	planID, userID uuid.UUID
	plans          *fakePlans
	gen            *fakeGenerator
	store          *memStore
	counter        *memCounter
	svc            *Service
}

func newFixture(items []guided.PlanItem) *fixture {
	f := &fixture{
		planID: uuid.New(), userID: uuid.New(),
		plans:   &fakePlans{profile: PlanProfile{Type: "learning", Focus: "Get through the graph chapter"}, items: items},
		gen:     &fakeGenerator{},
		store:   newMemStore(),
		counter: newMemCounter(),
	}
	f.svc = NewService(f.plans, f.gen, guided.NewSnippetSource(guided.NoRetrieval{}, noMemory{}), f.store, f.store, f.counter)
	f.svc.now = func() time.Time { return today }
	return f
}

func (f *fixture) key() guided.DayKey {
	d, _ := time.Parse(guided.DateLayout, todayDate)
	return guided.DayKey{PlanID: f.planID, UserID: f.userID, Date: d}
}

func (f *fixture) planMyDay() (DayPlan, error) {
	return f.svc.PlanMyDay(context.Background(), f.planID, f.userID, todayDate)
}

func (f *fixture) getDayPlan() (DayPlan, error) {
	return f.svc.GetDayPlan(context.Background(), f.planID, f.userID, todayDate)
}

type noMemory struct{}

func (noMemory) HasPersonalRows(context.Context, uuid.UUID) (bool, error) { return false, nil }

func ptr[T any](v T) *T { return &v }

// treeFixture is a small plan: two open top-level items, one with an open and
// a done child, plus a note and an archived task that are not in the tree.
type treeFixture struct {
	graphs, dp, bfs, dfs, note, archived uuid.UUID
}

func newTreeFixture() treeFixture {
	return treeFixture{graphs: uuid.New(), dp: uuid.New(), bfs: uuid.New(), dfs: uuid.New(), note: uuid.New(), archived: uuid.New()}
}

func (tf treeFixture) items() []guided.PlanItem {
	tomorrow := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	return []guided.PlanItem{
		{ID: tf.graphs, Description: "Graphs", Type: "task", Scope: "longterm", DueDate: &tomorrow},
		{ID: tf.bfs, ParentID: &tf.graphs, Description: "Solve 3 BFS problems", Type: "task", Scope: "longterm"},
		{ID: tf.dfs, ParentID: &tf.graphs, Description: "Read the DFS chapter", Type: "task", Scope: "longterm", Done: true},
		{ID: tf.dp, Description: "Dynamic programming", Type: "task", Scope: "longterm"},
		{ID: tf.note, Description: "remember the visited set", Type: "note", Scope: "longterm"},
		{ID: tf.archived, Description: "Old topic", Type: "task", Scope: "longterm", Archived: true},
	}
}

// genEntry is one entry as the model would return it.
type genEntry struct {
	Kind         string  `json:"kind"`
	ItemID       *string `json:"item_id"`
	ParentItemID *string `json:"parent_item_id"`
	Text         string  `json:"text"`
	Reason       string  `json:"reason"`
	Minutes      int     `json:"minutes"`
}

type genGroup struct {
	Label   string     `json:"label"`
	Entries []genEntry `json:"entries"`
}

func modelOutput(groups ...genGroup) string {
	b, err := json.Marshal(map[string]any{"groups": groups})
	if err != nil {
		panic(err)
	}
	return string(b)
}

func existing(id uuid.UUID, text string, minutes int) genEntry {
	s := id.String()
	return genEntry{Kind: "existing_item", ItemID: &s, Text: text, Reason: "due soon", Minutes: minutes}
}

func newStep(parent uuid.UUID, text string, minutes int) genEntry {
	s := parent.String()
	return genEntry{Kind: "new_step", ParentItemID: &s, Text: text, Reason: "gets it started", Minutes: minutes}
}
