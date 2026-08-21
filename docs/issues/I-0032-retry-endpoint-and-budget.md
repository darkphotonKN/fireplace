---
id: I-0032
status: open
implements: FS-0006
blocked_by: [I-0022, I-0023]
labels: [feature]
title: FS-0006 slice 8: retry endpoint, cooldown and attempt cap
---

Implements FS-0006 §Requirements, §API surface

## What to Build

`POST /api/plans/{id}/generation/retry` → **202**, no body.

- Emits a **fresh `plan.items_requested`** outbox row. Each emission is a new `event_id`, so retry
  needs no special-casing anywhere in the dedup path — that is the entire reason there is one
  trigger event rather than two.
- Guarded transition `failed → generating` or `ready → generating`.
- **409** when the plan is already `generating` — well-formed request, state forbids it, and it
  must not silently enqueue a second chain.
- **429 `GENERATION_COOLDOWN`** inside the cooldown or past the attempt cap, **refused before any
  LLM call**. An LLM call sits behind this button; twenty clicks must not be twenty paid
  generations. `retry_count` and `last_retry_at` carry the budget.
- Retry does **not** require insights to be reachable at click time — it is asynchronous, and a
  broken chain fails normally into `failed`.

## Acceptance Criteria

- [ ] Retry writes a `plan.items_requested` row with a **different** `event_id` each time.
- [ ] Retry while `generating` → 409, no new outbox row.
- [ ] Retry inside the cooldown → 429, no outbox row, no LLM call, remaining cooldown surfaced.
- [ ] Retry past the attempt cap → refused; the plan stays `failed`.
- [ ] A non-owner gets 403.
- [ ] Tests pass; gates pass.

## Blocked By

I-0022, I-0023.

## Spec Reference

FS-0006 §Requirements R32, R34 · §API surface (`retryPlanGeneration`)

## TDD Approach

- RED: two retries inside the cooldown; assert one outbox row and a 429.
- GREEN: guarded transition + budget check ahead of the emit.
