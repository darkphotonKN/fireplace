---
id: I-02JBZ-8
status: done
implements: FS-02JBZ
blocked_by: [I-02JBZ-4, I-DF4ZB-8]
labels: [feature]
title: "FS-02JBZ slice 8: the Day plan section replaces the plan page's Daily card"
---
Implements FS-02JBZ §Requirements (R62–R74), §Amendments (A5)

## What to Build

Today's moves becomes the **Day plan** section (D11, A5), reading the day plan instead of the
brief. Ticks are I-02JBZ-9, revision UI I-02JBZ-10.

- **Placement (R62).** In `client/src/app/plan/[planId]/layout.tsx`, replace
  `<Todo fixedTaskType="daily" dailyAIOnly />` in the Daily card with the new section component
  (proposed `client/src/components/DayPlanSection.tsx`), same card styling. Evolve I-DF4ZB-7's
  Today's moves component into it rather than writing a second one; it is no longer mounted
  from `Todo.tsx`. With the last daily `<Todo>` mount gone, delete Todo's daily path:
  `fixedTaskType = 'daily'`, `dailyAIOnly`, daily fetches and any Today's-moves mount.
- **Header and tooltip (R63).** "Day plan" with an "i" button and hover tooltip explaining daily
  progression (plan your day, tick what you did today, revisions and unfinished items come back),
  same markup pattern as the plan-focus tooltip in the same file (mind the existing stacking-
  context comment: the title card is `relative z-10` so its tooltip paints above this card).
- **Read (R64).** `getDayPlan(planId, localDate)` on mount and when the browser-local date changes.
- **Not planned (R65).** A short line and **Plan my day** → `planMyDay`, loading inside the
  section only.
- **Planned (R66, R67).** Groups in order with labels; each entry shows its milestone name (top-
  level ancestor resolved from loaded long-term items), text, reason, minutes when present.
  Resolution hides entries whose item is deleted/archived and `new_step` entries whose parent is
  gone/archived/closed; a done item shows completed (still a row; ticks land in I-02JBZ-9).
  Put resolution in a pure helper (proposed `client/src/lib/dayPlan.ts`, evolving I-DF4ZB-7's
  `lib/moves.ts`).
- **Plan again (R68).** Shows `refreshesRemaining`, retires at 0, "nothing has changed" when
  `generatedAt` is unchanged, `429 · BRIEF_REFRESH_LIMIT_REACHED` retires the action and keeps the
  plan; switch on `code`.
- **Basic plan (R69).** `source = rules` → "basic plan" label with a one-line reason, Plan my day
  still offered.
- **Not reachable (R70).** A 404 renders no section (the page's own not-here state applies).
- **Loading / error (R71).** Skeleton; quiet retry on failure, never a banner.
- **Toggle (R72).** Keep the show/hide toggle and its `localStorage` key.
- **Add (R73).** `new_step` → Add creates a long-term child of the entry's parent via the existing
  create endpoint; the entry then shows added. Failed create keeps it and shows the existing
  error. Never daily scope (R84).
- **Dismiss (R74, S18).** Hides for the session; records nothing.

## Acceptance Criteria

- [ ] The plan page shows "Day plan" where the Daily card was, with the "i" tooltip; `Todo.tsx`
      has no daily path and the layout mounts no daily `<Todo>`.
- [ ] Not planned → Plan my day → groups render with labels, milestone names, text, reasons and
      minutes.
- [ ] Entries pointing at deleted/archived items, or new steps under a gone/closed parent, are not
      shown.
- [ ] Plan again shows the count, retires at 0, reports "nothing has changed", and retires on a
      429 by `code`.
- [ ] A rules plan is labelled "basic plan".
- [ ] Add creates a long-term child of the parent; failure keeps the entry; no daily item.
- [ ] Skeleton while loading; quiet retry on failure; show/hide preference remembered.
- [ ] The client no longer calls `getDailyBrief`/`refreshDailyBrief`.
- [ ] Tests pass.

## Blocked By

- I-02JBZ-4 — `getDayPlan`/`planMyDay` and their generated types.
- I-DF4ZB-8 — the Today's moves block (with refresh) this evolves.

## Expected write set

- `client/src/components/DayPlanSection.tsx` (new, evolved from `TodaysMoves.tsx`) + test
- `client/src/lib/dayPlan.ts` (new, evolved from `lib/moves.ts`) + test
- `client/src/app/plan/[planId]/layout.tsx` **(hotspot: plan page layout)** (+ `layout.test.tsx`)
- `client/src/components/Todo.tsx` **(hotspot)**: delete the daily path
- `client/src/components/TodaysMoves.tsx`, `client/src/lib/moves.ts` (removed or folded in)
- `client/src/api/insights.ts` **(hotspot)**: add `getDayPlan`, `planMyDay`; stop using the brief calls
- Todo test files that pass `fixedTaskType="daily"` or `dailyAIOnly` (mechanical)

## Spec Reference

FS-02JBZ §Decisions D7, D11; §Amendments A5; §Requirements R62–R74, R84; §Spec defaults S18;
§User Stories 1–11, 18, 25, 26, 30; §Edge States ("First visit of the day", "Plan has no open
long-term task", "Fourth counted draw", "Entry's item archived or deleted", "Local midnight
passes", "Insights-service down").

## TDD Approach

- RED: a resolver table test (live, archived, deleted, done item; new step under live, closed and
  archived parents) → shown / hidden / completed.
- GREEN: `lib/dayPlan.ts`.
- Then: section tests for not-planned → Plan my day, planned render, Plan again (count, retire,
  unchanged, 429), basic label, Add, dismiss, skeleton and quiet retry; a layout test that no
  daily `<Todo>` is mounted and the tooltip exists.
