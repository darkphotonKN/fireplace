# FS-F6AEE: Duplicate an item

> Status: work-order · SPECIFICATION.md: client `## Checklists` "Duplicate an item"; plan-service `### Checklist Items` "Duplicate an item" → this FS · Related FS: FS-12ZMF (owner-or-member access, R7), FS-0008 (page jump for added items, R16/R17), FS-0009 (sibling-set order), FS-043WP (milestone page; its R17 menu list is widened here, see R14), FS-KSJFR (touched-today stamp, R12) · Related ADRs: docs/adr/0002-contract-planes-code-first-openapi.md, docs/adr/0004-error-representation-rfc9457.md, docs/adr/0005-request-validation-and-contract-design.md, docs/adr/0014-milestones-are-a-client-reading-of-items.md, docs/adr/0015-plan-access-is-a-relation.md

## Summary

Any live checklist item can be duplicated from its menu. The copy is new work, so it starts not
done. It sits directly after the item it was copied from, with the same text, intention, type,
scope, parent and dates. Duplicating a top-level item brings its live children along too, so copying a
milestone copies its steps. The copy is made in one go on the server, so it either arrives whole
or not at all.

Terms: on the client, a top-level task is a **milestone** and its children are its **steps**
(`client/CONTEXT.md`, ADR-0014). The server knows only checklist items, parents and children,
and this FS uses those words for everything that crosses the wire. No new domain term is
introduced: "duplicate" is the plain verb, and its rules are the requirements below.

## Background: what ships today

- `ItemActions` (`client/src/components/ItemActions.tsx`) is the one menu for an item. It is
  mounted by the list view (`Todo.tsx`, one per live row), the grid (`ChecklistGrid.tsx`, on each
  card and each step row inside a card), and the milestone page (`MilestonePage.tsx`, once, for
  the milestone itself; its steps carry no menu). Optional handlers are "offered only where they
  are handed over". The archived list view renders a delete button in place of the menu, so an
  archived item never shows `ItemActions` today.
- The checklist endpoints are already Huma-typed (`/api/plans/{id}/checklists/...` in
  `services/api-gateway/openapi.yaml`), and every operation asks the access seam for
  owner-or-member (FS-12ZMF R7). **No slice ⓪ is needed.**
- plan-service `ChecklistService` has `CreateItem` (one item; no copy semantics) and
  `ReorderItems` (rewrites one sibling set densely in one transaction). Two-tier nesting is
  enforced in the service (`validateParent`) and by the trigger `trg_checklist_two_tier`.
  `chk_note_not_done` forbids a done note.
- Item mutations in `client/src/api/checklists.ts` stamp *touched today* once they resolve
  (FS-KSJFR R12).

## Requirements

Decision source: every requirement below is a **charter default**, approved by the owner under
the charter "Duplicate checklist items" (not challenged in `challenge-me`). The reason sits with
each one.

### Server (plan-service)

1. **R1. One operation duplicates one item.** `ChecklistService.DuplicateItem` takes the source
   item id, the plan id the caller addressed, and the caller's user id. It returns the items it
   created.
2. **R2. The copy's fields.** The copy takes the source's `description` (same text, no "(copy)"
   suffix), `intention` (verbatim; a NULL `intention` stays NULL), `type`, `scope`, `parent_id`,
   `start_date` and `due_date`. It gets a new id and new timestamps, and `archived = false`.
   Nothing else is copied. *Reason: the adjacent placement already shows which is which, so a
   suffix would only be text to delete. USER DECISION on `intention`: a duplicate carries the
   item's content, and intention is content.*
3. **R3. The copy is never done.** `done = false` on the copy whatever the source's state.
   *Reason: a copy is new work, and it keeps a copied note valid under `chk_note_not_done`.*
4. **R4. A top-level item brings its live children.** When the source has no parent, each of
   its **non-archived** children is copied under the copy, with the same R2 fields (the parent
   becoming the copy) and `done = false` (R3) whatever the child's state. Archived children are
   not copied. A child source copies only itself: two-tier nesting leaves it nothing below.
   *Reason: one level is all the two-tier model allows. USER CONFIRMED that no copy carries over
   done.*
5. **R5. Placement.** The copy sits **directly after the source** in the source's sibling set
   (the same plan, scope and parent, live items only, in stored order as FS-0009 defines it). No
   other sibling changes its order relative to the others. Copied children keep the source's
   child order. Placement is written server-side, in the same transaction, so every client sees
   the same order. Rewriting the sibling set's sequences densely, as `ReorderItems` does, is the
   expected way.
