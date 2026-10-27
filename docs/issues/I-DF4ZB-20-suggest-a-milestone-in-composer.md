---
id: I-DF4ZB-20
status: open
implements: FS-DF4ZB
blocked_by: [I-DF4ZB-17, I-DF4ZB-19]
labels: [feature]
title: "FS-DF4ZB slice 20: Suggest a milestone from the grid composer (first to cut)"
---
Implements FS-DF4ZB §Requirements (R69, R70)

## What to Build

The grid composer's resting state offers **Suggest a milestone** (S13). **First to cut** (L19).

- Asking calls `suggestItemTree(planId)` and shows the proposal: title, reason and its steps,
  with a loading state and a quiet error.
- **Accept** creates a top-level long-term task with that title, then each step as a child of it,
  through the existing create endpoint, then records `accepted`. If a step create fails, what was
  created stays and the existing error is shown (R70).
- **Dismiss** records `dismissed` and returns the composer to rest.
- The new milestone shows in the strip and grid like any other (FS-FSFEC).

**Owner decision still open: O2** (no cap on item-tree suggestions in v1). The UI must not invent
a limit.

## Acceptance Criteria

- [ ] The grid composer offers "Suggest a milestone" in its resting state.
- [ ] Accepting creates the task and its steps as children, then records `accepted`.
- [ ] A step create failing leaves what was created and shows the existing error.
- [ ] Dismiss records `dismissed`.
- [ ] Tests pass.

## Blocked By

- I-DF4ZB-19 — `suggestItemTree` and its generated client types.
- I-DF4ZB-17 — serializes client edits around the grid and `insights.ts`, following the FS order.

## Expected write set

- `client/src/components/MilestoneComposer.tsx` (+ test, or the existing
  `Todo.milestone-composer*.test.tsx`)
- `client/src/api/insights.ts` (add `suggestItemTree`)
- `client/src/components/Todo.tsx` **(hotspot)**, only if the composer needs a create callback it
  does not already receive

## Spec Reference

FS-DF4ZB §Requirements R69, R70; §Decisions L19; §Spec defaults S13; §Open items O2; §User Stories
17, 18; §Edge States ("Item-tree accept partly fails"); FS-FSFEC (the grid composer).

## TDD Approach

- RED: a composer test that the resting state shows "Suggest a milestone".
- GREEN: the action.
- Then: accept creates the top-level task then children with its id, records `accepted`; a
  failing second step leaves the first two and shows the error; dismiss records `dismissed`.
