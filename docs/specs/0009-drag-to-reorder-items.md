# FS-0009: Drag to reorder checklist items

> Status: work-order · SPECIFICATION.md: client `## Checklists` entry → this FS, plan-service `### Checklist Items` entry → this FS · Related ADRs: docs/adr/0002-contract-planes-code-first-openapi.md, docs/adr/0005-request-validation-and-contract-design.md

## Summary

A checklist has an order its owner chose, and they choose it by dragging. Items can be
dragged into position in the list view and in the blocks view, and because the order lives on
the item rather than in either view, moving something in one moves it in the other. Today the
order is the order things were created in and nothing can change it.

## Background: the field already exists, and nothing can write it

`sequence` is already carried end to end — `checklist_items.sequence` (int) in plan-service,
`ChecklistItem.sequence` in the proto, `ChecklistResp.sequence` (required, string) in the
gateway's OpenAPI document, and therefore in the generated TS client. Both list queries
already `ORDER BY sequence ASC`, so what the client renders is already sequence order.

Two things are missing, and only the first is this feature:

1. **No write path.** Neither `UpdateChecklistReq` (HTTP) nor `UpdateItemRequest` (gRPC)
   carries `sequence`. Order is create-order, permanently.
2. **The value is assigned from a global count.** `service.Create` calls
   `repo.CountItems()`, which is `SELECT COUNT(id) FROM checklist_items` across every plan
   and every user, and stores `count+1`. Within one plan that still ascends, so ordering
   works, but the numbers are global, gappy, and racy: two creates in flight read the same
   count and store the same sequence. See §Out of Scope.

## Requirements

**R1 — The order is a property of the item, not of a view.**
R1.1 An item's position is its `sequence`, read from the server, never held only in a view.
R1.2 Both views render the same order; a move in one is visible in the other with no reload.
R1.3 Order survives a reload, a re-login, and a different device.

**R2 — Order is written for a whole sibling set at once.**
R2.1 One request carries the complete, final order of one sibling set: the top-level items of
a (plan, scope), or the children of one parent.
R2.2 The write is atomic — all of the set's sequences change, or none do.
R2.3 The write is idempotent: sending the order a set already has changes nothing and errors
on nothing.
R2.4 Per-item sequence writes are NOT added; `UpdateChecklistReq` gains no `sequence` field.
A drag is one move of one set, and N single-item writes would race each other.
R2.5 The server assigns dense positions (1..N) across the set in the order given, and returns
the reordered siblings so the client reconciles against what was stored rather than what it
hoped for.

**R3 — Dragging in the list view.**
R3.1 A row can be picked up and dropped between any two rows of its own sibling set.
R3.2 A top-level row carries its children with it, collapsed or not.
R3.3 While dragging, the place the item would land is shown, and the row being dragged is
visibly lifted out of the flow.
R3.4 A drag that ends where it began is a no-op and sends nothing.
R3.5 Escape during a drag abandons it and restores the original order.

**R4 — Dragging in the blocks view.**
R4.1 A card can be picked up and dropped into any position among the cards on the page.
R4.2 A step can be dragged within its own card.
R4.3 The drop target is shown as the gap the card will occupy, not as a highlight on a
neighbour.
R4.4 R3.4 and R3.5 apply identically.

**R5 — A drag never re-parents.**
R5.1 A drop outside the dragged item's own sibling set is refused: the item returns to where
it started and nothing is written.
R5.2 Nesting keeps its existing affordances — Tab / Shift+Tab, and the actions panel's "Nest
under above" / "Move out" (FS-0007).

**R6 — There is a way to reorder without a mouse.**
R6.1 The actions panel offers "Move up" and "Move down", each moving the item one place
within its sibling set.
R6.2 Both are absent at the ends of the set — no "Move up" on the first item.
R6.3 Each uses the same write as a drag, so the two paths cannot diverge.

**R7 — A type filter does not scramble what it hides.**
R7.1 Dragging stays available while the Notes or Checklist filter is on.
R7.2 The client sends the whole sibling set, including items the filter hides, so hidden
items keep their relative positions.
R7.3 A dragged item lands immediately before the visible item it was dropped above; items the
filter hides do not move around it.

**R8 — Paging (FS-0008) bounds a drag, it does not break it.**
R8.1 A drag reorders within the page it happens on.
R8.2 The sibling set sent is still the whole set, so items on other pages keep their places.
R8.3 Dragging past the top or bottom edge of a page does not change page — see §Out of Scope.