6. **R6. All or nothing.** The copy, its children and the placement write happen in **one
   plan-service transaction**. Any failure rolls the whole thing back, so a half-copied
   milestone never exists.
7. **R7. Response order.** The created items come back as a list: the copy first, then its
   copied children in their order. A child source, a note or a childless item returns a list of
   one.
8. **R8. Refusals.** The source must exist and belong to the addressed plan (ADR-0015 §5); if
   not, the answer is NotFound, exactly as for a missing item. A source that is archived is
   refused as InvalidArgument (*reason: duplicating something archived would really be a
   restore*). Nothing is written in either case.
9. **R9. No events.** `DuplicateItem` publishes nothing. *Reason: `CreateItem` publishes nothing,
   and the copy is never done, so no `checklist_item.completed` is owed.*
10. **R10. No schema change.** No migration and no new column. Existing RPCs and messages are
    unchanged; the new RPC may reuse `ListItemsResponse`.

### Gateway (api-gateway)

11. **R11. One new typed operation.** `POST /api/plans/{id}/checklists/{checklist_id}/duplicate`
    is registered through Huma beside the other checklist operations, so it lands in the
    generated `openapi.yaml` and the generated TS client. It takes no request body.
12. **R12. Access.** The operation requires **owner-or-member**, through the same access seam
    and in the same position as `createChecklist` (FS-12ZMF R7, ADR-0015). A stranger gets 404.
    A member may duplicate.
13. **R13. Errors pass through the existing mapping.** plan-service NotFound → `404 NOT_FOUND`,
    InvalidArgument → `400 VALIDATION_FAILED`, Unavailable → `503 SERVICE_UNAVAILABLE`, via
    `apierr.ProblemFor`. The gateway restates no plan-service rule (ADR-0005). No new `errcode`
    is added: the client never offers the archived case, so it has nothing to switch on.

### Client

14. **R14. A "Duplicate" row in the one menu.** `ItemActions` gains an optional `onDuplicate`
    handler and shows a "Duplicate" row only when it is handed over. It is handed over everywhere
    the menu renders for a live item: list rows (`Todo`), grid cards and the step rows inside them
    (`ChecklistGrid`), and the milestone page's menu (`MilestonePage`). One shared menu, so no
    per-view variants. *This widens FS-043WP R17's page menu (Rename, Dates, Archive, Delete) by
    one row, Duplicate. That is not one of the changes R17 rules out (flip, move, indent):
    duplicating leaves the page's subject as it is.*
15. **R15. Never offered on an archived item.** `onDuplicate` is not handed over for an item whose
    `archived` is true. Today no archived item renders the menu at all; this requirement holds if
    that changes.
16. **R16. Wait, then insert.** Choosing Duplicate calls the server and waits. On success, the
    returned items go into the view's list directly after the source (the copy, then its
    children under it), with no refetch. Nothing is shown before the server answers. *Reason:
    smaller v1, and no temporary ids to reconcile.*
17. **R17. Failure.** On failure nothing is inserted and the view shows its existing error
    message for a failed item action (the same surface its other item actions use). No new
    error UI.
18. **R18. Paging.** In the paged list (FS-0008), if the copy lands on a page other than the
    current one, the view jumps to the page holding it, by the same rule FS-0008 applies to an
    added item ("jump to the page the item lands on", R16/R17). No new paging rule.
19. **R19. The API wrapper.** `client/src/api/checklists.ts` gains one function calling the new
    operation through the generated client (ADR-0002 §4: no hand-written fetch). It stamps
    *touched today* on success like every other mutation there (FS-KSJFR R12).
20. **R20. The milestone page stays put.** Duplicating from the milestone page leaves the page on
    the source milestone. The returned items go into the page's loaded plan list, so the copy is
    already in place, right after the source, when Back returns to the plan. *Derived default:
    the charter says only "insert", and staying is the reading that adds no navigation rule. See
    Out of Scope.*

## User Stories

