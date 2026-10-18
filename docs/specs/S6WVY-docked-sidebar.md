# FS-S6WVY: Docked sidebar that remembers open or hidden

> Status: shipped · SPECIFICATION.md: client `## Navigation` "Docked sidebar that remembers open or hidden" → this FS (retrofit; the existing sidebar is `Plans sidebar → FS-none`) · **Supersedes FS-KSJFR R45** ("the sidebar still starts collapsed on `/`") and the matching KSJFR acceptance line · Related FS: FS-KSJFR (R46 discovery toast, kept), FS-12ZMF (R26 shared mark in the sidebar, kept) · Related ADRs: none · Governing doc: `client/docs/design-guideline.md`

## Summary

The plans sidebar becomes part of the page instead of a slab that slides over it. It docks under
a top bar that never moves, so the logo and "Fireplace" title stay put whatever the sidebar does.
Hidden, it shrinks to a slim rail that keeps its toggle in view; hovering the rail peeks the panel
over the page without shoving anything. It is open by default, and whatever the user last chose
with the toggle is what every page, every reload and every other tab shows, until they choose
again.

Client-only. No backend, API, route or data change.

## Background: what ships today

Recorded so the work order can be checked against the code it replaces.

- **The preference is overwritten by navigation.** `LayoutContent.tsx` runs an effect on `/` that
  calls `setIsCollapsed(true)` (FS-KSJFR R45). That setter (`LayoutWrapper.tsx`) also writes
  `localStorage.sidebarCollapsed = 'true'`, so every visit home silently replaces a pinned-open
  choice. This is the user's "it still keeps changing even though I'm using the same tab".
- **Reload flashes.** `LayoutWrapper` starts `isCollapsed` at `true` and reads storage in a mount
  effect, so the first paint is always collapsed and then snaps to the stored value.
- **The whole site slides.** The top bar's inline `left` and main's `ml-64` both follow the
  sidebar, so the logo and title travel 16rem on every toggle. On `/` the sidebar is forced out of
  the layout entirely (`ml-0`).
- **It looks bolted on.** `Sidebar.tsx` paints its own raw-hex/rgb background per theme (via
  `useTheme`), a coral-filled Projects/Learning switch with `text-white`, amber blur/glow/gradient
  hover layers, a `translate-x-1` text nudge, inline SVG chevrons, and a coral "+ New Plan" in
  Title Case. Collapsed, it is a 20px tab at mid-height plus an invisible 50px hover strip.
- **The discovery toast** (`LayoutContent.tsx`, FS-KSJFR R46) fires on plan pages for a
  first-timer and as a 24h reminder keyed on `lastSidebarOpen`, regardless of whether the panel is
  already open.

## Requirements

Decisions marked *charter default* were set by the charter run that scoped this item, each with
its reason; none was challenged (client chrome, cheap to reverse).

### Remembered state

- **R1** — The sidebar has exactly two remembered states, **open** and **hidden**. With no
  stored preference it is **open**. *Charter default — the user asked for open by default.*
- **R2** — The preference lives under **one** `localStorage` key, `sidebar`, holding `open` or
  `hidden`. Any other value, or a missing key, means "no preference" (R1).
- **R3** — The **only writer** of that key is the explicit pin/hide toggle (R16). Navigation,
  route effects, hover/peek, the discovery tip and cross-tab sync never write it. *Charter
  default — "remembering shouldn't be that hard"; a single writer makes it impossible for
  anything else to overwrite the choice.*
- **R4** — The legacy key `sidebarCollapsed` is **not read**. Its `true` has been written by the
  home effect for anyone who visited `/`, so it does not record a choice; honouring it would
  hide the sidebar for nearly every existing user and break R1. The toggle's first write removes
  it. (Scoping call, not charter text.)
- **R5** — The stored state is applied **before first paint** by a head script in
  `client/src/app/layout.tsx`, beside the theme script and using the same pattern, so a reload
  never shows open then hidden (or the reverse). The script tolerates blocked storage. *Charter
  default — no flicker on load.*
