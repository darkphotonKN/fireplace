---
id: I-FSFEC-2
status: done
implements: FS-FSFEC
blocked_by: [I-FSFEC-1]
labels: [feature]
title: "FS-FSFEC slice 2: the progress strip on home's Continue cards"
---
Implements FS-FSFEC §Requirements (R23–R28), §Edge States

## What to Build

Each *Continue where you left off* card on home shows its plan's milestone strip, from data home
already holds.

**Derivation, no new fetch (R24).** `useHomeData` already fetched every item of the three focused
plans (ADR-0013). Run them through slice 1's `milestones.ts`, filtered per R1: long-term scope,
non-archived, top-level, task. The fetch carries every scope, so the scope filter matters here
in a way it does not inside `Todo`. Home must still issue exactly `1 + min(planCount, 3)`
requests.

**Ticks from Today count (R25).** Rows ticked from *Today* this visit already count toward the
card's done/total through `tickedIds` (`progressOf` in `client/src/lib/homePlans.ts`). The strip
must read the same: one derivation, not two blocks disagreeing about one plan.

**What the card shows (R23, R26).** With at least one milestone: the strip and the milestone
count in place of done/total. With none: exactly today's done/total. Loading or failed: today's
empty slot, never an empty strip or a zero.

**No height change (R27).** The strip sits in the card's existing footer row, between the type
pill and the count, so it settling in moves nothing under the cursor (FS-KSJFR R22).

**Neutral (R28).** Draw the strip in neutral tokens. Coral on the dashboard belongs to the Today
block alone (FS-KSJFR R49).

## Acceptance Criteria

- [ ] A Continue card for a plan with milestones shows its strip and "N of M milestones".
- [ ] A plan with no milestones (none, only notes, or only daily items) shows done/total as today.
- [ ] A plan whose items are loading shows the empty slot; one whose items failed keeps it empty.
- [ ] Ticking a milestone or a step from Today updates that plan's strip in the same visit.
- [ ] Home issues `1 + min(planCount, 3)` requests at 1, 3, 4 and 12 plans.
- [ ] A card's height is the same before and after its items land.
- [ ] The strip uses neutral tokens; no coral outside the Today block.
- [ ] No raw hex, `rgb(...)` literal or `*-gray-*` utility in the files this slice touches.
- [ ] Tests pass.

## Blocked By

I-FSFEC-1 — needs `milestones.ts` and `MilestoneStrip`.

## Spec Reference

FS-FSFEC §Requirements R23–R28 (dashboard strip, charter default C10); §Edge States (items
loading or failed, plan with only daily items). ADR-0013 owns the request cap; FS-KSJFR R22 and
R49 own height stability and the coral rule.

## TDD Approach

- RED: a `Dashboard` test with a stubbed API where one focused plan has daily items, archived
  items and two long-term milestones; assert the card reads "0 of 2 milestones" and the request
  count is unchanged.
- GREEN: derive in the card from the fetched items.
- Then: ticking from Today moves the strip; failed and no-milestone plans render as today.
