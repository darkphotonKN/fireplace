package dayplan

import (
	"reflect"
	"testing"
	"time"
)

func openRung(n int, scheduledOn, dueOn time.Time) rung {
	return rung{Rung: n, ScheduledOn: scheduledOn, DueOn: dueOn}
}

func doneRung(n int, scheduledOn, dueOn, completedOn time.Time) rung {
	c := completedOn
	return rung{Rung: n, ScheduledOn: scheduledOn, DueOn: dueOn, CompletedOn: &c}
}

func TestOnTick_TheLadder(t *testing.T) {
	tests := []struct {
		name  string
		state ladderState
		date  time.Time
		want  ladderChange
	}{
		{"first tick ever schedules revision 1 at +1",
			ladderState{FirstTick: true}, oct(20),
			ladderChange{Schedule: &rung{Rung: 1, ScheduledOn: oct(20), DueOn: oct(21)}}},
		{"ticking revision 1 completes it and schedules revision 2 at +3",
			ladderState{Rungs: []rung{openRung(1, oct(20), oct(21))}}, oct(21),
			ladderChange{Complete: ptr(1), Schedule: &rung{Rung: 2, ScheduledOn: oct(21), DueOn: oct(24)}}},
		{"ticking revision 2 completes it and schedules revision 3 at +7",
			ladderState{Rungs: []rung{doneRung(1, oct(20), oct(21), oct(21)), openRung(2, oct(21), oct(24))}}, oct(24),
			ladderChange{Complete: ptr(2), Schedule: &rung{Rung: 3, ScheduledOn: oct(24), DueOn: oct(31)}}},
		{"ticking revision 3 completes it and ends the ladder",
			ladderState{Rungs: []rung{doneRung(1, oct(1), oct(2), oct(2)), doneRung(2, oct(2), oct(5), oct(5)), openRung(3, oct(5), oct(12))}}, oct(12),
			ladderChange{Complete: ptr(3)}},
		{"a slipped revision counts the next rung from the day it is ticked",
			ladderState{Rungs: []rung{openRung(1, oct(10), oct(11))}}, oct(15),
			ladderChange{Complete: ptr(1), Schedule: &rung{Rung: 2, ScheduledOn: oct(15), DueOn: oct(18)}}},
		{"a tick before the open rung is due schedules nothing",
			ladderState{Rungs: []rung{openRung(2, oct(21), oct(24))}}, oct(22),
			ladderChange{}},
		{"a tick after the ladder ended schedules nothing",
			ladderState{Rungs: []rung{doneRung(1, oct(1), oct(2), oct(2)), doneRung(2, oct(2), oct(5), oct(5)), doneRung(3, oct(5), oct(12), oct(12))}}, oct(20),
			ladderChange{}},
		{"a later tick that is not the first, with no ladder, starts nothing",
			ladderState{}, oct(20),
			ladderChange{}},
		{"revisions switched off: nothing completes, nothing is scheduled",
			ladderState{FirstTick: true, Off: true}, oct(20),
			ladderChange{}},
		{"switched back on: the next tick schedules the next rung",
			ladderState{Rungs: []rung{doneRung(1, oct(1), oct(2), oct(2))}, Resume: true}, oct(20),
			ladderChange{Schedule: &rung{Rung: 2, ScheduledOn: oct(20), DueOn: oct(23)}}},
		{"switched back on with nothing completed: revision 1 at +1",
			ladderState{Resume: true}, oct(20),
			ladderChange{Schedule: &rung{Rung: 1, ScheduledOn: oct(20), DueOn: oct(21)}}},
		{"switched back on after the ladder ended: never past 3",
			ladderState{Rungs: []rung{doneRung(1, oct(1), oct(2), oct(2)), doneRung(2, oct(2), oct(5), oct(5)), doneRung(3, oct(5), oct(12), oct(12))}, Resume: true}, oct(20),
			ladderChange{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := onTick(tt.state, tt.date)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("onTick = %+v, want %+v", show(got), show(tt.want))
			}
		})
	}
}

func TestOnUntick_UndoesWhatThatTickDid(t *testing.T) {
	tests := []struct {
		name  string
		rungs []rung
		date  time.Time
		want  ladderChange
	}{
		{"untick of the first tick removes revision 1",
			[]rung{openRung(1, oct(20), oct(21))}, oct(20),
			ladderChange{Remove: []int{1}}},
		{"untick of a revision tick reopens it and removes the next rung",
			[]rung{doneRung(1, oct(20), oct(21), oct(21)), openRung(2, oct(21), oct(24))}, oct(21),
			ladderChange{Reopen: ptr(1), Remove: []int{2}}},
		{"untick of the tick that ended the ladder reopens revision 3",
			[]rung{doneRung(1, oct(1), oct(2), oct(2)), doneRung(2, oct(2), oct(5), oct(5)), doneRung(3, oct(5), oct(12), oct(12))}, oct(12),
			ladderChange{Reopen: ptr(3)}},
		{"untick of a tick that scheduled nothing changes nothing",
			[]rung{openRung(2, oct(21), oct(24))}, oct(22),
			ladderChange{}},
		{"removing a rung removes the rungs that followed from it",
			[]rung{doneRung(1, oct(20), oct(21), oct(21)), openRung(2, oct(21), oct(24))}, oct(20),
			ladderChange{Remove: []int{1, 2}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := onUntick(tt.rungs, tt.date)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("onUntick = %+v, want %+v", show(got), show(tt.want))
			}
		})
	}
}

// show is c with its pointers read, for failure messages.
func show(c ladderChange) map[string]any {
	m := map[string]any{"remove": c.Remove}
	if c.Complete != nil {
		m["complete"] = *c.Complete
	}
	if c.Reopen != nil {
		m["reopen"] = *c.Reopen
	}
	if c.Schedule != nil {
		m["schedule"] = *c.Schedule
	}
	return m
}
