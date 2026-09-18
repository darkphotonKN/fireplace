---
id: I-0049
status: open
implements: FS-0008
blocked_by: []
labels: [feature]
title: FS-0008 slice 1: page the list, ten top-level items at a time
---
Implements FS-0008 §Requirements (R1–R12), §Acceptance Criteria "Paging", §Edge States

## What to Build

Bound the checklist's height by paging it in the client (`client/src/components/Todo.tsx`).
No request changes: the component already holds every item.

- Page over **top-level items** (`rowGroups`, after the type filter), never individual rows, so a
  parent and its children always land on the same page. A collapsed parent still counts as one.
- 10 top-level items per page.
- Quiet controls bottom-right, below the list: previous · `1 of 3` · next. Muted by default,
  warming to `text-primary` on hover/focus, matching the header's collapse-all toggle. Not
  rendered at all when everything fits on one page.
- Previous disabled on the first page, next on the last; disabled controls are out of the tab
  order and marked disabled for assistive tech, with "Previous page" / "Next page" labels.
- Changing the type filter, or switching daily / long-term, returns to page 1. A fresh mount
  starts on page 1 (the page is never stored — collapse state still is, per FS-0007 R6).
- If the current page stops existing (items archived, deleted, outdented, filtered away), clamp
  to the new last page rather than showing an empty one.
- Counts, filters, collapse state and the guide rail keep working over the **whole** set.
- Announce page changes politely to assistive tech.

Put the page maths in a pure helper (e.g. extend `client/src/lib/nesting.ts` or a new
`client/src/lib/paging.ts`) so I-0051 and I-0053 can reuse it.

## Acceptance Criteria

- [ ] 25 top-level items render 10 on page 1 with controls reading `1 of 3`.
- [ ] A parent with 8 children counts as one of the ten and renders with its children on one page.
- [ ] 10 or fewer top-level items render no page controls.
- [ ] Previous is disabled on page 1 and next on the last page; neither is keyboard-reachable while disabled.
- [ ] Switching the type filter returns to page 1; switching daily / long-term does too.
- [ ] Remounting shows page 1 while collapse state is still restored.
- [ ] Archiving every item on the last page moves the view to the new last page, never an empty one.
- [ ] Parent counts, filter tabs and collapse state are computed over the full set while one page shows.
- [ ] Page controls carry "Previous page" / "Next page" labels and page changes are announced politely.
- [ ] No network request is made when changing pages.
- [ ] Regression: FS-0007 behaviour holds (rail, chevrons, counts, remembered collapse, Tab-to-nest).
- [ ] Client test suite passes.

## Blocked By

None

## Spec Reference

FS-0008 §Requirements R1–R12; §User Stories 1–4, 15–17, 19–21, 23, 24; §Acceptance Criteria "Paging"; §Edge States (empty list, exactly 10, filter empties the list, page's items archived, family larger than a page, reduced motion, narrow screens).

## TDD Approach

- RED: page helper — 25 groups with size 10 gives pages of 10 / 10 / 5; a group's children never split; page 4 clamps to 3; an empty set gives one page.
- GREEN: pure helper, then wire `renderedRows` to the current page's groups and render the controls.
