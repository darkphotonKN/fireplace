---
id: I-S6WVY-1
status: done
implements: FS-S6WVY
blocked_by: []
labels: [feature]
title: "FS-S6WVY slice 1: the sidebar remembers open or hidden, and nothing else overwrites it"
---
Implements FS-S6WVY §Remembered state, §Home and the discovery tip

## What to Build

Fix the user-visible bug first: "it still keeps changing even though I'm using the same tab".
After this slice the sidebar is open by default, only the pin/hide toggle changes the remembered
choice, a reload paints the right state on the first frame, other tabs follow, and `/` stops
forcing a collapse. The current look and geometry (16rem panel, top bar offset) stay as they are;
slice 2 replaces them.

- **One store, one writer.** A new `localStorage` key `sidebar` (`open` | `hidden`). No key, or
  any other value, means open. The legacy `sidebarCollapsed` key is never read; the toggle's first
  write removes it. The toggle (`setIsCollapsed` called from `Sidebar`'s pin/hide button and the
  collapsed tab) is the only writer. Suggested home: `client/src/lib/sidebarState.ts` owning the
  key, the parse rule and the head-script source string (FS §Design notes).
- **Pre-paint apply.** A head script in `client/src/app/layout.tsx`, beside the theme script,
  reads the key (inside try/catch) and sets an attribute on `<html>` (e.g. `data-sidebar`). The
  existing geometry (main's left offset, the top bar's `left`, the panel's slide) is driven from
  that attribute through CSS in `globals.css`, so it is right before hydration. React reads the
  state via `useSyncExternalStore` with a server snapshot of `open`; no hydration mismatch.
- **No animation on load.** Width/slide transitions are enabled only after the first frame.
- **Cross-tab.** Subscribe to the `storage` event for `sidebar`; update state and the `<html>`
  attribute; never write.
- **Home respects the choice.** Delete the `/` effect that calls `setIsCollapsed(true)` (this
  supersedes FS-KSJFR R45), and drop the `isHomePage ? "ml-0"` / `left: 0` special cases so `/`
  uses the same offset as every other route.
- **Tip only when hidden.** On plan pages the discovery toast fires only if the sidebar is hidden
  on arrival, judged from the resolved remembered state (not the server snapshot). If it is open,
  skip and leave `hasSeenSidebarHint` unspent. Toggling on the page doesn't re-run the decision.
  Never on `/` (R46 kept). Copy, timing, the 24h reminder and `lastSidebarOpen` are unchanged.

Files expected to touch:
- `client/src/lib/sidebarState.ts` (new) + `client/src/lib/sidebarState.test.ts` (new)
- `client/src/components/LayoutWrapper.tsx` + `client/src/components/LayoutWrapper.test.tsx` (new)
- `client/src/components/LayoutContent.tsx` + `client/src/components/LayoutContent.test.tsx`
- `client/src/components/Sidebar.tsx` (only: geometry classes follow the `<html>` attribute; toggle
  stays the writer. No restyle here.)
- `client/src/app/layout.tsx` (head script)
- `client/src/app/globals.css` (attribute-driven geometry, transition gate)

## Acceptance Criteria

- [ ] With no `sidebar` key, an authenticated page loads with the sidebar open.
- [ ] Hiding with the toggle writes `sidebar=hidden`; reload and every route (including `/`) show
      it hidden. Pinning writes `sidebar=open`; same holds.
- [ ] Navigating between `/`, a plan page and `/myplans` never writes `sidebar` (a test spies on
      storage writes).
- [ ] Hover/peek never writes `sidebar`.
- [ ] Legacy `sidebarCollapsed=true` with no `sidebar` key loads open; the first toggle removes
      `sidebarCollapsed`.
- [ ] An unknown value in `sidebar` loads open.
- [ ] A reload with `sidebar=hidden` paints hidden on the first frame with no slide/width
      animation, and React hydrates without a mismatch warning (manual check in the browser).
- [ ] A `storage` event for `sidebar` from another tab updates this tab without this tab writing.
- [ ] With storage throwing on read and write, the sidebar loads open and the toggle still works.
- [ ] The `/` collapse effect is gone; `LayoutContent.test.tsx`'s R45 block is replaced by tests
      that `/` neither collapses nor writes.
- [ ] The toast never fires on `/`.
- [ ] On a plan page with the sidebar open: no toast, `hasSeenSidebarHint` stays unset.
- [ ] On a plan page with the sidebar hidden: first-timer toast and 24h reminder fire as before.
- [ ] Toggling on a plan page does not fire the toast.
- [ ] `npm run lint`, `npm run build` and the test suite pass.

## Blocked By

None

## Spec Reference

FS-S6WVY §Remembered state (R1–R9), §Home and the discovery tip (R10–R13), plus R19's "`/`
included" (same offset on home). User stories 3–7, 20–23. Edge States rows: no preference,
legacy key, garbage value, storage blocked, reload with hidden, two tabs, arrive on plan page
open, hide on plan page, `/` while authenticated.

## TDD Approach

- RED: `sidebarState.test.ts`: parse returns `open` for missing, `open` for `"true"`/garbage,
  `hidden` for `"hidden"`; legacy `sidebarCollapsed` is ignored.
- GREEN: the parse rule and key in `sidebarState.ts`.
- RED: `LayoutWrapper.test.tsx`: default open; `setIsCollapsed(true)` writes `sidebar=hidden` and
  removes `sidebarCollapsed`; a dispatched `StorageEvent` flips state with no `setItem` call;
  throwing storage still toggles.
- RED: `LayoutContent.test.tsx`: rendering `/` calls no setter and writes nothing; plan page with
  the sidebar open fires no toast and leaves `hasSeenSidebarHint` unset; hidden still fires.
