---
id: I-0028
status: open
implements: FS-0006
blocked_by: [I-0021, I-0022]
labels: [feature]
title: FS-0006 slice 6: guided and custom creation
---

Implements FS-0006 §Requirements, §API surface

## What to Build

Both creation paths on **one operation on one resource**. Guided and custom differ in *how the
draft is obtained*, not in what is created, so they are not separate endpoints.

- **`POST /api/plans`** gains a required `mode` (`guided` | `custom`). `mode=custom` takes
  `planDraft` (1–20000); `mode=guided` takes `seed` (10–500).
- **Validation split** (ADR-0005): per-field constraints are **shape** → 422 at the boundary. The
  conditional rule — the field matching `mode` present, the other absent — is **domain**, owned by
  plan-service, returning 400 `VALIDATION_FAILED` through the existing mapping. **No new error
  code**; a body whose mode and payload disagree is a first-party client bug.
- **plan-service dials insights directly.** `SetupServices` stops discarding its
  `discovery.Registry` parameter (currently `_`); `internal/plan/insights_client.go` follows the
  existing pattern — one cached `*grpc.ClientConn`, never dialled per RPC, gRPC status codes
  translated into local domain sentinels. Clone `insights-service/internal/insights/plan_client.go`.
- **The `GenerateDraft` call is OUTSIDE the transaction.** Folding it into `ExecTx` alongside the
  outbox write would hold a Postgres transaction open for a 10–15 second LLM call.
- **Fail-closed**: if generation fails, **no plan row commits**. Committing with the seed as the
  draft would silently hand the user a custom plan *and* generate items from the degraded input —
  one failure becoming two. The client keeps the seed so retry is one action.
- **Creation writes the `plan.items_requested` outbox row in the same transaction as the plan
  row** — both modes. `plan.created` stays a general-purpose lifecycle fact with no insights
  binding, and is never re-emitted.
- **`description` is prefilled from the draft** and independently editable thereafter.
- **Client**: the form offers both paths; custom is fully skippable.

## Interim state

Until I-0027 lands, `mode=guided` returns **501 `NOT_IMPLEMENTED`** — wired end to end in
`apierr` and described in `errcode.go` as a delivery statement rather than a failure. `mode=custom`
works fully. Do not fall through to 500 or 503 for this.

## Acceptance Criteria

- [ ] `mode=custom` + valid `planDraft` → 201, exactly one `plans` row, **zero** `checklist_items`.
- [ ] `mode=guided` + valid `seed` → the persisted draft is the generated text, not the seed.
- [ ] `mode=guided` carrying `planDraft` (or the inverse) → 400 `VALIDATION_FAILED`, no row, no LLM call.
- [ ] `seed` under 10 or over 500 → 422, **no LLM call**.
- [ ] `GenerateDraft` failure → no `plans` row exists.
- [ ] Exactly one `plan.items_requested` outbox row per creation, in the plan's transaction.
- [ ] `SetupServices` no longer discards its registry; the client dials once and reuses the conn.
- [ ] The `GenerateDraft` call is demonstrably outside `ExecTx`.
- [ ] `plan-service/CLAUDE.md`'s "No outbound gRPC — this service is a leaf" line is updated.
- [ ] Tests pass; gates pass.

## Blocked By

I-0021, I-0022. **Not** blocked by I-0027 — see Interim state.

## Spec Reference

FS-0006 §Requirements R3, R7–R14, R50, R58, R59 · §API surface (`createPlan`)

## TDD Approach

- RED: transport test — `mode=guided` with a 4-char seed returns 422 and the fake generator is never called.
- GREEN: discriminator + shape bounds, then the client and the outbox write.
