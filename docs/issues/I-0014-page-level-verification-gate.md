---
id: I-0014
status: open
blocked_by: [I-0009, I-0010, I-0011, I-0012, I-0013]
implements: FS-0003
labels: [enhancement]
title: "FS-0003 slice 8: page-level verification gate"
---
Implements FS-0003 §Acceptance Criteria, §Edge States

## What to Build

Not new sections — the checks that **cannot** be performed section-by-section because they are
properties of the assembled page. Per-section theme and accessibility checks already happened in
I-0008–I-0013; this slice covers only what needed the whole tour to exist.

Anything failing here is fixed here, in whichever section owns the defect.

### Whole-page structure

- Six sections render in the specified order: HERO → PLAN → DAILY → REVIEW → NUDGE → RETURN.
- Total scroll length equals visual content length — nothing pins, nothing pads.
- No section is pinned or sticky; scrolling is never intercepted.
- No horizontal overflow at **any** breakpoint.
- Exactly one coral heading exists across the entire page.

### Motion, end to end

- At ≥640px, background / mock art / text move at distinguishable rates (≈0.2× / ≈0.6× / 1.0×).
- Below 640px, no depth translation anywhere; entrance transitions and the hero glow remain;
  **section content and order are identical to desktop** — only the parallax drops.
- Under `prefers-reduced-motion: reduce`, no translation or entrance animation runs anywhere and
  every section is visible and legible at rest.
- **Rapid scroll / fling to the bottom:** every section skipped past renders at rest and fully
  visible. An entrance animation that never fired must never leave content hidden — this is the
  classic scroll-animation bug and it fails silently.
- Only `transform`/`opacity` animate in the scroll path; no layout-triggering properties, no
  per-scroll-event layout reads.

### Access and degradation

- Tab order proceeds sequentially through all six sections and the chrome bar, reaching every
  CTA; mocks are not focusable and expose no interactive roles.
- **JavaScript disabled / pre-hydration:** all six sections' copy and CTAs are present and
  usable. Motion is enhancement, never a gate on content.
- **Crawler / no-JS indexer:** the tour's copy is present in server-rendered markup.
- **200% browser zoom:** layout stays readable, no clipped copy, no horizontal scroll.
- **Very short viewport** (landscape phone): sections stack and scroll normally; no section
  assumes a minimum viewport height to be readable.
- **Very tall / ultrawide viewport:** content stays within the guideline's max container width;
  the ember glow scales without tiling seams.
- **Back-navigation from `/auth`:** the page restores coherently — already-entered sections show
  at rest rather than mid-animation, and scroll restoration does not strand a section invisible.
- **Theme toggled mid-scroll:** colours swap in place; scroll position preserved; no entrance
  animation replays.
- **Print / reader mode:** content degrades to plain readable text; decorative layers do not
  obscure copy.

### Repo-level gates

- `grep` across the landing directory: no raw hex colours, no `*-gray-*` utilities.
- `grep` across the tour's mocks: no `@/components` app imports, no `useAuth`/context
  consumption, no `fetch`/service-layer calls.
- **Isolation holds:** nothing outside the logged-out branch imports from the landing directory.
  This is the only structural brake on the decorative-motion exemption (FS-0003 §Known
  tensions 1) — verify it rather than assume it.
- `LayoutContent.tsx` and the `Dashboard` branch are unmodified.
- The existing `fadeIn` keyframe is unmodified and unused by the tour.
- `npm run lint` and `npm run build` pass.

## Acceptance Criteria

Every unchecked box in FS-0003 §Acceptance Criteria is verified against the assembled page, and
every row of FS-0003 §Edge States behaves as specified. Both documents are the checklist — this
slice does not restate them, it closes them.

- [ ] All FS-0003 §Acceptance Criteria verified on the assembled page.
- [ ] All FS-0003 §Edge States rows manually walked, including the device-dependent ones
      (touch scrolling, 200% zoom, print, back-navigation).
- [ ] Any defect found is fixed in the owning section rather than deferred.

## Blocked By

I-0009, I-0010, I-0011, I-0012, I-0013 — every section must exist before page-level properties
can be verified.

## Spec Reference

FS-0003 §Acceptance Criteria (all), §Edge States (all), §Requirements R1.2, R9, R10, R11, R12,
R18, R20 · Covers user stories 10, 15, 16, 17, 18, 20, 23, 25, 29.

## TDD Approach

Largely manual and device-dependent by nature. Where automatable: assert document scroll width
never exceeds viewport width across breakpoints; assert all six sections' copy is present in the
server-rendered HTML; run the grep gates as a scripted check.
