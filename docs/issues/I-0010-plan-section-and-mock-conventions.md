---
id: I-0010
status: open
blocked_by: [I-0008]
implements: FS-0003
labels: [enhancement]
title: "FS-0003 slice 4: PLAN section + the mock component conventions"
---
Implements FS-0003 §Requirements (R2, R13, R16, R17, R20)

## What to Build

Section 2 of the loop — **"Every arc gets a plan."** — illustrated by a mocked plan card with
soft Gantt bars. This is the first mocked section, so it also establishes the contract every
later mock follows. Get the contract right here; slices 5–7 inherit it.

### The mock contract (binding on all later sections)

1. **Stylized-honest, never a replica.** Same tokens, type scale, radii, and coral as the real
   product — but simplified. A plan card is a title and a few rows; a Gantt is a few soft bars.
   No window chrome, no toolbars, no dense real-app detail. A near-replica **silently becomes a
   lie** the first time the real UI changes, because hand-built mocks cannot track the app; a
   stylized mock reads as *a picture of the idea* and stays true.
2. **Bespoke and isolated.** No imports from real app components (`@/components/...`), no
   context consumption (`useAuth`, `useSidebar`, `useTheme`), no service-layer calls, no
   `fetch`. Importing real components would drag in `AuthContext`, the service layer, and client
   state, coupling this marketing surface to product internals and breaking the landing whenever
   the real checklist is refactored.
3. **Decorative to assistive tech.** Mocks are not focusable, expose no interactive roles, and
   are not announced as controls. A screen-reader user should hear the section's copy, not a
   recital of fake checkbox labels. Fake checkboxes must not be real `<input>`s.
4. **Token-only colour.** No raw hex, no `*-gray-*`. Warm neutrals only — no cold grays, no pure
   `#FFFFFF`, no pure black.
5. **Static.** Nothing in a mock mutates as a function of scroll position — no self-ticking
   checkboxes, no self-drawing bars. That was the rejected "cinematic" option.

### This section specifically

- Copy conveying that a plan holds a whole arc of work, not a flat to-do list.
- Heading takes `foreground`, **not** coral — the hero holds the page's only coral heading.
- The mock art sits on the ≈0.6× parallax layer; the copy stays at 1.0×, so text never moves
  relative to the reader.
- Entrance via the I-0007 reveal primitive.

## Acceptance Criteria

- [ ] Section 2 renders after the hero with a plan-card mock including Gantt-style bars.
- [ ] The section heading renders at `foreground`; no second coral heading exists on the page.
- [ ] Mock art moves at ≈0.6× and copy at 1.0× at ≥640px; neither translates below 640px.
- [ ] `grep` over the mock components finds no `@/components` app imports, no `useAuth`/context
      consumption, and no `fetch`/service-layer calls.
- [ ] `grep` over the section finds no raw hex and no `*-gray-*` utilities.
- [ ] Mocks are not in the tab order and expose no interactive roles; fake checkboxes are not
      real `<input>` elements.
- [ ] Nothing in the mock changes as a function of scroll position.
- [ ] Under `prefers-reduced-motion: reduce`, the section is static and fully legible.
- [ ] Verified in light and dark mode.
- [ ] `npm run lint` and `npm run build` pass.

## Blocked By

I-0008 — needs the landing directory and the section scaffolding.

## Spec Reference

FS-0003 §Requirements R2 (section table), R13 (tokens), R16 (stylized-honest), R17 (bespoke
mocks), R20 (accessibility) · §Design decisions D6, D7 · Covers user stories 3, 8, 9, 19, 26, 27, 28.

## TDD Approach

- RED: assert the mock module's import graph contains no `@/components` app component and no
  service-layer module; assert no element inside the mock is focusable.
- GREEN: build the mock from primitives local to the landing directory.
