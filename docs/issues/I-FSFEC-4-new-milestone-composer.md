---
id: I-FSFEC-4
status: done
implements: FS-FSFEC
blocked_by: [I-FSFEC-1, I-FSFEC-3]
labels: [feature]
title: "FS-FSFEC slice 4: the new-milestone composer — rest, name, fill, with a traced card"
---
Implements FS-FSFEC §Requirements (R29–R36, R40, R42–R45), §Edge States

## What to Build

In grid view the pinned bottom bar stops being the list's add bar and becomes the composer that
names a new milestone and then fills it with steps. The happy path and its keys; the edges that
pull the trace away come in slice 5.

**Its own component.** Build the state machine in a new `client/src/components/MilestoneComposer.tsx`
(resting → naming → filling) and have `Todo.tsx` render it in place of the add bar's form only
when `isGrid`. Keep the composer's logic out of `Todo.tsx`, which is 2,500 lines already: Todo
hands it the create calls and the paging jump, and it owns its state. The list's add bar,
`handleAddBarKeyDown`, `findAddBarTarget` and Tab-to-nest are untouched (R29).

**Resting (R30).** A quiet "+ New milestone" button. Click, Enter or Space opens naming with
focus. No global shortcut.

**Naming (R31).** Placeholder "Name a new milestone…". Enter with text creates a top-level task
through the existing create path, then moves to filling with that milestone traced. The view
jumps to its page by the existing `pendingJumpId` (FS-0008 R17). Empty Enter does nothing; a
failed create stays in naming with the text kept and the existing error.

**Filling (R32, R34).** Label "↳ steps for ‹title›", placeholder "Add a step…". Enter with text
creates a task step under the traced milestone (reuse `addChildTodo`, which already opens a
collapsed parent), stays in filling, clears the input, keeps focus. No type control while
filling.

**The trace (R33).** `ChecklistGrid` takes a `tracedId` and marks that card (ring and ✦,
plus a `data-traced` hook for tests). The input's accessible label is "Add a step to ‹title›".
Only the milestone the composer just created is ever traced.

**Keys (R35, R36).** In filling, Tab and Shift+Tab go to naming with text kept and the trace
cleared. In naming both are swallowed: no change, focus stays. Esc from naming or filling goes
to resting, clears the trace, discards the text.

**Quiet edges (R40, R42–R45).** Blur keeps the state and the trace. State is never persisted;
every load starts resting. Enter during an in-flight create is ignored with focus kept
(read-only, not disabled). The FS-0007 "Tab to nest" tip never fires from the composer. A card's
own "+ add step" keeps working and does not touch the composer.

## Acceptance Criteria

- [ ] Grid view shows "+ New milestone" at rest; list view shows today's add bar unchanged.
- [ ] Click, Enter or Space on the resting bar opens naming with the input focused.
- [ ] Naming + Enter creates a top-level task, jumps to its page, and enters filling with that
      card traced and named in the input.
- [ ] Empty Enter in naming does nothing; a failed create keeps naming and the text.
- [ ] Filling + Enter adds a task step to the traced card; the input clears and keeps focus.
- [ ] A step added to a collapsed traced card opens it.
- [ ] Tab and Shift+Tab in filling go to naming with the text kept and no card traced.
- [ ] Tab and Shift+Tab in naming change nothing and focus stays in the input.
- [ ] Esc from naming or filling goes to resting with no card traced.
- [ ] Clicking a card or ticking a step while filling keeps filling and the trace.
- [ ] Reloading mid-fill starts at resting.
- [ ] Enter during an in-flight create is ignored and focus stays.
- [ ] The "Tab to nest" tip never appears from the grid composer.
- [ ] A card's "+ add step" still adds to that card and leaves the composer's state alone.
- [ ] Existing list add-bar tests (`Todo.add-bar-nest`, `Todo.add-page-jump`,
      `Todo.pinned-add-bar`, `Todo.filter-add-type`) still pass in list view.
- [ ] Tests pass.

## Blocked By

I-FSFEC-1 and I-FSFEC-3. `Todo.tsx` is the hotspot slice 1 edits, and `ChecklistGrid.tsx` is
the file slice 3 edits; this slice touches both, so it runs after them, not alongside.

## Spec Reference

FS-FSFEC §Requirements R29–R36, R40, R42–R45 (the composer, charter defaults C4, C6, C7);
§Edge States (new milestone on another page, traced milestone collapsed, create fails, Enter in
flight, reload mid-fill); §Decisions D4–D6.

## TDD Approach

- RED: a `Todo` test in grid view that opens the composer, types a name and Enter, then a step
  and Enter, and asserts one top-level create, one child create with that `parentId`, and the
  new card carrying `data-traced`.
- GREEN: `MilestoneComposer` and the wiring.
- Then: Tab/Shift+Tab/Esc transitions, blur keeps state, list view untouched.
