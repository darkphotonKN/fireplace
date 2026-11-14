---
id: I-49CQ2-6
status: done
implements: FS-49CQ2
blocked_by: [I-49CQ2-1, I-49CQ2-2, I-49CQ2-3, I-49CQ2-4, I-49CQ2-5]
labels: [feature]
title: "FS-49CQ2 slice 6: no sideways scroll or clipping on any signed-in page at 375, 768, 1024 and 1440px"
---
Implements FS-49CQ2 §Acceptance bar

## What to Build

With the shell, grids and top bar done, walk every authenticated route at the four widths and fix
what still overflows, with the smallest local change. "Legit and workable", not a redesign.

- Routes: `/` (dashboard, and the focus prompt when the day gate shows it), `/plan/[planId]`
  (milestone and checklist readings, Day plan, notes, video suggestions), `/plan/[planId]/m/[milestoneId]`,
  `/myplans`, `/calendar`, `/profile`, `/create-plan`.
- Widths: 375, 768, 1024, 1440. States: sidebar open and hidden at ≥768; drawer closed and open at
  375. Both themes.
- Check: `document.documentElement.scrollWidth <= innerWidth`, nothing clipped, every primary
  action reachable (create, tick, open, add, navigate, plans panel, theme, profile, sign out).
  Content inside its own scroller may scroll within itself.
- Fix per R23: wrap, `min-w-0`, `max-w-full`, `truncate`, or stack a row below `sm`. Tokens only
  for anything new. No page redesign, no new dependency, no change to `/learning/*` or the
  logged-out landing.
- Record in the PR description a short table: route × width → pass, or what was fixed.

Files expected to touch (as found; likely candidates):
- `client/src/app/calendar/page.tsx`, `client/src/components/calendar/Calendar.tsx`
- `client/src/app/profile/page.tsx`
- `client/src/app/create-plan/page.tsx`
- `client/src/app/plan/[planId]/layout.tsx`, `client/src/app/plan/[planId]/page.tsx`
- `client/src/components/MilestonePage.tsx`, `client/src/components/MilestoneStrip.tsx`
- `client/src/components/Todo.tsx` and its add bar / item rows
- `client/src/components/DayPlanSection.tsx`, `client/src/components/ChecklistGrid.tsx`
- `client/src/components/home/*`
- `client/src/app/myplans/page.tsx`
- `client/src/components/notes/*`

## Acceptance Criteria

- [ ] At 375, 768, 1024 and 1440px on every route above, in every listed state, there is no
      horizontal page scroll, nothing is clipped and every primary action is reachable.
- [ ] Checked in light and dark.
- [ ] Each fix is local (no redesign); new classes use tokens only.
- [ ] The PR carries the route × width result table.
- [ ] `npm run lint`, `npm run build` and `npm test` pass.

## Blocked By

I-49CQ2-1, I-49CQ2-2, I-49CQ2-3, I-49CQ2-4, I-49CQ2-5

## Spec Reference

FS-49CQ2 §Acceptance bar: R22, R23; user stories 25–26; FS-49CQ2 §Acceptance Criteria
"Acceptance bar (checked in a browser)".
