---
id: I-49CQ2-4
status: done
implements: FS-49CQ2
blocked_by: [I-49CQ2-2]
labels: [feature]
title: "FS-49CQ2 slice 4: the top bar fits a 375px phone with thumb-sized controls"
---
Implements FS-49CQ2 §Gutters and top bar (R16–R20)

## What to Build

Title plus menu, home, theme and profile don't fit 375px. Trim the bar at small widths without
changing any control's behaviour.

- **Title (R16).** Below `sm` the "Fireplace" `<h1>` text is hidden (e.g. `hidden sm:block` or
  `sr-only` below sm); the logo stays and still links home; the home link keeps an accessible name.
- **Tap targets (R17).** At narrow widths menu (from slice 2), Home, theme and profile buttons are
  each at least 40×40px (e.g. `h-10 min-w-10` below md), icons unchanged.
- **Profile (R18).** Below `sm` the profile button shows no display-name text (icon/chevron only,
  or an initial) and keeps an accessible name (`aria-label` with the name). At `sm`+ unchanged.
- **Left padding (R19).** At narrow widths the bar's left padding matches its right (`pl-4`); at
  docked widths `pl-2.5` stays (logo centred over the rail). The logo/title never move when the
  drawer opens or closes.
- **Tokens (R20).** New/changed classes use tokens. Where a touched control is drifted (e.g.
  `ThemeToggle`'s `hover:bg-black/5 dark:hover:bg-white/10`, `HomeButton`'s same hover,
  `UserProfile`'s `text-foreground/80`), retheme the touched classes to tokens
  (`hover:bg-muted`, `text-muted-foreground`, …). Coral `focus-visible` ring on each. Re-theming
  `UserProfile`'s dropdown panel is welcome but not required.

Files expected to touch:
- `client/src/components/LayoutContent.tsx` (title visibility, bar padding)
- `client/src/components/HomeButton.tsx` (+ `HomeButton.test.tsx` if it asserts classes)
- `client/src/components/ThemeToggle.tsx`
- `client/src/components/UserProfile.tsx`
- `client/src/components/PlansDrawer.tsx` (or wherever slice 2 put the menu button: size only)
- `client/src/components/LayoutContent.test.tsx` (only if structure changes)

## Acceptance Criteria

- [ ] Below `sm` the "Fireplace" title text is hidden; the logo links home with an accessible
      name.
- [ ] Below `sm` the profile control shows no name text and has an accessible name containing the
      user's name; at `sm`+ it is as before.
- [ ] At narrow widths menu, home, theme and profile each measure at least 40×40px.
- [ ] At 375px with a long display name the top bar does not overflow.
- [ ] New/changed classes use tokens only; no hex, rgb or gray/amber utilities added.
- [ ] Checked in light and dark.
- [ ] `npm run lint`, `npm run build` and `npm test` pass.

## Blocked By

I-49CQ2-2 (the menu button it sizes)

## Spec Reference

FS-49CQ2 §Gutters and top bar: R16–R20; user stories 20–22; edge state "Long user name at 375px".
