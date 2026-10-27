package dayplan

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
)

// revisionsLabel is the rules plan's first group (R38).
const revisionsLabel = "Revisions"

// untitledLabel stands in for an item whose description is blank: as a top-
// level item's group label, so every group keeps a 1–40 character label
// (S14), and as any item's entry text (entryText).
const untitledLabel = "Untitled"

// dueRevision is one revision due on the plan date: the item, its rung (1–3)
// and the day it fell due, from the revision ladder (R44).
type dueRevision struct {
	ItemID uuid.UUID
	Rung   int
	DueOn  time.Time
}

// rulesInput is everything a rules draw reads. It is all data the service
// already holds; nothing here calls out.
type rulesInput struct {
	// Tree is the plan tree, in plan-service's order.
	Tree guided.PlanTree
	// UpdatedAt is each item's last update, for the next-step rule.
	UpdatedAt map[uuid.UUID]time.Time
	// Date is the plan date.
	Date time.Time
	// DueRevisions are the revisions due on the plan date (R44).
	DueRevisions []dueRevision
	// Pinned are the groups of entries ticked today (R39), kept as they are,
	// in place.
	Pinned []Group
}

// drawRules is the rules plan (R37, R38): due revisions (oldest due first),
// then overdue items (most overdue first), then items with a due date
// (soonest first), then the next open step of each top-level item, most
// recently updated first. Ties keep plan order; an item ranks once, at its
// first rule.
//
// Revisions sit in a "Revisions" group; every other entry under its top-level
// ancestor, labelled with that item's description cut to 40 characters.
// Pinned groups come first and count toward the bounds: entries are taken in
// rank order until 6, skipping any that would open a fourth group, and an
// entry whose label matches a group already there joins it. Every entry is an
// existing item with no minutes and a reason that names its rule. Entry ids
// are left for the store; pinned entries keep theirs.
func drawRules(in rulesInput) []Group {
	r := newRanker(in)
	groups := make([]Group, 0, MaxGroups)
	seen := map[uuid.UUID]bool{}
	total := 0
	for _, g := range in.Pinned {
		groups = append(groups, Group{Label: g.Label, Entries: slices.Clone(g.Entries)})
		for _, e := range g.Entries {
			if e.ItemID != nil {
				seen[*e.ItemID] = true
			}
		}
		total += len(g.Entries)
	}

	for _, c := range r.candidates() {
		if total >= MaxEntries {
			break
		}
		if seen[*c.entry.ItemID] {
			continue
		}
		gi := slices.IndexFunc(groups, func(g Group) bool { return g.Label == c.label })
		if gi < 0 {
			if len(groups) == MaxGroups {
				continue
			}
			groups = append(groups, Group{Label: c.label})
			gi = len(groups) - 1
		}
		groups[gi].Entries = append(groups[gi].Entries, c.entry)
		seen[*c.entry.ItemID] = true
		total++
	}
	return groups
}

// candidate is one ranked entry and the label of the group it belongs in.
type candidate struct {
	label string
	entry Entry
}

// ranker holds the tree indexed for the rules.
type ranker struct {
	in    rulesInput
	items map[uuid.UUID]guided.TreeItem
}

func newRanker(in rulesInput) ranker {
	items := make(map[uuid.UUID]guided.TreeItem, len(in.Tree.Items))
	for _, it := range in.Tree.Items {
		items[it.ID] = it
	}
	return ranker{in: in, items: items}
}

// candidates is every rule's entries, in rank order, duplicates included.
func (r ranker) candidates() []candidate {
	var out []candidate
	out = append(out, r.revisions()...)
	out = append(out, r.overdue()...)
	out = append(out, r.dueSoonest()...)
	out = append(out, r.nextSteps()...)
	return out
}

// revisions are the due revisions whose item is still in the tree, done or
// not (S7), oldest due first.
func (r ranker) revisions() []candidate {
	revs := slices.Clone(r.in.DueRevisions)
	slices.SortStableFunc(revs, func(a, b dueRevision) int { return day(a.DueOn).Compare(day(b.DueOn)) })
	var out []candidate
	for _, rev := range revs {
		it, ok := r.items[rev.ItemID]
		if !ok {
			continue
		}
		c := r.candidate(it, fmt.Sprintf("Revision %d due", rev.Rung))
		c.label = revisionsLabel
		c.entry.Revision = &rev.Rung
		out = append(out, c)
	}
	return out
}

