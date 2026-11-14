---
id: I-XCTMG-3
status: done
implements: FS-XCTMG
blocked_by: [I-XCTMG-2]
labels: [feature]
title: "FS-XCTMG slice 3: the inverted Today tile with a Day plan mock"
---
Implements FS-XCTMG §Requirements (R11, R12, R13–R15, R16–R18, R24, R25)

## What to Build

The second tile, and the one the hero badge points at.

**Inverted shell (R12).** TODAY uses `BentoTile`'s `inverted` mode: `bg-secondary
text-secondary-foreground` in both themes, inner colours drawn from `secondary-foreground` at
reduced opacity (for example `text-secondary-foreground/70`, `border-secondary-foreground/15`),
never from page tokens that would not read on it. Footer row inverted too. The tile carries
`id="today"` so the hero badge (`#today`, slice 1) lands on it.

**Day plan mock (R13, R15).** `TodayTileMock`, styled like the production Day plan section: a few
AI-suggested entries drawn from the open steps in `mocks/content.ts` (for example SQL Tools Drill
Practice, DLQ setup, Kafka review), grouped under short labels with rough time estimates, and a
partly filled day meter. End state: ticks land on entries one after another and the day meter
fills to match (`scaleX`, staggered delays). Say "entry" in code and tests, not "move"
(§Known tensions 5). Any Day plan copy the mock needs goes in `content.ts`.

Placed second in the grid's DOM order (after PLAN). Its keyframes or transitions go in the
landing CSS module.

Files expected to touch:
- `client/src/components/landing/bento/BentoGrid.tsx`
- `client/src/components/landing/bento/BentoTile.tsx` (only if the `inverted` mode from slice 2
  needs finishing)
- `client/src/components/landing/bento.module.css`
- `client/src/components/landing/mocks/content.ts`
- `client/src/components/landing/mocks/TodayTileMock.tsx` (new) + a test

Hotspots: `BentoGrid.tsx`, `content.ts` and the CSS module are shared with I-XCTMG-4..6. Land one
tile slice at a time in this worktree.

## Acceptance Criteria

- [ ] TODAY renders second in the grid, with `id="today"`; clicking the hero badge scrolls to it.
- [ ] The tile uses `bg-secondary text-secondary-foreground` in both themes; no page-token text
      colour (`text-foreground`, `text-muted-foreground`) inside it.
- [ ] It shows grouped entries with time estimates and a partly filled day meter, using steps
      from `content.ts`.
- [ ] End state: every visible entry ticked and the day meter full, reached on hover, on
      `:focus-visible`, once on view with `(hover: none)`, and statically under reduced motion.
- [ ] Only `transform`/`opacity` animate.
- [ ] Link to `/auth?tab=signup`, accessible name "Today … Try it"; `expectMockContract` passes.
- [ ] Token-only; no x.ai-pass file in `git diff --stat`.
- [ ] `npm test` and `npm run lint` pass in `client/`.

## Blocked By

I-XCTMG-2 (the grid, the tile shell and its reveal contract).

## Spec Reference

FS-XCTMG §Requirements R11–R12 (shell, inverted tile), R13–R15 (mock, content, TODAY row),
R16–R18 (reveal contract), R24–R25 (tokens, themes); §Known tensions 5. User stories 6, 11, 12.

## TDD Approach

- RED: render the grid and assert the second tile has `id="today"` and the inverted classes, and
  contains no `text-foreground`/`text-muted-foreground`.
- GREEN: `TodayTileMock` in the inverted shell.
- Then: entries read from `content.ts`; the `end` state marks every entry ticked.
