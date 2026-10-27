package dayplan

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
)

// validate checks the model's groups against the plan tree and the day plan's
// bounds (R8, R9), dropping what fails, one entry at a time. Nothing is
// repaired and nothing is regenerated: a dropped entry is gone, a group left
// with no entries is dropped, and the result may be empty. The one adjustment
// is the estimate: an out-of-range minutes is left out, keeping the entry.
//
// pinned are the groups of entries ticked today (R12). They come first, as
// they are, and count toward the bounds: the model's entries fill the slots
// left, an entry for a pinned item is a second entry for it and dropped, and a
// model group whose label equals a group already kept joins it rather than
// opening another.
//
// due are the revisions due on the plan date: an existing item with one due is
// a candidate even when it is done (R8, S7).
//
// Every id is checked against the tree; none is trusted. An existing item's
// text and parent come from the tree, which is authoritative, not from the
// model's wording of them. Entries get their ids when the plan is stored.
func validate(tree guided.PlanTree, pinned []Group, due []dueRevision, generated []generatedGroup) []Group {
	idx := indexTree(tree, due)
	groups := make([]Group, 0, MaxGroups)
	seenItems := map[uuid.UUID]bool{}
	seenTexts := map[string]bool{}
	total := 0
	for _, g := range pinned {
		groups = append(groups, Group{Label: g.Label, Entries: slices.Clone(g.Entries)})
		for _, e := range g.Entries {
			if e.ItemID != nil {
				seenItems[*e.ItemID] = true
			}
			total++
		}
	}

	for gi, g := range generated {
		if gi >= MaxGroups {
			break
		}
		label := strings.TrimSpace(g.Label)
		if label == "" || utf8.RuneCountInString(label) > MaxLabelLength {
			continue
		}
		target := slices.IndexFunc(groups, func(kept Group) bool { return kept.Label == label })
		if target < 0 && len(groups) == MaxGroups {
			continue
		}
		var kept []Entry
		for _, ge := range g.Entries {
			if total == MaxEntries {
				break
			}
			e, ok := idx.entry(ge)
			if !ok {
				continue
			}
			if e.Kind == EntryExistingItem {
				if seenItems[*e.ItemID] {
					continue
				}
				seenItems[*e.ItemID] = true
			} else {
				text := guided.NormalizeText(e.Text)
				if seenTexts[text] {
					continue
				}
				seenTexts[text] = true
			}
			kept = append(kept, e)
			total++
		}
		switch {
		case len(kept) == 0:
		case target < 0:
			groups = append(groups, Group{Label: label, Entries: kept})
		default:
			groups[target].Entries = append(groups[target].Entries, kept...)
		}
	}
	return groups
}

// treeIndex is the plan tree keyed for validation, with the items that have
// a revision due.
type treeIndex struct {
	items    map[uuid.UUID]guided.TreeItem
	children map[uuid.UUID][]guided.TreeItem
	revising map[uuid.UUID]bool
}

func indexTree(tree guided.PlanTree, due []dueRevision) treeIndex {
	idx := treeIndex{items: map[uuid.UUID]guided.TreeItem{}, children: map[uuid.UUID][]guided.TreeItem{}, revising: map[uuid.UUID]bool{}}
	for _, r := range due {
		idx.revising[r.ItemID] = true
	}
	for _, it := range tree.Items {
		idx.items[it.ID] = it
		if it.ParentID != nil {
			idx.children[*it.ParentID] = append(idx.children[*it.ParentID], it)
		}
	}
	return idx
}

// entry applies R8 to one entry in isolation: an existing_item names an open
// task of the tree (top-level or child), or one with a revision due, done or
// not; a new_step names an open top-level task and does not repeat one of its
// children's text. Every entry needs a reason.
func (idx treeIndex) entry(g generatedEntry) (Entry, bool) {
	reason := strings.TrimSpace(g.Reason)
	if reason == "" {
		return Entry{}, false
	}
	e := Entry{Kind: EntryKind(g.Kind), Reason: reason, Minutes: minutes(g.Minutes)}

	switch e.Kind {
	case EntryExistingItem:
		item, ok := idx.candidateItem(g.ItemID)
		if !ok {
			return Entry{}, false
		}
		e.ItemID, e.ParentItemID, e.Text = &item.ID, item.ParentID, entryText(item.Description)
		return e, true

	case EntryNewStep:
		parent, ok := idx.openItem(g.ParentItemID)
		text := strings.TrimSpace(g.Text)
		if !ok || parent.ParentID != nil || text == "" {
			return Entry{}, false
		}
		for _, c := range idx.children[parent.ID] {
			if guided.NormalizeText(c.Description) == guided.NormalizeText(text) {
				return Entry{}, false
			}
		}
		e.ParentItemID, e.Text = &parent.ID, text
		return e, true

	default:
		return Entry{}, false
	}
}

// openItem resolves a model-given id to an open item of the tree.
func (idx treeIndex) openItem(id *string) (guided.TreeItem, bool) {
	item, ok := idx.item(id)
	if !ok || item.Done {
		return guided.TreeItem{}, false
	}
	return item, true
}

// candidateItem resolves a model-given id to an item an existing_item entry
// may name: open, or with a revision due (S7).
func (idx treeIndex) candidateItem(id *string) (guided.TreeItem, bool) {
	item, ok := idx.item(id)
	if !ok || (item.Done && !idx.revising[item.ID]) {
		return guided.TreeItem{}, false
	}
	return item, true
}

// item resolves a model-given id to an item of the tree.
func (idx treeIndex) item(id *string) (guided.TreeItem, bool) {
	if id == nil {
		return guided.TreeItem{}, false
	}
	parsed, err := uuid.Parse(*id)
	if err != nil {
		return guided.TreeItem{}, false
	}
	item, ok := idx.items[parsed]
	return item, ok
}

// minutes keeps an estimate within bounds and leaves out any other (S14).
func minutes(m int) *int {
	if m < MinMinutes || m > MaxMinutes {
		return nil
	}
	return &m
}