// overdue are the open items due before the plan date, most overdue first.
func (r ranker) overdue() []candidate {
	return r.dated(func(due time.Time) bool { return due.Before(day(r.in.Date)) }, "Overdue since %s")
}

// dueSoonest are the open items due on or after the plan date, soonest first.
func (r ranker) dueSoonest() []candidate {
	return r.dated(func(due time.Time) bool { return !due.Before(day(r.in.Date)) }, "Due %s")
}

// dated is the open items whose due day passes keep, earliest due day first,
// each with reason format applied to that day.
func (r ranker) dated(keep func(due time.Time) bool, format string) []candidate {
	var items []guided.TreeItem
	for _, it := range r.in.Tree.Items {
		if !it.Done && it.DueDate != nil && keep(day(*it.DueDate)) {
			items = append(items, it)
		}
	}
	slices.SortStableFunc(items, func(a, b guided.TreeItem) int { return day(*a.DueDate).Compare(day(*b.DueDate)) })
	out := make([]candidate, 0, len(items))
	for _, it := range items {
		out = append(out, r.candidate(it, fmt.Sprintf(format, day(*it.DueDate).Format("Jan 2"))))
	}
	return out
}

// nextSteps takes the top-level items, most recently updated first; each
// contributes its first open child in stored order, or itself when it has no
// open child and is open.
func (r ranker) nextSteps() []candidate {
	var tops []guided.TreeItem
	for _, it := range r.in.Tree.Items {
		if it.ParentID == nil {
			tops = append(tops, it)
		}
	}
	slices.SortStableFunc(tops, func(a, b guided.TreeItem) int {
		return cmp.Compare(r.in.UpdatedAt[b.ID].UnixNano(), r.in.UpdatedAt[a.ID].UnixNano())
	})
	var out []candidate
	for _, top := range tops {
		step, ok := r.firstOpenChild(top.ID)
		if !ok {
			if top.Done {
				continue
			}
			step = top
		}
		out = append(out, r.candidate(step, "Next step"))
	}
	return out
}

func (r ranker) firstOpenChild(parentID uuid.UUID) (guided.TreeItem, bool) {
	for _, it := range r.in.Tree.Items {
		if it.ParentID != nil && *it.ParentID == parentID && !it.Done {
			return it, true
		}
	}
	return guided.TreeItem{}, false
}

// candidate is it as an existing-item entry under its top-level ancestor.
func (r ranker) candidate(it guided.TreeItem, reason string) candidate {
	id := it.ID
	return candidate{
		label: groupLabel(r.topLevel(it).Description),
		entry: Entry{Kind: EntryExistingItem, ItemID: &id, ParentItemID: it.ParentID, Text: entryText(it.Description), Reason: reason},
	}
}

// topLevel walks it up to its highest ancestor in the tree. An ancestor
// missing from the tree (archived, or not a task) ends the walk.
func (r ranker) topLevel(it guided.TreeItem) guided.TreeItem {
	for depth := 0; it.ParentID != nil && depth < len(r.items); depth++ {
		parent, ok := r.items[*it.ParentID]
		if !ok {
			break
		}
		it = parent
	}
	return it
}

// groupLabel is a top-level item's description, trimmed and cut to the label
// bound.
func groupLabel(description string) string {
	label := strings.TrimSpace(description)
	if runes := []rune(label); len(runes) > MaxLabelLength {
		label = strings.TrimSpace(string(runes[:MaxLabelLength]))
	}
	if label == "" {
		return untitledLabel
	}
	return label
}

// entryText is an existing item's entry text: its description as the tree
// has it, or untitledLabel when that is blank, so a stored entry's text is
// never blank (day_plan_entries.text) and a blank item still gets planned.
func entryText(description string) string {
	if strings.TrimSpace(description) == "" {
		return untitledLabel
	}
	return description
}

// day is t's UTC calendar day: due dates compare by day, not instant.
func day(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
