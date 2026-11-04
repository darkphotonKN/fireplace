---
id: I-XCTMG-8
status: done
implements: FS-XCTMG
blocked_by: [I-XCTMG-7]
labels: [feature]
title: "FS-XCTMG slice 8: verify the landing on :3011 in both themes, desktop and mobile"
---
Implements FS-XCTMG §Requirements (R25, R29), §Acceptance Criteria, §Edge States

## What to Build

No code. Run the client from this worktree on :3011 (nvm Node 22, no turbopack on symlinked deps;
the gateway is not needed since the landing fetches nothing) and check `/` logged out in a real
browser.

Capture screenshots:
- light and dark, at desktop (≥1280px) and mobile (375px): four full-page shots;
- one desktop shot of a tile in its end state via hover, and one via keyboard focus;
- the mobile grid after scrolling, showing tiles that played once and rest at their end state.

Walk the §Edge States table where a browser can show it: reduced motion (emulated), theme toggled
while a tile is revealed, 320px width, 200% zoom, fling to the bottom, back from `/auth`. Confirm
mocks are crisp (no raster blur) in both themes and that TODAY reads as inverted in both.

Anything that fails goes back as a finding against the slice that owns it; this slice does not
fix code. Screenshots go in the PR description, not in the repo.

Files expected to touch: none.

## Acceptance Criteria

- [ ] Four full-page screenshots (light/dark × desktop/mobile) attached to the PR.
- [ ] Hover and focus end-state screenshots attached.
- [ ] Mobile shows tiles at rest in their end state after scrolling.
- [ ] No horizontal scroll at 320px or at 200% zoom.
- [ ] Under emulated reduced motion, every tile shows its end state with no animation.
- [ ] Every finding is filed against its owning slice, or the slice is reported clean.
- [ ] No file in the repo is added or changed by this slice.

## Blocked By

I-XCTMG-7.

## Spec Reference

FS-XCTMG §Requirements R25 (both themes), R29 (verification); §Design decisions D11; §Edge
States; §Acceptance Criteria (the screenshot criterion).
