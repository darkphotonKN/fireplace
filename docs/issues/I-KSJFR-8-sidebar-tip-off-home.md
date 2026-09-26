---
id: I-KSJFR-8
status: in-progress
implements: FS-KSJFR
blocked_by: [I-KSJFR-3]
labels: [feature]
title: "FS-KSJFR slice 8: stop the sidebar tip firing on home"
---
Implements FS-KSJFR §Requirements (R45–R46)

## What to Build

A small correction that only becomes correct once the rest of this feature ships.

`client/src/components/LayoutContent.tsx` fires a discovery toast — "Tip: Your plans live in the
side panel" — on the authenticated home and on plan pages, gated by `localStorage` with a 24h
reminder. Home now lists plans on the page, so on `/` that tip points at a panel showing what
the user can already see, which reads as an unfinished redesign (R46).

- The toast **no longer fires on `/`**.
- It **continues to fire on plan pages**, unchanged — same first-timer hint, same 24h reminder,
  same storage keys.
- The sidebar **still starts collapsed on `/`** (R45). That behavior is deliberate and stays:
  home keeps its clean full-width entry.

Touch only the home condition. This slice is not a refactor of the hint system.

## Acceptance Criteria

- [ ] The discovery toast does not fire on `/`, first visit or later.
- [ ] The discovery toast still fires on a plan page for a first-timer.
- [ ] The 24h reminder behavior on plan pages is unchanged.
- [ ] The sidebar still starts collapsed on `/`.
- [ ] No change to the sidebar's pinned-state behavior on other routes.
- [ ] Tests pass.

## Blocked By

I-KSJFR-3 — the tip is only redundant once home actually lists plans.

## Spec Reference

FS-KSJFR §Requirements (R45–R46).

## TDD Approach

- RED: a test asserting no toast is raised when the authenticated home mounts with the hint
  storage cleared.
- GREEN: drop `/` from the condition.
- Then: the companion test that a plan page still raises it under the same cleared storage.
