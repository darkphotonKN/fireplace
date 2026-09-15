---
id: I-0044
status: in-progress
implements: FS-0007
blocked_by: [I-0043]
labels: [feature]
title: FS-0007 slice 2: collapse and expand a parent
---
Implements FS-0007 §Requirements (R5.1–R5.6), §Acceptance Criteria "Collapse"

## What to Build

Give every Parent Item with at least one child a chevron in the left gutter (outside the
checkbox column) that hides and shows its children. Down when expanded, right when collapsed.
On hover-capable devices the chevron is hidden until the row is hovered/focused while expanded,
and always visible while collapsed; under `(hover: none)` it is always visible at low opacity.

Toggling via click/tap, Enter or Space never changes the parent's done state or starts editing.
Hidden children leave the layout, the tab order, and the accessibility tree. The chevron has
`aria-expanded` and an action label. A collapsed parent shows a count after its text: `done/total`
over task children only, or `N notes` / `1 note` when every child is a note. Short height+opacity
transition; reduced motion respected. State is in-memory for this slice (persistence is I-0045).

## Acceptance Criteria

- [x] Clicking/tapping a parent's chevron hides its children; clicking again shows them. Done state and edit mode unaffected.
- [x] Chevron is keyboard-focusable, toggles on Enter and Space, and exposes `aria-expanded` and an action label.
- [ ] Expanded parent: chevron hidden until hover/focus on hover devices; low-opacity visible under `(hover: none)`. Collapsed parent: always visible.
- [x] Collapsed parent with 2 task children (1 done) and 1 note child shows `1/2`.
- [x] Collapsed parent with 3 note children shows `3 notes`; with 1 note child shows `1 note`.
- [x] No count on an expanded parent.
- [x] Hidden children are not in the tab order and not exposed to assistive tech.
- [ ] With reduced motion enabled, toggling has no visible transition.
- [x] Rail (I-0043) is hidden with the children when collapsed.
- [x] Regression checks from I-0043 still pass; client test suite passes.

## Blocked By

I-0043

## Spec Reference

FS-0007 §Requirements R5.1–R5.6; §User Stories 3–6, 19–23, 26, 27; §Acceptance Criteria "Collapse"; §Edge States (touch device, very many children, parent loses last child).

## TDD Approach

- RED: count helper — `[task done, task, note]` → `1/2`; `[note, note, note]` → `3 notes`; `[note]` → `1 note`.
- GREEN: pure count formatter; then a render test that toggling the chevron removes children from the document and flips `aria-expanded`.
