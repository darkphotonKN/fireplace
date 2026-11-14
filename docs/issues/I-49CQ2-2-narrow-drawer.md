---
id: I-49CQ2-2
status: done
implements: FS-49CQ2
blocked_by: []
labels: [feature]
title: "FS-49CQ2 slice 2: below md the sidebar is a drawer behind a menu button, and the page runs full width"
---
Implements FS-49CQ2 §Sidebar below md: drawer, not dock, §Gutters and top bar (R15)

## What to Build

Fix "RWD doesn't work with the expanding bar". Below 768px the dock goes away and the plans panel
becomes an overlay drawer. At 768px and up nothing about FS-S6WVY changes.

- **No dock below md (R1, R2, R10).** In `globals.css`, inside a `max-width: 767.98px` media
  query, main's offset (`.fp-sidebar-offset`) is 0 and the dock/rail (`.fp-sidebar-dock`,
  including its toggle) is not shown, whatever `html[data-sidebar]` says. CSS only, so the first
  paint is right and hydration matches. The head script and `data-sidebar` keep working for
  docked widths.
- **Menu button (R3, R8, R10).** At narrow widths only, a lucide `Menu` button at the left of the
  top bar, before the logo. Accessible name "Open plans panel" / "Close plans panel",
  `aria-expanded`, `aria-controls` → the drawer. It toggles the drawer. Coral `focus-visible`
  ring. (Sizing to ≥40px is slice 4; give it a sensible size now.)
- **Drawer (R4, R9).** Reuse the panel body of `Sidebar.tsx` (filter, plan list with shared mark,
  "New plan") — e.g. extract it into one component rendered by both the dock and the drawer, so
  FS-S6WVY R25–R31's look is met once. Drawer sits beneath the top bar, full remaining height,
  16rem wide but at most ~85% of the viewport, `bg-layout`, `border-border` right hairline,
  `shadow-lg`; list scrolls inside, "New plan" pinned at the bottom. A scrim dims the page below the
  top bar (token colour, e.g. `bg-foreground/40` or an existing overlay token). Slide-in ~300ms on
  `cubic-bezier(0.22, 1, 0.36, 1)`; instant or plain fade under `prefers-reduced-motion`.
- **Closing (R5, R7).** Closes on route change, scrim tap, `Escape`, the menu button, and a tap on
  any link inside it (so the current plan or Home on `/` still closes it). Crossing to ≥768px
  while open closes it.
- **Focus and scroll (R8).** Opening moves focus to the drawer's first control; closing by
  `Escape`, scrim or menu button returns focus to the menu button. While open, main is `inert` and
  the page does not scroll; the top bar stays usable. Closed, the drawer is `inert`/hidden.
- **Never remembered (R6).** Drawer state is local (not `LayoutWrapper`'s sidebar context), closed
  on every load and navigation, never reads or writes the `sidebar` key, never touches
  `sidebarCollapsed`. `lastSidebarOpen` may be stamped on open (FS-S6WVY R9), not required.
- **Gutters (R15).** Main is `p-4` below md and `p-8` at md and up; `pt-20` unchanged.
- Tokens only for every new class; no new npm dependency (no dialog library is installed).

Files expected to touch:
- `client/src/app/globals.css` (narrow media query for dock/offset; drawer/scrim motion and
  reduced motion)
- `client/src/components/Sidebar.tsx` (extract the shared panel body)
- new `client/src/components/PlansPanel.tsx` (or similar: the shared filter/list/New plan body)
- new `client/src/components/PlansDrawer.tsx` (or similar: menu button + drawer + scrim, local
  state)
- `client/src/components/LayoutContent.tsx` (render the menu button/drawer, main padding, `inert`
  on main while open)
- `client/src/components/Sidebar.test.tsx` (stays green; adjust only for the extraction)
- new `client/src/components/PlansDrawer.test.tsx`
- `client/src/components/LayoutContent.test.tsx` (only if the top-bar structure breaks mocks)
- `client/src/test/motion.ts` (only if the `matchMedia` stub needs a `max-width`/`min-width: 768px`
  answer)

## Acceptance Criteria

- [ ] At a narrow width, the dock and rail are not shown and main has no left offset, with
      `sidebar=open`, `sidebar=hidden` and no key alike; a reload paints full width on the first
      frame with no hydration warning.
- [ ] The menu button shows only at narrow widths, with the right name, `aria-expanded` and
      `aria-controls`.
- [ ] Opening shows the drawer over a scrim with the filter, plan list, shared mark and "New plan";
      focus moves into it.
- [ ] It closes on route change, scrim tap, `Escape`, the menu button and a link tap inside it;
      after `Escape`, scrim or menu-button close, focus is on the menu button.
- [ ] While open, main is not focusable and the page does not scroll; closed, drawer contents are
      not focusable.
- [ ] A test spying on storage shows opening, closing and navigating never write `sidebar`; the
      drawer is closed on load whatever `sidebar` holds.
- [ ] Crossing to ≥768px with the drawer open closes it and the dock shows the remembered state.
- [ ] No slide animation under `prefers-reduced-motion`.
- [ ] Main is `p-4` below md and `p-8` at md and up.
- [ ] At ≥768px the FS-S6WVY behaviour is unchanged (`Sidebar.test.tsx`, `LayoutWrapper.test.tsx`,
      `LayoutContent.test.tsx` green).
- [ ] Checked at 375px in light and dark.
- [ ] `npm run lint`, `npm run build` and `npm test` pass.

## Blocked By

None

## Spec Reference

FS-49CQ2 §Sidebar below md: drawer, not dock: R1–R10; §Gutters and top bar: R15; user stories
1–11, 19, 27, 28; edge states for narrow width, drawer navigation, resize/rotate, blocked storage,
many plans, reduced motion. Supersedes FS-S6WVY R17, R19 below md only.

## TDD Approach

- RED: with `matchMedia` reporting narrow, render the shell and assert a "Open plans panel"
  button exists; click it and assert the drawer is visible, then press `Escape` and assert it is
  gone, focus is on the button, and `localStorage.setItem` was never called with `sidebar`.
- GREEN: the drawer component with local state and the close paths.
