---
id: I-0013
status: open
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
