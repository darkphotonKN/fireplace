---
id: I-49CQ2-5
status: done
implements: FS-49CQ2
blocked_by: []
labels: [feature]
title: "FS-49CQ2 slice 5: the sidebar discovery tip stays quiet below md"
---
Implements FS-49CQ2 §Discovery toast

## What to Build

The tip says "Hover or click the left edge", which a phone can't do and where there is no rail.

- In `LayoutContent.tsx`'s tip effect, skip when the viewport is below 768px
  (`matchMedia('(min-width: 768px)')` false), before reading or writing any flag, so
  `hasSeenSidebarHint` stays unspent for a later docked visit.
- At docked widths nothing changes: FS-KSJFR R46 and FS-S6WVY R12–R13 hold (never on `/`, only when
  hidden on arrival, decided on arrival, same copy and timing).

Files expected to touch:
- `client/src/components/LayoutContent.tsx` (tip effect guard)
- `client/src/components/LayoutContent.test.tsx` (narrow cases; existing cases run as docked)
- `client/src/test/motion.ts` (only if the stub needs a 768px answer)

## Acceptance Criteria

- [ ] At a narrow width on a plan page with `sidebar=hidden`, no toast fires and
      `hasSeenSidebarHint` stays unset.
- [ ] At docked widths the existing tip tests stay green.
- [ ] `npm run lint`, `npm run build` and `npm test` pass.

## Blocked By

None

## Spec Reference

FS-49CQ2 §Discovery toast: R21; user stories 23–24; edge state "Narrow width, plan page,
first-ever visit".

## TDD Approach

- RED: stub `(min-width: 768px)` false, render on `/plan/p1` with `sidebar=hidden`, advance
  timers past 1.5s, assert no toast and `hasSeenSidebarHint` unset.
- GREEN: the width guard at the top of the effect.
