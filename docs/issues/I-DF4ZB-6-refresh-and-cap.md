---
id: I-DF4ZB-6
status: done
implements: FS-DF4ZB
blocked_by: [I-DF4ZB-5]
labels: [feature]
title: "FS-DF4ZB slice 6: refreshDailyBrief with an atomic cap of 3 and refreshesRemaining"
---
Implements FS-DF4ZB §Requirements (R16, R17, R20, R51, R53, R56), §API surface

## What to Build

`POST /api/insights/daily-brief/refresh` with body `{planId, date}` redraws the brief for the key.

- **Refresh (R16).** Runs through the rule gate (I-DF4ZB-5). A refresh served by the gate does
  not count and returns the stored moves. A refresh that reaches the model counts. With no stored
  brief for the key it behaves as a read, and counts only if it reaches the model.
- **Cap (R17).** At most 3 counted refreshes per (plan, user, date). The check and the increment
  are one atomic step (concurrent refreshes cannot overspend). A refresh that would be the fourth
  is refused with `BRIEF_REFRESH_LIMIT_REACHED` as `ErrorInfo` detail on `ResourceExhausted`
  (→ 429 via I-DF4ZB-3) and the stored brief is untouched. A failed generation does not count
  (R54) and leaves the stored brief.
- **`refreshesRemaining` (R20).** Computed from the counter, 0–3, on both read and refresh.
- Add `BRIEF_REFRESH_LIMIT_REACHED` to `common/errcode` (R53).
- **RPC** `RefreshDailyBrief`; **gateway** typed Huma handler `refreshDailyBrief`, camelCase body,
  identity from the JWT; regenerate `openapi.yaml` and the client; gates green.

The client's refresh action is I-DF4ZB-8.

## Acceptance Criteria

- [ ] A refresh with an unchanged plan makes no model call and does not lower `refreshesRemaining`.
- [ ] Three counted refreshes succeed; the fourth returns `429 · BRIEF_REFRESH_LIMIT_REACHED` and
      leaves the stored brief unchanged.
- [ ] Concurrent refreshes at count 2: one succeeds, one gets 429.
- [ ] A failed generation during refresh returns 503, stores nothing and spends no refresh.
- [ ] A refresh with no stored brief for the key behaves as a read and counts only if it
      reaches the model.
- [ ] `refreshesRemaining` is correct on read and refresh.
- [ ] `openapi.yaml` and the generated client are regenerated; gates green.
- [ ] Tests pass.

## Blocked By

I-DF4ZB-5 — whether a refresh counts depends on whether the gate served it.

Not blocked by H2: built against the R29 seam/stub.

## Expected write set

- `common/api/proto/insights/insights.proto` + regenerated stubs **(hotspot)**
- `common/errcode/errcode.go`
- `services/insights-service/internal/guided/` (refresh, cap counter)
- `services/insights-service/internal/insights/handler.go` (or the guided handler)
- `services/insights-service/migrations/00000N_*` (counter column/table, if not already present)
- `services/api-gateway/internal/insights/typed_insights.go` (+ test),
  `services/api-gateway/internal/gateway/insights/client.go`
- `services/api-gateway/openapi.yaml` + `client/src/api/generated/` **(hotspot, regenerated)**

## Spec Reference

FS-DF4ZB §Requirements R16, R17, R20, R51, R53 (`BRIEF_REFRESH_LIMIT_REACHED`), R56; §Decisions
L5; §Spec defaults S2, S4; §API surface (refreshDailyBrief); §Edge States ("Refresh with nothing
changed", "Fourth counted refresh", "Two tabs refresh at once at count 2", "Plan edited after the
brief was drawn").

## TDD Approach

- RED: a repository test that fires N concurrent increments at count 2 and asserts exactly one
  succeeds.
- GREEN: one atomic conditional update.
- Then: service tests for gated (no count), counted, fourth (429, brief unchanged), failed (no
  count); the handler and regenerated contract.
