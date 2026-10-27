---
id: I-DF4ZB-8
status: done
implements: FS-DF4ZB
blocked_by: [I-DF4ZB-6, I-DF4ZB-7]
labels: [feature]
title: "FS-DF4ZB slice 8: refresh action in Today's moves, with refreshes left and an unchanged notice"
---
Implements FS-DF4ZB §Requirements (R63)

## What to Build

A **refresh** action in the Today's moves block, calling `refreshDailyBrief`.

- Shows the refreshes left from `refreshesRemaining`. At zero it is retired (hidden or disabled),
  with no request needed to discover that.
- A refresh the rule gate served comes back with the same moves; the block says nothing has
  changed since these moves were drawn. The response carries no explicit "gate served" flag, so
  detect it from the response itself (same `generatedAt` as the brief on screen, or equivalently
  `refreshesRemaining` not lowered). Pick one, document it in the component, and test it.
- `429 · BRIEF_REFRESH_LIMIT_REACHED` (another tab spent the last one) retires the action and
  keeps the moves on screen. Switch on `code`, never on `detail`. Other failures use the block's
  quiet retry.
- After a counted refresh, the new moves replace the old; ticked/added/dismissed local state for
  moves no longer present is dropped.

## Acceptance Criteria

- [ ] Refresh shows the remaining count and retires at zero.
- [ ] An unchanged redraw says nothing has changed.
- [ ] A 429 retires the action and keeps the current moves; the client switches on `code`.
- [ ] A counted refresh replaces the moves.
- [ ] Tests pass.

## Blocked By

- I-DF4ZB-6 — `refreshDailyBrief` and `refreshesRemaining`.
- I-DF4ZB-7 — the block it lives in.

## Expected write set

- `client/src/components/TodaysMoves.tsx` (+ test)
- `client/src/api/insights.ts` (add `refreshDailyBrief`)

## Spec Reference

FS-DF4ZB §Requirements R63; §User Stories 10–12, 31; §Edge States ("Refresh with nothing
changed", "Fourth counted refresh", "Two tabs refresh at once at count 2"); ADR-0004 (switch on
`code`).

## TDD Approach

- RED: a component test where refresh resolves with the same `generatedAt` and asserts the
  unchanged notice and an unchanged count.
- GREEN: the comparison.
- Then: count at zero retires the action; a 429 problem response retires it and keeps the moves.
