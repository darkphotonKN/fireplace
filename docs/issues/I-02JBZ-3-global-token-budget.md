---
id: I-02JBZ-3
status: done
implements: FS-02JBZ
blocked_by: [I-DF4ZB-3, I-DF4ZB-4]
labels: [feature]
title: "FS-02JBZ slice 3: global daily token budget across every insights model call (LLM_BUDGET_EXHAUSTED)"
---
Implements FS-02JBZ §Requirements (R31–R35, R58 `LLM_BUDGET_EXHAUSTED`, R60), §API surface (getChecklistSuggestion, getSuggestedVideos error rows)

## What to Build

The circuit breaker of ADR-0019 clause 3–4, at the one seam every insights model call goes
through (the generator in `services/insights-service/internal/ai/`).

- **Record everything (R31, A4).** Every model call records usage in I-DF4ZB-4's ledger: the
  guided generators already do; add the single checklist suggestion (kind
  `checklist_suggestion`) and the video search terms (kind `video_terms`). Each transport retry
  records what the provider reported. Embedding calls are **not** recorded against the budget
  (S8).
- **Check before calling (R32).** Before any model call, sum today's tokens from the ledger,
  where today is the server's **UTC** calendar day and tokens = input + output (cached input is
  part of input). Compare with the budget.
- **Refuse over budget (R33).** At or over budget, make no call; fail with
  `LLM_BUDGET_EXHAUSTED` as `ErrorInfo` detail on `ResourceExhausted` (I-DF4ZB-3 maps it to 429).
  Add the code to `common/errcode` (R58).
- **Config (R34).** `LLM_DAILY_TOKEN_BUDGET` read at boot; unset → 200000. Set 200000 in
  `services/insights-service/.env` / `.env.example` and the `insights-service` environment in
  `docker-compose.yml`; `docker-compose.prod.yml` sets `${LLM_DAILY_TOKEN_BUDGET:-2000000}`.
- **Soft edge (R35).** Check-then-call; document in code that concurrent calls may overshoot by
  their output ceilings, and do not add locking to prevent it.
- **Gateway (R60).** `getChecklistSuggestion` and `getSuggestedVideos` document
  `429 · LLM_BUDGET_EXHAUSTED`; regenerate `openapi.yaml` and the client if the document changes.
  Nothing else about them changes.

The day plan's catch-and-fall-back is I-02JBZ-5. A generator added later (day plan, step and
item-tree suggestions) is covered automatically by going through the same seam; say so in the
seam's doc comment.

## Acceptance Criteria

- [ ] With today's ledger sum at or over the budget, no provider call is made and the generator
      returns `LLM_BUDGET_EXHAUSTED`; through the gateway, `getChecklistSuggestion` and
      `getSuggestedVideos` answer `429 · LLM_BUDGET_EXHAUSTED`.
- [ ] Under budget, behavior is unchanged.
- [ ] The single suggestion and the video search terms write ledger rows with their kinds;
      retries write rows; embeddings do not count.
- [ ] Yesterday's (UTC) usage does not count toward today.
- [ ] Unset budget = 200000; dev config 200000; prod compose default 2000000.
- [ ] `common/errcode` gains only `LLM_BUDGET_EXHAUSTED` in this slice.
- [ ] Gates green.
- [ ] Tests pass.

## Blocked By

- I-DF4ZB-4 — the usage ledger and the per-generator seam this guards.
- I-DF4ZB-3 — `ResourceExhausted` → 429 and `ErrorInfo` reasons at the gateway seam.

## Expected write set

- `services/insights-service/internal/ai/generator.go` (+ test) — the guard at the call seam
- `services/insights-service/internal/guided/` usage repository (sum for a UTC day) — or wherever I-DF4ZB-4 put the ledger
- `services/insights-service/internal/insights/service.go` (legacy generators record usage)
- `services/insights-service/migrations/00000N_*` only if the ledger's kind enum is a CHECK constraint
- `services/insights-service/config/` **(hotspot: insights composition root)**, `.env.example`
- `docker-compose.yml`, `docker-compose.prod.yml`
- `common/errcode/errcode.go` **(hotspot)**
- `services/api-gateway/internal/insights/typed_insights.go` (error docs)
- `services/api-gateway/openapi.yaml` + `client/src/api/generated/schema.d.ts` **(hotspot, regenerated if changed)**

## Spec Reference

FS-02JBZ §Decisions D8, D16; §Amendments A4; §Requirements R31–R35, R58, R60; §Spec defaults S8;
§API surface (getChecklistSuggestion, getSuggestedVideos rows); §Edge States ("Ledger sum crosses
the budget mid-flight"); ADR-0019 clauses 3–4.

## TDD Approach

- RED: a generator test with a fake ledger reporting `budget` tokens for today and a fake
  provider: assert zero provider calls and an `ErrorInfo` reason `LLM_BUDGET_EXHAUSTED` on
  `ResourceExhausted`.
- GREEN: the pre-call check.
- Then: under-budget passthrough; UTC-day boundary; retries recorded; legacy kinds recorded;
  config default; gateway 429 mapping.
