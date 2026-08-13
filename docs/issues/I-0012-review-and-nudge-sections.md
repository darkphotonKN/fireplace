---
id: I-0012
status: open
blocked_by: [I-0010]
implements: FS-0003
labels: [enhancement]
title: "FS-0003 slice 6: REVIEW + NUDGE sections"
---
Implements FS-0003 §Requirements (R2, R2.1, R13, R16, R17, R20)

## What to Build

Sections 4 and 5 of the loop, built together because they are causally linked — *close the day,
and it tells you what's next* — and because their **relative proportion** is a requirement that
is easiest to get right in one diff.

**Section 4 — REVIEW: "Close the day honestly."**
Mocked daily review with a streak indicator. The beat to land: the product expects reflection,
not just checking boxes.

**Section 5 — NUDGE: "It notices what you keep skipping."**
A **single** mocked suggestion card. This section is **deliberately the smallest on the page** —
less vertical space than REVIEW — so it reads as a grace note rather than padding (R2.1). Resist
the pull to give it parity; the proportion is the point. The suggestion shown should feel
grounded in the mocked plan rather than generic, since that is what makes the AI beat credible.

Both follow the mock contract from I-0010 without restating it. Both headings take `foreground`;
the hero keeps the page's only coral heading. Mock art on the ≈0.6× layer, copy at 1.0×,
entrance via the I-0007 reveal primitive.

## Acceptance Criteria

- [ ] Sections 4 and 5 render in order after section 3.
- [ ] Section 5 occupies **visibly less vertical space** than section 4 at desktop widths.
- [ ] Section 5 contains exactly one suggestion card.
- [ ] Both headings render at `foreground`; coral use is limited to at most one small accent per
      section.
- [ ] Mock art moves at ≈0.6× and copy at 1.0× at ≥640px; neither translates below 640px.
- [ ] The mock contract holds for both: no app-component imports, no context, no fetches, no raw
      hex, no `*-gray-*`, nothing focusable, no scroll-driven mutation.
- [ ] Under `prefers-reduced-motion: reduce`, both sections are static and fully legible.
- [ ] Verified in light and dark mode.
- [ ] `npm run lint` and `npm run build` pass.

## Blocked By

I-0010 — inherits the mock component conventions established there.

## Spec Reference

FS-0003 §Requirements R2 (section table), R2.1 (NUDGE proportion), R13, R16, R17, R20 ·
Covers user stories 5, 6, 8, 9, 19, 27.

## TDD Approach

- RED: assert section 5's rendered height is less than section 4's at a desktop viewport;
  assert exactly one suggestion card is present.
- GREEN: build both sections, sizing NUDGE deliberately.
