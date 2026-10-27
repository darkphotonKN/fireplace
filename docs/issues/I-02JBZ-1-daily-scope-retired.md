---
id: I-02JBZ-1
status: done
implements: FS-02JBZ
blocked_by: [I-DF4ZB-3]
labels: [enhancement]
title: "FS-02JBZ slice 1: daily scope retired (migration, DAILY_SCOPE_RETIRED, no Daily tab)"
---
Implements FS-02JBZ §Requirements (R50, R51, R56, R58 `DAILY_SCOPE_RETIRED`, R91 tab and home parts), §API surface (createChecklist · updateChecklist error row), §Edge States

## What to Build

The daily scope goes, without data loss (D12). Retirement is kept separate from the day-plan
slices: nothing here needs the day plan, and nothing in the day plan needs this.

**plan-service.**

- **Migration 000022 (R50).** `UPDATE checklist_items SET scope = 'longterm' WHERE scope = 'daily'`,
  archived or not. Nothing deleted; `done`, dates, `parent_id`, `intention`, `archived` kept.
  Migrated rows are placed **after** the existing long-term siblings at the same level (same plan,
  same parent), keeping their relative order (S16). Write a down migration that is honest about
  being lossy (it cannot know which rows were daily) or make it a documented no-op; say which.
- **Refusal (R51).** `Create` with `scope = 'daily'`, and `Update` to `'daily'`, are refused with
  `DAILY_SCOPE_RETIRED` as `ErrorInfo` detail (I-DF4ZB-3's helper in `common/grpcerror`) on
  `InvalidArgument`. The gateway seam maps it to `400 · DAILY_SCOPE_RETIRED`. Listing and
  reordering with `scope = daily` stay accepted and find nothing (S15). The CHECK constraint and
  the proto `scope` field stay.
- Add `DAILY_SCOPE_RETIRED` to `common/errcode` (R58).

**Gateway.**

- `createChecklist` / `updateChecklist` keep their request shape (the `daily | longterm` enum
  stays, so the request is well-formed and the refusal is domain, ADR-0005). Document the new
  `400 · DAILY_SCOPE_RETIRED` row if the handlers list their errors; regenerate `openapi.yaml`
  and the client if that changes the document; run gates.
- **AI notes (R56, S13).** `internal/notes/service.go` `GenerateAINotes` reads the plan's
  long-term **task** items as context instead of `scope := "daily"`.

**Client (R91, tab and home parts only).**

- The legacy `/plans/[planId]` route's `<Todo />` loses its Daily | Long-term tab buttons and
  shows the long-term list (internal task type defaults to `longterm`).
- Home's `isTodays` (`client/src/lib/homeWork.ts`) drops the `item.scope === "daily"` clause.
- **Not here:** the plan page's Daily card (`<Todo fixedTaskType="daily" dailyAIOnly />` in
  `layout.tsx`) still hosts Today's moves until I-02JBZ-8 replaces it; leave Todo's
  `fixedTaskType = 'daily'` path working. The daily-reset toggle is I-02JBZ-2.

## Acceptance Criteria

- [ ] After migrating, no `daily` row exists; migrated rows keep `done`, dates, parents,
      intention and archived state, and sort after the existing long-term siblings in their old
      relative order.
- [ ] Creating or updating an item to `daily` returns `400 · DAILY_SCOPE_RETIRED` (asserted on
      the problem `code`, through the gateway seam).
- [ ] Listing with `scope=daily` returns an empty list, not an error.
- [ ] AI notes builds its context from long-term task items.
- [ ] The legacy route shows no Daily tab; home's Today ignores scope.
- [ ] `common/errcode` gains only `DAILY_SCOPE_RETIRED` in this slice.
- [ ] `make -C services/api-gateway gates` green (no breaking finding expected here).
- [ ] Tests pass.

## Blocked By

- I-DF4ZB-3 — the domain code must cross gRPC as `ErrorInfo`; plan-service uses its helper.

## Expected write set

- `services/plan-service/migrations/000022_retire_daily_scope.{up,down}.sql` (new)
- `services/plan-service/internal/checklistitem/service.go` (+ tests)
- `common/errcode/errcode.go` **(hotspot: every slice adding a code)**
- `services/api-gateway/internal/notes/service.go` (+ test)
- `services/api-gateway/internal/gateway/plan/typed_checklists.go` (only if documenting the error)
- `services/api-gateway/openapi.yaml` + `client/src/api/generated/schema.d.ts` **(hotspot, regenerated, only if the document changes)**
- `client/src/components/Todo.tsx` **(hotspot)**: tab buttons and internal default only
- `client/src/lib/homeWork.ts` (+ `homeWork.test.ts`)

## Spec Reference

FS-02JBZ §Decisions D12; §Requirements R50, R51, R56, R58, R91 (legacy tab, home Today rule);
§Spec defaults S13, S15, S16; §API surface (createChecklist · updateChecklist row);
§Acceptance Criteria (daily-scope retirement); §Edge States ("Existing daily items at
migration", "Old client creates a daily item", "Daily item that was done at migration").

## TDD Approach

- RED: a migration test against the test database: a plan with long-term siblings and daily rows
  (top-level and nested, one done, one archived) → after `up`, all long-term, order as R50.
- GREEN: the migration.
- Then: service table test for create/update with `daily` → `ErrorInfo` reason
  `DAILY_SCOPE_RETIRED`; gateway problem mapping; notes context; client tests for the legacy
  route and `isTodays`.
