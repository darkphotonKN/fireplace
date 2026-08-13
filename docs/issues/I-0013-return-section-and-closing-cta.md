---
id: I-0013
status: done
blocked_by: [I-0008]
implements: FS-0003
labels: [enhancement]
title: "FS-0003 slice 7: RETURN section + closing CTA"
---
Implements FS-0003 §Requirements (R2.2, R5, R14)

## What to Build

Section 6 — the close. The tour's whole reason for using the loop as its spine is that the
scroll **is** the cycle, so this section must visually return to the hero's imagery (the
hearth), making the loop legible rather than merely stated. A visitor should end feeling they
have seen a cycle they'd be joining, not a list of features.

*(This is the HITL part: whether the reprise actually **reads** as a return is a visual
judgment, not something acceptance criteria can fully capture. Expect to iterate on it.)*

**Closing CTA — one primary action, not a pair.** The section presents a single primary button
(`Start your plan`) plus a quiet `Sign in` **text link**. By section 6 the visitor is warm; two
equal-weight buttons read as hesitation. This is deliberately *not* symmetrical with the hero's
CTA pair.

- The primary button uses `text-primary-foreground` (defined in I-0007) — no hardcoded
  `text-white`.
- Targets are unchanged: `/auth?tab=signup` and `/auth`.
- Section heading takes `foreground`; the hero keeps the page's only coral heading. The coral
  here belongs to the button.
- The reprised hero imagery sits on the slow (≈0.2×) layer, consistent with the hero.

## Acceptance Criteria

- [ ] Section 6 renders last and visually reprises the hero's imagery.
- [ ] Exactly one primary button and one text-link `Sign in` are present — not two buttons.
- [ ] The primary button uses `text-primary-foreground`; no hardcoded `text-white`.
- [ ] CTAs resolve to `/auth?tab=signup` and `/auth`.
- [ ] The section heading renders at `foreground`; the page still has exactly one coral heading.
- [ ] The reprised imagery moves at ≈0.2× at ≥640px and not at all below.
- [ ] Under `prefers-reduced-motion: reduce`, the section is static and fully legible.
- [ ] Verified in light and dark mode.
- [ ] `npm run lint` and `npm run build` pass.

## Blocked By

I-0008 — needs the hero's imagery in order to reprise it.

## Spec Reference

FS-0003 §Requirements R2.2 (loop closure), R5 (closing CTA), R14 (coral discipline), R15
(on-coral token) · §Design decisions D3 · Covers user stories 7, 11, 21.

## TDD Approach

- RED: assert section 6 renders exactly one `<button>`/button-styled CTA plus one text link, and
  that the link targets `/auth`.
- GREEN: build the section with the asymmetric CTA pairing.

---

## Implementation notes

**The reprise is literal, not thematic.** `ReturnSection` renders the same `EmberGlow`
component the hero does, on the same `GLOW_SPEED` layer — verified as **2** radial-gradient
layers in SSR markup. The loop has to *look* like it returns, not merely say so.

**Asymmetric CTAs, as specified.** One filled primary (`Start your plan` → `/auth?tab=signup`)
and one quiet text link (`Sign in` → `/auth`). The test asserts exactly two links, that the
primary carries `bg-primary` and the quiet one does not, and that the primary uses
`text-primary-foreground` and never `text-white`.

**`CtaLink` extracted** (`primary` / `outline` / `quiet` variants, plus the two auth hrefs as
constants). The coral button now has one definition instead of a second copy here and a third
in I-0009's chrome bar. It is also the single place the on-coral token is applied, which is
what keeps the 2.86:1 `text-white` regression from creeping back per-CTA.

**The section that justified the earlier accessibility fix.** RETURN is below the fold and
carries the page's closing CTAs, so it is precisely where hiding-by-opacity would have stranded
a keyboard visitor on an invisible button. A test asserts the reveal wrapper is `inert` before
entry and not after — the scenario, not just the mechanism.

**Shared test helpers.** `stubElementTop` / `BELOW_FOLD` moved into `src/test/motion.ts`. jsdom
reports all-zero rects, which reads as "already on screen", so without the stub a test cannot
exercise the below-the-fold path at all — a trap worth having in one documented place rather
than rediscovered per test file. `Section` now emits `data-landing-reveal` so any section's
wrapper is addressable.

**Verified server-side** (dev server, Node 22): all six beats in order, 1 `<h1>` and 5 `<h2>`,
4 mocks, both auth targets present twice each, `text-primary-foreground` present, `text-white`
**zero**, `inert` **zero** in SSR, and 2 glow layers.

**Not machine-verified:** whether the close actually *reads* as a return rather than a repeat,
and light/dark appearance. Both need eyes.
