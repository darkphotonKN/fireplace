---
id: I-0007
status: done
implements: FS-0003
blocked_by: []
labels: [enhancement]
title: "FS-0003 slice 1: landing motion primitives + the --primary-foreground token gap"
---
Implements FS-0003 §Requirements (R8, R10, R11, R12, R15), §Starting state

## What to Build

The groundwork the whole tour stands on: one missing token, one missing accessibility
behavior, and the two motion primitives every section will use.

**1. Define `--primary-foreground`** in `client/src/app/globals.css` for **both** `:root` and
`.dark`. It is consumed by `tailwind.config.js:48` (`hsl(var(--primary-foreground))`) but
declared nowhere, so `text-primary-foreground` currently resolves to nothing — which is why
today's landing hardcodes `text-white` on the coral button. The design guideline prescribes
the value: the on-coral text colour, cream `#F2F0E3` (light) / near-white (dark), expressed as
an HSL channel triplet like every other token in the file.

**2. Add `prefers-reduced-motion: reduce` handling.** No such handling exists anywhere in
`globals.css` today. Under the reduce preference, all parallax translation and all entrance
animation must be inert — content sits at its final position, fully visible and legible.

**3. Add a new entrance keyframe** (e.g. `riseIn`): `opacity 0→1`, `translateY(+16px)→0`,
≈400ms ease-out. **Do not modify or reuse the existing `fadeIn` keyframe** — it is reserved
for newly added checklist items landing in a list (600ms, `translateY(-8px)`, coral halo), a
*communicating* motion. The tour's entrance is a different intent and must not overload it.

**4. Build two reusable motion primitives** in the landing's own directory:

- a **parallax-layer** wrapper taking a speed factor, driving `transform` only;
- a **reveal-on-enter** wrapper applying the entrance keyframe when a section enters view.

Both must satisfy:

- **≥640px gate** — below `sm`, the parallax wrapper performs no translation at all. The
  reveal wrapper still runs. Touch devices therefore never run scroll-linked transforms, so
  iOS momentum scrolling cannot desynchronise from layer positions.
- **Reduced-motion no-op** — both collapse to static under the reduce preference.
- **Fling-safe** — a section scrolled past before its entrance fires must render **at rest and
  fully visible**. An animation that never ran must never leave content hidden. This is the
  classic scroll-animation bug and it fails silently; test it deliberately.
- **`transform` and `opacity` only** in the scroll path. No layout-triggering properties, no
  per-scroll-event layout reads. Drive with `requestAnimationFrame` or CSS.
- `"use client"` sits on these primitives, not on a whole page — they are the client island.

**5. Prove the token resolves** by switching the existing landing's coral CTA from hardcoded
`text-white` to `text-primary-foreground`. This is the visible, verifiable behavior this slice
delivers; the rest of the landing is untouched here.

## Acceptance Criteria

- [ ] `--primary-foreground` is defined in both `:root` and `.dark` in `globals.css`.
- [ ] `text-primary-foreground` resolves to a visible colour in both themes; the existing
      landing CTA uses it and no longer hardcodes `text-white`.
- [ ] Contrast of the on-coral text against `--primary` is legible in both themes.
- [ ] A new entrance keyframe exists; `fadeIn` is byte-for-byte unmodified.
- [ ] The parallax primitive applies no transform below 640px.
- [ ] Under `prefers-reduced-motion: reduce`, neither primitive animates and wrapped content is
      visible at its final position.
- [ ] Content wrapped in the reveal primitive is fully visible after a fast scroll/fling that
      skips past it.
- [ ] Only `transform`/`opacity` are written in the scroll path; no layout reads per scroll event.
- [ ] `npm run lint` and `npm run build` pass.

## Blocked By

None.

## Spec Reference

FS-0003 §Requirements R8 (entrance), R10 (mobile), R11 (reduced motion), R12 (performance),
R15 (token) · §Starting state (the verified `globals.css` / `tailwind.config.js` facts —
do not re-derive them) · Covers user stories 15, 17, 24.

## TDD Approach

- RED: assert `getComputedStyle` on a `text-primary-foreground` element returns a non-empty
  colour; assert the parallax primitive's transform is identity at a 375px-wide viewport and
  under a mocked `prefers-reduced-motion: reduce` match.
- GREEN: define the token; add the breakpoint and media-query guards inside the primitives.

---

## Implementation notes

**Token value deviates from FS-0003 R15 — deliberately, with approval.** R15 specifies cream
`#F2F0E3` / near-white, inherited from the design guideline. Measured against coral `#F76F53`,
cream is **2.50:1** and white **2.86:1**; WCAG AA needs **4.5:1** for normal text, and CTA
labels at `text-base`/`font-medium` are normal text. The token ships as warm dark `#2E2E2E`
(`0 0% 18%`) at **4.76:1**. This honors the guideline's stated goal — "a guaranteed contrast
pair" — where its prescribed value could not. `client/docs/design-guideline.md` has been
corrected so the doc no longer prescribes a failing value. **Coral surfaces now carry dark
text, not light.**

**Test infrastructure added.** The client had none (no runner, no config, no test files).
Added vitest + @testing-library/react + jsdom, `npm test` / `npm run test:watch`, and
`src/test/motion.ts` helpers stubbing `matchMedia` and `IntersectionObserver`. `jsdom` is
pinned to `^25` and `@vitejs/plugin-react` to `^4` — jsdom 30 needs a newer Node than this
environment has, and plugin-react 6 pulls a second copy of vite that breaks vitest's types.

**A real bug surfaced during the loop.** Media queries resolve in an effect, so the first
commit always runs with `prefersReducedMotion === false`. `RevealOnEnter` therefore hid
below-fold content on mount, and when the preference then flipped true the effect returned
early — leaving content **permanently invisible for exactly the visitor who opted out of
motion**. Fixed by actively resetting to `idle` rather than returning early.

**The fling guard is mutation-verified.** Reverting `isIntersecting || boundingClientRect.top < 0`
to the naive `isIntersecting` fails the fling test and only that test, so the test is not vacuous.

**`npm run lint` / `npm run build`: the toolchain runs, the repo does not pass — pre-existing.**
The default shell's Node is 18.17.1, below Next 15's `^18.18.0` floor, which initially looked
like a hard block. It is not: **nvm has v22.13.1 installed** (`~/.nvm/versions/node/v22.13.1`),
and every gate runs there. Use it.

On Node 22 the true state is:

- `npm run lint` → **68 pre-existing errors** (unused vars, empty interfaces) across
  `ui/*`, `NotesContext.tsx`, `notesService.ts`, and others. **Zero in new code.**
- `npm run build` → fails at the lint stage. With `--no-lint` it reports
  **`✓ Compiled successfully`**, then fails typecheck on a pre-existing Next 15 migration miss:
  `src/app/plans/[planId]/page.tsx` still types `params` as a plain object where Next 15 requires
  a `Promise`.

So the client cannot build today for reasons that predate FS-0003, and the `lint`/`build`
acceptance criterion is unsatisfiable repo-wide until that debt is cleared. New code compiles,
lints, and typechecks clean. Also verified: full vitest suite green, and a direct Tailwind CLI
compile confirming `--primary-foreground` emits in both themes, `.text-primary-foreground`
resolves, `@keyframes riseIn` + `.animate-riseIn` are generated, and the
`prefers-reduced-motion` block is present.
