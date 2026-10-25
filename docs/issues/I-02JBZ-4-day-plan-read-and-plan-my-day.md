---
id: I-02JBZ-4
status: done
implements: FS-02JBZ
blocked_by: [I-DF4ZB-6]
labels: [feature]
title: "FS-02JBZ slice 4: getDayPlan (never generates) and planMyDay (grouped, sized, free first draw, shared cap)"
---
Implements FS-02JBZ §Requirements (R1–R11, R13–R23, R57, R59, R61), §Amendments (A1, A3), §API surface (getDayPlan, planMyDay, DayPlan, DayPlanGroup, DayPlanEntry)

## What to Build

The day plan's AI path end to end, server and gateway. Ticks are I-02JBZ-6, revisions I-02JBZ-7,
the rules fallback I-02JBZ-5. Until I-02JBZ-5 lands, a failed generation answers
`503 · SERVICE_UNAVAILABLE` like DF4ZB R54.

**insights-service** (new package, proposed `internal/dayplan/`, reusing I-DF4ZB-4..6's parts
rather than copying them):

- **Storage (R1–R5, S1).** A day plan per (plan, user, date), **beside** the brief, in
  `fireplace_insights`: `source` (`ai` | `rules`), drawn-at, plan-state hash, groups (label 1–40)
  and entries (`existing_item` | `new_step`, item id, parent item id, text, reason, minutes 5–240
  or null, revision 1–3 or null). Entry ids are uuids usable as suggestion ids later (FS O1).
  Leave room for ticks and revisions (I-02JBZ-6/7) without designing them here.
- **Read (R14).** `GetDayPlan` returns the stored plan or `not_planned`. **No model call, no rules
  draw, no copy-forward**, ever (A1, ADR-0019).
- **Plan my day (R6, R7, R13).** One strict structured call, generator kind `day_plan` in the
  ledger, own model/ceiling config (default `gpt-5-mini`, minimal reasoning). Prompt in DF4ZB R2
  order; static rules ask for topic-affinity groups with short labels, nearer due dates and
  overdue first, a minutes estimate per entry, ≤6 entries in ≤3 groups (D2, D14). Plan data adds
  plan type and focus (fetch `plan_type` from `GetPlan`; `PlanContext` lacks it today), today's
  date, and placeholders for due revisions and ticked entries (empty until I-02JBZ-6/7). The
  schema has **no** revision field (R10).
- **Candidates and validation (R8, R9, S2, S14).** `existing_item` → any open, non-archived,
  long-term task, top-level or child; `new_step` → parent is an open top-level long-term task,
  text not duplicating its children. Drop (never repair): bad item, duplicate item, duplicate
  new-step text, empty reason, fourth group, seventh entry; an out-of-range `minutes` is nulled,
  not dropped; empty groups dropped; zero entries is valid.
- **Replace atomically (R11).**
- **Free draw and cap (R15, R16, R19, R20, A3).** The first successful AI draw for the key is
  free; later successful AI draws count on **I-DF4ZB-6's counter for the same key** (shared with
  `refreshDailyBrief`), max 3, fourth → `BRIEF_REFRESH_LIMIT_REACHED` on `ResourceExhausted`,
  plan untouched. A failed generation spends nothing. `refreshesRemaining` 0–3 (3 before the
  free draw).
- **Gate (R17, S3).** Hash = DF4ZB R19's tree + plan type + focus + date (+ due revisions and
  ticked items once I-02JBZ-6/7 exist; build the hash input so they slot in). Equal hash on a
  stored `ai` plan → return it, no call, no count. A stored `rules` plan never satisfies the gate.
- **No candidates (R18).** Store an empty planned `ai` plan, no call, no count, free draw unused.
- **Concurrency (R21).** Two requests on one key: one plan stored, both receive it, at most one
  draw's allowance spent; a lost generation is still recorded.
- **Date (R22).** Outside server UTC ±1 day → `BRIEF_DATE_OUT_OF_RANGE`. **Access (R23).**
  Through plan-service `GetPlan` as today; per user.
- **RPCs** `GetDayPlan`, `PlanMyDay` in `insights.proto`; names say item / parent item, never
  milestone (R61).