**R9 — The move is shown before it is saved, and taken back if it fails.**
R9.1 The new order renders immediately on drop.
R9.2 On failure the previous order is restored and a toast says the move could not be saved.
R9.3 Nothing about the failure is silent — a reverted move without a message reads as a bug in
the drag.

**R10 — Order is defined even when sequences collide.**
R10.1 Items with equal `sequence` are ordered deterministically by a documented tiebreak, so a
list never flaps between renders. (Today's global counter can produce ties; see §Out of Scope.)

## User Stories

1. As someone planning a project, I want to drag a milestone to the top, so that what matters
   most is what I see first.
2. As someone planning a project, I want the order I chose to still be there tomorrow, so that
   arranging my plan is worth the effort.
3. As a learner, I want to order steps the way I will actually do them, so that the list reads
   as a path rather than a pile.
4. As someone who added things out of order, I want to fix the order afterwards, so that I can
   capture quickly and tidy later.
5. As someone using the blocks view, I want to drag a card into position, so that the grid
   reflects my priorities.
6. As someone who switches views, I want the order I set in blocks to be the order in the list,
   so that I do not have to arrange my plan twice.
7. As someone who switches views, I want the reverse too — an order set in the list showing in
   the blocks — so that neither view is the "real" one.
8. As someone dragging a parent, I want its steps to come with it, so that a family stays
   together.
9. As someone dragging a collapsed parent, I want the same, so that folding is about reading,
   not about structure.
10. As someone dragging a step, I want it to move within its own block, so that a drag cannot
    accidentally restructure my plan.
11. As someone who drops a step somewhere it cannot go, I want it to return visibly, so that I
    know nothing happened.
12. As someone who started a drag by accident, I want Escape to abandon it, so that I am never
    committed by a twitch.
13. As someone who dropped an item back where it started, I want nothing to be saved, so that
    the list does not flicker for a move I did not make.
14. As a keyboard user, I want "Move up" and "Move down" in the actions panel, so that ordering
    is not mouse-only.
15. As a screen-reader user, I want an item's move announced, so that I know it landed.
16. As someone filtering to Notes, I want to reorder what I can see, so that a filter does not
    take a capability away.
17. As someone filtering to Notes, I want my tasks to keep their places, so that a filtered
    drag does not scramble what is hidden.
18. As someone with a long list, I want to reorder within the page I am on, so that paging and
    dragging do not fight.
19. As someone with a long list, I want items on other pages left alone, so that a local move
    stays local.
20. As someone who moved an item, I want to see it move at once, so that the drag feels
    finished when I let go.
21. As someone whose network dropped, I want the move taken back and to be told, so that I do
    not trust an order that was never saved.
22. As someone with two tabs open, I want the second to show the saved order when it next
    loads, so that the order is the server's, not a tab's.
23. As a plan owner, I want reordering to obey the same permissions as any other edit, so that
    a viewer cannot rearrange my plan.
24. As someone who just added an item, I want it at the end, so that adding stays predictable
    after I have reordered things.
25. As someone dragging in a narrow window, I want the same behaviour as a wide one, so that
    the feature is not desktop-only.

## Acceptance Criteria

- [ ] An item dragged to a new position in the list is in that position after a reload.
- [ ] An item dragged in the list view is in the same position when the blocks view is opened,
      and the reverse.
- [ ] A top-level item dragged in either view takes its children with it, collapsed or not.
- [ ] A drop that would change an item's parent is refused, and nothing is written.
- [ ] A drop in the same place writes nothing (no request is made).
- [ ] Escape mid-drag restores the order the list had before the drag.
- [ ] "Move up" / "Move down" in the actions panel move an item one place, and are absent at
      the ends of the set.
- [ ] Reordering under the Notes filter leaves the hidden tasks in their relative order.
- [ ] Reordering on page 2 leaves page 1's order untouched.
- [ ] A failed save restores the previous order and raises a toast.
- [ ] Two items with the same `sequence` render in a stable, documented order across reloads.
- [ ] The reorder endpoint is idempotent: sending the current order twice changes nothing.
- [ ] Contract gates pass, and the change is additive (no breaking finding from oasdiff).
- [ ] Client test suite passes.

## Edge States

- **Empty / single item.** A set of zero or one item has nothing to reorder; no drag handles,
  no Move up / Move down.
- **Concurrent reorder.** Two clients reorder the same set. Last write wins over the whole
  set; the loser sees the winner's order on its next load. No merge is attempted.
- **Item deleted mid-drag.** The dragged item is deleted in another tab before the drop lands.
  Its id now exists nowhere, so the write fails `404 · NOT_FOUND`, the order reverts, and the
  list refetches.
- **Item archived mid-drag.** The same gesture, a different answer: an archived item still
  exists, it has just left the set (archived rows are never siblings). The ids are therefore no
  longer a permutation of the set and the write fails `400 · VALIDATION_FAILED`. The two
  statuses answer different questions — *does this id exist at all* versus *does it belong to
  this set* — and a client that cannot tell them apart cannot tell a vanished item from a
  misaddressed request.
- **Ids that are not the sibling set.** A request whose ids are not exactly a permutation of
  the set — missing one, containing a stranger, or mixing two parents — is refused whole.
- **An item added while dragging.** The new item is appended and is not part of the set the
  client sent; the write is refused as above and the client refetches.
- **Filter hides the drop neighbour.** Covered by R7.3: the landing place is defined against
  the visible item dropped above, not against the hidden ones around it.
- **A page holding one item.** Nothing to reorder on that page; the drag is available but has
  no target.
- **Archived view.** Not reorderable — see §Out of Scope.
- **Unauthorized.** A user with view-only access to a shared plan gets `403 · FORBIDDEN`, and
  the client does not offer drag handles at all. **Declared, not yet reachable** — see
  §Out of Scope.
- **Touch.** A press-and-hold starts a drag; a scroll does not.

## API surface

One new operation. Additive — no existing path, schema, or field changes, so the breaking
gate stays green.

| Op | Method + Path | Query/Params | Request body | Response | Errors |
|----|---------------|--------------|--------------|----------|--------|
| Reorder a sibling set | `PATCH /api/plans/{id}/checklists/order` | `id` (path, uuid, plan) | `ReorderChecklistReq`: `scope` (`daily`\|`longterm`, required), `parentId` (uuid or null — null means the top-level set, required), `ids` (uuid[], required, non-empty, unique, exactly the set's members in their new order) | `200` — `ChecklistResp[]`, the reordered siblings in their new order | plan or an id unknown: `404 · NOT_FOUND` · ids are not a permutation of the sibling set, or mix parents/scopes: `400 · VALIDATION_FAILED` · unknown body member: `422 · VALIDATION_FAILED` · no token: `401 · UNAUTHENTICATED` · view-only on a shared plan: `403 · FORBIDDEN` |

Downstream, plan-service gains `rpc ReorderItems(ReorderItemsRequest) returns (ListItemsResponse)`,
carrying the same four fields plus `user_id`, and writes the set's sequences in one
transaction.

## Out of Scope

- **Fixing the global sequence counter.** `Create` will keep using `CountItems()+1` over the
  whole table. Dense renumbering within a plan does not collide with it — a new item's number
  is far larger than any renumbered set, so new items still land last. **Trigger to revisit:**
  the first duplicate-sequence bug report, or the first time a plan's items need numbers that
  mean something on their own. R10's tiebreak is what keeps that latent bug from being visible
  as a flapping list.
- **Dragging across parents.** A drag that would re-parent is refused (R5). Nesting stays with
  Tab and the actions panel. **Trigger:** users asking for it, once ordering itself is settled.
- **Dragging between pages.** No edge-hover auto-page, no drop onto a page control. **Trigger:**
  a plan long enough that moving something across a page boundary is common — and note that
  FS-0008 §Out of Scope already parks server-side paging behind a similar trigger.
- **Reordering archived items.** The archived view is a record, not a working list.
- **Ordering across scopes.** Daily and long-term are separate sets; nothing moves between them
  by drag. Moving scope stays the existing menu action.
- **Sorting.** No sort-by-date, sort-by-done, or "clean up" action. This feature is about the
  order a person chose, and an automatic sort would destroy it.
- **Ordering plans themselves.** Only items within a plan.
- **Enforcing the `403`.** The contract declares it and the client hides drag handles from a
  view-only user, but no checklist operation in this repo checks plan ownership — `user_id` is
  accepted on every checklist RPC and used by none, which predates this feature and applies to
  the whole surface. Reorder inherits that gap rather than closing it here: bolting an
  ownership check onto one endpoint would make the surface inconsistent and would put an
  authorization decision in a feature that is about ordering. **Owned by FS-0005**
  (per-user authorization on plan-scoped resources), which is where the check belongs.
  **Trigger:** FS-0005 shipping, at which point this row becomes reachable with no change here.
