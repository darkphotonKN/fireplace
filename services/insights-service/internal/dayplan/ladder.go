package dayplan

import (
	"slices"
	"time"
)

// The revision ladder (FS-02JBZ R41–R47, D5, D6, S6, S7): timing is a rule,
// never model output (ADR-0019 clause 5). Chained: each tick schedules only
// the next rung, a fixed number of days after that tick's plan date.

// MaxRevision is the last rung: ticking it ends the ladder.
const MaxRevision = 3

// rungOffsetDays is how many days after the tick that schedules it each rung
// falls due: revision 1 at +1, revision 2 at +3, revision 3 at +7.
var rungOffsetDays = [MaxRevision + 1]int{1: 1, 2: 3, 3: 7}

// rung is one revision of one item for one user on one plan: its number, the
// plan date of the tick that scheduled it, the day it falls due, and the plan
// date of the tick that completed it (nil while open).
type rung struct {
	Rung        int
	ScheduledOn time.Time
	DueOn       time.Time
	CompletedOn *time.Time
}

// ladderState is what a new day tick of an item reacts to: the item's rungs,
// whether this tick is its first day tick ever (for this user, on this plan,
// R42), whether its revisions are switched off, and whether they were switched
// back on since the last rung was cancelled (Resume, R47).
type ladderState struct {
	Rungs     []rung
	FirstTick bool
	Off       bool
	Resume    bool
}

// ladderChange is what a tick or an untick does to an item's ladder: at most
// one rung completed or reopened, at most one scheduled, and rungs removed.
type ladderChange struct {
	Complete *int
	Schedule *rung
	Reopen   *int
	Remove   []int
}

// onTick is what a new day tick on date does to the ladder in st.
//
// An open rung due on or before date is completed and the next one, if any,
// scheduled from date (R43); an open rung not yet due makes the tick schedule
// nothing. With no open rung, the item's first tick ever starts the ladder at
// revision 1 (R42), and a tick after the switch was turned back on resumes it
// at the rung after the last completed (R47); never past MaxRevision. With
// revisions switched off, nothing happens.
func onTick(st ladderState, date time.Time) ladderChange {
	if st.Off {
		return ladderChange{}
	}
	completed := 0
	for _, r := range st.Rungs {
		if r.CompletedOn == nil {
			if r.DueOn.After(date) {
				return ladderChange{}
			}
			return ladderChange{Complete: &r.Rung, Schedule: schedule(r.Rung+1, date)}
		}
		completed = max(completed, r.Rung)
	}
	switch {
	case len(st.Rungs) == 0 && st.FirstTick:
		return ladderChange{Schedule: schedule(1, date)}
	case st.Resume:
		return ladderChange{Schedule: schedule(completed+1, date)}
	default:
		return ladderChange{}
	}
}

// schedule is rung n scheduled by a tick on date, or nil past the ladder.
func schedule(n int, date time.Time) *rung {
	if n > MaxRevision {
		return nil
	}
	return &rung{Rung: n, ScheduledOn: date, DueOn: date.AddDate(0, 0, rungOffsetDays[n])}
}

// onUntick undoes what the item's tick on date did (R46, D5): the rung that
// tick completed is reopened, the rung it scheduled is removed, and so is
// every later rung, since each followed from the one removed. A tick that did
// neither leaves the ladder as it is.
func onUntick(rungs []rung, date time.Time) ladderChange {
	var c ladderChange
	removeFrom := 0
	for _, r := range rungs {
		if r.CompletedOn != nil && r.CompletedOn.Equal(date) {
			c.Reopen = &r.Rung
		}
		if r.ScheduledOn.Equal(date) && (removeFrom == 0 || r.Rung < removeFrom) {
			removeFrom = r.Rung
		}
	}
	if removeFrom > 0 {
		for _, r := range rungs {
			if r.Rung >= removeFrom {
				c.Remove = append(c.Remove, r.Rung)
			}
		}
		slices.Sort(c.Remove)
	}
	return c
}
