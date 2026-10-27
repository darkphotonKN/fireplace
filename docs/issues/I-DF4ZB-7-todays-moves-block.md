---
id: I-DF4ZB-7
status: done
implements: FS-DF4ZB
blocked_by: [I-DF4ZB-4]
labels: [feature]
title: "FS-DF4ZB slice 7: Today's moves on the daily tab (read, resolve, tick, add as child, dismiss)"
---
Implements FS-DF4ZB §Requirements (R57–R62, R64, R65), §Edge States

## What to Build

Replace the daily tab's suggestions block with **Today's moves**, reading `getDailyBrief` for the
browser-local date. Recording feedback is I-DF4ZB-10 and the refresh action is I-DF4ZB-8; this
slice ships the block working without either.

- **Placement (R57).** Same spot on the **daily** tab, same show/hide toggle, same remembered
  `localStorage` preference (`showDailyInsights`). Build the block as its own component so
  `Todo.tsx` only mounts it and passes the handlers it already owns.
- **Read (R58).** `getDailyBrief(planId, localDate)` through the generated client in
  `client/src/api/insights.ts`. `localDate` is the browser-local `YYYY-MM-DD`; a later read after
  local midnight sends the new date.
- **Resolve (R59).** Moves are resolved against the plan's **long-term** items at render time. On
  the daily tab `todos` holds daily items only, so the block needs the long-term list (today's
  `fetchDailyInsights` already fetches it with `fetchChecklist(planId, 'longterm')`; reuse that,
  don't add a second loader). A move whose milestone is gone, archived, closed or no longer
  top-level is hidden; an `existing_step` move whose step is gone or archived is hidden. Each
  shown move names its milestone, its text and its reason. Never more than three.
- **Tick (R60).** An `existing_step` move has a checkbox. Ticking marks that step done through the
  existing item update, optimistic, reverted on failure. A ticked move stays in the block, shown
  done, for the rest of the day. A step ticked elsewhere renders done from live items.
- **Add (R61, L18).** A `new_step` move has **Add**: create a long-term task **as a child of the
  move's milestone** via the existing create endpoint (`createChecklistItem` with the parent id);
  the move then shows as added. A failed create keeps the move and shows the existing error. No
  daily-scope item is created; `addInsightAsTodo` and the old block are removed from `Todo.tsx`.
- **Dismiss (R62).** Every move can be dismissed; it hides. (Recording `dismissed` lands in
  I-DF4ZB-10.)
- **Empty and error states (R64, R65).** No milestones → an invitation to start one. Milestones
  but no moves → "nothing to suggest right now". Loading → a skeleton of the block. A failed read
  → a quiet retry, never an error banner over the checklist (this includes
  `BRIEF_DATE_OUT_OF_RANGE` and 503).

After this slice the client no longer calls `checklist-suggestion-daily`; removing the endpoint is
I-DF4ZB-9.

## Acceptance Criteria

- [ ] The daily tab shows "Today's moves" in place of the old block, with the same show/hide
      toggle and remembered preference.
- [ ] Each move names its milestone, its text and its reason; never more than three.
- [ ] Ticking an existing-step move marks that step done and reverts on failure; the move stays,
      shown done.
- [ ] Adding a new-step move creates a long-term child of that milestone; no daily-scope item is
      created; a failed create keeps the move and shows the existing error.
- [ ] Moves pointing at deleted, archived or closed items, or at a milestone no longer top-level,
      are not shown.
- [ ] Dismissing hides a move.
- [ ] No milestones → invitation; milestones but no moves → "nothing to suggest right now".
- [ ] Loading shows a skeleton; a failed read shows a quiet retry, not a banner.
- [ ] The client no longer calls `checklist-suggestion-daily`.
- [ ] Tests pass.

## Blocked By

I-DF4ZB-4 — `getDailyBrief` and its generated client types.

## Expected write set

- `client/src/components/TodaysMoves.tsx` (new) + `TodaysMoves.test.tsx`
- `client/src/lib/moves.ts` (new, proposed: resolve moves against long-term items) + test
- `client/src/components/Todo.tsx` **(hotspot)**: remove the old block, `dailyInsights` state,
  `addInsightAsTodo`; mount the new block. Keep the edit to the daily-tab region.
- `client/src/api/insights.ts` (add `getDailyBrief`; stop using `getDailyInsights`)

## Spec Reference

FS-DF4ZB §Requirements R57–R62 (recording deferred to I-DF4ZB-10), R64, R65; §Decisions L1, L18;
§User Stories 1–8, 13, 23; §Edge States ("User ticks a move's step elsewhere", "Move's milestone
archived or deleted after drawing", "Local midnight passes", "Client clock wrong", "Accept create
fails"); ADR-0014 (milestone is a client reading).

## TDD Approach

- RED: a table test for the resolver: a brief with moves pointing at a live milestone, an
  archived milestone, a closed milestone, a demoted (now child) milestone, an archived step and a
  done step → which are shown and which render done.
- GREEN: the resolver.
- Then: component tests for tick (optimistic + revert), add (child create with the parent id,
  never daily scope), dismiss, empty states, skeleton and quiet retry; a `Todo` test that the
  daily tab mounts the block and never calls `checklist-suggestion-daily`.
