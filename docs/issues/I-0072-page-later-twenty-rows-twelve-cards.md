---
id: I-0072
status: done
implements: FS-0008
blocked_by: []
labels: [enhancement]
title: "FS-0008 §Change slice 1: page later, twenty rows and twelve cards"
---
Implements FS-0008 §Change: taller board, later paging (R25, R26, R30, R32)

## What to Build

Paging starts at twice today's size. A list page holds 20 top-level items (`PAGE_SIZE` in
`client/src/lib/paging.ts`, was 10) and a grid page holds 12 cards (`GRID_PAGE_SIZE` in
`client/src/components/Todo.tsx`, was 6). Paging stays count-based: the page unit is still the
top-level group, a family is never split, and a collapsed parent still counts as one. The page
size never depends on rendered height or the viewport. The archived view and the daily plan have
no paging and are not touched.

This slice changes only the counts. The list area's height classes are slice 2 (I-0073).

## Acceptance Criteria

- [ ] `PAGE_SIZE` is 20 and `GRID_PAGE_SIZE` is 12. The comments next to both constants give the new reasons (FS-0008 R25, R26).
- [ ] List view: 25 top-level items show 20 on page 1, reading `1 of 2`. Page 2 shows the other 5.
- [ ] List view: exactly 20 top-level items show no page controls; 21 show `1 of 2`.
- [ ] List view: a parent with 8 children counts as one of the twenty, and its children render on the parent's page.
- [ ] Grid view: 13 top-level items show 12 cards on page 1, reading `1 of 2`. Exactly 12 show no controls.
- [ ] Changing the viewport size does not change which items sit on which page.
- [ ] FS-0008 R5–R18 hold at the new sizes: quiet bottom-right controls, Tab-to-nest on the current page, the page jump on add (including a new milestone and a duplicate jumping to their page), clamping, and the page-1 reset on a filter or view change.
- [ ] The archived view and the daily plan are unchanged.
- [ ] Fixtures that relied on 10-row or 6-card pages are resized so they still cover the same behaviour. Client test suite passes.

## Blocked By

None.

**Coordination, not a code dependency:** two of the test files below pin 6-card grid pages:
`Todo.milestone-composer.test.tsx` (12 cards read `1 of 2`, 13 read `3 of 3`) and
`Todo.milestone-composer-edges.test.tsx` (7 cards span two pages). Both fail at
`GRID_PAGE_SIZE = 12`. Both currently carry uncommitted edits from other work in this worktree.
Pick this slice up only after that work lands or hands the files back.

## Files expected to touch

- `client/src/lib/paging.ts`: `PAGE_SIZE` and its comment
- `client/src/lib/paging.test.ts`: the 25 / 10 / 8-children cases, moved to the new size
- `client/src/components/Todo.tsx`: `GRID_PAGE_SIZE` and its comment only
- Fixtures sized to the old pages:
  - `client/src/components/Todo.paging.test.tsx` (25 rows → `1 of 3`)
  - `client/src/components/Todo.add-page-jump.test.tsx` (`… of 3`)
  - `client/src/components/Todo.duplicate.test.tsx` (`… of 3`, list and grid)
  - `client/src/components/Todo.collapse-all-paging.test.tsx` (12 rows over two pages of ten)
  - `client/src/components/Todo.drag.test.tsx` (13 rows, then Next page)
  - `client/src/components/Todo.view-toggle.test.tsx` (7 cards → `1 of 2`)
  - `client/src/components/Todo.milestone-strip.test.tsx` (8 cards + a note → `1 of 2`)
  - `client/src/components/Todo.milestone-composer.test.tsx` (see Coordination)
  - `client/src/components/Todo.milestone-composer-edges.test.tsx` (see Coordination)

## Spec Reference

FS-0008 §Change: taller board, later paging: R25 (list 20), R26 (grid 12), R30 (count-based),
R32 (long-term list and grid only); the count and viewport criteria under its Acceptance
Criteria; Edge States "Exactly 20 rows / 12 cards", "A single family taller than the cap".

## TDD Approach

- RED: `paging.test.ts`: `pageCount(25)` is 2, `pageCount(PAGE_SIZE)` is 1, and `pageSlice` cuts 20 groups per page. In `Todo.paging.test.tsx`, 25 rows read `1 of 2` with 20 on page 1.
- GREEN: set `PAGE_SIZE = 20`. Then a grid case (13 cards read `1 of 2`, 12 show no controls) goes RED, and `GRID_PAGE_SIZE = 12` turns it green. Resize the remaining fixtures.