1. As a planner, I want to duplicate an item from its menu, so that I don't retype something I need twice.
2. As a planner, I want the copy right after the original, so that I can see at once which item it came from.
3. As a planner, I want the copy to have exactly the same text, so that I'm not left deleting a "(copy)" suffix.
4. As a planner, I want the copy to keep the original's type, so that a note stays a note and a task stays a task.
5. As a planner, I want the copy to keep the original's scope, so that a daily item stays daily and a long-term one stays long-term.
6. As a planner, I want the copy to keep the original's start and due dates, so that I only adjust the dates rather than re-enter them.
7. As a planner, I want a copy of a done item to start not done, so that the copy reads as new work.
8. As a planner duplicating a milestone, I want its steps copied with it, so that I can reuse a block of work as a template.
9. As a planner duplicating a milestone, I want every copied step to start not done, so that the new milestone's progress starts from zero.
10. As a planner duplicating a milestone, I want the copied steps in the same order as the original's, so that the sequence I planned survives.
11. As a planner, I want archived steps left behind when I duplicate a milestone, so that work I set aside doesn't come back.
12. As a planner, I want to duplicate a single step, so that I can repeat it inside the same milestone, right after itself.
13. As a planner, I want a duplicate either to arrive whole or not at all, so that I never find a milestone copied with half its steps.
14. As a planner, I want Duplicate in the same menu in the list, the grid and on the milestone page, so that I don't have to learn three menus.
15. As a planner looking at my archived items, I want no Duplicate there, so that restoring and copying stay different things.
16. As a planner on page 1 whose copy lands on page 2, I want the view to follow it, so that I see what I just made.
17. As a planner, I want a failed duplicate to say so and leave my list as it was, so that I'm never unsure whether a copy exists.
18. As a planner, I want the copy to appear only once it really exists, so that I never act on an item the server doesn't have.
19. As a planner on a milestone's page, I want to duplicate the milestone and find the copy next to it when I go back, so that the page I'm on doesn't change under me.
20. As a member of a shared plan, I want to duplicate items like the owner can, so that sharing the plan means sharing the work.
21. As a stranger guessing a plan or item id, I should get "not found", so that duplicating can't be used to learn what exists.
22. As a planner using two devices, I want the copy in the same place on both, so that the order doesn't depend on which device made it.
23. As a client developer, I want the operation in the generated client, so that a contract change breaks my build, not my users.
24. As a dashboard user, I want duplicating to count as touching my plans today, so that the day gate treats it like any other edit.

## Acceptance Criteria

- [ ] `DuplicateItem` on a childless top-level task returns one item: a new id, same description, intention, type, scope, dates, no parent, `done = false`, not archived, placed directly after the source.
- [ ] The copy and each copied child carry their own source's `intention` verbatim; a source with no `intention` (NULL) gives a copy with none.
- [ ] `DuplicateItem` on a done source returns a copy with `done = false`.
- [ ] `DuplicateItem` on a top-level task with children returns the copy then one copy per non-archived child, in the source's child order, each `done = false` whatever its source's state, each parented to the copy.
- [ ] Archived children of the source are not copied.
- [ ] `DuplicateItem` on a child returns one item under the same parent, directly after the source among that parent's live children.
- [ ] `DuplicateItem` on a note returns a note, not done.
- [ ] After a duplicate, listing the sibling set shows every other sibling in its previous relative order, with the copy immediately after the source.
- [ ] A failure partway through (e.g. a forced error after the parent insert) leaves no copy and no changed sequence.
- [ ] An archived source is refused with InvalidArgument, and nothing is written.
- [ ] A source id that doesn't exist, or exists in another plan than the one addressed, answers NotFound, and nothing is written.
- [ ] No message is published on `plan.events` by a duplicate.
- [ ] `POST /api/plans/{id}/checklists/{checklist_id}/duplicate` appears in the regenerated `openapi.yaml` with the shape in §API surface; `make -C services/api-gateway gates` is green, with no breaking change reported.
- [ ] The owner and a member each get `201` with the created items; a stranger gets `404 NOT_FOUND`; no token gets `401 UNAUTHENTICATED`.
- [ ] An archived source answers `400 VALIDATION_FAILED`; a missing or other-plan source answers `404 NOT_FOUND`; a malformed path uuid answers `422 VALIDATION_FAILED`.
- [ ] The generated TS client is regenerated, and the client calls the operation only through it.
- [ ] `ItemActions` shows a "Duplicate" row when `onDuplicate` is handed over, and none when it isn't.
- [ ] Duplicate is offered on list rows, grid cards, grid step rows and the milestone page's menu; it is not offered on any archived item.
- [ ] After a successful duplicate in the list view, the copy (and its children) appear directly after the source without a refetch; nothing appears before the server answers.
- [ ] After a successful duplicate in the grid, the copy card appears directly after the source card with its steps.
- [ ] A duplicate whose copy lands on another page of the paged list moves the view to that page.
- [ ] A failed duplicate inserts nothing and shows the view's existing item-action error.
- [ ] Duplicating from the milestone page keeps the page on the source; after Back, the copy is directly after the source in the plan.
- [ ] A successful duplicate stamps *touched today*; a failed one doesn't.

