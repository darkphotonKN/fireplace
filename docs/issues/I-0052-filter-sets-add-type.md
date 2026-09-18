---
id: I-0052
status: open
implements: FS-0008
blocked_by: []
labels: [feature]
title: FS-0008 slice 4: the type filter sets what the add bar creates
---
Implements FS-0008 §Requirements (R19–R22), §Acceptance Criteria "Filter sets add type"

## What to Build

Make the All | Notes | Checklist filter (the `listTypeFilter` tabs in
`client/src/components/Todo.tsx`) also decide what the add bar creates, so switching to Notes
and typing produces a note without flipping the icon.

- Choosing **Notes** sets `newTodoType` to `note`; choosing **Checklist** sets it to `task`.
- Choosing **All** leaves the current type untouched.
- The filter sets the type only at the moment it is chosen: the add bar's type icon stays
  clickable and a manual override afterwards sticks.
- Placeholder, `aria-label` and the icon follow the resulting type, as they already do.

## Acceptance Criteria

- [ ] Choosing Notes sets the add bar to note and the placeholder follows.
- [ ] Choosing Checklist sets it to task.
- [ ] Choosing All leaves the current type unchanged.
- [ ] Clicking the type icon after a filter set the type overrides it, and the override sticks.
- [ ] Re-choosing the same filter does not fight a manual override made since.
- [ ] Regression: filtering still filters the list, and the add bar's other behaviour is unchanged.
- [ ] Client test suite passes.

## Blocked By

None

## Spec Reference

FS-0008 §Requirements R19–R22; §User Stories 11–14; §Acceptance Criteria "Filter sets add type".

## TDD Approach

- RED: clicking Notes makes the add input's placeholder the note one; clicking the type icon afterwards flips it back to task and it stays.
- GREEN: set `newTodoType` from the filter handler for the two typed filters only.
