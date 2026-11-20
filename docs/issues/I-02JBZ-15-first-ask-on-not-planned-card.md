---
id: I-02JBZ-15
status: done
implements: FS-02JBZ
blocked_by: [I-02JBZ-13]
labels: [feature]
title: "FS-02JBZ slice 15: the not-planned card asks how much time a day, once per plan"
---
Implements FS-02JBZ §Change: Hours a day (§Requirements R108–R113)

## What to Build

The first ask, client only, in `client/src/components/DayPlanSection.tsx` (`NotPlanned`). When
the day plan is `not_planned`, `settings.asked` is false and Plan my day isn't retired, the card
asks "How much time do you have for this plan each day?" with chips **30m · 1h · 2h · 3h · 4h**
and a quiet "Skip, plan anyway", in place of its line and button (R108, R112).

- A chip calls `setDayPlanSettings(planId, minutes)` then `planMyDay`, one click, with the
  section's planning skeleton throughout; Skip does the same with `null` (R109). The returned
  DayPlan's `settings` replaces the state, so the ask is gone.
- Save fails (any code but `NOT_FOUND`): still plans; the PlanBar notice line says "Couldn't save
  your hours, so this plan uses a typical day." The ask returns next time (R110). `NOT_FOUND`
  follows R70. Save succeeds but the plan fails: today's quiet retry, no ask again (R111). Switch
  on `code`, never `detail`.
- Style per `client/docs/design-guideline.md` and the card's existing tokens: rounded chip
  controls in a wrapping row (FS-49CQ2 narrow widths), `PRIMARY_ACTION` vocabulary, Skip as a
  muted text action (R113). Accessible: the question labels the chip group; chips are buttons.

## Acceptance Criteria

- [ ] Not planned + `asked: false`: the question, five chips and "Skip, plan anyway" show; the
      plain line and Plan my day button don't.
- [ ] Not planned + `asked: true`: the card is as today.
- [ ] Retired (`refreshesRemaining` 0): no ask.
- [ ] Clicking "2h" calls `setDayPlanSettings(planId, 120)` then `planMyDay`, once each, in that
      order; Skip sends `null` then plans.
- [ ] After either, the section shows the drawn plan and a later `not_planned` read with
      `asked: true` shows no ask.
- [ ] A failed save still calls `planMyDay` and shows the quiet notice.
- [ ] A successful save followed by a failed plan shows the quiet retry and no ask.
- [ ] Tests pass (`npm test`, `npm run lint`, `npm run build` in client).

## Blocked By

I-02JBZ-13 (`settings` on DayPlan and `setDayPlanSettings` in the generated client)

## Spec Reference

FS-02JBZ §Change: Hours a day: R108–R113; Edge States "Never asked, never saved", "Skip", "Chip
save fails", "Chip save succeeds, plan fails", "Second device after answering", "Not-planned card
with Plan my day retired".

## TDD Approach

- RED: in `DayPlanSection.test.tsx`, a `not_planned` read with `settings: {asked: false}` renders
  the question and the chips; clicking "1h" calls the mocked `setDayPlanSettings` with 60 and then
  `planMyDay`.
- GREEN: the ask branch in `NotPlanned` and the chained handler.

## Files expected to touch

- `client/src/components/DayPlanSection.tsx`
- `client/src/components/DayPlanSection.test.tsx`
