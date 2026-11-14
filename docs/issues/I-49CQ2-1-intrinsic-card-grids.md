---
id: I-49CQ2-1
status: done
implements: FS-49CQ2
blocked_by: []
labels: [feature]
title: "FS-49CQ2 slice 1: card grids take their column count from the space they get"
---
Implements FS-49CQ2 §Card grids follow the space they get

## What to Build

Fix "the grid is too small". The six card grids stop counting columns from the viewport and use
intrinsic columns, so the count follows the real width beside the sidebar at every width.

- Add ONE shared definition, `repeat(auto-fill, minmax(min(100%, 18rem), 1fr))`, as a Tailwind
  `theme.extend.gridTemplateColumns` entry (e.g. `grid-cols-cards`) or one class in `globals.css`
  (R13). The 18rem minimum lives only there.
- Replace the viewport column counts with it in:
  - `ChecklistGrid.tsx` (`sm:grid-cols-2 xl:grid-cols-3`)
  - `DayPlanSection.tsx` (`sm:grid-cols-2 xl:grid-cols-3`)
  - `app/plan/[planId]/layout.tsx` video suggestions (`md:grid-cols-2 lg:grid-cols-3`)
  - `app/myplans/page.tsx` (`md:grid-cols-2 xl:grid-cols-3`)
  - `components/home/Dashboard.tsx` Continue cards (`md:grid-cols-3`)
  - `components/home/FocusPrompt.tsx` choices (`md:grid-cols-2`)
- Keep gaps, card padding, `items-start` and card content as they are.
- Leave non-card grids alone: `calendar/Calendar.tsx` 7-day grids, `TabsList grid-cols-*`
  (`NotesContainer.tsx`, `app/auth/page.tsx`) (R14).

Files expected to touch:
- `client/tailwind.config.js` (or `client/src/app/globals.css`) — the shared definition
- `client/src/components/ChecklistGrid.tsx`
- `client/src/components/DayPlanSection.tsx`
- `client/src/app/plan/[planId]/layout.tsx`
- `client/src/app/myplans/page.tsx`
- `client/src/components/home/Dashboard.tsx`
- `client/src/components/home/FocusPrompt.tsx`
- the matching `*.test.tsx` only if one asserts the old classes (none found at slicing time)

## Acceptance Criteria

- [ ] One shared intrinsic-column definition with an 18rem minimum exists; the six grids use it.
- [ ] None of the six grids carries `sm:`/`md:`/`lg:`/`xl:grid-cols-*`.
- [ ] Calendar, tab-list and auth grids are unchanged.
- [ ] In a browser, with the sidebar open at ~800px viewport the milestone grid shows one or two
      readable columns (no ~160px cards), and at 1440px with it hidden it shows more columns.
- [ ] A grid narrower than 18rem shrinks to one full-width column with no overflow.
- [ ] `npm run lint`, `npm run build` and `npm test` pass.

## Blocked By

None

## Spec Reference

FS-49CQ2 §Card grids follow the space they get: R12, R13, R14; user stories 15–18; edge states
"One card in a grid", "Home with three Continue cards", "Grid narrower than 18rem".
