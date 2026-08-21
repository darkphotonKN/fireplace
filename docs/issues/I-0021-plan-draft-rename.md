---
id: I-0021
status: open
implements: FS-0006
blocked_by: []
labels: [enhancement]
title: FS-0006 slice 1: rename focus to plan_draft end to end
---

Implements FS-0006 §Requirements, §API surface

## What to Build

The field rename, complete through every layer, with nothing else changing behaviourally.

- **plan-service**: migration renaming `plans.focus` → `plan_draft` (TEXT NOT NULL); `Plan`,
  `SearchResult`, `CreatePlanInput`, `UpdatePlanInput`; repository queries.
- **common/proto**: `PlanCreatedEvent.focus` → `plan_draft`, **reusing the existing field
  number**. insights is its only consumer and is mid-development, so a straight rename is safe —
  but the number must not be bumped.
- **api-gateway**: `CreatePlanReq`, `UpdatePlanReq`, `PlanResp` publish `planDraft`. `focus` is
  removed, not aliased. `planDraft` carries the 1–20000 bound (shape → 422).
- **client**: regenerate the typed client; update the create form and `myplans`. The list's
  `focus` render is **removed** — that duplication is what the split exists to fix.
- **chore in scope**: delete migration `000003`'s `-- learning or development.` comment. There
  has never been a `development` plan type.

`/api/plans` is already serialized (FS-0004), so no slice ⓪ — the change flows through the
typed Huma handlers and `openapi.yaml` regenerates.

## Acceptance Criteria

- [ ] Every existing `focus` value survives the migration unchanged; no other backfill.
- [ ] `focus` appears nowhere in `PlanResp`, `CreatePlanReq` or `UpdatePlanReq`.
- [ ] `PlanCreatedEvent`'s field number is unchanged; only its name moved.
- [ ] The plans list renders `description`, never the draft.
- [ ] `make -C services/api-gateway gates` passes; the client diff is committed.
- [ ] Tests pass.

## Blocked By

None. This is the foundation slice.

## Spec Reference

FS-0006 §Requirements R1–R6, R49, R57 · §API surface (`createPlan`, `updatePlan`,
`listPlans`/`getPlan` rows)

## TDD Approach

- RED: a transport test asserting `PlanResp` has no `focus` key and does have `planDraft`.
- GREEN: rename through model → repo → proto → transport → generated client.
