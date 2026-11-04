---
id: I-XCTMG-5
status: done
implements: FS-XCTMG
blocked_by: [I-XCTMG-2]
labels: [feature]
title: "FS-XCTMG slice 5: the wide Milestones tile with a 14-segment progress strip"
---
Implements FS-XCTMG §Requirements (R10, R11, R13–R15, R16–R18, R24, R25)

## What to Build

The wide tile in row 2.

**Milestones mock (R13, R15).** `MilestonesTileMock`, styled like the production plan header: a
plan overview (Learn microservices, "N of 14 milestones" where N counts closed milestones) above a
14-segment **progress strip**, each segment filled to its milestone's own progress. Database (1/5)
and Microservices (6/8) are two of the 14 and read their counts from `content.ts`; the other 12
milestones get short plausible names and progress in `content.ts` too. End state: segments fill
one by one (`scaleX`, staggered `transition-delay`), and the headline count cross-fades to match.

**Span (R10).** At `lg` the tile spans two columns. It is fourth in DOM order.

Files expected to touch:
- `client/src/components/landing/bento/BentoGrid.tsx`
- `client/src/components/landing/bento/BentoTile.tsx` (only if the span prop from slice 2 needs
  finishing)
- `client/src/components/landing/bento.module.css`
- `client/src/components/landing/mocks/content.ts`
- `client/src/components/landing/mocks/MilestonesTileMock.tsx` (new) + a test

Hotspots: shared with I-XCTMG-3, 4 and 6. Land one tile slice at a time in this worktree.

## Acceptance Criteria

- [ ] MILESTONES renders fourth and spans two columns at `lg`.
- [ ] The strip has exactly 14 segments; the headline reads "N of 14 milestones" with N equal to
      the closed milestones in `content.ts`.
- [ ] Database's and Microservices' segments are filled to 1/5 and 6/8 in the start state, the
      same counts the PLAN tile shows.
- [ ] End state: segments fill in sequence; reached on hover, `:focus-visible`, once on view with
      `(hover: none)`, and statically under reduced motion.
- [ ] Segments fill by `scaleX`, never by `width`.
- [ ] Link to `/auth?tab=signup`, accessible name "Milestones … Try it"; `expectMockContract`
      passes.
- [ ] Token-only; no x.ai-pass file in `git diff --stat`.
- [ ] `npm test` and `npm run lint` pass in `client/`.

## Blocked By

I-XCTMG-2.

## Spec Reference

FS-XCTMG §Requirements R10 (lg span), R11, R13–R15 (mock, content, MILESTONES row), R16–R18,
R24–R25; `client/CONTEXT.md` "progress strip". User story 14.

## TDD Approach

- RED: render the tile and assert 14 segments and a headline count derived from `content.ts`.
- GREEN: `MilestonesTileMock` reading the shared content.
- Then: the Database and Microservices segments' fill fractions match the PLAN tile's counts.
