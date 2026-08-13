---
id: I-0009
status: open
blocked_by: [I-0008]
implements: FS-0003
labels: [enhancement]
title: "FS-0003 slice 3: scroll-triggered chrome bar"
---
Implements FS-0003 §Requirements (R4, R14, R20)

## What to Build

A slim warm bar that is **absent over the hero** and fades in once the visitor scrolls past it,
so the first impression stays uninterrupted while the CTA remains one click away through
sections 2–6.

- Carries: the existing `Logo` component (`src/components/Logo.tsx`, wrapping `FireIcon`), the
  existing `ThemeToggle` (`src/components/ThemeToggle.tsx`), and a primary CTA.
- **The tour owns this bar.** `LayoutContent.tsx:83` short-circuits for logged-out `/`
  (`if (isHomePage && !isAuthenticated) return <>{children}</>`), so the landing inherits no
  chrome and none can be obtained from the layout. Do not modify `LayoutContent` to provide it —
  that would leak the landing's chrome into the layout's concerns.
- The theme toggle matters specifically because the warm palette in *both* themes is the point
  of the redesign — a visitor should be able to see it in the mode they'd actually use.
- Appearance/disappearance is opacity-driven; the bar must not cause layout shift in the content
  behind it, and must not introduce horizontal overflow.
- The bar's CTA is coral; section headings elsewhere stay `foreground` (coral discipline, R14).
- Keyboard reachable, and its appearance must not scramble tab order — see the mid-scroll focus
  criterion below.

## Acceptance Criteria

- [ ] No bar is visible at scroll position 0.
- [ ] The bar fades in after the hero is scrolled past, carrying logo, theme toggle, and a
      primary CTA.
- [ ] The bar's CTA resolves to the same auth target as the hero's primary CTA.
- [ ] Toggling the theme from the bar swaps colours in place, preserving scroll position and
      **without replaying** any section's entrance animation.
- [ ] The bar's appearance causes no layout shift and no horizontal overflow.
- [ ] Tabbing while the bar is visible reaches its controls in a sensible order; focus is not
      stolen or lost when the bar appears or disappears.
- [ ] Under `prefers-reduced-motion: reduce`, the bar appears without a fade transition and
      remains fully usable.
- [ ] `LayoutContent.tsx` is unmodified.
- [ ] Verified in light and dark mode.
- [ ] `npm run lint` and `npm run build` pass.

## Blocked By

I-0008 — needs the hero to exist in order to have something to scroll past.

## Spec Reference

FS-0003 §Requirements R4 (chrome bar), R14 (coral discipline), R20 (accessibility) ·
§Edge States (theme toggled mid-scroll) · Covers user stories 11, 13, 14, 18.

## TDD Approach

- RED: assert no bar element is rendered at scroll offset 0; assert it is rendered past the
  hero's height.
- GREEN: gate the bar on scroll offset via the I-0007 primitives.
