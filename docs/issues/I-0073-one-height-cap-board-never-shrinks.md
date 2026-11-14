---
id: I-0073
status: done
implements: FS-0008
blocked_by: [I-0072]
labels: [enhancement]
title: "FS-0008 §Change slice 2: one height cap, so the board never shrinks when page 2 appears"
---
Implements FS-0008 §Change: taller board, later paging (R27, R28, R29, R31, R32)

## What to Build

The list area in `Todo.tsx` (the `totalPages > 1 ? 'h-[min(70vh,40rem)]…' : 'max-h-[70vh]…'`
block) uses one cap, `min(80vh, 60rem)`, for both states, in list and grid alike. One page hugs
its content up to the cap and scrolls inside past it, with the add bar pinned. From two pages on,
the area is fixed at exactly the cap on every page, including a short last page. The space under
a short page stays empty, and the add bar and the controls stay where they are. When the set drops
back to one page (archive, delete, filter), the area hugs again. The archived view (no list area)
and the daily plan (its own section) are not touched.

The block's explanatory comment is rewritten. Its "a full page of ten rows plus the bar is about
40rem" rationale no longer holds.

## Acceptance Criteria

- [ ] One page: the list area carries `max-h-[min(80vh,60rem)]` with `overflow-y-auto`, and no fixed height, so a short list hugs its rows.
- [ ] Two or more pages: the list area carries `h-[min(80vh,60rem)]` on page 1, on a middle page and on a short last page.
- [ ] The single-page bound and the paged height are the same `min(80vh,60rem)` expression, in list and grid alike, so adding the item that opens page 2 never makes the board shorter.
- [ ] Taking a 21-item list back to 20 by archiving, deleting or filtering removes the controls, and the area goes back to `max-h-[min(80vh,60rem)]`.
- [ ] The pinned add bar still never covers the last row (R14), and the controls stay bottom-right on every page (R5).
- [ ] No `70vh` or `40rem` bound is left on the list area.
- [ ] The archived view and the daily plan are unchanged: same markup, same classes.
- [ ] Client test suite passes.

## Blocked By

I-0072. Its new page sizes decide how many items a test needs to reach two pages. Building on
them avoids sizing the fixtures twice.

## Files expected to touch

- `client/src/components/Todo.tsx`: the list-area `className` block and the comment above it only
- `client/src/components/Todo.pinned-add-bar.test.tsx`: the `h-[min(70vh,40rem)]` / `max-h-[70vh]` assertions become the new cap. Add a grid case, and a case that drops back to one page.

## Spec Reference

FS-0008 §Change: taller board, later paging: R27 (one cap), R28 (one page hugs), R29 (paged
area fixed at the cap on every page), R31 (back to hugging), R32 (archived and daily unchanged);
the height criteria under its Acceptance Criteria; Edge States "21st item added", "Short last
page", "Short screen", "Tall screen", "Narrow screen, grid in one column", "Collapse all on a
paged board".

## TDD Approach

- RED: in `Todo.pinned-add-bar.test.tsx`, a 4-item list's area matches `max-h-[min(80vh,60rem)]`. A 24-item list's area matches `h-[min(80vh,60rem)]` on page 1 and on page 2.
- GREEN: change the two class strings in the list-area block. Then add the grid and drop-back-to-one-page cases.
