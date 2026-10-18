---
id: I-F6AEE-1
status: done
implements: FS-F6AEE
blocked_by: []
labels: [feature]
title: "FS-F6AEE slice 1: duplicate an item (with its live children) end to end through the API"
---
Implements FS-F6AEE §Requirements (R1–R13, R19), §API surface

## What to Build

The whole server side of duplicating, from the plan-service transaction to a typed function the
client can call. After this slice, `POST /api/plans/{id}/checklists/{checklist_id}/duplicate`
returns `201` with the created items. There is no UI yet: slice 2 adds the menu row.

**plan-service (R1–R10).** Add `rpc DuplicateItem(DuplicateItemRequest) returns
(ListItemsResponse)` to `ChecklistService`. `DuplicateItemRequest` carries `id`, `user_id` and
`plan_id`, with `plan_id` required and scoping the row like the other item-by-id requests. Add
it with a comment in the style of `ReorderItems`, and regenerate with `make proto-gen`. Existing
messages and RPCs stay as they are.

- Service: load the source with `GetInPlan(planID, id)`. A missing row, or one in another plan,
  is `ErrNotFound`. An archived source is `ErrInvalidInput`. Nothing is written in either case.
- Repository: one `commonhelpers.ExecTx` that:
  1. inserts the copy with the source's `description`, `type`, `scope`, `parent_id`,
     `start_date` and `due_date`, and with `done = false` and `archived = false`. It does not
     copy `intention` (R2, R3);
  2. if the source is top-level, reads its **non-archived** children in stored order and inserts
     one copy each under the new id, with the same fields and `done = false` (R4). Their
     sequences are 1..N in the source's child order;
  3. rewrites the source's sibling set (same plan, scope and parent, `archived = false`, the
     stored order of `ListSiblings` including the tiebreak) densely 1..N, with the copy directly
     after the source (R5). Reusing the `unnest … WITH ORDINALITY` write that `Reorder` uses is
     the expected path;
  4. reads the created rows back: the copy first, then its children in order (R7).
  Any error rolls the whole transaction back (R6).
- Handler: map to the proto as `ReorderItems` does. Errors go through `grpcerror`, so NotFound
  maps to `NotFound` and InvalidArgument to `InvalidArgument`.
- No publish of any kind (R9). No migration (R10).

**Gateway (R11–R13).** In `typed_checklists.go`, register `duplicateChecklist` as `POST
/api/plans/{id}/checklists/{checklist_id}/duplicate` with `DefaultStatus: 201`, the shared
`writeErrs` list, and no request body. Put `authorize(ctx, "duplicate checklist item",
in.PlanID)` first (owner-or-member, same as `createChecklist`). Return `ChecklistListOutput`.
Add `DuplicateChecklist` to the `ChecklistsClient` interface and to `client.go`, mapping each
item with `checklistFromProto`. Errors go through `apierr.ProblemFor`, with no new `errcode`.
The operation's description states the copy semantics in a sentence or two: what is copied, not
done, children, placement, and the 400 for an archived source.

**Contract and client (R19).** Regenerate `services/api-gateway/openapi.yaml` (`make -C
services/api-gateway openapi`) and `client/src/api/generated/schema.d.ts` (`make -C
services/api-gateway client`). Then run `make -C services/api-gateway gates`: the diff must be
additive only. In `client/src/api/checklists.ts`, add `duplicateChecklistItem(planId,
checklistId): Promise<ChecklistItem[]>` through the generated `api.POST`, with `apiErrorFrom` on
error and `markTouchedToday()` only on success.

No server identifier, proto name or schema name says "milestone" (ADR-0014).

Files expected to touch:
- `common/api/proto/plan/checklist.proto` (+ generated `checklist.pb.go`, `checklist_grpc.pb.go`)
- `services/plan-service/internal/checklistitem/{model,repository,service,handler}.go`
- `services/plan-service/internal/checklistitem/repository_duplicate_test.go` (new),
  `service_test.go` / a new `service_duplicate_test.go`, `handler_test.go`
- `services/api-gateway/internal/gateway/plan/{client,typed_checklists}.go`
- `services/api-gateway/internal/gateway/plan/checklist_duplicate_test.go` (new); the
  `checklists_access_test.go` table gains the new route
- `services/api-gateway/openapi.yaml`, `client/src/api/generated/schema.d.ts`
- `client/src/api/checklists.ts`

## Acceptance Criteria

- [ ] Duplicating a childless top-level task returns one item with a new id, the same
      description, type, scope and dates, no parent, `done = false`, not archived and no
      `intention`.
- [ ] A done source gives a copy with `done = false`; a note source gives a note.
- [ ] A top-level source with children returns the copy, then one copy per **non-archived**
      child in the source's child order, each `done = false` whatever its source's state, and
      each parented to the copy. Archived children are not copied.
- [ ] A child source returns one item under the same parent.
- [ ] After a duplicate, the sibling set lists every other sibling in its previous relative
      order, with the copy directly after the source. This holds for a top-level item, for a
      child, and for the last item of a set.
- [ ] A forced failure after the first insert leaves no copy and no rewritten sequence (the
      transaction rolls back).
- [ ] An archived source answers InvalidArgument and writes nothing. A missing id, or an id in
      another plan than `plan_id`, answers NotFound and writes nothing.
- [ ] Nothing is published to `plan.events`.
- [ ] HTTP: the owner and a member get `201` with `ChecklistResp[]`. A stranger gets `404 ·
      NOT_FOUND`, no token gets `401 · UNAUTHENTICATED`, an archived source gets `400 ·
      VALIDATION_FAILED`, a missing or other-plan item gets `404 · NOT_FOUND`, and a malformed
      path uuid gets `422 · VALIDATION_FAILED`.
- [ ] `openapi.yaml` and `schema.d.ts` are regenerated, `make -C services/api-gateway gates`
      passes, and the contract diff adds the one path and nothing else.
- [ ] `duplicateChecklistItem` stamps *touched today* on success and not on failure.
- [ ] Tests pass.

## Blocked By

None.

## Spec Reference

FS-F6AEE §Requirements R1–R13 (server and gateway) and R19 (API wrapper). §API surface (the
one row, plus the gRPC note). §Acceptance Criteria (the server, HTTP, contract and
touched-today items). §Edge States (child source, note source, only-archived children, last of
set, archived source, other-plan id, concurrent duplicates, ties in sequence, plan-service
unreachable). User stories 3–13, 20–24.

## TDD Approach

- RED: a plan-service repository test against the test DB. Seed a top-level task (done) with
  two children (one done, one archived) between two siblings. Call Duplicate, then assert the
  returned rows (copy plus the one live child, both not done, in order) and that `ListSiblings`
  order is `[a, source, copy, b]`.
- GREEN: the transactional repository method.
- Then the service refusals (archived → ErrInvalidInput, other plan → ErrNotFound), the handler
  mapping, the gateway transport and access tests, the regeneration, the gates, and the client
  wrapper.
