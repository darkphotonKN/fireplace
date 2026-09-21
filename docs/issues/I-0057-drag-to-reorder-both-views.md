---
id: I-0057
status: open
implements: FS-0009
blocked_by: [I-0056]
labels: [feature]
title: "FS-0009 slice 4: drag an item into place, in the list and in the blocks"
---
Implements FS-0009 §Requirements (R3, R4, R5, R7, R8), §Edge States

## What to Build

Dragging, in both views, over the write path I-0056 already built and proved. Nothing about the
request changes here — this slice is the gesture.

**Use `@dnd-kit` (`@dnd-kit/core` + `@dnd-kit/sortable`), a new client dependency, chosen for
this slice.** It is pointer-based rather than HTML5 drag-and-drop: it works on touch, it is
keyboard-operable, and its events can be driven in tests, where native DnD cannot be — jsdom
does not implement it.

List view (`Todo.tsx`):

- A row can be picked up and dropped between any two rows of its own sibling set (R3.1).
- A top-level row carries its children, collapsed or not (R3.2).
- While dragging, the place the item would land is shown, and the dragged row is visibly lifted
  out of the flow (R3.3).
- A drop where the drag began sends nothing (R3.4). Escape abandons and restores (R3.5).

Blocks view (`ChecklistGrid.tsx`):

- A card can be dropped into any position among the page's cards (R4.1); a step drags within its
  own card (R4.2).
- The drop target reads as the gap the card will occupy, not as a highlight on its neighbour
  (R4.3).
- R3.4 and R3.5 hold identically (R4.4).

Rules both views obey:

- **A drag never re-parents** (R5.1). A drop outside the item's own sibling set is refused: the
  item returns to where it started and nothing is written. Nesting keeps Tab / Shift+Tab and the
  panel's "Nest under above" / "Move out".
- Dragging stays available under a type filter, and still writes the whole sibling set (R7).
- A drag reorders within the page it happens on; items on other pages keep their places, and
  dragging past a page edge does not change page (R8).
- Press-and-hold starts a drag on touch; a scroll does not (§Edge States).
- A view-only user on a shared plan gets no drag handles at all.

## Acceptance Criteria

- [ ] A row dragged to a new position in the list holds that position after a reload.
- [ ] A card dragged in the blocks view holds its position, and shows in the list view (R1.2).
- [ ] A dragged top-level item takes its children with it, collapsed or not.
- [ ] A drop that would change an item's parent is refused and writes nothing.
- [ ] A drop in the same place makes no request.
- [ ] Escape mid-drag restores the order the view had before the drag.
- [ ] Reordering by drag under the Notes filter leaves hidden tasks in their relative order.
- [ ] A drag on page 2 leaves page 1's order untouched.
- [ ] The keyboard path from I-0056 still works and goes through the same write.
- [ ] Client test suite passes.
- [ ] HITL: the lift, the drop indicator and the settle are checked in a browser, in both views,
      in both themes, and on a narrow window.

## Blocked By

I-0056 — the write path, the optimistic apply and the revert all come from there. This slice
adds no request of its own.

## Spec Reference

FS-0009 §Requirements R3 (list), R4 (blocks), R5 (never re-parents), R7 (filters), R8 (paging);
§User Stories 1–13, 18, 19, 25; §Edge States (item deleted mid-drag, an item added while
dragging, a page holding one item, touch, unauthorized).

## TDD Approach

- RED: drive dnd-kit's sensors to drag row 3 above row 1 and assert the rendered order, and that
  the API was called with the full set in that order.
- GREEN: the sortable context and the drop handler that maps a drop to the same reorder call.

Note for whoever picks this up: the *behaviour* here is testable and must be tested, but the
*feel* is not — jsdom computes no layout and paints nothing. Budget a browser pass, and do not
report the slice done on green tests alone.
