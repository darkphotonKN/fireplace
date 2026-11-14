---
id: I-XCTMG-1
status: done
implements: FS-XCTMG
blocked_by: []
labels: [feature]
title: "FS-XCTMG slice 1: the hero at x.ai scale and the chrome bar's two pills"
---
Implements FS-XCTMG §Requirements (R4–R8, R23, R24, R26), §Edge States

## What to Build

The first screen becomes the x.ai-scale hero, and the chrome bar gets its two pills. The old story
sections stay below for now; slice 7 removes them.

**CTA pills (R8).** `CtaLink` gains a solid action pill (`bg-action text-action-foreground
rounded-control`) and a grey pill (`bg-muted text-foreground rounded-control`). The existing
`primary` variant moves to `--action` too, so every solid CTA on the page uses it. No hardcoded
colour.

**Hero (R4–R6).** Top to bottom: a small eyebrow (logo + "Fireplace" wordmark, not a heading); the
pill badge "New · Day plan with AI moves ↗", verbatim, as an in-page link to `#today` (the Today
tile gets that id in slice 3; until then the anchor is a no-op, which is fine); the single `h1`
with the anchor phrase verbatim on two lines at `text-5xl`+ on phones and `text-7xl`–`text-8xl`
at `lg`, coloured `foreground` (override the global coral `h1`), with a coral (`primary`)
underline on "fire" as the page's one coral moment; the existing description verbatim as the
subhead at ≈ `text-xl`–`text-2xl`; "Start your plan" (action pill, `/auth?tab=signup`) and "Sign
in" (grey pill, `/auth`). The quiet "See how a day here runs ↓" hint may stay. Keep the `{' '}`
before the line break so the headline is spoken with a pause. `EmberGlow` stays behind it. No
overflow at 320px: the headline wraps.

**Chrome bar (R7).** Same hide-over-hero / fade-in / `inert` behaviour. Left: logo + wordmark.
Right, in order: theme toggle, "Sign in" (grey pill, `/auth`), "Start your plan" (action pill,
`/auth?tab=signup`), sized ≈ `text-sm px-4 py-2`. No profile menu.

Files expected to touch:
- `client/src/components/landing/CtaLink.tsx`
- `client/src/components/landing/sections/HeroSection.tsx`
- `client/src/components/landing/sections/HeroSection.test.tsx`
- `client/src/components/landing/ChromeBar.tsx`
- `client/src/components/landing/ChromeBar.test.tsx`
- `client/src/components/landing/sections/ReturnSection.test.tsx` only if the `CtaLink` change
  breaks an assertion there (ReturnSection itself is slice 7's)

Hotspots: `CtaLink.tsx` is also used by `ReturnSection` (slice 7). Do not edit `globals.css`,
`tailwind.config.js`, `ThemeToggle.tsx` or any other x.ai-pass file.

## Acceptance Criteria

- [ ] The hero shows eyebrow, badge, `h1`, subhead and two pill CTAs in that order, with the copy
      verbatim.
- [ ] Exactly one `h1` on the page; it reads "Start your plan now. Sit down by the fire." with a
      pause between the halves in `textContent`.
- [ ] The `h1` carries `text-7xl` or `text-8xl` at `lg` and at least `text-5xl` below `md`, and a
      `text-foreground` class; "fire" is wrapped in an element with a coral underline.
- [ ] The badge is a link to `#today` and is reachable by Tab.
- [ ] Hero "Start your plan" → `/auth?tab=signup` with `bg-action`; "Sign in" → `/auth` with
      `bg-muted`; both `rounded-control`.
- [ ] No chrome bar at scroll 0; once visible it holds logo, theme toggle, "Sign in", "Start your
      plan" in that order, and no profile menu.
- [ ] No raw hex, `rgb(`, `*-gray-*`, `bg-white` or `text-white` in the touched files.
- [ ] The hero copy is in server-rendered markup; nothing waits on `isLoading`.
- [ ] No x.ai-pass file appears in `git diff --stat`.
- [ ] `npm test` and `npm run lint` pass in `client/`.

## Blocked By

None.

## Spec Reference

FS-XCTMG §Requirements R4 (hero), R5 (sizing, no overflow), R6 (glow), R7 (chrome bar), R8
(CTA pills), R23 (focus order: badge → hero CTAs), R24 (token-only), R26 (first paint);
§Relationship to FS-0003 (R3/R3.1/R4 rows); §Known tensions 3 and 5; §Edge States (auth
resolving, 320px, JS disabled). User stories 1–6, 25–27.

## TDD Approach

- RED: render `HeroSection` and assert one `h1` whose `textContent` is the anchor phrase with a
  space between the halves, carrying `text-foreground` and a `text-7xl`/`text-8xl` `lg:` size.
- GREEN: the new hero markup.
- Then: badge href `#today`; CTA hrefs and variant classes; chrome bar order of controls after a
  mocked scroll past 0.8 viewports, and no profile menu.
