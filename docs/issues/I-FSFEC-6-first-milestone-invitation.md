---
id: I-FSFEC-6
status: open
implements: FS-FSFEC
blocked_by: [I-FSFEC-5]
labels: [feature]
title: "FS-FSFEC slice 6: an empty plan invites its first milestone"
---
Implements FS-FSFEC §Requirements (R46, R47), §Edge States

## What to Build

An empty long-term plan in grid view stops looking broken.

**The invitation (R46).** When the long-term list holds no items at all and the filter is All
or Checklist, the grid shows a single "Start your first milestone" invitation in place of the
"No items yet. Add one below!" text, and the resting bar is not drawn beneath it. Activating it
(click, Enter or Space) opens the composer in naming with focus. Once a milestone exists, the
normal grid and resting bar return.

**Filter-empty stays as it is (R47).** When items exist but the filter hides them all, keep
today's "No notes yet." / "No tasks yet." with the composer below. Under the Notes filter on an
empty plan, the composer is in note mode (slice 5), so it shows today's notes-empty text, not
the milestone invitation.

List view's empty state is unchanged.

## Acceptance Criteria

- [ ] An empty long-term plan in grid view shows "Start your first milestone" and no resting bar.
- [ ] Activating the invitation opens naming with the input focused.
- [ ] Naming the first milestone replaces the invitation with the grid; the composer is filling.
- [ ] Items present but all hidden by the filter: today's filter-empty text, composer below.
- [ ] Empty plan under the Notes filter: "No notes yet." with the composer in note mode.
- [ ] List view's empty state is unchanged.
- [ ] Tests pass.

## Blocked By

I-FSFEC-5 — opens the composer slice 4 built and leans on slice 5's note mode; edits the empty
branch of `Todo.tsx`, so it is sequenced after the other `Todo.tsx` slices.

## Spec Reference

FS-FSFEC §Requirements R46, R47; §Edge States (empty long-term list, filter hides all);
§Decisions D5 (the empty-plan exception).

## TDD Approach

- RED: a `Todo` test on an empty long-term list in grid view asserts the invitation is shown and
  the resting bar is not, then activates it and asserts the naming input has focus.
- GREEN: the empty branch for grid.
- Then: filter-empty and Notes-filter cases; list view unchanged.
