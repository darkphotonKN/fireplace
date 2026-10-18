---
id: I-S6WVY-2
status: done
implements: FS-S6WVY
blocked_by: [I-S6WVY-1]
labels: [feature]
title: "FS-S6WVY slice 2: docked sidebar under a fixed top bar, rail and overlay peek, warm retheme"
---
Implements FS-S6WVY §Docked shell, §Peek, §Look

## What to Build

Fix "it shifts the entire site sideways" and "it doesn't feel like part of the site". On top of
slice 1's remembered state and `<html>` attribute, rebuild the shell and restyle the sidebar.

- **Fixed top bar.** Full viewport width on every authenticated route and state; drop its
  sidebar-driven `left`. Logo and "Fireplace" title never move. Hairline `border-border` bottom
  edge.
- **Docked sidebar.** Beneath the top bar, full remaining height, on `bg-layout` (the page's own
  surface) with a hairline `border-border` right edge. Open = 16rem; hidden = a ~3rem **rail**
  holding only the toggle (replacing the mid-height tab and the invisible 50px hover strip).
  Toggle (lucide `PanelLeftClose` / `PanelLeftOpen`) sits at the top of the column in both
  states; accessible name "Hide plans panel" / "Show plans panel", `aria-expanded`,
  `aria-controls`.
- **Reflow only on toggle.** Width animates ~300ms on `cubic-bezier(0.22, 1, 0.36, 1)`; main's
  left offset follows in sync (driven by the same attribute/custom property slice 1 set up).
  Hidden contents are `inert` unless peeking.
- **Peek as overlay.** Hovering the rail for ~150ms shows the full-width panel as an overlay
  (`shadow-lg`, same surface and edge) above content without reflow. Closes on pointer leave,
  `Escape`, focus-out, and navigation. Peek state is local to `Sidebar`, never written. Toggle
  during a peek pins open.
- **Retheme `Sidebar.tsx` to tokens.** Remove `useTheme`, every hex/rgb literal, inline colour
  `style`, gray/amber/orange utilities, `dark:` overrides, inline SVG, and the blur/glow/gradient/
  translate hover layers.
  - Projects/Learning: segmented control, `bg-muted` track, active segment `bg-card
    text-foreground`, inactive `text-muted-foreground`; `aria-pressed`; no coral.
  - Active plan: `text-primary` + thin left coral bar, `aria-current="page"` (the one coral
    moment).
  - Rows: `hover:bg-muted transition-colors`.
  - "New plan": outline/ghost (`border-border text-foreground hover:bg-muted`) with lucide `Plus`,
    sentence case, pinned at the bottom.
  - Coral `focus-visible` ring (`ring-ring`) on every control. Loading/error/empty rows muted.
  - Shared mark (FS-12ZMF R26) unchanged.
- **Reduced motion.** Width change and peek instant (or a plain opacity fade).
- Token values unchanged. If a new semantic token is truly needed, add it to both themes in
  `globals.css` and record it in `client/docs/design-guideline.md`.

Files expected to touch:
- `client/src/components/LayoutContent.tsx` (top bar, main offset)
- `client/src/components/Sidebar.tsx` (dock, rail, peek, restyle)
- `client/src/components/Sidebar.test.tsx` (extend)
- `client/src/app/globals.css` (rail/open widths, transition curve, reduced motion)
- `client/src/components/LayoutContent.test.tsx` (only if the top-bar structure breaks existing
  mocks)
- `client/docs/design-guideline.md` (only if a new semantic token is added)

## Acceptance Criteria

- [ ] The top bar is full width at every state; logo/title position is identical open, hidden and
      peeking.
- [ ] Open sidebar is 16rem with matching content offset; hidden is a ~3rem rail with the toggle,
      with matching content offset.
- [ ] Sidebar and rail share the page surface with a hairline `border-border` edge; the top bar
      has the matching bottom hairline.
- [ ] Toggling animates width ~300ms on the guideline curve with content in sync; with
      `prefers-reduced-motion` it is instant.
- [ ] Hovering the rail ~150ms peeks the panel as a shadowed overlay; content does not move;
      pointer leave, `Escape`, focus-out or navigation closes it.
- [ ] Clicking the toggle during a peek pins open and writes `open`; peeking alone writes nothing.
- [ ] Hidden panel contents are not focusable unless peeking.
- [ ] Toggle exposes "Show plans panel" / "Hide plans panel", `aria-expanded`, `aria-controls`.
- [ ] `Sidebar.tsx` has no hex/rgb literals, colour `style`, gray/amber/orange utilities, `dark:`
      overrides, `useTheme` or inline SVG.
- [ ] Segmented control: muted track, active segment card + foreground, `aria-pressed`, no coral.
- [ ] Active plan: `text-primary`, left coral bar, `aria-current="page"`; no other coral in the
      sidebar.
- [ ] Row hover is a `bg-muted` colour shift only.
- [ ] "New plan" is outline/ghost with a `Plus` icon, sentence case.
- [ ] Every sidebar control shows a coral `focus-visible` ring.
- [ ] The shared mark still renders (existing `Sidebar.test.tsx` case green).
- [ ] Verified in light and dark in the browser.
- [ ] `npm run lint`, `npm run build` and the test suite pass.

## Blocked By

I-S6WVY-1 — this slice builds on its `<html>` attribute and store, and all three components are
shared, so the slices run serially.

## Spec Reference

FS-S6WVY §Docked shell (R14–R20), §Peek (R21–R24), §Look (R25–R34). User stories 1, 2, 8–19,
24, 25. Edge States rows: peek then click a plan, peek then toggle, quick cursor crossing, focus
in/out of a peek, reduced motion, plans fail to load, no plans of the type, many plans,
toggle in another tab while peeking.

## TDD Approach

- RED: `Sidebar.test.tsx`: the toggle is named "Hide plans panel" with `aria-expanded="true"` when
  open; the active plan link has `aria-current="page"`; the Projects segment has
  `aria-pressed="true"` and clicking Learning flips it.
- GREEN: the restyled `Sidebar` markup.
- RED: hidden state renders the toggle "Show plans panel" and the panel is `inert`; hovering the
  rail (fake timers, past ~150ms) shows the panel without calling `setIsCollapsed`; `Escape`
  closes it.
- Visual (manual): both themes, reduced motion on, title stays put across toggle and peek.
