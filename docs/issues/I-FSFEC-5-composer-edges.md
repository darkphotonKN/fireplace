---
id: I-FSFEC-5
status: done
implements: FS-FSFEC
blocked_by: [I-FSFEC-4]
labels: [feature]
title: "FS-FSFEC slice 5: the composer under paging, filters, lost targets and view switches"
---
Implements FS-FSFEC §Requirements (R37–R39, R41), §Edge States

## What to Build

The composer keeps its footing when the plan moves under it.

**Paged away while filling (R37, charter default C1).** Filling and the trace survive paging.
Enter adds the step to the traced milestone and the view jumps back to the page holding it, by
the same `pendingJumpId` → `pageOfItem` path FS-0008 R16 uses for nested adds.

**Traced milestone lost (R38, charter default C2).** If the traced milestone is archived,
deleted, converted to a note, nested under another item, or hidden by the type filter, drop to
naming with the typed text kept. Resolve eligibility every render the way `resolveAddBarTarget`
does for the list bar (FS-0007 R8.11); prefer reusing it over a second rule.

**Notes (R39, charter default C3).** Under the Notes filter, or with the type icon set to note,
the composer creates top-level note cards and never enters filling: after Enter it stays in
naming, input cleared, focus kept. Resting reads "+ New note" and naming "Jot down a note…". The
filter sets the type only when chosen and the icon wins afterwards (FS-0008 R19–R21); the icon
is offered in naming only. A task named while the Notes filter hides tasks is created hidden, so
R38 keeps the composer in naming.

**View switch (R41).** Switching to list mid-fill clears the trace; the list's add bar starts at
top level (FS-0007 R8.10) with any typed text kept. Back in grid: naming if text remains,
otherwise resting.

## Acceptance Criteria

- [ ] Filling, then paging to page 1, then Enter adds the step to the traced milestone and jumps
      back to its page with the trace still shown.
- [ ] Archiving the traced milestone drops the composer to naming with text kept.
- [ ] Deleting it does the same.
- [ ] Converting it to a note does the same.
- [ ] Switching to the Notes filter while filling does the same.
- [ ] Under the Notes filter the resting bar reads "+ New note"; Enter creates a top-level note
      card and the composer stays in naming.
- [ ] With the type icon on note under All, the same holds; setting it back to task restores
      milestone creation.
- [ ] Choosing the Notes filter sets the type once; the icon can still override it.
- [ ] No type control is shown while filling.
- [ ] Switching to list mid-fill: the list bar is at top level with the text kept; nothing is
      nested.
- [ ] Switching back to grid opens naming when text remains, resting when it doesn't.
- [ ] Tests pass.

## Blocked By

I-FSFEC-4 — extends its composer, and both edit `Todo.tsx`.

## Spec Reference

FS-FSFEC §Requirements R37–R39, R41 (charter defaults C1, C2, C3); §Edge States (paged away
while filling, traced milestone archived/deleted/converted/nested/filtered out, task named
under the Notes filter, switch to list mid-fill).

## TDD Approach

- RED: a `Todo` test that fills a milestone on page 2, pages to 1, adds a step, and asserts the
  child create's `parentId` and that the page is 2 again.
- GREEN: keep the trace across paging and jump on add.
- Then: one test per way of losing the target; the Notes filter and icon cases; view switch.
