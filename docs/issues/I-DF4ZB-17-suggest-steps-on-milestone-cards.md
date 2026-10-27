---
id: I-DF4ZB-17
status: open
implements: FS-DF4ZB
blocked_by: [I-DF4ZB-16]
labels: [feature]
title: "FS-DF4ZB slice 17: Suggest steps on open milestone cards in the grid"
---
Implements FS-DF4ZB §Requirements (R66–R68)

## What to Build

In the grid (the milestone reading), every **open** milestone card offers **Suggest steps**.
Closed milestones do not (S12). The list view is unchanged.

- Asking calls `suggestSteps(planId, parentItemId)` and shows the suggestions under the card,
  with a loading state and a quiet error.
- Each suggestion has **Add** (create a long-term task as a child of that milestone through the
  existing create endpoint, then record `accepted` via `recordSuggestionFeedback`) and dismiss
  (record `dismissed`, hide it).
- Adding one leaves the rest in place. The panel closes when the user closes it or the list is
  empty.
- Closing the panel records nothing (R68); only an explicit dismiss feeds the filter.
- A failed create keeps the suggestion and shows the existing error; no `accepted` is recorded.
- An added step counts toward the milestone's progress strip (FS-FSFEC) because it is an ordinary
  child.

**Owner decision still open: O2** (no cap on step suggestions in v1). The UI must not invent a
limit or a "remaining" counter.

## Acceptance Criteria

- [ ] Open milestone cards offer "Suggest steps"; closed ones do not; the list view has no such
      action.
- [ ] Added steps land as children of that milestone and the rest stay listed.
- [ ] Add records `accepted`; dismiss hides and records `dismissed`; closing records nothing.
- [ ] A failed create keeps the suggestion, shows the existing error, records nothing.
- [ ] The panel closes when emptied or closed.
- [ ] Tests pass.

## Blocked By

I-DF4ZB-16 — `suggestSteps` and its generated client types.

## Expected write set

- `client/src/components/SuggestedSteps.tsx` (new) + test
- `client/src/components/ChecklistGrid.tsx` (+ `ChecklistGrid.test.tsx`)
- `client/src/api/insights.ts` (add `suggestSteps`)
- `client/src/components/Todo.tsx` **(hotspot)**, only if the grid needs a create-child handler
  threaded through that it does not already receive; prefer reusing what `ChecklistGrid` gets

## Spec Reference

FS-DF4ZB §Requirements R66–R68; §Decisions L2, L3, L18; §Spec defaults S12; §Open items O2; §User
Stories 15, 16, 18; §Edge States ("Accept create fails"); §Out of Scope ("Suggested steps in the
list view").

## TDD Approach

- RED: a `ChecklistGrid` test that an open milestone card renders "Suggest steps" and a closed one
  does not.
- GREEN: the action.
- Then: the panel: add creates a child with the parent id and records `accepted`; dismiss records
  `dismissed`; close records nothing; empty closes.
