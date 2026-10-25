---
id: I-02JBZ-11
status: done
implements: FS-02JBZ
blocked_by: [I-02JBZ-4]
labels: [feature]
title: "FS-02JBZ slice 11: Day plan roll-up on home (cached reads only, rule picks, ADR-0013 bound 4 → 7)"
---
Implements FS-02JBZ §Requirements (R85–R90)

## What to Build

A model-free home block (D10, ADR-0019 clause 2).

- **Reads (R85, R90).** In `useHomeData` (or a sibling hook), after the focused plans are known,
  call `getDayPlan(planId, localDate)` for each of ADR-0013's three focused plans, in parallel and
  settled independently like the checklist fetches. **Never** `planMyDay`. Total requests on load:
  1 + 3 + 3 = 7. Update the `useHomeData` doc comment and say plainly that ADR-0013's bound rises
  from 4 to 7 while its decision (same three plans, shipped endpoints, no aggregate) holds.
- **Stored plan (R86).** First two entries in plan order that resolve against home's items (reuse
  `lib/dayPlan.ts` resolution); ticked entries shown ticked.
- **No plan / failed read (R87).** Two client-side rule picks from home's items: overdue → due
  soonest → next open step of the most recently updated milestone (`updatedAt`); no revisions.
  Marked "basic". Proposed `client/src/lib/dayPlanRollup.ts`.
- **Counts (R88).** Two per plan, up to 6; a single focused plan fills the block (up to 6). Rows
  are read-only and link to `/plan/<planId>`.
- **Empty (R89).** A plan with nothing contributes nothing; no rows → no block.
- **Placement (S11).** Above the Today block in `Dashboard.tsx`.

## Acceptance Criteria

- [ ] Home issues exactly 7 requests on load with three focused plans and never calls
      `planMyDay`.
- [ ] Two entries per plan from stored day plans; six from a single plan.
- [ ] Plans without a day plan today (or whose read failed) contribute rule picks marked basic.
- [ ] Rows are read-only and link to their plan; ticked entries show ticked.
- [ ] No rows → no block.
- [ ] Tests pass.

## Blocked By

- I-02JBZ-4 — `getDayPlan` and its generated types.

Hotspot note: shares `client/src/api/insights.ts` and `lib/dayPlan.ts` with I-02JBZ-8; if run
after I-02JBZ-8, reuse its `getDayPlan` wrapper and resolver instead of adding parallel ones.

## Expected write set

- `client/src/components/home/DayPlanRollup.tsx` (new) + test
- `client/src/lib/dayPlanRollup.ts` (new) + test
- `client/src/components/home/useHomeData.ts` (+ test)
- `client/src/components/home/Dashboard.tsx` (+ `Dashboard.test.tsx`)
- `client/src/api/insights.ts` **(hotspot)** (only if `getDayPlan` isn't there yet)
- `client/src/lib/dayPlan.ts` **(hotspot, shared with I-02JBZ-8)** (only if not there yet)

## Spec Reference

FS-02JBZ §Decisions D10; §Requirements R85–R90; §Spec defaults S11; §User Stories 27–29;
§Edge States ("Home: one focused plan", "Home: a plan's day plan read fails", "Home: plan with no
items"); ADR-0013 (bound 4 → 7, decision unchanged); ADR-0019 clause 2.

## TDD Approach

- RED: a table test for `dayPlanRollup`: three plans (stored plan, not planned, failed read), one
  plan alone, a plan with no items → expected rows, counts and basic marks.
- GREEN: the pure function.
- Then: hook test counting requests (7) and asserting `planMyDay` is never called; dashboard
  render and placement.
