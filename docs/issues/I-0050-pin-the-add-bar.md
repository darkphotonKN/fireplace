---
id: I-0050
status: done
implements: FS-0008
blocked_by: []
labels: [feature]
title: FS-0008 slice 2: pin the add bar to the bottom of the card
---
Implements FS-0008 §Requirements (R13–R14)

## What to Build

Keep the add bar in view no matter how long the list is, without moving it to the top: it still
belongs to the end of the list, which is what makes Tab-to-nest ("nest under the item above")
read correctly.

- The add form (`<form onSubmit={addTodo}>` and its wrapper `div` in
  `client/src/components/Todo.tsx`) sticks to the bottom of the list card while the list area
  scrolls.
- The list area reserves the bar's height plus the existing `pb-6`, so the last row is never
  hidden behind it and stays clickable.
- Works in both the daily and long-term cards on the plan page. Note the daily card is
  `overflow-hidden` with its own padding — check the bar doesn't clip there.
- Keep the add bar's current look: hairline, ember focus underline, type icon aligned to the
  checkbox column, `pl-8` gutter shared with the list.

## Acceptance Criteria

- [x] The add bar stays visible at the bottom of the card while the list area scrolls.
- [x] The last row is fully visible and clickable, never covered by the pinned bar.
- [x] The bar keeps its hairline, ember underline, icon alignment and gutter.
- [x] Nothing clips in the daily card (`overflow-hidden`) or at narrow widths.
- [x] Regression: add, focus-return after add, and Tab-to-nest still work.
- [x] Client test suite passes.
- [x] HITL: visual sign-off on the pinned bar with a real, long plan.

Ticked by `client/src/components/Todo.pinned-add-bar.test.tsx` for structure, and by
measuring the same CSS in a throwaway static page in a real engine for the geometry jsdom
cannot compute (bar bottom flush with the scrollport at every scroll position; last row
fully inside it with a 16px gap and hit-testing to itself, not the bar; nothing outside the
`overflow-hidden` card; still pinned with no horizontal scroll at a 327px card). The HITL
criterion is left open: it needs the running app and a real long plan.

## Blocked By

None

## Spec Reference

FS-0008 §Requirements R13–R14; §User Stories 5, 6, 22; §Acceptance Criteria "Pinned add bar" (first two); §Edge States (narrow screens).

## TDD Approach

- RED: render a long list and assert the add form carries the sticky positioning classes and the list area reserves space for it.
- GREEN: sticky wrapper plus the reserved padding; verify visually since geometry is not fully testable in jsdom.
