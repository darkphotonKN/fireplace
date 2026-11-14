# FS-49CQ2: Responsive layout on phone, tablet and desktop

> Status: work-order · SPECIFICATION.md: client `## Layout` "Responsive layout on phone, tablet and desktop" → this FS · **Below md (768px) supersedes FS-S6WVY R17 (rail), R19 (content offset) and R21–R22 (peek)** for that width range only; FS-S6WVY is shipped and not edited, and its behaviour at ≥768px is unchanged apart from the card-grid rule (R11–R13 here), which applies at every width · Related FS: FS-S6WVY (docked sidebar; its Out of Scope deferred "mobile / narrow-screen behaviour" to later work, which is this FS), FS-KSJFR (discovery toast, R46) · Related ADRs: none · Governing doc: `client/docs/design-guideline.md`

## Summary

The signed-in app becomes workable on a phone and on a tablet, not only on a desktop. Below
tablet width the plans sidebar stops taking a slice of the screen: the page runs full width and a
menu button in the top bar slides the plans panel in over the page, then out again. Card grids
stop guessing their column count from the screen width and fit as many readable cards as the
space beside the sidebar actually allows, so cards never shrink to a sliver. The top bar, page
gutters and tips are trimmed so nothing is clipped and nothing scrolls sideways at 375px.

The bar is "legit and workable", not polished. Client-only: no backend, API, route or data
change, and no new npm dependency.

## Background: what ships today

Recorded so the work order can be checked against the code it replaces.

- **The sidebar docks at every width.** `Sidebar.tsx` and `globals.css` (`.fp-sidebar-dock`,
  `.fp-sidebar-offset`, `--fp-sidebar-*`, driven by `html[data-sidebar]`) implement FS-S6WVY:
  16rem open or a 3rem rail, and main's `margin-left` follows. On a 375px phone with the sidebar
  open, the content column is about 55px wide.
- **Peek is hover-only, and touch fakes hover.** The dock's `onMouseEnter` starts the 150ms peek;
  a tap on a touch screen emits a synthetic `mouseenter`, so tapping the rail can pop an overlay
  that only closes on `mouseleave`.