- **R6** — Width transitions are **suppressed on the initial paint** and enabled only after the
  page has settled, so a hidden-by-preference load does not animate closed.
- **R7** — **Other tabs follow.** A toggle in one tab updates every other open tab of the app
  (via the `storage` event) without those tabs writing anything. *Charter default.*
- **R8** — Blocked storage (private mode, quota, policy) never breaks the sidebar: it opens by
  default and the toggle still works for the life of the page; it just isn't remembered.
- **R9** — `lastSidebarOpen` (the toast's 24h reminder stamp) is **not** the open/hidden state and
  keeps its current job; it may still be stamped by opening or peeking. It is not read by R1–R8.

### Home and the discovery tip

- **R10** — `/` (the authenticated home) **respects the remembered state** like every other page.
  The home effect that forces a collapse is removed. **This supersedes FS-KSJFR R45.** *Charter
  default — R45's forced collapse is what overwrote the user's preference.* (FS-KSJFR itself is
  not edited; this header records the supersession.)
- **R11** — The "your plans live in the side panel" discovery toast still **never fires on `/`**
  (FS-KSJFR R46, kept).
- **R12** — On plan pages the toast fires **only when the sidebar is actually hidden** when the
  user arrives on the page, judged from the remembered state, never from a pre-hydration default.
  When it is open, the toast is skipped and the first-timer flag (`hasSeenSidebarHint`) is **left
  unspent**. *Charter default — a tip pointing at an already-open panel is noise.*
- **R13** — Using the toggle on a plan page does not itself trigger the toast; the tip is decided
  on arrival, not re-evaluated on every toggle. Copy, timing (1.5s first-timer delay), position
  and the 24h reminder are otherwise unchanged.

### Docked shell

- **R14** — The top bar spans the **full viewport width** at every sidebar state and on every
  authenticated route. The logo and "Fireplace" title **never move** when the sidebar opens,
  hides or peeks. *Charter default — the sliding title is the user's main complaint.*
- **R15** — The sidebar docks **beneath** the top bar, full remaining height, on the **same
  surface as the page** (`bg-layout`), separated from the content by a hairline `border-border`
  right edge; the top bar carries the same hairline as its bottom edge so the two read as one
  frame. No differently coloured panel. *Charter default — "doesn't feel like part of the site".*
- **R16** — The pin/hide **toggle** sits at the top of the sidebar column in the same place in
  both states (open: in the sidebar header; hidden: at the top of the rail), so the pointer
  doesn't hunt for it. Icons are lucide-react `PanelLeftClose` / `PanelLeftOpen`.
- **R17** — **Open** width is 16rem. **Hidden** is a slim **rail** of ~3rem holding only the
  toggle, not zero width. *Charter default — a smaller, legible shift; a visible handle beats an
  invisible edge.*
- **R18** — Toggling animates the sidebar's **width** over ~300ms on the design-guideline curve
  (`cubic-bezier(0.22, 1, 0.36, 1)`), and the main content's left offset **reflows in sync** with
  it. Nothing else on the page moves.
- **R19** — The main content's left offset equals the sidebar's current docked width (16rem or the
  rail) on every authenticated route, `/` included.
- **R20** — When hidden, the panel's contents (filter, plan links, "New plan") are **not
  focusable or reachable by assistive tech** (e.g. `inert`) unless the panel is peeking.

### Peek

- **R21** — Hovering the rail **peeks** the panel: it appears at full open width as an **overlay**
  above the content with a soft shadow (`shadow-lg`, the guideline's overlay elevation), on the
  same surface and hairline edge. The content **does not reflow**. *Charter default — peeking
  shouldn't shove the page around.*
- **R22** — The peek opens after a short hover-intent delay (~150ms) so a cursor crossing the left
  edge doesn't flash it, and closes when the pointer leaves the panel, on `Escape`, when focus
  leaves it, and on navigation.
- **R23** — Peeking **never writes** the stored state (R3). Clicking the toggle while peeking pins
  the sidebar open (writes `open`, content reflows per R18).
- **R24** — Only the toggle changes the docked width. Peek and un-peek animate opacity/transform of
  the overlay, not layout.

### Look

All of these follow `client/docs/design-guideline.md`, which wins on any visual disagreement.
*Charter default for the whole section — "warm minimalism", "one coral moment", "motion
communicates, never decorates".*

- **R25** — **Tokens only.** No raw hex, no `rgb(...)`/`rgba(...)` literals, no `*-gray-*` /
  `*-amber-*` / `*-orange-*` utilities, no inline colour `style`, no per-component dark-mode
  branches. `Sidebar` no longer reads `useTheme`. Token **values** are unchanged; if a new
  semantic token is truly needed it is added to both themes in `globals.css` and recorded in the
  guideline.
- **R26** — **Projects / Learning** is a quiet segmented control: a `bg-muted` track; the active
  segment is the card surface (`bg-card`) with `text-foreground`; inactive segments are
  `text-muted-foreground`, darkening on hover. No coral fill. Labels in sentence case. The active
  segment is exposed to assistive tech (`aria-pressed` or equivalent).
- **R27** — **Active plan** is the view's one coral moment: `text-primary` plus a thin coral bar
  on the row's left edge, and `aria-current="page"`. No coral background wash.
- **R28** — Plan rows hover with a calm colour shift (`hover:bg-muted`, `transition-colors`).
  The blur, glow, gradient layers and the `translate-x` nudge are removed.
- **R29** — **"New plan"** is an outline/ghost button (`border-border`, `text-foreground`,
  `hover:bg-muted`) with a lucide `Plus` icon, in sentence case ("New plan"), pinned to the bottom
  of the panel. Not coral.
- **R30** — Every interactive element in the sidebar and rail (toggle, segments, plan links, New
  plan) shows a visible **coral `focus-visible` ring** (`ring-ring`).
- **R31** — Loading, error and empty rows use `text-muted-foreground`; empty copy stays
  invitational ("No projects yet" / "No learning plans yet").
- **R32** — The shared mark on member plans (FS-12ZMF R26) is unchanged.
- **R33** — Under `prefers-reduced-motion` the width change and peek are instant (or a plain
  opacity fade); no width or transform animation.
- **R34** — Both themes are verified; every colour comes from a token defined in both.

## User Stories

1. As a user, I want the "Fireplace" title and logo to stay exactly where they are when I open or
   hide the sidebar, so that the site doesn't feel like it lurches sideways.
2. As a user, I want the sidebar to look like part of the page rather than a separate coloured
   slab, so that the app feels like one warm, calm place.
3. As a new user, I want the sidebar open the first time I sign in, so that I can see my plans
   without discovering a hidden panel.
4. As a user who hid the sidebar, I want it to stay hidden on every page I visit, so that I don't
   have to hide it again after each click.
5. As a user who pinned the sidebar open, I want visiting home not to hide it, so that my choice
   isn't quietly undone.
6. As a user, I want my choice to survive a reload without the panel flickering open then shut,
   so that the page settles once.
7. As a user with two tabs open, I want hiding the sidebar in one to hide it in the other, so that
   the app behaves like one app.
8. As a user with the sidebar hidden, I want a slim rail with a visible toggle, so that I can
   always find the way back.
9. As a user with the sidebar hidden, I want to glance at my plans by hovering the rail without
   the page jumping, so that I can check something and carry on.
10. As a user moving my mouse past the left edge, I want nothing to pop open unless I pause there,
    so that the panel doesn't flash at me.
11. As a user peeking at the panel, I want to pin it open with the same toggle, so that I don't
    have to learn a second control.
12. As a user, I want the open/hide change to be a short, smooth width change that the content
    follows, so that I can see what changed and where.
13. As a user who prefers reduced motion, I want the sidebar to change instantly, so that the
    interface doesn't animate at me.
14. As a user, I want the current plan to stand out in the list in the brand colour, so that I
    always know where I am.
15. As a user, I want Projects and Learning to be a quiet switch, so that the filter doesn't
    compete with the plan I'm in.
16. As a user, I want hovering a plan to be a gentle highlight rather than glows and slides, so
    that the list feels calm and productive.
17. As a user, I want "New plan" to be easy to find but not shout, so that creating a plan is
    available without pulling focus.
18. As a keyboard user, I want to toggle the sidebar and reach every plan with the keyboard and a
    visible focus ring, so that I'm never stuck needing a mouse.
19. As a keyboard or screen-reader user with the sidebar hidden, I want its hidden links out of
    the tab order, so that I don't tab through invisible items.
20. As a returning user on a plan page with the sidebar open, I want no tip telling me my plans
    are in the side panel, so that I'm not told what I can already see.
21. As a returning user on a plan page with the sidebar hidden, I want the discovery tip to still
    help me find it, so that the rail isn't a mystery.
22. As a user who just hid the sidebar on a plan page, I want no tip to appear as a reaction, so
    that hiding it doesn't nag me.
23. As a user in a private window, I want the sidebar to still open and hide, so that blocked
    storage doesn't break navigation.
24. As a user in dark mode, I want the sidebar to match the page's warm charcoal, so that both
    themes feel equally finished.
25. As a member of a shared plan, I want its "shared" mark to stay in the sidebar list, so that the
    redesign doesn't lose what sharing added.

## Acceptance Criteria

Remembered state
- [ ] With no `sidebar` key, an authenticated page loads with the sidebar open.
- [ ] Hiding with the toggle writes `sidebar=hidden`; reload and every route (including `/`) show
      it hidden; pinning writes `sidebar=open` and the same holds.
- [ ] Navigating between `/`, a plan page and `/myplans` never writes the `sidebar` key (verified
      by a test that spies on storage writes).
- [ ] Hover/peek never writes the `sidebar` key.
- [ ] A legacy `sidebarCollapsed=true` with no `sidebar` key loads open; the first toggle removes
      `sidebarCollapsed`.
- [ ] An unknown value in `sidebar` loads open.
- [ ] A reload with `sidebar=hidden` paints hidden on the first frame with no width animation (no
      open-then-hidden flash), and React hydrates without a mismatch warning.
- [ ] A `storage` event for `sidebar` from another tab updates this tab's sidebar without this
      tab writing.
- [ ] With storage throwing on read and write, the sidebar loads open and the toggle still works.

Home and the tip
- [ ] The home effect that forces a collapse is gone; `/` shows the remembered state (R45
      superseded; `LayoutContent.test.tsx`'s R45 expectations are replaced by R10's).
- [ ] The toast never fires on `/` (R46 kept).
- [ ] On a plan page with the sidebar open, the toast does not fire and `hasSeenSidebarHint` stays
      unset.
- [ ] On a plan page with the sidebar hidden, the first-timer toast and the 24h reminder fire as
      before.
- [ ] Toggling on a plan page does not fire the toast.

Docked shell and peek
- [ ] The top bar is full width at every state; the logo/title's position is identical open,
      hidden and peeking.
- [ ] Open sidebar is 16rem and the content offset matches; hidden is a ~3rem rail with the
      toggle, and the content offset matches.
- [ ] Sidebar and rail share the page surface with a hairline `border-border` edge; top bar has
      the matching bottom hairline.
- [ ] Toggling animates width ~300ms on the guideline curve with content in sync; with
      `prefers-reduced-motion` it is instant.
- [ ] Hovering the rail for ~150ms peeks the panel as a shadowed overlay; content does not move;
      leaving, `Escape`, focus-out or navigation closes it.
- [ ] Clicking the toggle during a peek pins open and writes `open`.
- [ ] Hidden panel contents are not focusable unless peeking.
- [ ] The toggle has an accessible name ("Show plans panel" / "Hide plans panel"),
      `aria-expanded`, and `aria-controls` pointing at the panel.

Look
- [ ] `Sidebar.tsx` contains no hex/rgb literals, no colour `style`, no gray/amber/orange
      utilities, no `dark:` overrides, no `useTheme`, and no inline SVG (lucide only).
- [ ] Projects/Learning: muted track, active segment card surface + foreground, no coral;
      active state exposed to assistive tech.
- [ ] Active plan: `text-primary`, thin left coral bar, `aria-current="page"`; no other coral in
      the sidebar.
- [ ] Rows hover with `bg-muted` colour shift only; no blur/glow/gradient/translate.
- [ ] "New plan" is an outline/ghost button with a `Plus` icon, sentence case.
- [ ] Every sidebar control shows a coral `focus-visible` ring.
- [ ] The FS-12ZMF shared mark still renders on member plans (`Sidebar.test.tsx` stays green).
- [ ] Checked in both light and dark.
- [ ] `npm run lint`, `npm run build` and the test suite pass.

## Edge States

| Situation | Behaviour |
|---|---|
| No stored preference | Open (R1). |
| Legacy `sidebarCollapsed` only | Ignored; open. Removed on first toggle (R4). |
| Garbage in `sidebar` | Treated as no preference; open. |
| Storage blocked | Open; toggle works in-page; not remembered (R8). Head script swallows the error. |
| Reload with `hidden` | First paint hidden, no animation (R5–R6). |
| Two tabs, toggle in A | B follows via `storage`; B writes nothing (R7). |
| Toggle in A while B is peeking | B's docked width updates; its peek closes or stays as an overlay, never writes. |
| Peek, then click a plan | Navigates; peek closes on route change; stored state unchanged. |
| Peek, then click toggle | Pins open, writes `open`, content reflows (R23). |
| Cursor crosses the rail quickly | No peek (hover-intent delay, R22). |
| Keyboard focus moves into a peek, then out | Peek closes on focus-out. |
| Reduced motion | Instant width change; peek is instant or an opacity fade (R33). |
| Arrive on a plan page with sidebar open | No toast; first-timer flag unspent (R12). |
| Hide the sidebar while on a plan page | No toast as a reaction (R13). |
| `/` while authenticated | Remembered state, same geometry as other routes (R10, R19). |
| Signed out, or `/auth` | No sidebar and no top-bar change, as today. |
| Plans fail to load | Muted error row; toggle and rail still work. |
| No plans of the selected type | Muted invitational empty row. |
| Many plans | The plan list scrolls inside the panel; "New plan" stays pinned at the bottom. |

## Design notes (non-binding guidance for the implementer)

- A small module (e.g. `client/src/lib/sidebarState.ts`) can own the key, the parse rule (R2/R4),
  and the head-script source string, so the pre-paint script and the React store share one
  definition and both are unit-testable.
- The theme pattern generalises: the head script sets an attribute on `<html>` (e.g.
  `data-sidebar="open|hidden"`), and CSS drives the docked width from it (e.g. a
  `--sidebar-width` custom property), so geometry is right before React hydrates. React reads
  the same state through `useSyncExternalStore` (server snapshot `open`), so hydration matches
  and the client re-renders only ARIA/labels, not geometry.
- Transition suppression (R6) can be a second attribute set after the first frame that turns
  the width transition on.
- Peek state is local to `Sidebar`; it is never part of the context or storage.

## Out of Scope

- Mobile / narrow-screen behaviour (off-canvas drawer, breakpoints below tablet). Desktop and
  tablet only, as today. *Charter default.*
- Any backend, API, route or data change. *Charter default.*
- Changing token values, the type scale or `--radius`. *Charter default.*
- A keyboard shortcut for the toggle.
- Resizable sidebar width; remembering the Projects/Learning filter; reordering or searching
  plans in the sidebar.
- Rewording the discovery toast or changing its timing/storage keys beyond R12–R13.
- Editing FS-KSJFR (supersession is recorded here only).
- Moving `bg-layout`'s raw values into tokens (pre-existing drift in `globals.css`; R25 governs
  `Sidebar.tsx` and the new shell code, not that class).
