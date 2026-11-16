---
id: I-S6WVY-3
status: done
implements: FS-S6WVY
blocked_by: []
labels: [feature]
title: "FS-S6WVY slice 3: All filter segment, first and on by default"
---
Implements FS-S6WVY §Change: All filter, on by default

## What to Build

Fix "if you're on a project or learning plan but the sidebar doesn't show your filter". The plans
panel's filter gains a third segment, **All**, placed first and selected by default, so with
nothing picked the current plan is always listed and gets R27's active mark. One behaviour, end to
end, in the shared panel that the dock and the drawer both render.

- **State** (`client/src/components/PlansList.tsx`): `PlanFilter` gains `'all'`; the provider's
  initial state is `'all'`. Still in memory only, never stored (R37). Comments that say
  "Projects / Learning choice" name the three-way choice.
- **Panel** (`client/src/components/PlansPanel.tsx`): `FILTERS` reads All · Projects · Learning
  (R35, R36), text labels, same R26 segment styling. Under `'all'` the list is every plan in API
  order with no grouping or badge, unknown `planType` included (R39); the two type filters keep
  their exact-match rule. Empty copy under All is "No plans yet"; the other two keep R31's
  strings (R40).
- An explicit Projects/Learning pick is never overridden by navigation (R38). No effect watches
  the route.

## Acceptance Criteria

- [ ] Three segments, "All", "Projects", "Learning", in that order, in the dock and the drawer.
- [ ] First render: All `aria-pressed="true"`, Projects and Learning `"false"`.
- [ ] Under All, a project and a learning plan are both listed, in API order, no group headings
      or type badge.
- [ ] On a learning plan's page with nothing picked, its row is listed with `aria-current="page"`
      and the coral mark; same for a project plan.
- [ ] Pick Learning, navigate to a project plan: Learning stays pressed and the project plan is
      not listed.
- [ ] Pick Projects or Learning, then All: every plan is listed again.
- [ ] A shell remount starts on All; nothing is written to `localStorage`, `sessionStorage`, the
      URL or the API.
- [ ] A plan with an unknown `planType` is listed under All only.
- [ ] No plans: All shows "No plans yet"; Projects and Learning keep "No projects yet" and
      "No learning plans yet".
- [ ] A pick in the dock shows in the drawer and the reverse (FS-49CQ2 R4).
- [ ] Opening the drawer focuses the All segment.
- [ ] All follows R26 and R30: muted track, card-surface active state, no coral, coral
      `focus-visible` ring.
- [ ] Tests updated: `Sidebar.test.tsx` (default pressed segment), `PlansDrawer.test.tsx` (focus
      lands on All; drawer pick still shared), `PlansList.test.tsx` (shared three-way choice).
- [ ] `npm test`, `npm run lint` and `npm run build` pass in `client/`.

## Blocked By

None. (I-S6WVY-1 and I-S6WVY-2, which built the shell and the segmented control, are done.)

## Spec Reference

FS-S6WVY §Change: All filter, on by default: R35–R41, all of its acceptance criteria and edge
states. R26 and R31 of the original work order stand for the look and the two existing empty
strings.

Files expected to change: `client/src/components/PlansList.tsx`,
`client/src/components/PlansPanel.tsx`, `client/src/components/Sidebar.test.tsx`,
`client/src/components/PlansDrawer.test.tsx`, `client/src/components/PlansList.test.tsx`. No
backend, route, persisted filter or new dependency.

## TDD Approach

- RED: in `Sidebar.test.tsx`, render the dock with one project and one learning plan and expect
  an "All" segment pressed first, with both plans listed.
- GREEN: add `'all'` to `PlanFilter`, default the provider to it, prepend the segment to
  `FILTERS`, and skip the type filter under `'all'`.
- Then: the learning-plan-highlighted case, the held pick across navigation, "No plans yet", the
  unknown `planType`, and the drawer's focus on All.
