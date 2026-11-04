---
id: I-XCTMG-2
status: done
implements: FS-XCTMG
blocked_by: []
labels: [feature]
title: "FS-XCTMG slice 2: the bento grid, the BentoTile reveal contract, and the Plan tile"
---
Implements FS-XCTMG §Requirements (R2, R9–R11, R13–R21, R23–R25), §Edge States

## What to Build

The grid exists with its first tile, and the one shell every tile uses owns the whole
hover/focus/touch/reduced-motion contract. Later tile slices only add a mock and a span.

**Grid (R9, R10).** A `BentoGrid` section mounted in `LandingTour` directly after the hero (the
old four sections stay below it until slice 7). Column template: one column below `md`, two at
`md`, three at `lg`. Visual order equals DOM order: no `grid-flow-dense`, no `order-*`. Wrapped in
`Section` so it enters with `riseIn` (R20). Not parallaxed (R21).

**`BentoTile` shell (R11).** A single `next/link` to `/auth?tab=signup` covering the tile;
`rounded-tile bg-card border border-border shadow-tile`; mock on top, footer row with the label
bottom-left and "Try it →" bottom-right (arrow `aria-hidden`); accessible name = label + "Try it".
Fixed box: the reveal never resizes the tile or moves anything outside it. Accepts an `inverted`
prop for TODAY (slice 3) and a span prop for slices 5–6.

**Reveal contract (R16–R18).** The tile exposes its state as a data attribute (e.g.
`data-reveal="start|playing|end"`). The state drives the mock's end-state classes.
1. `(hover: hover)`: `:hover` plays the reveal, leaving returns to start. CSS only, under
   `@media (hover: hover)` in a new landing-local CSS module (e.g.
   `client/src/components/landing/bento.module.css`). Not Tailwind `hover:` (ungated in v3).
2. `:focus-visible` on the link plays it on every device.
3. `(hover: none)`: plays once on first scroll into view, then rests at `end`; a tile flung past
   with no intersecting frame lands at `end` (reuse the `RevealOnEnter` fling rule).
4. `prefers-reduced-motion: reduce`: `end`, statically, from hydration.
Server render / no-JS = `start`, fully legible. Add a hover-capability query to
`motion/useMediaQuery.ts`. Transform and opacity only; bars fill by `scaleX`; changing numbers are
two pre-rendered labels cross-faded.

**Content (R14).** `mocks/content.ts` gains the persona (Chris Anderson), the plan (Learn
microservices) and both milestones with every step and done flag from R14, including Database's
"Indexes and query plans". Keep `GATEWAY_TASK` (the old sections still use it until slice 7).

**Plan tile (R13, R15).** `PlanTileMock`: the Database (1/5) and Microservices (6/8) milestone
cards with progress bars, styled like the production milestone cards, a few steps each visible.
End state: one open Database step ticks, its count cross-fades 1/5 → 2/5 and its bar fills to
match; further steps reveal inside the cards. Inside `Mock` (decorative, `aria-hidden`). Update
`Mock.tsx`'s doc comment from "stylized-honest, never replicas" to "styled like production,
still bespoke and decorative" (R13).

Files expected to touch:
- `client/src/components/landing/LandingTour.tsx` (mount the grid)
- `client/src/components/landing/bento/BentoGrid.tsx` (new)
- `client/src/components/landing/bento/BentoTile.tsx` (new) + `BentoTile.test.tsx` (new)
- `client/src/components/landing/bento.module.css` (new; or `bento/bento.module.css`)
- `client/src/components/landing/motion/useMediaQuery.ts`
- `client/src/components/landing/mocks/content.ts`
- `client/src/components/landing/mocks/Mock.tsx` (doc comment only)
- `client/src/components/landing/mocks/PlanTileMock.tsx` (new) + a test

Hotspots: `LandingTour.tsx` (slice 7), `BentoGrid.tsx`, `content.ts` and the CSS module
(slices 3–6). Paths above are a suggestion; keep everything under `components/landing/`.

## Acceptance Criteria

- [ ] `/` logged out shows a grid after the hero containing the PLAN tile.
- [ ] The PLAN tile is one link to `/auth?tab=signup`, accessible name "Plan … Try it" (label +
      "Try it"), label bottom-left, "Try it →" bottom-right.
- [ ] The tile uses `rounded-tile`, `bg-card`, `border-border`, `shadow-tile`.
- [ ] PLAN shows Database 1/5 and Microservices 6/8 with the R14 steps and done flags, read from
      `mocks/content.ts`.
- [ ] With `(hover: none)` mocked, the tile goes `start → playing → end` on first intersection and
      stays `end`; a tile above the viewport on observe lands at `end`.
- [ ] With reduced motion mocked, the tile renders `end` and never `playing`.
- [ ] Server render (no effects run) shows `start` with all steps legible.
- [ ] Hover and focus rules live in the CSS module under `@media (hover: hover)` and
      `:focus-visible`; the module animates only `transform` and `opacity`.
- [ ] `expectMockContract` passes on the tile; `PlanTileMock` imports nothing from
      `@/components` product components, no context, no fetch.
- [ ] No raw hex, non-`var()` `rgb(`/`hsl(`, `*-gray-*`, `bg-white`/`text-white`/`bg-black` in the
      touched files, CSS module included.
- [ ] No x.ai-pass file appears in `git diff --stat`.
- [ ] `npm test`, `npm run lint` and `npm run build` pass in `client/` (the build proves the CSS
      module compiles).

## Blocked By

None. Runs in parallel with I-XCTMG-1 (no shared files).

## Spec Reference

FS-XCTMG §Requirements R2 (sealed dir, CSS module), R9–R11 (grid, layout, tile shell), R13–R15
(mocks, content, PLAN row), R16–R18 (reveal contract), R19–R21 (no scroll mutation, entrance, no
parallax on tiles), R23–R25 (access, tokens, themes); §Edge States (touch, hybrid, reduced motion,
fast hover, fling, JS disabled, theme toggle mid-reveal). User stories 7–10, 16–18, 21, 23, 24,
33–35.

## TDD Approach

- RED: render `BentoTile` with `matchMedia('(hover: none)')` true and a stubbed
  `IntersectionObserver`; fire an intersection and assert `data-reveal` reaches `end` and stays
  there after a second intersection.
- GREEN: the state machine in the shell.
- Then: reduced motion → `end` with no `playing`; initial render → `start`; link href and
  accessible name; `expectMockContract`; PLAN counts come from `content.ts`.
