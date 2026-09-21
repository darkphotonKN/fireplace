---
id: I-0055
status: in-progress
implements: FS-0009
blocked_by: []
labels: [feature]
title: "FS-0009 slice 2: one endpoint that writes the order of a sibling set"
---
Implements FS-0009 §Requirements (R2), §API surface, §Edge States

## What to Build

The write path the order has never had. One request carries the complete, final order of one
sibling set and the server stores it in a single transaction.

**No migration.** `checklist_items.sequence` exists; so does `ChecklistItem.sequence` in the
proto and `ChecklistResp.sequence` in the OpenAPI document. Only the write is missing.

plan-service (`internal/checklistitem/`):

- `Reorder(ctx, in ReorderInput)` on the service: validates that `ids` is exactly a permutation
  of the sibling set identified by (plan, scope, parentId) — no missing member, no stranger, no
  mixing of parents or scopes — then hands off to the repository.
- Repository `Reorder`: one transaction, assigning dense positions `1..N` in the order given.
- `rpc ReorderItems(ReorderItemsRequest) returns (ListItemsResponse)` in
  `common/api/proto/plan/checklist.proto`, carrying `plan_id`, `user_id`, `scope`,
  `optional parent_id`, `repeated ids`. Returns the reordered siblings.
- Idempotent: sending the order a set already has writes the same numbers and errors on nothing.

api-gateway:

- `PATCH /api/plans/:id/checklists/order`, typed handler, `ReorderChecklistReq`
  (`scope`, `parentId` nullable, `ids`), proxying to the RPC.
- Errors per the §API surface table: unknown plan or id `404 · NOT_FOUND`; ids that are not a
  permutation, or that mix parents/scopes `400 · VALIDATION_FAILED`; unknown body member
  `422 · VALIDATION_FAILED`; no token `401 · UNAUTHENTICATED`; view-only on a shared plan
  `403 · FORBIDDEN`.
- Regenerate the OpenAPI document and the TS client, and run the contract gates. This is
  **additive** — a new path, no existing schema touched — so the breaking check must stay green.
  If it does not, stop and surface it rather than opening a PR.

**Per-item `sequence` writes are not part of this.** `UpdateChecklistReq` gains no `sequence`
field (R2.4): a drag is one move of one set, and N single-item writes would race each other.

## Acceptance Criteria

- [ ] A valid reorder stores dense `1..N` across the set and returns the siblings in that order.
- [ ] The same request sent twice leaves the same state and raises no error.
- [x] The write is atomic: a request that fails part-way leaves every sequence as it was.
- [ ] Ids missing a member of the set are refused `400 · VALIDATION_FAILED`.
- [ ] Ids containing an item from another parent, another scope, or another plan are refused
      the same way.
- [ ] An unknown item id, or an unknown plan, is `404 · NOT_FOUND`.
- [ ] A top-level set is addressed with `parentId: null`, and a child set with its parent's id.
- [ ] `UpdateChecklistReq` still has no `sequence` field.
- [ ] Contract gates pass and oasdiff reports no breaking change.
- [ ] plan-service and api-gateway tests pass.

## Blocked By

None. (I-0054 is independent; it makes later assertions stable but is not required here.)

## Spec Reference

FS-0009 §Requirements R2.1–R2.5; §API surface (the one new operation); §Edge States
(concurrent reorder, item deleted mid-drag, ids that are not the sibling set).

## TDD Approach

- RED: service test — reorder three items and assert their sequences come back `1, 2, 3` in the
  order asked for.
- GREEN: the repository transaction and the service validation that gets there.


## Integration note (parent session)

The slice arrived with every criterion met except atomicity, which was marked PARTIAL because
the worktree it was built in had no database: a fake repository proves the service *calls* the
write, never that the SQL does anything. That gap is now closed against the real
`fireplace_plans` Postgres, using the harness I-0054 introduced —
`repository_reorder_test.go` covers the dense `1..N` write through
`unnest($1::uuid[]) WITH ORDINALITY`, the read-back order, and the row-count guard. The guard
was proved load-bearing by deleting it: the vanished-id test then reports success where it
should abandon the transaction.

One gap the merge itself created: `ListSiblings` is a list query added after I-0054 made every
list order total, and it ended at `sequence ASC`. The set it returns is what the client renders
and drags against, so it now carries the same tiebreak. Found by reading the merged result, not
by any test that existed.
