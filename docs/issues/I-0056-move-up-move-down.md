---
id: I-0056
status: done
implements: FS-0009
blocked_by: [I-0055]
labels: [feature]
title: "FS-0009 slice 3: move an item up or down, in both views, without dragging"
---
Implements FS-0009 §Requirements (R1, R6, R7.2, R9), §Edge States

## What to Build

The client's whole reorder path — the call, the optimistic apply, the revert — driven by two
actions in the item panel rather than by any drag. `ItemActions` is shared by the list and the
blocks view, so this lands reordering in **both** at once, and lands the keyboard path first
rather than as an afterthought.

- `reorderChecklistItems(planId, { scope, parentId, ids })` in `client/src/services/api.ts`,
  wrapping the generated client (I-0055).
- "Move up" and "Move down" in `ItemActions`, each moving the item one place within its sibling
  set. Both absent at the ends of the set — no "Move up" on the first item (R6.2) — and absent
  entirely for a set of one.
- **The request always carries the whole sibling set, including items the type filter hides**
  (R7.2). The client holds every item already, so it can compute the full order; a filtered view
  must never reorder only what it can see, or the hidden items scramble.
- Optimistic: the new order renders immediately (R9.1). On failure, restore the previous order
  **and** raise a toast saying the move could not be saved (R9.2) — a silent revert reads as a
  bug in the control.
- The item's new position is announced politely for assistive tech.
- Nothing is sent when the move is a no-op (an end-of-set move that cannot happen).

Both views read order from the server, so no view-local ordering state is introduced (R1.1).

## Acceptance Criteria

- [ ] "Move up" moves an item one place, and the list re-renders in the new order at once.
- [ ] "Move down" likewise, and the pair round-trips an item back to where it started.
- [ ] "Move up" is absent on the first item of a set; "Move down" on the last; both are absent
      for a set of one.
- [ ] A move made in the list view shows in the blocks view, and the reverse (R1.2).
- [ ] The order survives a remount, read back from the server rather than from local state.
- [ ] Under the Notes filter, moving a note leaves the hidden tasks in their relative order.
- [ ] A failed save restores the previous order and raises a toast.
- [ ] A child's move reorders within its parent only, leaving other families untouched.
- [ ] Client test suite passes.

## Blocked By

I-0055 — there is nothing to call until the endpoint exists.

## Spec Reference

FS-0009 §Requirements R1 (order belongs to the item), R2.3 (idempotent), R6 (the non-mouse
path), R7.2 (the whole set, not the visible set), R9 (optimistic, and honest when it fails);
§User Stories 14, 15, 16, 17, 20, 21; §Edge States (empty / single item).

## TDD Approach

- RED: render a list, use "Move down" on the first item, assert the two items have swapped and
  that the API was called with the full set of ids in the new order.
- GREEN: the reorder call plus the optimistic state update behind the two menu actions.