- **The top bar is sized for a desktop.** `LayoutContent.tsx`: logo, the "Fireplace" title,
  `HomeButton` (already hides its "Home" label below `sm`), `ThemeToggle` (`p-2` around a 16px
  icon, about 32px) and `UserProfile` (the user's display name as text plus a chevron). The left
  padding (`pl-2.5`) centres the logo over the rail.
- **Main has desktop gutters.** `<main className="pt-20 p-8">` with a `max-w-7xl` inner column.
- **Card grids count columns from the viewport, not the space they get.**
  `ChecklistGrid.tsx` and `DayPlanSection.tsx` (`sm:grid-cols-2 xl:grid-cols-3`),
  `app/plan/[planId]/layout.tsx` video suggestions (`md:grid-cols-2 lg:grid-cols-3`),
  `app/myplans/page.tsx` (`md:grid-cols-2 xl:grid-cols-3`), `home/Dashboard.tsx`
  (`md:grid-cols-3`), `home/FocusPrompt.tsx` (`md:grid-cols-2`). With the sidebar open at a
  640–900px viewport, cards come out around 160px wide.
- **The discovery toast talks about hovering.** On plan pages with the sidebar hidden,
  `LayoutContent.tsx` shows "Hover or click the left edge" (FS-S6WVY R12–R13).
- Tailwind 3.4 with no container-queries plugin; Next 15, React 19. `useMediaQuery`
  (`components/landing/motion/useMediaQuery.ts`) exists and starts `false` on the server and the
  first client render. `src/test/motion.ts` has a `matchMedia` stub.

## Requirements

Decisions marked *charter default* were set by the charter run that scoped this item, each with
its reason, and approved by the user under that charter; none was challenged (client chrome,
cheap to reverse). Decisions marked *scoping call* were settled while writing this spec, to close
a gap the charter left.

Widths below use the guideline's Tailwind breakpoints: **narrow** is below `md` (<768px),
**docked** is `md` and up (≥768px). `sm` is 640px.

### Sidebar below md: drawer, not dock

- **R1** — At narrow widths the sidebar **never docks**: no 16rem column and no rail, and main's
  left offset is **0**, so content runs the full viewport width. This holds whatever the
  remembered `sidebar` value is. Supersedes FS-S6WVY R17 and R19 below md. *Charter default — a
  16rem dock can't share a phone screen.*
- **R2** — The narrow geometry comes from **CSS media queries** on the existing sidebar classes
  (or equivalent), not from JavaScript, so the first paint at a narrow width is already full width,
  with no flash of a dock and no hydration mismatch. *Scoping call — FS-S6WVY R5's no-flicker rule
  extended to this width range.*
- **R3** — At narrow widths the top bar shows a **menu button** at its left edge, before the logo.
  It opens the plans panel as an **overlay drawer** from the left, over a **scrim** that dims the
  rest of the page. *Charter default.*
- **R4** — The drawer holds the **same panel** as the dock: the Projects / Learning filter, the
  plan list (with the FS-12ZMF shared mark) and "New plan", with the same look (FS-S6WVY R25–R31).
  It sits beneath the top bar, full remaining height, 16rem wide but never wider than ~85% of the
  viewport, on `bg-layout` with the `border-border` right hairline and the overlay elevation
  (`shadow-lg`). The plan list scrolls inside it and "New plan" stays pinned at its bottom.
  *Scoping call.*
- **R5** — The drawer **closes** on: navigation (any route change, including tapping a plan or "New
  plan"), a tap on the scrim, `Escape`, and the menu button pressed again. *Charter default
  (navigation, scrim, Escape); the menu button as a toggle is a scoping call, so the control that
  opened it closes it, as FS-S6WVY R16 does for the dock.*
- **R6** — The drawer **never writes** the `sidebar` localStorage key, and nothing done at a
  narrow width reads it to decide whether the drawer is open. The drawer's open state is local to
  the page and is never remembered: it is **closed** on every load and every navigation. The
  remembered open/hidden choice for docked widths is untouched by anything done below md.
  *Charter default — keeps FS-S6WVY R3's single writer.*
- **R7** — If the viewport grows to docked width while the drawer is open (resize, rotation), the
  drawer **closes** and the dock shows the remembered state; shrinking back to narrow shows the
  page full width with the drawer closed. *Scoping call.*
- **R8** — The menu button has an accessible name ("Open plans panel" / "Close plans panel"),
  `aria-expanded`, and `aria-controls` pointing at the drawer. Opening moves focus into the drawer
  (its first control); closing by `Escape`, scrim or the menu button returns focus to the menu
  button. While the drawer is open the page content behind the scrim is not focusable (e.g.
  `inert` on main) and does not scroll; the top bar stays usable. When closed, the drawer's
  contents are not focusable or reachable by assistive tech (as FS-S6WVY R20). *Scoping call.*
- **R9** — Opening the drawer is a short slide-in on the guideline curve
  (`cubic-bezier(0.22, 1, 0.36, 1)`, ~300ms) with the scrim fading; under
  `prefers-reduced-motion` it is instant or a plain opacity fade (FS-S6WVY R33). Width-transition
  suppression on first paint (FS-S6WVY R6) is unaffected.
- **R10** — The dock's own toggle and the rail are **not shown** at narrow widths; the menu button
  is **not shown** at docked widths. Exactly one plans-panel control is visible at any width.
  *Scoping call — follows from R1 and R3.*

### Peek needs a real hover

- **R11** — The rail's hover **peek** (FS-S6WVY R21–R22) starts only when the device can hover,
  i.e. under `(hover: hover)`. On a touch-only device a tap on the rail does not open a peek; the
  toggle on the rail still pins the sidebar open as before (an explicit toggle, so writing
  `sidebar` stays R3-legal). At narrow widths there is no rail, so no peek at all (R1). *Charter
  default — touch has no hover.*

### Card grids follow the space they get

- **R12** — The card grids listed in *Background* (checklist milestone grid, Day plan groups, video
  suggestions, My Plans, home *Continue where you left off*, focus-prompt choices) lay out with
  **intrinsic columns**, `repeat(auto-fill, minmax(min(100%, 18rem), 1fr))`, instead of viewport
  `sm:`/`md:`/`lg:`/`xl:` column counts. This applies at **every** width, docked included.
  *Charter default — the column count follows the real width beside the sidebar, without a
  container-query plugin.*
- **R13** — One definition serves every grid (a named Tailwind `gridTemplateColumns` entry or one
  CSS class), so the 18rem minimum is a single tunable rather than six copies. *Scoping call — the
  guideline asks to avoid arbitrary values.*
- **R14** — Only card grids change. Grids that are not card lists keep their own columns: the
  calendar's 7-day grid, tab lists (`TabsList grid-cols-*`), and the auth page. Gaps, card padding
  and card content are unchanged. *Scoping call.*

### Gutters and top bar

- **R15** — Main's padding is **`p-4` at narrow widths and `p-8` at docked widths**; the top
  clearance for the fixed top bar (`pt-20`) is unchanged. *Charter default — 32px gutters eat a
  phone.*
- **R16** — Below `sm` the **"Fireplace" title text hides** and the logo stays, still linking home;
  the home link keeps an accessible name. *Charter default — title plus four buttons don't fit
  375px.*
- **R17** — The top-bar controls (menu, home, theme, profile) are **at least 40×40px tap targets**
  at narrow widths, without changing their icons or behaviour. *Charter default.*
- **R18** — Below `sm` the profile control shows no display-name text, only its icon/chevron (a
  long name would push the bar past 375px), and keeps an accessible name. *Scoping call — needed to
  meet R22 with a real user name.*
- **R19** — At narrow widths the top bar's left padding matches its right (`pl-4`), since there is
  no rail to centre the logo over; at docked widths it is unchanged (FS-S6WVY R14). The logo and
  title still never move when the drawer opens or closes. *Scoping call.*
- **R20** — Tokens only, per the guideline: new and changed classes use semantic tokens, no raw
  hex, `rgb(...)` or gray/amber utilities. A drifted top-bar control that is touched for R17–R18
  has its touched hover/background classes rethemed to tokens as it goes (guideline *Known
  drift*). No token value changes. *Charter default (tokens); the as-you-go retheme is the
  guideline's own rule.*

### Discovery toast

- **R21** — The sidebar discovery toast (FS-KSJFR R46, FS-S6WVY R12–R13) **does not fire at narrow
  widths**, and `hasSeenSidebarHint` is left unspent there, so a phone visit does not use up the
  first-timer tip a desktop visit would show. Its copy, timing and behaviour at docked widths are
  unchanged. *Charter default (no toast below md — its copy says "hover the left edge"); leaving
  the flag unspent is a scoping call, matching FS-S6WVY R12.*

### Acceptance bar

- **R22** — At **375, 768, 1024 and 1440px** viewport widths, on every authenticated route in
  scope (`/` dashboard, `/plan/[planId]`, `/plan/[planId]/m/[milestoneId]`, `/myplans`,
  `/calendar`, `/profile`, `/create-plan`), with the sidebar open and hidden at docked widths and
  the drawer closed and open at narrow widths: there is **no horizontal page scroll**, **nothing
  is clipped**, and **every primary action is reachable** (create, tick, open, add, navigate, the
  plans panel, theme, profile, sign out). Content inside its own scroller (a long plan list in the
  drawer, a wide table) may scroll within itself. *Charter default — "legit and workable", not
  polished.*
- **R23** — Where a route fails R22 for a reason outside the shell and grids (a fixed-width
  element, a non-wrapping row), the fix is the smallest local change that removes the overflow
  (wrap, `min-w-0`, `max-w-full`, stacking a row below `sm`). No redesign of the page. *Scoping
  call.*

## User Stories

1. As a phone user, I want the page to use the whole screen width, so that I can read my plan
   instead of a 55px sliver.
2. As a phone user, I want a menu button in the top bar, so that I can still get to my plans list.
3. As a phone user, I want the plans panel to slide over the page rather than squeeze it, so that
   the page doesn't reflow every time I open it.
4. As a phone user, I want the panel to close when I pick a plan, so that I land on the plan
   without an extra tap.
5. As a phone user, I want tapping the dimmed area to close the panel, so that dismissing it is
   obvious.
6. As a keyboard user on a narrow window, I want `Escape` to close the panel and put focus back on
   the menu button, so that I don't lose my place.
7. As a screen-reader user, I want the menu button to say whether the panel is open, so that I
   know what it will do.
8. As a phone user, I want the page behind an open panel not to scroll or take focus, so that my
   taps and swipes go where I expect.
9. As a desktop user, I want my pinned-open or hidden sidebar choice to survive anything I do on
   my phone, so that the desktop stays the way I set it.
10. As a user who resizes a window or rotates a tablet, I want the layout to switch cleanly
    between drawer and dock, so that I never see both or neither.
11. As a phone user reloading a page, I want it to paint full width straight away, so that nothing
    flashes a sidebar and then jumps.
12. As a tablet user without a mouse, I want tapping the rail not to pop a panel I can't dismiss,
    so that the sidebar does only what I asked.
13. As a tablet user, I want the rail's toggle to still pin the sidebar open, so that I can browse
    my plans with a tap.
14. As a desktop user with a mouse, I want the hover peek exactly as before, so that nothing I rely
    on changes.
15. As a user with the sidebar open on a mid-size screen, I want milestone cards wide enough to
    read, so that the grid doesn't turn into slivers.
16. As a user on a wide screen with the sidebar hidden, I want the grid to use the extra space for
    more columns, so that I see more at once.
17. As a phone user, I want cards in a single full-width column, so that each card is readable.
18. As a user, I want the Day plan groups, My Plans cards, home cards and video suggestions to
    behave the same way, so that the app feels consistent.
19. As a phone user, I want smaller page gutters, so that content gets the space.
20. As a phone user, I want the top bar to fit without overflowing, so that the theme and profile
    buttons are never cut off.
21. As a phone user, I want top-bar buttons big enough to hit with a thumb, so that I don't tap
    the wrong one.
22. As a phone user with a long name, I want the profile button not to push the bar off screen,
    so that the bar stays usable.
23. As a phone user, I want no tip telling me to hover the left edge, so that I'm not told to do
    something my screen can't.
24. As a user who first opens the app on a phone, I want the sidebar tip still to greet me on the
    desktop, so that I'm not robbed of it by a phone visit.
25. As a user on any authenticated page at any common width, I want no sideways scrolling, so that
    the page feels built for my screen.
26. As a user on a phone, I want to be able to create a plan, tick items, open milestones and sign
    out, so that the app works away from my desk.
27. As a user who prefers reduced motion, I want the drawer to appear without sliding, so that the
    interface doesn't animate at me.
28. As a user in dark mode on a phone, I want the drawer and scrim to match the theme, so that
    both themes feel finished.

## Acceptance Criteria

Drawer below md
- [ ] At a narrow width, the dock and rail are not shown and main has no left offset, with
      `sidebar=open`, `sidebar=hidden` and no key alike.
- [ ] A reload at a narrow width paints full width on the first frame (geometry from CSS media
      queries), with no hydration mismatch warning.
- [ ] At a narrow width the top bar shows a menu button with "Open plans panel" / "Close plans
      panel", `aria-expanded` and `aria-controls`; at docked widths it is not shown.
- [ ] Opening shows the drawer over a scrim with the filter, plan list, shared mark and "New plan";
      focus moves into the drawer.
- [ ] The drawer closes on route change, scrim tap, `Escape` and the menu button; after `Escape`,
      scrim or menu-button close, focus is on the menu button.
- [ ] While open, main is not focusable and the page does not scroll behind the scrim; while
      closed, the drawer's contents are not focusable.
- [ ] Opening, closing and navigating with the drawer never write the `sidebar` key (test spies on
      storage writes), and the drawer is closed on every load whatever `sidebar` holds.
- [ ] Crossing to docked width with the drawer open closes it and shows the remembered dock state.
- [ ] Under `prefers-reduced-motion` the drawer has no slide animation.

Peek
- [ ] With `(hover: hover)` false, entering the rail does not peek; the rail toggle still pins open
      and writes `open`.
- [ ] With `(hover: hover)` true, the FS-S6WVY peek behaves as before (`Sidebar.test.tsx` peek
      cases stay green).

Grids
- [ ] The six card grids use one shared intrinsic-column definition with an 18rem minimum, and
      none carries `sm:`/`md:`/`lg:`/`xl:grid-cols-*`.
- [ ] Calendar, tab-list and auth grids are unchanged.

Gutters, top bar, toast
- [ ] Main is `p-4` below md and `p-8` at md and up; `pt-20` unchanged.
- [ ] Below `sm` the "Fireplace" title text is hidden and the logo still links home with an
      accessible name; the profile control shows no name text and keeps an accessible name.
- [ ] At narrow widths menu, home, theme and profile are each at least 40×40px.
- [ ] At a narrow width on a plan page with `sidebar=hidden`, the discovery toast does not fire
      and `hasSeenSidebarHint` stays unset; at docked widths the FS-S6WVY R12–R13 behaviour holds
      (`LayoutContent.test.tsx` stays green).
- [ ] New and changed classes use tokens only; no raw hex, rgb or gray/amber utilities added.

Acceptance bar (checked in a browser)
- [ ] At 375, 768, 1024 and 1440px, on `/`, `/plan/[planId]`, `/plan/[planId]/m/[milestoneId]`,
      `/myplans`, `/calendar`, `/profile` and `/create-plan`, with the sidebar open and hidden (or
      the drawer open and closed), `document.documentElement.scrollWidth` does not exceed the
      viewport width, nothing is clipped and every primary action is reachable.
- [ ] Checked in both light and dark.
- [ ] `npm run lint`, `npm run build` and `npm test` pass.

## Edge States

| Situation | Behaviour |
|---|---|
| Narrow width, `sidebar=open` stored | Full width, drawer closed; the key is neither read for the drawer nor written (R1, R6). |
| Narrow width, storage blocked | Same as above; the drawer works for the life of the page. |
| Drawer open, tap a plan | Navigates; drawer closes on route change (R5). |
| Drawer open, tap "New plan" | Navigates to `/create-plan`; drawer closes. |
| Drawer open, tap the current plan | Route doesn't change; the tap still closes the drawer. |
| Drawer open, tap Home or the logo in the top bar | Navigates home; drawer closes (or, already on `/`, the tap closes it). |
| Drawer open, rotate or resize to ≥768px | Drawer closes; dock shows the remembered state (R7). |
| Docked, resize to <768px | Dock and rail go; full width; drawer closed; nothing written. |
| Peek open at docked width, resize to narrow | No rail, so no peek; drawer closed. |
| Touch tablet ≥768px | Dock and rail as FS-S6WVY, but no peek on tap (R11); the toggle pins/hides and is remembered. |
| Hybrid device (touch screen and mouse) | Follows `(hover: hover)`; with a mouse the peek works as before. |
| Plans fail to load / no plans | Same muted rows as the dock, inside the drawer. |
| Many plans | List scrolls inside the drawer; "New plan" stays pinned. |
| One card in a grid on a wide screen | One 18rem-plus card; trailing tracks stay empty (auto-fill keeps tracks; accepted). |
| Home with three Continue cards on a very wide content column | Up to four tracks fit, so the third card's row may leave one track empty; accepted under "workable". |
| Grid narrower than 18rem (very small phone) | `min(100%, 18rem)` lets the single column shrink to the container; no overflow. |
| Long user name at 375px | No name text below `sm`; at `sm` and up the name shows as today. |
| Narrow width, plan page, first-ever visit | No toast; `hasSeenSidebarHint` stays unset for a later docked visit (R21). |
| Reduced motion | Drawer and scrim appear without slide (R9). |
| Signed out, `/auth`, logged-out landing | Unchanged: no sidebar, no menu button (FS-0003 owns the landing). |

## Design notes (non-binding guidance for the implementer)

- Narrow geometry can live entirely in `globals.css`: inside `@media (max-width: 767.98px)`, set
  `.fp-sidebar-offset { margin-left: 0 }` and hide `.fp-sidebar-dock`, so the head script and
  `html[data-sidebar]` keep working untouched for docked widths.
- The drawer can reuse the panel's contents by extracting the filter/list/"New plan" body of
  `Sidebar.tsx` into a component rendered by both the dock and the drawer, so the look requirements
  are met once.
- Drawer state is local (a small hook or state in `LayoutContent`/a `MobileNav` component), not
  in `LayoutWrapper`'s sidebar context, which belongs to the remembered state.
- `useMediaQuery` starts `false`, which suits R7 and R11 (closed drawer, no peek until known);
  for R11 a `(hover: hover)` check at peek time, or `pointerType === 'mouse'` on a pointer event,
  avoids touch-emulated `mouseenter`.
- A Tailwind `theme.extend.gridTemplateColumns.cards` entry gives `grid-cols-cards` for R13.
- No dialog library is installed (only Radix checkbox, progress, slot); the drawer is a small
  hand-rolled overlay, per the no-new-dependency bound.

## Out of Scope

- The logged-out landing and product tour (FS-0003 owns them). *Charter default.*
- `/learning/*` pages. *Charter default.*
- Touch drag-to-reorder (FS-0009). *Charter default.*
- Swipe gestures to open or close the drawer.
- Anything beyond "workable": no mobile-specific redesign of any page, no bottom navigation, no
  per-route mobile layouts, no PWA install work. *Charter default.*
- Any backend, API, route or data change; any new npm dependency (including a container-queries
  plugin or a dialog library). *Charter default.*
- Changing FS-S6WVY behaviour at docked widths, other than the grid rule. *Charter default.*
- Changing token values, the type scale or `--radius`.
- Rewording the discovery toast.
- Editing FS-S6WVY or FS-KSJFR (supersession is recorded here only).
- Updating the guideline's "stay usable down to `sm`" line; this FS goes below it (375px) without
  contradicting any visual rule.
