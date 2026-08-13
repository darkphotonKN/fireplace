---
id: I-0011
status: open
blocked_by: [I-0010]
implements: FS-0003
labels: [enhancement]
title: "FS-0003 slice 5: DAILY section"
---
Implements FS-0003 §Requirements (R2, R13, R16, R17, R20)

## What to Build

Section 3 of the loop — **"Today is a short list."** — illustrated by a mocked checklist that
makes the **dual-scope idea** legible at a glance: the daily list is visibly separate from
long-term goals. A visitor should grasp that distinction without reading documentation, since it
is one of the product's genuinely distinctive ideas.

Follows the mock contract established in I-0010 without restating it: stylized-honest, bespoke
and isolated, decorative to assistive tech, token-only, static under scroll. A few checklist
rows with a couple completed — not a dense replica of the real `Todo` component, and **not an
import of it**.

- Heading takes `foreground`; the hero keeps the page's only coral heading.
- Mock art on the ≈0.6× layer, copy at 1.0×; entrance via the I-0007 reveal primitive.
- If a completed row needs a coral tick, that is the section's **one** permitted small accent.

## Acceptance Criteria

- [ ] Section 3 renders after section 2 with a checklist mock in which daily and long-term
      scopes are visually distinguishable.
- [ ] The section heading renders at `foreground`; coral use is limited to at most one small
      accent.
- [ ] Mock art moves at ≈0.6× and copy at 1.0× at ≥640px; neither translates below 640px.
- [ ] The mock contract holds: no app-component imports, no context, no fetches, no raw hex, no
      `*-gray-*`, nothing focusable, no real `<input>` elements, no scroll-driven mutation.
- [ ] Under `prefers-reduced-motion: reduce`, the section is static and fully legible.
- [ ] Verified in light and dark mode.
- [ ] `npm run lint` and `npm run build` pass.

## Blocked By

I-0010 — inherits the mock component conventions established there.

## Spec Reference

FS-0003 §Requirements R2 (section table), R13, R16, R17, R20 · Covers user stories 4, 8, 9, 19, 27.

## TDD Approach

- RED: assert the section renders both scope groupings and that no element within the mock is
  focusable.
- GREEN: build the checklist mock from the I-0010 primitives.
