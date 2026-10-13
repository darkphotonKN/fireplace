---
id: I-12ZMF-1
status: done
implements: FS-12ZMF
blocked_by: []
labels: [feature]
title: "FS-12ZMF slice 1: plan-service answers the relation, items are scoped to their path plan, user.deleted clears shares"
---
Implements FS-12ZMF §Requirements (R1–R3, R11–R13, R25), §API surface (plan-service RPCs: GetPlanAccess, item-by-id `plan_id`)

## What to Build

The plan-service foundation for ADR-0015, plus the end-to-end IDOR fix for checklist items.

**Relation RPC (R1–R3).** Add `GetPlanAccess(plan_id, user_id) → { relation, owner_id }` to
`PlanService`, with an enum `PlanRelation { UNSPECIFIED = 0; NONE; OWNER; MEMBER }`.
- `owner`: `plans.user_id = user`.
- `member`: a `plan_shares` row exists for the pair. `role` is ignored.
- Anything else, **including a missing plan**, is `NONE`. Never return `NotFound` for a missing
  plan.

One query (plan LEFT JOIN share) is enough. Add the matching method to the gateway's
`plangw.Client` (`GetPlanAccess(ctx, planID, userID) (Relation, ownerID, error)`). No handler
uses it yet; slice 2 does.

**Item belongs to the path plan (R11–R13).** Add `plan_id` to `GetItemRequest`,
`UpdateItemRequest`, `UpdateItemDatesRequest`, `ArchiveItemRequest` and `DeleteItemRequest`
(additive fields).
- plan-service scopes those operations `WHERE id = $1 AND plan_id = $2`. An item in another plan
  is `ErrNotFound`, indistinguishable from a missing one.
- An empty `plan_id` is `ErrInvalidInput` (R12).
- Watch the update and archive paths that reload by id after writing. The reload must not answer
  for a row the scoped write didn't touch.
- On the gateway, thread `in.PlanID` from the typed checklist handlers through
  `ChecklistsClient` (`GetChecklist`, `UpdateChecklist`, `UpdateChecklistDates`,
  `ArchiveChecklist`, `DeleteChecklist`) into the requests.

Create, list and reorder are already plan-scoped. Leave their guards as they are (R13).

**Account deletion (R25).** `CascadeDeleteForUser` also runs
`DELETE FROM plan_shares WHERE user_id = $1`. It's idempotent, and errors follow the consumer's
existing ack/requeue policy.

## Acceptance Criteria

- [ ] `GetPlanAccess` answers OWNER for the owner, MEMBER for a user with a share row, and NONE
      for a stranger and for a nonexistent plan id. `owner_id` is set whenever the plan exists.
- [ ] A malformed uuid → `InvalidArgument`.
- [ ] Each item-by-id RPC with the item's own `plan_id` behaves exactly as before.
- [ ] Each item-by-id RPC with another plan's `plan_id` → `NotFound`, and the row is unchanged
      (assert in the DB for update / dates / archive / delete).
- [ ] Each item-by-id RPC with empty `plan_id` → `InvalidArgument`.
- [ ] Through the gateway: `PATCH /api/plans/{A}/checklists/{item of B}` → 404, and B's item is
      unchanged. The same holds for GET, `/dates`, `/archive` and DELETE.
- [ ] After `user.deleted` for U, no `plan_shares` row has `user_id = U`. A second delivery
      succeeds as a no-op.
- [ ] `buf lint` is clean on the changed protos, and the generated `*.pb.go` are regenerated.
- [ ] Tests pass.

## Blocked By

None. This is the foundation slice.

## Spec Reference

FS-12ZMF §Requirements R1–R3 (relation), R11–R13 (item belongs to path plan), R25 (user.deleted);
§API surface "plan-service RPCs"; §Edge States (item from another plan, missing plan id,
user.deleted twice); ADR-0015 §Decision 1, 2, 5. User stories 19, 20, 21.

Files expected:
- `common/api/proto/plan/plan.proto`, `common/api/proto/plan/checklist.proto`, and the regenerated
  `*.pb.go`
- `services/plan-service/internal/plan/{repository,service,handler,model}.go`
- `services/plan-service/internal/checklistitem/{repository,service,handler,model}.go`, plus tests
- `services/api-gateway/internal/gateway/plan/{client.go,typed_checklists.go}`, plus the
  checklist transport tests

## TDD Approach

- RED: a checklistitem service test that updates an item with another plan's id and expects
  `ErrNotFound`, with the row unchanged.
- GREEN: `plan_id` in `UpdateItemInput` and the repo's `WHERE id AND plan_id`.
- RED: a plan service test where `GetPlanAccess` for a random plan id answers NONE (not an error).
