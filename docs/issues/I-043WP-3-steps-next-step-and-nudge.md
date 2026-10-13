---
id: I-043WP-3
status: done
implements: FS-043WP
blocked_by: [I-043WP-2]
labels: [feature]
title: "FS-043WP slice 3: work the steps on the page, see the next one, and get the same nudge"
---
Implements FS-043WP §Requirements (R21, R26–R32), §Edge States

## What to Build

The page becomes a place to move the milestone forward. You can tick, add and rename steps, see
the one next step, and get the offer to close the milestone once every step is done.

**Steps (R29–R31, G8, G9).** Task steps tick and untick. Any step renames inline: click its
text, Enter or blur saves, Esc cancels, and an empty rename keeps the old text. Steps have no
other actions on the page. An always-visible "+ add step" opens a composer:
- Enter with text creates a **task** step at the end of this milestone; the input clears and
  keeps focus.
- Esc closes the composer.
- Enter is ignored while a create is in flight.
- A failed create keeps the text and shows the existing error.

Mutations go through `client/src/api/checklists.ts`, as `Todo`'s do. They are optimistic where
today's lists are and revert with the existing error text on failure (R32).

**Next step (R26–R28, G11, G12).** Derive it and never store it: it is the first non-archived,
not-done task step in stored order. Put the derivation in `lib/milestones.ts` next to
`isComplete`, with a table test. Show the next step on its own near the top, with its checkbox.
Ticking it promotes the step after it. The same step is also marked as next in the full list.
When there is no open task step, the slot shows:
- all done and the milestone not closed: the completion nudge;
- no task steps (none at all, or only notes): an invitation to add the first step, which opens
  the composer;
- the milestone closed: nothing.

**One nudge (R21, C7).** The grid card's nudge has its markup, Close through the done toggle,
and the dismiss memory that lasts until the progress pair changes (`pairOf`). Move all of it out
of `ChecklistGrid.tsx` into one shared component, and render that component on the card and on
the page. The grid's behaviour and its existing tests must not change.

Files expected: `client/src/components/MilestonePage.tsx` + test,
`client/src/components/CompletionNudge.tsx` (new; any name that fits) + test,
`client/src/components/ChecklistGrid.tsx` (switches to the shared nudge),
`client/src/lib/milestones.ts` + test.

## Acceptance Criteria

- [ ] Task steps tick and untick on the page; a failed tick reverts with the existing error.
- [ ] Any step renames inline; an empty rename keeps the old text; Esc cancels.
- [ ] "+ add step" adds task steps at the end, keeps focus, ignores Enter mid-create, and keeps
      the text on failure.
- [ ] The next step is the first open task step in order. It shows on its own with a checkbox
      and is marked in the list. Ticking it promotes the next one.
- [ ] Notes are never the next step.
- [ ] With no task steps, the page shows an invitation that opens the composer.
- [ ] With all task steps done and the milestone not closed, the nudge shows in the next-step
      slot. Close closes the milestone; dismiss hides the nudge until the progress pair changes.
- [ ] A closed milestone shows no next step.
- [ ] The grid card and the page render the same nudge component, and the existing
      completion-nudge tests pass unchanged.
- [ ] Tests pass.

## Blocked By

I-043WP-2: needs `MilestonePage` and its load.

## Spec Reference

FS-043WP §Requirements R21 (shared completion nudge, charter default C7), R26–R28 (next step,
charter default C6, gap defaults G11, G12), R29–R32 (steps, gap defaults G8, G9). §Edge States
(no steps, only notes, all done, closed with steps open, dismiss then re-tick, dismiss then
reopen, next-step tick fails, next step renamed).

## TDD Approach

- RED: a `lib/milestones.ts` table test for the next step, covering open steps in order, notes
  skipped, archived steps skipped, all done, and none.
- GREEN: the derivation.
- Then: `MilestonePage` tests for ticking the next step promoting the following one, the
  invitation, the nudge in the slot, and the add composer. Move the nudge out of the grid while
  keeping `Todo.completion-nudge.test.tsx` and `ChecklistGrid.test.tsx` green throughout.
