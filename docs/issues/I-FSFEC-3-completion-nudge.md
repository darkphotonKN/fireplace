---
id: I-FSFEC-3
status: open
implements: FS-FSFEC
blocked_by: [I-FSFEC-1]
labels: [feature]
title: "FS-FSFEC slice 3: complete milestones offer to close, closed ones stay faded"
---
Implements FS-FSFEC §Requirements (R5, R8–R14), §Edge States

## What to Build

In the grid, a milestone whose task steps are all done says so and offers to close itself.

**Complete state (R10).** At least one task step, all done, milestone not done: full meter, a
glow on the card, and a nudge with **Close** and a dismiss. Read completeness from slice 1's
`milestones.ts` rather than the card's local `taskProgress`, so the strip and the card can't
disagree.

**Close (R11).** Close calls the same `onToggleDone` the card's checkbox already uses: optimistic,
reverting with the existing error on failure. No new prop or API is needed, so `Todo.tsx` stays
untouched by this slice.

**Dismiss (R12).** Hides the nudge until that milestone's done/total changes. In memory in
`ChecklistGrid`, keyed by milestone and its progress pair. No server state and no storage, so a
reload or remount shows it again.

**Never auto-close (R13).** Adding or unticking a step ends the complete state. Nothing sets
`done` but a person.

**Closed milestones (R8, R9).** Stay in place, faded (the existing `opacity-60` path). The
checkbox stays hand-settable at any progress: closing early and reopening both work.

**Notes (R5).** A top-level note card never shows a nudge or a checkbox. Grid only (R14): no
change to list rows.

## Acceptance Criteria

- [ ] A milestone with all task steps done and not itself done shows a full meter, a glow and a
      Close nudge.
- [ ] A milestone with no task steps never shows the nudge.
- [ ] Close marks the milestone done; it stays in its grid position, faded.
- [ ] A failed Close reverts the milestone and shows the existing error.
- [ ] Dismiss hides the nudge; unticking then re-ticking a step brings it back.
- [ ] Remounting the grid brings a dismissed nudge back.
- [ ] Adding a step to a complete milestone ends the complete state.
- [ ] A milestone is never set done without a click on its checkbox or Close.
- [ ] Note cards never show the nudge.
- [ ] List view rows are unchanged.
- [ ] Tests pass.

## Blocked By

I-FSFEC-1 — reads completeness from `milestones.ts`. Does not touch `Todo.tsx`, so it can run
alongside I-FSFEC-2.

## Spec Reference

FS-FSFEC §Requirements R5, R8–R14 (completion nudge, charter default C8); §Edge States (all
steps done not closed, closed early, reopened, step added to complete milestone, dismiss then
reload, Close fails); §Decisions D7, D8.

## TDD Approach

- RED: a `ChecklistGrid` test with a milestone whose two task steps are done asserts the nudge
  is shown and that Close calls `onToggleDone` with the milestone's id.
- GREEN: the complete state and nudge.
- Then: dismiss holds until the progress pair changes; no-step and note cards never nudge.
