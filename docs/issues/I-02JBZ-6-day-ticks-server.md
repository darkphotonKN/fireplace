---
id: I-02JBZ-6
status: done
implements: FS-02JBZ
blocked_by: [I-02JBZ-4]
labels: [feature]
title: "FS-02JBZ slice 6: day ticks (tick/untick an entry, bind an added step, pinned through redraws)"
---
Implements FS-02JBZ §Requirements (R12, R24–R30, R58 `DAY_PLAN_ENTRY_NOT_TICKABLE`), §API surface (tickDayPlanEntry, untickDayPlanEntry, DayPlanEntry.ticked)

## What to Build

"Done for today" on an entry, server and gateway. The revision ladder hooks onto ticks in
I-02JBZ-7; this slice must leave that hook (a tick result the ladder can react to, and an untick
that can undo it) without scheduling anything.

**insights-service.**

- **Storage (R24, D1).** Ticks per (user, plan, item, date), date = the day plan's date, with the
  entry reference.
- **Tick (R25, S5).** `TickDayPlanEntry(entryId, itemId?, user_id)`. `existing_item` entry: no
  `itemId` or its own. Unbound `new_step` entry: `itemId` required; verify through plan-service
  that it is a non-archived **task**, a **child of the entry's parent**, in the **same plan**;
  then bind (entry becomes `existing_item` for that item, text and reason kept) and tick.
- **Refusals (R26).** `DAY_PLAN_ENTRY_NOT_TICKABLE` (`ErrorInfo` on `FailedPrecondition` → 400)
  when: unbound `new_step` without a valid `itemId`; the entry's item is archived or deleted
  (check plan-service); `itemId` contradicts a bound entry. Unknown or foreign entry →
  `NotFound`. Add the code to `common/errcode` (R58).
- **Idempotent (R27).** Re-tick returns the entry unchanged.
- **Independent of `done` (R28).** Never call plan-service to change `done`.
- **Untick (R29).** `UntickDayPlanEntry(entryId)` removes the tick; unticking an unticked entry
  succeeds.
- **Date window (R30, S17).** Tick/untick only while the entry's plan date is within server UTC
  ±1 day; otherwise `BRIEF_DATE_OUT_OF_RANGE`.
- **Pinned through redraws (R12, R39, S4).** Plan my day (AI and rules) keeps every entry ticked
  today in place with its group, text and tick; the prompt lists them as fixed and done; the
  model fills `6 − ticked` slots, joining a ticked group by exact label or opening new ones within
  3 groups; R9 applies to what it adds. Ticked items join the hash input (R17).
- **Read model.** `DayPlanEntry.ticked` reflects the tick for the plan's date.
- **RPCs** `TickDayPlanEntry`, `UntickDayPlanEntry`.

**Gateway.** Typed Huma `tickDayPlanEntry` (`PUT /api/insights/day-plan/entries/{id}/tick`, body
`{itemId?}` with `omitempty`, `200` DayPlanEntry) and `untickDayPlanEntry` (`DELETE …/tick`,
`204`). Regenerate `openapi.yaml` and the client; gates green.

## Acceptance Criteria

- [ ] Ticking an `existing_item` entry stores the tick and does not change the item's `done`;
      ticking again changes nothing.
- [ ] Unbound `new_step` without `itemId`, or with an item that is not a non-archived task child
      of its parent in this plan → `400 · DAY_PLAN_ENTRY_NOT_TICKABLE`; with a valid one the entry
      is bound and ticked.
- [ ] Entry whose item was archived or deleted → `400 · DAY_PLAN_ENTRY_NOT_TICKABLE`; another
      user's entry → `404 · NOT_FOUND`.
- [ ] Untick removes the tick; untick of an unticked entry → `204`.
- [ ] Out-of-window plan date → `400 · BRIEF_DATE_OUT_OF_RANGE`.
- [ ] Plan my day (AI and rules) keeps ticked entries in place; never more than 6 entries or 3
      groups; changing ticks changes the hash.
- [ ] `common/errcode` gains only `DAY_PLAN_ENTRY_NOT_TICKABLE` here.
- [ ] Regenerated contract; gates green.
- [ ] Tests pass.

## Blocked By

- I-02JBZ-4 — the day plan and its entries.

Not blocked by I-02JBZ-5, but if I-02JBZ-5 has landed, pin into the rules draw too (R39); if not,
I-02JBZ-5 does it.

## Expected write set

- `common/api/proto/insights/insights.proto` + regenerated stubs **(hotspot)**
- `common/errcode/errcode.go` **(hotspot)**
- `services/insights-service/internal/dayplan/` (ticks repository, service, prompt pinning) (+ tests)
- `services/insights-service/internal/insights/plan_client.go` (item lookup, if not already there)
- `services/insights-service/migrations/00000N_create_day_ticks.{up,down}.sql` (new)
- `services/insights-service/config/services.go` **(hotspot)**
- `services/api-gateway/internal/insights/typed_insights.go` (+ test), `services/api-gateway/internal/gateway/insights/client.go`
- `services/api-gateway/openapi.yaml` + `client/src/api/generated/schema.d.ts` **(hotspot, regenerated)**

## Spec Reference

FS-02JBZ §Decisions D1, D3, D4; §Requirements R12, R17 (ticks in hash), R24–R30, R39, R58; §Spec
defaults S4, S5, S17; §API surface (tickDayPlanEntry, untickDayPlanEntry, DayPlanEntry);
§User Stories 12, 15, 17, 18, 31; §Edge States ("Entry's item archived or deleted", "Tick on
yesterday's plan", "Shared plan").

## TDD Approach

- RED: service test: tick an `existing_item` entry → one tick row, plan-service `UpdateItem` never
  called.
- GREEN: tick path.
- Then: bind-on-tick table (valid child, wrong parent, other plan, note, archived); idempotency;
  untick; date window; pinning golden test (prompt lists ticked entries; merged output bounded).