**Gateway (R57, R59).** Typed Huma `getDayPlan` (`GET /api/insights/day-plan?planId&date`) and
`planMyDay` (`POST /api/insights/day-plan`, body `{planId, date}`), responses exactly as
§API surface (`groups` never null; `ticked` false and `revisionsEnabled` absent until later
slices). Identity from the JWT. Regenerate `openapi.yaml` and the client; gates green.

`getDailyBrief`/`refreshDailyBrief` keep working untouched; they are retired in I-02JBZ-12.

## Acceptance Criteria

- [ ] `getDayPlan` with nothing stored returns `not_planned`, empty `groups`, and makes no model
      call (assert the fake generator was never invoked).
- [ ] `planMyDay` makes at most one structured call and stores ≤6 entries in ≤3 non-empty groups
      with labels 1–40 chars; a second `getDayPlan` returns it with no call.
- [ ] Prompt carries plan type, focus and today's date; static rules byte-identical across calls.
- [ ] Invalid entries dropped per R9; bad `minutes` nulled; nothing repaired or regenerated.
- [ ] First successful AI draw leaves `refreshesRemaining` 3; three more lower it to 0; the next
      returns `429 · BRIEF_REFRESH_LIMIT_REACHED` with the plan unchanged.
- [ ] A counted brief refresh and a counted Plan my day on one key share the cap.
- [ ] Unchanged hash → stored plan returned, no call, no count.
- [ ] Concurrent Plan my day on one key → one stored plan, both receive it.
- [ ] No candidates → empty planned plan, no call, no count.
- [ ] Out-of-window date → `400 · BRIEF_DATE_OUT_OF_RANGE` on both ops.
- [ ] Usage recorded as `day_plan`.
- [ ] No new path, type or proto name contains "milestone".
- [ ] `openapi.yaml` and `client/src/api/generated` regenerated; gates green.
- [ ] Tests pass.

## Blocked By

- I-DF4ZB-6 — the shared counter and cap; transitively the plan tree, prompt assembly, structured
  call, validation, ledger (I-DF4ZB-4), hash (I-DF4ZB-5) and error seam (I-DF4ZB-3).

## Expected write set

- `common/api/proto/insights/insights.proto` + regenerated stubs **(hotspot: insights proto)**
- `services/insights-service/internal/dayplan/` (new: model, prompt, schema, validation, service, repository) (+ tests)
- `services/insights-service/internal/guided/` (extract reusable parts only if needed; no behavior change to the brief)
- `services/insights-service/internal/insights/plan_client.go` (plan type)
- `services/insights-service/internal/insights/handler.go` or a dayplan handler
- `services/insights-service/migrations/00000N_create_day_plans.{up,down}.sql` (new)
- `services/insights-service/config/services.go` **(hotspot)**, `.env.example`
- `services/api-gateway/internal/insights/typed_insights.go` (+ test), `services/api-gateway/internal/gateway/insights/client.go`
- `services/api-gateway/openapi.yaml` + `client/src/api/generated/schema.d.ts` **(hotspot, regenerated)**

## Spec Reference

FS-02JBZ §Decisions D1, D2, D7, D13, D14, D15; §Amendments A1, A3; §Requirements R1–R11, R13–R23,
R57, R59, R61; §Spec defaults S1, S2, S3, S14; §API surface (getDayPlan, planMyDay, DayPlan,
DayPlanGroup, DayPlanEntry); §User Stories 1–11, 31; §Edge States ("First visit of the day",
"Plan has no open long-term task", "Model returns only invalid entries", "Fourth counted draw",
"Two tabs press Plan my day at once", "Plan edited after the plan was drawn", "Local midnight
passes", "Client clock off"); ADR-0019 clauses 1–2.

## TDD Approach

- RED: service test with a fake plan client, a fake generator and an in-memory repo:
  `GetDayPlan` on an empty key → `not_planned` and zero generator calls.
- GREEN: read path.
- Then: Plan my day happy path (one call, stored, validated); validation table (each drop rule);
  free draw then counted draws then 429 against the shared counter; gate hit; no-candidates;
  concurrency on the repository; date window; prompt golden test; Huma handlers and regen.
