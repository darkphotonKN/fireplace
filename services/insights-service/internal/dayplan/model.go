// Package dayplan is FS-02JBZ's day plan: up to six entries in up to three
// labelled groups for one user's day on one plan, drawn only when the user asks
// (Plan my day) and otherwise only read (ADR-0019).
//
// It is built on the guided flow's machinery rather than beside it: the plan
// tree, its plan-state hash, the per (plan, user, date) refresh counter, the
// date rule, retrieval, and the metered structured generator.
//
// Server names say item and parent item, never milestone (ADR-0014); only
// prompt prose may.
package dayplan

import (
	"time"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
)

// Bounds of a day plan (FS-02JBZ D2, R5).
const (
	MaxGroups      = 3
	MaxEntries     = 6
	MaxLabelLength = 40
	MinMinutes     = 5
	MaxMinutes     = 240
)

// Source says how a stored day plan was drawn.
type Source string

const (
	SourceAI    Source = "ai"
	SourceRules Source = "rules"
)

// EntryKind says whether an entry points at an existing item or proposes a
// new child of a parent item.
type EntryKind string

const (
	EntryExistingItem EntryKind = "existing_item"
	EntryNewStep      EntryKind = "new_step"
)

// Entry is one thing to do on the plan date (R3). ItemID is set iff Kind is
// EntryExistingItem. ParentItemID is the parent a new step would go under, or
// an existing item's parent when it has one. Minutes is nil on a rules entry
// or when the model's estimate was out of range; Revision is set only by the
// server (R10), never by the model. Ticked is the day tick on the entry's item
// for the plan date (R24), read beside the entry, never stored on it.
// RevisionsEnabled is the item's revisions switch (R49), set when served and
// only on existing_item entries of learning plans; nil everywhere else.
type Entry struct {
	ID           uuid.UUID
	Kind         EntryKind
	ItemID       *uuid.UUID
	ParentItemID *uuid.UUID
	Text         string
	Reason       string
	Minutes      *int
	Revision     *int
	Ticked       bool

	RevisionsEnabled *bool
}

// Group is a short label over related entries, in order. It has no id.
type Group struct {
	Label   string
	Entries []Entry
}

// DayPlan is the day plan for one key. Planned is false for the not_planned
// state a read answers when nothing is stored, which has no source, draw time
// or groups. RefreshesRemaining is derived from the refresh counter when
// served, never stored.
//
// DrawID names the stored draw and TickVersion counts the ticks and unticks
// on the key; a redraw replaces the plan only if neither moved since it
// started (R21, Ref). FreeDrawUsed records that a successful AI draw has been
// stored for the key (R15).
type DayPlan struct {
	Key                guided.DayKey
	Planned            bool
	Source             Source
	GeneratedAt        time.Time
	PlanStateHash      string
	DrawID             uuid.UUID
	TickVersion        int64
	FreeDrawUsed       bool
	Groups             []Group
	RefreshesRemaining int
}

// DrawRef is the stored state a redraw starts from: the draw the key serves
// and its tick version. The zero DrawRef means nothing is stored. A redraw's
// save is conditional on both, so a tick that lands while a redraw is in
// flight (whose draw never saw it) makes the redraw lose.
type DrawRef struct {
	DrawID      uuid.UUID
	TickVersion int64
}

// Ref is the stored state p is, for a redraw starting from it.
func (p DayPlan) Ref() DrawRef {
	return DrawRef{DrawID: p.DrawID, TickVersion: p.TickVersion}
}