## Edge States

- **Source is a child (a step):** the copy goes under the same parent, right after the source.
  It has no children to copy.
- **Source is a top-level note:** notes cannot be parents, so only the note is copied.
- **Source has only archived children:** only the source is copied; the response has one item.
- **Source is the last of its sibling set:** the copy becomes the new last.
- **Source is archived:** the menu doesn't offer it (R15). A direct call is refused with
  `400 VALIDATION_FAILED` (R8), and nothing is written.
- **Source deleted or archived between opening the menu and choosing Duplicate:** 404 or 400.
  The client shows its error (R17) and inserts nothing. The deleted row's removal reaches the
  view the way it does today; this feature doesn't add a refetch.
- **Item id from another plan in the path:** 404, never a copy into either plan (ADR-0015 §5).
- **Two duplicates of the same source in flight:** each copy goes directly after the source at
  the time it commits, so the later one sits between the source and the earlier copy. Both
  copies exist and neither is half-written.
- **A child of the source is archived or added mid-request:** the copy reflects the children as
  the transaction reads them. There is no partial result.
- **Sequences already tied** (the global `CountItems()+1` counter can tie, FS-0009 §Out of
  Scope): "directly after" is defined by the stored list order including its tiebreak. A dense
  rewrite of the set (R5) removes the tie for that set.
- **Daily item while daily is AI-only:** the add bar is hidden then, but the menu still offers
  Duplicate. The AI-only rule is not enforced on the server today and this feature doesn't start
  enforcing it.
- **Duplicate in the grid while a composer is tracing a milestone:** the copy is a new card and
  is not traced. Tracing follows only what the composer itself created.
- **Plan-service unreachable:** `503 SERVICE_UNAVAILABLE`. The client shows its error, and the
  user can try again from the menu.
- **Member removed from the plan mid-session:** 404 (ADR-0015), shown as a failed action.

## API surface

One new operation. Additive: no existing path, schema or field changes, so the breaking gate
stays green. Downstream, plan-service gains `rpc DuplicateItem(DuplicateItemRequest) returns
(ListItemsResponse)`, where the request carries `id`, `plan_id` and `user_id`.

| Op | Method + Path | Query/Params | Request body | Response | Errors |
|----|---------------|--------------|--------------|----------|--------|
| Duplicate an item | `POST /api/plans/{id}/checklists/{checklist_id}/duplicate` | `id` (path, uuid, plan), `checklist_id` (path, uuid, the source item) | none | `201`: `ChecklistResp[]`, the copy first, then its copied children in order (one element for a child, a note or an item without live children) | no relation to the plan, item missing, or item not in the path plan: `404 · NOT_FOUND` · source archived: `400 · VALIDATION_FAILED` · malformed path uuid: `422 · VALIDATION_FAILED` · no token: `401 · UNAUTHENTICATED` · plan-service unreachable: `503 · SERVICE_UNAVAILABLE` |

The checklist operations share one declared error list (`400, 401, 403, 404, 422, 503`). This
operation may reuse it, but `403` is unreachable here because a member is allowed.

## Out of Scope

- **Optimistic insert.** The view waits for the server (R16). **Trigger:** duplicate latency that
  people notice.
- **A "(copy)" suffix or any rename on copy.**
- **Duplicating across plans or scopes**, or into a chosen position. The copy goes in the
  source's own set, right after it.
- **Duplicating several items at once, or a whole plan.**
- **Duplicating archived items** (that would be a restore, a separate action).
- **A per-step menu on the milestone page.** The page has one menu, for the milestone (FS-043WP
  R17). This feature adds Duplicate to it but adds no step menus.
- **Navigating to the copy** from the milestone page (R20). **Trigger:** people duplicating from
  the page and then hunting for the copy.
- **Events for duplication.** None is emitted (R9).
- **A server-side milestone** of any kind (ADR-0014). The server duplicates checklist items.
- **Undo.** The copy is removed by the existing Delete.
