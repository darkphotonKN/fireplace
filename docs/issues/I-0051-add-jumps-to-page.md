---
id: I-0051
status: done
implements: FS-0008
blocked_by: [I-0049, I-0050]
labels: [feature]
title: FS-0008 slice 3: adding takes you to the page the item lands on
---
Implements FS-0008 §Requirements (R15–R18), §Acceptance Criteria "Pinned add bar"

## What to Build

Join paging (I-0049) to the add bar so adding never drops an item out of sight.

- **Tab-to-nest targets the current page:** `findAddBarTarget` (in `client/src/lib/nesting.ts`)
  is given the page's groups, so the target is the nearest eligible top-level task walking up
  from the bottom of the rows *on that page*. `resolveAddBarTarget`'s eligibility rules are
  unchanged.
- **A nested add jumps to its family's page** when that family isn't on the current page, so the
  new child is seen arriving with `animate-fadeIn`.
- **A top-level add jumps to the last page**, where the appended item lands.
- **A jump keeps the add bar's state:** typed text, nesting, and input focus survive it
  (building on FS-0007 R8.6 and the focus-return in `addTodo`).

## Acceptance Criteria

- [x] On page 2 of 3, Tab nests under the last eligible top-level task of page 2.
- [x] Adding a child to a family on another page moves the view to that page and plays the arrival animation.
- [x] A top-level add moves the view to the last page and plays the arrival animation.
- [x] Typed text, nesting state and input focus survive a page jump caused by an add.
- [x] A nested add into a collapsed parent still expands it (FS-0007 R8.9) after the jump.
- [x] Regression: Tab / Shift+Tab, create failure keeping text and nesting, and target-ineligibility reset all still work.
- [x] Client test suite passes.
- [x] HITL: the jump read as following the item rather than as the list moving under you.

Ticked by `client/src/components/Todo.add-page-jump.test.tsx`, with the regression row resting
on the existing `Todo.add-bar-nest.test.tsx`, and the restored arrival assertion in
`Todo.pinned-add-bar.test.tsx`. Two notes on what the tests can and cannot say. The arrival
animation is asserted as the `animate-fadeIn` class on the new row, as elsewhere in this suite;
jsdom runs no animation. And a successful add clears the input by design (FS-0007), so the
"typed text survives" half is proved over a page change with text half-typed, while the
add-caused jump is proved to keep the nesting and the focus. The scroll position of the list
area is deliberately left alone on a page change — see the HITL row.

## Blocked By

I-0049, I-0050

## Spec Reference

FS-0008 §Requirements R15–R18; §User Stories 7–10; §Acceptance Criteria "Pinned add bar" (last four); §Edge States (add while on page 1 of 3, nested add bar whose target is filtered away).

## TDD Approach

- RED: with 25 items and the view on page 1, submitting a top-level add lands the view on page 3 with the new row visible and the input focused and empty.
- GREEN: page maths from I-0049 exposed as "page holding this item"; `addTodo` sets the page after a successful create.
