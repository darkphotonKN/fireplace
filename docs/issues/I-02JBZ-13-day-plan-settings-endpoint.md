---
id: I-02JBZ-13
status: done
implements: FS-02JBZ
blocked_by: []
labels: [feature]
title: "FS-02JBZ slice 13: day plan settings stored, returned on the day plan and saved by setDayPlanSettings"
---
Implements FS-02JBZ §Change: Hours a day (§Requirements R93–R95, R101–R107, §API surface)

## What to Build

The foundation of "Hours a day", end to end through every server layer and the generated client,
with no behaviour change to drawing yet. A per (user, plan) **day plan setting** (`dailyMinutes`
30–720 in 30-minute steps or unset, plus `asked`) is stored in `fireplace_insights`, returned as
`settings` on every DayPlan, and saved through one new operation.

- **Migration 000014** (`services/insights-service/migrations/000014_create_day_plan_settings.{up,down}.sql`):
  table `day_plan_settings`, PK (plan_id, user_id), nullable `daily_minutes` with a CHECK for
  30–720 and a multiple of 30, timestamps with the `update_updated_at_column()` trigger. A row
  means asked; NULL minutes means asked with no budget (R95).
- **insights-service** (`internal/dayplan`): a settings store (repository + its test against the
  real database, like `repository_test.go`); `Service.SetDayPlanSettings(planID, userID, *minutes)`
  checks plan access through plan-service as the caller (as `SetItemRevisions` does, R106),
  refuses an out-of-range or off-step value with `InvalidArgument` (R105), upserts, and returns the
  saved setting. It never draws, never touches the counter, the ledger or the stored day plan
  (R103). `GetDayPlan` and `PlanMyDay` attach the caller's setting to every DayPlan they return,
  in every status (R101). Wire the store in `config/services.go`.
- **Proto** (`common/api/proto/insights/insights.proto`): `DayPlanSettings { optional int32
  daily_minutes; bool asked; }`, `DayPlan.settings`, RPC `SetDayPlanSettings` (request `plan_id`,
  `user_id`, `optional int32 daily_minutes`; response carries `DayPlanSettings`). `make proto-gen`;
  never hand-edit the generated Go.
- **gRPC handler** (`internal/insights/handler.go`): the new RPC and the settings mapping on the
  two DayPlan responses, through the existing `ErrorInfo` seam.
- **Gateway** (`internal/gateway/insights/client.go`, `internal/insights/typed_day_plan.go`):
  `DayPlanResponse` gains `settings` (always present); new typed Huma op `setDayPlanSettings`,
  `PUT /api/insights/day-plan/settings`, body `planId` (uuid, required) and `dailyMinutes`
  (integer, **required and nullable**, minimum 30, maximum 720, multipleOf 30; contract-patterns
  §5, precedent `parentId`), response `200` DayPlanSettings, guarded by the same `authorizer` as
  the other day plan ops. Errors: 404, 422, plus the common set.
- **Contract**: `make openapi client` in `services/api-gateway`; commit `openapi.yaml` and
  `client/src/api/generated/` together; `make gates` green (additive).
- **Client api module** (`client/src/api/insights.ts`): `DayPlanSettings` type from the generated
  schema and a `setDayPlanSettings(planId, dailyMinutes | null)` wrapper that throws the
  problem's `code` like its siblings. No UI in this slice.

## Acceptance Criteria

- [ ] Migration 000014 up creates `day_plan_settings`; its CHECK refuses 45 and 750; down drops it.
- [ ] `getDayPlan` / `planMyDay` return `settings: {asked: false}` (no `dailyMinutes`) for a user
      who never saved, in `not_planned` and `planned` states.
- [ ] `setDayPlanSettings` with 120 returns `200 {dailyMinutes: 120, asked: true}`; the next
      `getDayPlan` carries it. With `null` it returns `200 {asked: true}` and the next read has no
      `dailyMinutes`.
- [ ] 0, 29, 45, 721, 750, `1.5`, a string, a missing `dailyMinutes`, an unknown member or a
      malformed `planId` return `422 · VALIDATION_FAILED` at the gateway, never forwarded.
- [ ] The RPC refuses an off-step value with `InvalidArgument` when called directly.
- [ ] A plan the caller can't access returns `404 · NOT_FOUND`.
- [ ] Saving makes no model call, no rules draw, no ledger row, and leaves the stored day plan and
      `refreshesRemaining` unchanged.
- [ ] On a shared plan, one member's setting doesn't change another's.
- [ ] `openapi.yaml`, `client/src/api/generated/` and the proto Go are regenerated, never
      hand-edited; `make gates` green.
- [ ] Tests pass (`make test` in insights-service and api-gateway; `npm test` in client).

## Blocked By

None

## Spec Reference

FS-02JBZ §Change: Hours a day: R93–R95 (the setting and its storage), R101–R107 (reads, the new
operation, validation, access, RPC and contract), §API surface; Edge States "Off-step or
out-of-range value", "Plan not accessible", "Shared plan".

## TDD Approach

- RED: a service test that `GetDayPlan` for a fresh (user, plan) returns settings with
  `Asked == false` and nil minutes, then `SetDayPlanSettings(…, 120)` followed by `GetDayPlan`
  returns 120 and asked.
- GREEN: the settings store and the attach step in `present`/the read path.
- Then the gateway: a typed-op test table for the 422 cases and the 404.

## Files expected to touch

- `services/insights-service/migrations/000014_create_day_plan_settings.up.sql`, `.down.sql`
- `services/insights-service/internal/dayplan/` (new `settings.go`, `settings_repository.go` and
  tests; `service.go`, `model.go`, `fakes_test.go`, `service_test.go`)
- `services/insights-service/internal/insights/handler.go`, `handler_day_plan_test.go`
- `services/insights-service/config/services.go`
- `common/api/proto/insights/insights.proto` and its generated Go
- `services/api-gateway/internal/gateway/insights/client.go`
- `services/api-gateway/internal/insights/typed_day_plan.go`, `typed_day_plan_test.go`
- `services/api-gateway/openapi.yaml` (generated)
- `client/src/api/generated/schema.d.ts` (generated), `client/src/api/insights.ts`
