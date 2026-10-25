---
id: I-02JBZ-2
status: done
implements: FS-02JBZ
blocked_by: [I-02JBZ-1]
labels: [enhancement]
title: "FS-02JBZ slice 2: remove the nightly daily reset and the daily-reset toggle"
---
Implements FS-02JBZ §Requirements (R52–R55, R91 settings part), §API surface (removed toggleDailyReset)

## What to Build

With no daily rows left (I-02JBZ-1), the reset machinery has nothing to do. Remove it end to end.

- **plan-service (R52).** Remove `ChecklistService.DailyReset` and `PlanService.ToggleDailyReset`
  from `common/api/proto/plan/{checklist,plan}.proto` with their request/response messages;
  regenerate (`make -C common`, never hand-edit). Remove the handlers, `service.DailyReset`,
  `service.ToggleDailyReset` and `repository.BulkResetDailyItems`, with their tests.
- **Gateway (R53, R54).** Remove `internal/jobs/daily_reset_job.go`, the job wiring in
  `config/routes.go` (and `internal/jobs/manager.go` if nothing else uses it), the adapter's
  `ResetDailyItems`, the plan client's `DailyReset` and `ToggleDailyReset`, and the
  `toggleDailyReset` operation in `internal/gateway/plan/typed_plans.go`. Regenerate
  `openapi.yaml` and the client. The breaking gate flags the removed operation: add the finding
  to the repo-root `.oasdiff-ignore` citing FS-02JBZ, per `docs/agents/contract.md`.
- **Kept (R55).** `plans.daily_reset`, `createPlan`/`updatePlan` accepting `dailyReset`, and
  plans returning it. No effect any more; dropping them is later cleanup.
- **Client (R91, settings part).** Remove the "Refresh daily tasks" setting, `dailyReset` state,
  `handleToggleDailyReset` and `isTogglingReset` from `Todo.tsx`; remove `toggleDailyReset` from
  `client/src/api/plans.ts`; drop `toggleDailyReset` from the `@/api/plans` mocks in the Todo test
  files. Sharing and "View archived tasks" stay in settings.

## Acceptance Criteria

- [ ] `DailyReset` and `ToggleDailyReset` are gone from the proto, generated code, plan-service
      and the gateway client; no cron job is scheduled at gateway boot.
- [ ] `PATCH /api/plans/{id}/toggle-daily-reset` returns 404/405 (route gone); the breaking-gate
      finding is allowlisted citing FS-02JBZ; `make -C services/api-gateway gates` is otherwise
      green.
- [ ] `createPlan`/`updatePlan` still accept `dailyReset`; `getPlan` still returns it.
- [ ] The settings view has no daily-reset toggle; sharing and archived still work.
- [ ] Generated code regenerated, not edited.
- [ ] Tests pass.

## Blocked By

- I-02JBZ-1 — rows are migrated before the reset stops (otherwise daily rows done at that moment
  would never reset again while still shown as daily); also serializes `Todo.tsx` and
  plan-service.

## Expected write set

- `common/api/proto/plan/plan.proto`, `common/api/proto/plan/checklist.proto` + regenerated stubs **(hotspot: plan proto)**
- `services/plan-service/internal/checklistitem/{handler,service,repository}.go` (+ tests)
- `services/plan-service/internal/plan/{handler,service}.go` (+ tests)
- `services/api-gateway/internal/jobs/` (deleted), `services/api-gateway/config/routes.go`
- `services/api-gateway/internal/gateway/plan/{adapter,client,typed_plans}.go` (+ tests)
- `services/api-gateway/openapi.yaml` + `client/src/api/generated/schema.d.ts` **(hotspot, regenerated)**
- `.oasdiff-ignore` (repo root) **(hotspot: shared allowlist)**
- `client/src/components/Todo.tsx` **(hotspot)**: settings region only
- `client/src/api/plans.ts`
- `client/src/components/Todo.*.test.tsx` mocks (mechanical)

## Spec Reference

FS-02JBZ §Decisions D12; §Requirements R52–R55, R91 (settings toggle); §API surface
(~~toggleDailyReset~~, plane 2 removals); §Acceptance Criteria (daily-scope retirement); ADR-0002
(removing a serialized endpoint; breaking gate allowlist).

## TDD Approach

- RED: a gateway route test asserting `toggle-daily-reset` is not registered and that the job
  manager starts no reset job; a Todo test asserting no "Refresh daily tasks" control for an owner.
- GREEN: the removals.
- Then: compile-driven deletion through proto regen; gates with the allowlist entry.
