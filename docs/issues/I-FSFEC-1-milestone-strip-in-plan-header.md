---
id: I-FSFEC-1
status: open
implements: FS-FSFEC
blocked_by: []
labels: [feature]
title: "FS-FSFEC slice 1: milestones, the plan-header progress strip, and the toggle's new names"
---
Implements FS-FSFEC §Requirements (R1–R9, R15–R22), §Edge States

## What to Build

The plan learns what a milestone is and shows its progress across them, in both views. Nothing
about the composer or the nudge yet.

**The derivation (R1, R5–R9).** One pure module, `client/src/lib/milestones.ts`, that takes a
list's items and returns its milestones in list order, each with its step progress and whether
it is closed, plus the plan summary (closed, total). A milestone is a non-archived, top-level,
long-term task (R1). Note children never count (R6). A milestone with no task steps reads empty
until done and full after (R7). A closed milestone reads full whatever its steps say (R8).
Notes at top level are not milestones (R5). This module is the single derivation every later
slice and the dashboard read, so keep its surface small: items in, milestones and a summary out.

**The strip (R15–R22).** `client/src/components/MilestoneStrip.tsx`: headline "N of M
milestones" (N = closed) and one segment per milestone, each filled to its own progress, equal
width, no cap, gaps tightening past 12, one line at any count (R18). The headline is the text
equivalent; each segment titles its milestone's name and progress (R21). Takes milestones in,
renders nothing for zero (R19). Tokens only; its colour comes from props or a variant so the
dashboard can draw it neutral later (R28).

**Wiring in `Todo.tsx`.** On the long-term tab only (not daily, archived or settings, R22), in
**both** views, the items section opens with the strip. Feed it from `todos`, the whole list,
not `rowGroups`/`pagedGroups`: it ignores paging, the type filter and collapse (R16). Because it
reads the same state the items do, an optimistic tick moves a segment at once and a revert moves
it back (R20).

**The toggle's names (R3).** "Card view" / "List view" become "Milestones" / "Checklist" as
accessible name and title. Put the pair in a group named "View" so it stays distinguishable from
the type filter's "Checklist" tab. Update `Todo.view-toggle.test.tsx` queries accordingly.

`Todo.tsx` is the hotspot: keep the edit to the header region and the toggle; later slices touch
the add bar.

## Acceptance Criteria

- [ ] `milestones.ts` returns top-level, non-archived, long-term tasks only, in list order.
- [ ] A milestone at 3/4 task steps reads 0.75; note steps change nothing.
- [ ] A milestone with no task steps (none, or only notes) reads 0 until done and 1 after.
- [ ] A closed milestone reads 1 and counts toward N.
- [ ] The long-term tab shows "N of M milestones" and the strip in grid view **and** list view.
- [ ] The strip ignores the page, the type filter and collapsed state.
- [ ] A plan with no milestones (empty, or only notes) shows no strip and no headline.
- [ ] At 30 milestones the strip stays on one line with no horizontal scroll; gaps at 13+ are
      tighter than at 12.
- [ ] Ticking a step updates its segment before the request resolves; a rejected tick reverts it.
- [ ] No strip on daily, archived or settings.
- [ ] The toggle's buttons are named "Milestones" and "Checklist", inside a group named "View";
      "Card view" appears nowhere.
- [ ] No API call or request body changes.
- [ ] Tests pass.

## Blocked By

None.

## Spec Reference

FS-FSFEC §Requirements R1–R9 (what a milestone is and how its progress reads), R15–R22 (the
plan-header strip), R3 (toggle labels, charter default C11); §Edge States (only notes,
all-notes children, closed early, 30+ milestones); charter defaults C5, C9, C11.

## TDD Approach

- RED: a table test for `milestones.ts` over a mixed list (top-level notes, archived items,
  daily items, all-notes children, a closed milestone with undone steps) asserting the milestone
  set, each progress and the summary.
- GREEN: the derivation.
- Then: `MilestoneStrip` renders M segments with the right fills and nothing at zero; a `Todo`
  test on a paged, filtered long-term list asserts the headline counts the whole list in both
  views; the toggle names.
