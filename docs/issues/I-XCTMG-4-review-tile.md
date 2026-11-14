---
id: I-XCTMG-4
status: done
implements: FS-XCTMG
blocked_by: [I-XCTMG-2]
labels: [feature]
title: "FS-XCTMG slice 4: the Review tile with end-of-day chat bubbles"
---
Implements FS-XCTMG §Requirements (R11, R13–R15, R16–R18, R24, R25)

## What to Build

The third tile of row 1.

**Review mock (R13, R15).** `ReviewTileMock`: an end-of-day exchange as chat bubbles, styled like
the product's chat surfaces. Chris Anderson's short note on the day (one or two lines, about the
plan's steps, e.g. finishing SQL Tools Drill Basics and skipping DLQ setup) and one reply. End
state: the next reply slides in (`translateY` + `opacity`). The slot for the hidden reply is
reserved in the start state, so the tile's box never changes. No long personal notes; copy lives
in `content.ts`.

Placed third in the grid's DOM order. Standard (non-inverted) `BentoTile`, label bottom-left,
"Try it →" bottom-right.

Files expected to touch:
- `client/src/components/landing/bento/BentoGrid.tsx`
- `client/src/components/landing/bento.module.css`
- `client/src/components/landing/mocks/content.ts`
- `client/src/components/landing/mocks/ReviewTileMock.tsx` (new) + a test

Hotspots: shared with I-XCTMG-3, 5 and 6. Land one tile slice at a time in this worktree.

## Acceptance Criteria

- [ ] REVIEW renders third in the grid; at `lg` PLAN, TODAY and REVIEW share row 1 equally.
- [ ] Start state shows Chris's note and one reply; end state adds the next reply, sliding in.
- [ ] The tile's box height is identical in start and end state.
- [ ] End state is reached on hover, `:focus-visible`, once on view with `(hover: none)`, and
      statically under reduced motion.
- [ ] Bubble copy comes from `content.ts`; nothing longer than two lines per bubble.
- [ ] Link to `/auth?tab=signup`, accessible name "Review … Try it"; `expectMockContract` passes.
- [ ] Token-only; no x.ai-pass file in `git diff --stat`.
- [ ] `npm test` and `npm run lint` pass in `client/`.

## Blocked By

I-XCTMG-2.

## Spec Reference

FS-XCTMG §Requirements R11 (shell, fixed box), R13–R15 (mock, content, REVIEW row), R16–R18
(reveal contract), R24–R25. User story 13.

## TDD Approach

- RED: render the REVIEW tile in `end` and assert the second reply is present and not hidden; in
  `start` assert it is in the DOM but marked hidden (opacity class or `aria-hidden` within the
  mock), so the box is reserved.
- GREEN: `ReviewTileMock`.
