---
id: I-02JBZ-12
status: done
implements: FS-02JBZ
blocked_by: [I-02JBZ-8]
labels: [enhancement]
title: "FS-02JBZ slice 12: retire getDailyBrief / refreshDailyBrief now the client reads the day plan"
---
Implements FS-02JBZ §Requirements (R92), §Amendments (A1, A2), §API surface (removed getDailyBrief, refreshDailyBrief)

## What to Build

The brief's read path generates on a miss, which ADR-0019 forbids, and nothing reads it after
I-02JBZ-8. Remove its HTTP and RPC surface; keep what the day plan reuses.

- Remove `getDailyBrief` and `refreshDailyBrief` from
  `services/api-gateway/internal/insights/typed_insights.go` and the gateway insights client.
- Remove `GetDailyBrief` and `RefreshDailyBrief` (and messages) from `insights.proto`; regenerate.
- Remove the brief-only generation path in insights-service (the brief service's read/refresh
  orchestration and its generator config). **Keep** the plan tree, prompt assembly, validation,
  plan-state hash, the per (plan, user, date) counter and the usage ledger, which the day plan
  uses (A2). The brief's tables stay (dropping them is later cleanup, like `daily_reset`).
- Remove `getDailyBrief`/`refreshDailyBrief` from `client/src/api/insights.ts` if I-02JBZ-8 left
  them.
- Regenerate `openapi.yaml` and the client. If the breaking gate flags the removals (it will if
  the operations already exist on the base the gate compares against), allowlist them in
  `.oasdiff-ignore` citing FS-02JBZ.

**Owner note (FS O1):** I-DF4ZB-10 and I-DF4ZB-11 were written against the brief and Today's
moves. This slice removes the brief's surface they mention; they need retargeting at day plan
entries before they are built. Do not edit them here.

## Acceptance Criteria

- [ ] `GET /api/insights/daily-brief`, `POST /api/insights/daily-brief/refresh`, `GetDailyBrief`
      and `RefreshDailyBrief` are gone from the gateway, the proto, insights-service and the
      client.
- [ ] The day plan still works end to end (its tests unchanged and green): hash, validation,
      counter and ledger are intact.
- [ ] No read path in insights-service calls the model (assert across the remaining read RPCs).
- [ ] Gates green, with any breaking finding allowlisted citing FS-02JBZ.
- [ ] Tests pass.

## Blocked By

- I-02JBZ-8 — the client no longer calls the brief.

## Expected write set

- `common/api/proto/insights/insights.proto` + regenerated stubs **(hotspot)**
- `services/insights-service/internal/guided/` (remove brief orchestration only) (+ tests)
- `services/insights-service/internal/insights/handler.go` (or the guided handler)
- `services/insights-service/config/services.go` **(hotspot)**
- `services/api-gateway/internal/insights/typed_insights.go` (+ test), `services/api-gateway/internal/gateway/insights/client.go`
- `services/api-gateway/openapi.yaml` + `client/src/api/generated/schema.d.ts` **(hotspot, regenerated)**
- `.oasdiff-ignore` (repo root) **(hotspot)**
- `client/src/api/insights.ts` **(hotspot)**

## Spec Reference

FS-02JBZ §Amendments A1, A2; §Requirements R92; §Spec defaults S1; §API surface (removed rows,
plane 2); §Open items O1; ADR-0019 clause 2; ADR-0002 (removing serialized endpoints).

## TDD Approach

- RED: a gateway route test asserting the two paths are not registered; an insights test that the
  day plan service still builds with the kept parts.
- GREEN: the removals, compile-driven from proto regen.
- Then: gates with the allowlist entry if needed.
