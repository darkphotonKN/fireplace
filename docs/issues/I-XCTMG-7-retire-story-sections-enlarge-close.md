---
id: I-XCTMG-7
status: done
implements: FS-XCTMG
blocked_by: [I-XCTMG-1, I-XCTMG-3, I-XCTMG-4, I-XCTMG-5, I-XCTMG-6]
labels: [feature]
title: "FS-XCTMG slice 7: retire the four story sections and enlarge the close"
---
Implements FS-XCTMG §Requirements (R3, R10, R19, R21, R22, R23, R24, R28), §Acceptance Criteria

## What to Build

The page reaches its final shape: hero, grid, close.

**Remove (R28).** Delete `PlanSection`, `DailySection`, `ReviewSection`, `NudgeSection` and their
tests; `SplitSection`; `ART_SPEED` from `motion/speeds.ts`; and the mocks only they used
(`ChecklistMock`, `PlanCardMock`, `DailyReviewMock`, `SuggestionMock`, and `CheckRow` unless a
tile mock now imports it); `GATEWAY_TASK` from `content.ts` once nothing uses it. Keep
`ParallaxLayer`, `GLOW_SPEED`, `EmberGlow`, `RevealOnEnter`, `Section` and
`client/src/test/mockContract.ts`. `LandingTour` renders `ChromeBar`, `HeroSection`, `BentoGrid`,
`ReturnSection` and nothing else (R3).

**Enlarge the close (R22).** `ReturnSection` keeps its copy and the ember glow, scaled up to sit
with the new hero: headline ≈ `text-4xl`–`text-6xl`, `foreground` (not a second coral moment),
body ≈ `text-lg`–`text-xl`. One solid "Start your plan" (`--action`, from slice 1's `CtaLink`)
plus the quiet "Sign in" text link.

**Whole-page checks.** With all five tiles in, confirm R10 at every breakpoint (visual order =
DOM order), R23's tab order (badge → hero CTAs → five tiles → close CTAs) and that no horizontal
overflow appears from 320px to 2560px. Update the remaining landing tests to the new page.

Files expected to touch:
- `client/src/components/landing/LandingTour.tsx`
- `client/src/components/landing/sections/ReturnSection.tsx` + `ReturnSection.test.tsx`
- delete: `sections/PlanSection.tsx`, `DailySection.tsx`, `ReviewSection.tsx`, `NudgeSection.tsx`
  and their `.test.tsx`; `SplitSection.tsx`; `mocks/ChecklistMock.tsx`, `PlanCardMock.tsx`,
  `DailyReviewMock.tsx`, `SuggestionMock.tsx`, possibly `CheckRow.tsx`
- `client/src/components/landing/motion/speeds.ts`
- `client/src/components/landing/mocks/content.ts`
- `client/src/app/page.tsx` only if the logged-out branch needs it (expected: no change)

## Acceptance Criteria

- [ ] `LandingTour` renders exactly chrome bar, hero, grid of five tiles, close.
- [ ] The removed files are gone and nothing imports them; `ART_SPEED` and `GATEWAY_TASK` are gone.
- [ ] The close's copy is unchanged; its headline is ≥ `text-4xl`, `foreground`; it shows one
      `bg-action` "Start your plan" and a quiet "Sign in" link; the ember glow is behind it.
- [ ] Exactly one coral moment on the page (the underline on "fire"), apart from CTAs.
- [ ] Visual order equals DOM order at `sm`, `md` and `lg`; tab order is badge, hero CTAs, the
      five tiles, the close CTAs.
- [ ] No horizontal overflow at 320, 375, 768, 1024, 1280 and 2560px.
- [ ] `grep` over `client/src/components/landing/` finds no raw hex, non-`var()` `rgb(`/`hsl(`,
      `*-gray-*`, `bg-white`/`text-white`/`bg-black`.
- [ ] No content changes as a function of scroll position; no pinning.
- [ ] No x.ai-pass file in `git diff --stat`.
- [ ] `npm run lint`, `npm test` and `npm run build` pass in `client/`.

## Blocked By

I-XCTMG-1 (hero, `CtaLink` pills), I-XCTMG-3, I-XCTMG-4, I-XCTMG-5, I-XCTMG-6 (all five tiles in
place, and `LandingTour.tsx`/`content.ts` free of in-flight edits).

## Spec Reference

FS-XCTMG §Requirements R3 (page order), R10 (final layout), R19 (no scroll mutation), R21 (art
layer removed), R22 (close), R23 (focus order), R24 (token grep), R28 (cleanup); §Relationship to
FS-0003 (R2, R5, R7 rows). User stories 19, 20, 22, 30, 36.

## TDD Approach

- RED: render `LandingTour` and assert its top-level sections are exactly hero, grid, close, and
  that the grid holds five tile links in DOM order PLAN, TODAY, REVIEW, MILESTONES, CALENDAR.
- GREEN: the trimmed `LandingTour` and deletions.
- Then: `ReturnSection` CTA count and classes; tab order across the page.
