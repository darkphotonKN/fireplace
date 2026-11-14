---
id: I-XCTMG-6
status: done
implements: FS-XCTMG
blocked_by: [I-XCTMG-2]
labels: [feature]
title: "FS-XCTMG slice 6: the Calendar tile with a month card"
---
Implements FS-XCTMG §Requirements (R10, R11, R13–R15, R16–R18, R24, R25)

## What to Build

The fifth tile, closing row 2.

**Calendar mock (R13, R15).** `CalendarTileMock`, styled like the production calendar card: one
month grid with a handful of dated items from the plan (step names from `content.ts`, e.g. "Kafka
review", "Migration strategies"). Use a fixed invented month and year in `content.ts`, never
`new Date()`, so the mock is deterministic and server/client renders agree. End state: one day
lights up (a highlight layer fading in, `opacity`/`scale`).

**Span (R10).** Fifth in DOM order. One column at `lg` (beside MILESTONES); at `md` it spans both
columns so the two-column grid has no orphan cell.

Files expected to touch:
- `client/src/components/landing/bento/BentoGrid.tsx`
- `client/src/components/landing/bento.module.css`
- `client/src/components/landing/mocks/content.ts`
- `client/src/components/landing/mocks/CalendarTileMock.tsx` (new) + a test

Hotspots: shared with I-XCTMG-3, 4 and 5. Land one tile slice at a time in this worktree.

## Acceptance Criteria

- [ ] CALENDAR renders fifth; one column at `lg`, both columns at `md`, full width below `md`.
- [ ] The month is fixed in `content.ts`; no `Date` call in the mock.
- [ ] Dated items use step names from `content.ts`.
- [ ] End state: one day highlighted; reached on hover, `:focus-visible`, once on view with
      `(hover: none)`, and statically under reduced motion.
- [ ] Link to `/auth?tab=signup`, accessible name "Calendar … Try it"; `expectMockContract`
      passes.
- [ ] Token-only; no x.ai-pass file in `git diff --stat`.
- [ ] `npm test` and `npm run lint` pass in `client/`.

## Blocked By

I-XCTMG-2.

## Spec Reference

FS-XCTMG §Requirements R10 (md span, lg placement), R11, R13–R15 (mock, content, CALENDAR row),
R16–R18, R24–R25; §Design decisions (derived: `md` grid). User story 15.

## TDD Approach

- RED: render the tile twice and assert identical markup (deterministic month); assert dated item
  names come from `content.ts`.
- GREEN: `CalendarTileMock`.
