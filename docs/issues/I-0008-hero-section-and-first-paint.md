---
id: I-0008
status: open
blocked_by: [I-0007]
implements: FS-0003
labels: [enhancement]
title: "FS-0003 slice 2: hero section + first paint"
---
Implements FS-0003 §Requirements (R1, R3, R6, R19), §Edge States

## What to Build

Section 1 of the tour, plus the isolated home all later sections live in, plus the first-paint
fix.

**1. Establish the landing directory.** All tour code — sections, mocks, motion primitives from
I-0007 — lives under one dedicated landing directory and is imported by nothing outside the
logged-out branch of `src/app/page.tsx`. This isolation is the **only** structural brake on the
decorative-motion exemption (FS-0003 §Known tensions 1): the exemption was deliberately not
recorded as a repo-wide ADR, so nothing but this boundary stops it being cited as precedent for
animating product surfaces. Treat it as load-bearing, not organisational taste.

**2. Build the HERO section.**

- The anchor phrase appears **verbatim**: *"Start your plan now. Sit down by the fire."* It is
  documented product voice in `client/docs/design-guideline.md`; rewording it is out of scope.
- The `h1` needs an **explicit size utility** to render at hero scale (≥38px). `globals.css`
  sets `h1 { font-size: 32px }` as a global element selector, so a bare `<h1>` silently renders
  small. The same rule already makes it coral — this `h1` is the page's single coral moment.
- **Replace** the current one-sentence description ("A developer and learning platform that
  houses your checklists…") with a sub-line that front-loads meaning rather than jargon and
  hints the page continues below. *(This is the HITL part — the copy needs a human eye.)*
- The hero renders **chrome-free**: no bar, no logo, no toggle. That is I-0009's job and it
  must not appear here.
- Ambient ember glow as the slowest parallax layer (≈0.2×), via the I-0007 primitive.
- CTAs route to `/auth` and `/auth?tab=signup` — unchanged targets.
- All colour from semantic tokens. No raw hex; the current `LandingPage` hardcodes
  `rgb(247, 111, 83)`, `#2e2e2e`, `#d1cfc0` and none of that survives.

**3. Fix first paint.** `Home` currently gates on `isLoading` and renders a bare `"Loading..."`
string — on the page that is both first impression and SEO surface. The hero is static and
identical for everyone, so it must paint immediately without waiting on auth resolution.

An authenticated visitor arriving at `/` may briefly see the hero before the dashboard replaces
it. **This flash is accepted, not a bug** — gating first paint on auth was the alternative and
was explicitly rejected. Do not "fix" it with a loading gate.

The `Dashboard` branch itself is not modified (FS-0003 D2).

## Acceptance Criteria

- [ ] The hero contains the anchor phrase verbatim: "Start your plan now. Sit down by the fire."
- [ ] The hero `h1` computes to ≥38px, overriding the global 32px element style.
- [ ] The hero is the only coral heading on the page.
- [ ] The new sub-line replaces the old description sentence.
- [ ] No chrome bar, logo, or theme toggle renders at scroll position 0.
- [ ] The ember glow moves at ≈0.2× scroll speed at ≥640px, and not at all below it.
- [ ] CTAs resolve to `/auth` and `/auth?tab=signup`.
- [ ] The hero appears in first paint without waiting on auth; the `"Loading..."` string no
      longer precedes the landing.
- [ ] An authenticated visitor still lands on the dashboard once auth resolves.
- [ ] The hero's copy is present in server-rendered markup.
- [ ] `grep` over the landing directory finds no raw hex and no `*-gray-*` utilities.
- [ ] Hero verified in light and dark mode.
- [ ] `npm run lint` and `npm run build` pass.

## Blocked By

I-0007 — needs the parallax primitive and the resolved `--primary-foreground` token.

## Spec Reference

FS-0003 §Requirements R1.1/R1.2 (scope + isolation), R3/R3.1 (hero + copy), R6 (CTA targets),
R19 (first paint), R13/R14 (tokens, coral discipline) · §Edge States (auth resolving,
authenticated visitor, slow network, crawler) · Covers user stories 1, 2, 12, 21, 22, 24, 25, 29, 30.

## TDD Approach

- RED: assert the hero renders with `isLoading: true` and no `"Loading..."` text is present;
  assert the anchor phrase string matches verbatim; assert no chrome-bar element at scroll 0.
- GREEN: render the hero outside the `isLoading` branch; build the section.
