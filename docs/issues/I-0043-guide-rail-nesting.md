---
id: I-0043
status: done
implements: FS-0007
blocked_by: []
labels: [feature]
title: FS-0007 slice 1: guide rail ties children to their parent
---
Implements FS-0007 §Requirements (R0.1–R0.3, R1–R4), §Acceptance Criteria "Attachment"

## What to Build

Make a Child Item read as attached to its Parent Item in the checklist list
(`client/src/components/Todo.tsx`). Derive, from the rendered rows, which top-level items are
parents with at least one visible child, and draw a single quiet guide rail: a vertical 1px
hairline starting beneath the parent's checkbox and running down alongside all of its children,
ending at the last child. Child rows sit to the right of the rail. The rail uses the same token
and weight as the add bar's resting hairline (`border-foreground/15`) in light and dark.

A parent with no children, or whose children are all hidden by the active type filter, renders
no rail. Children whose parent is not rendered keep today's behaviour (top level, no indent).

This slice is the foundation the collapse and add-bar slices build on: keep the parent/children
grouping as a small, testable unit rather than inline JSX logic.

## Acceptance Criteria

- [x] A parent with children shows one continuous hairline from beneath its checkbox to its last child; a parent without children shows none.
- [x] The rail is centred on the checkbox column and uses the same colour/weight as the add bar's resting hairline in light and dark themes.
- [x] With a type filter that hides a parent, its children render at top level with no rail (existing behaviour preserved).
- [x] Parent/children grouping is covered by tests (parents with children, childless parents, children whose parent is filtered out, notes as children).
- [x] Regression: row Tab/Shift+Tab indent/outdent, hover-menu actions (type toggle, indent, schedule, edit, archive), and add-bar focus-return after add still work.
- [x] Client test suite passes.
- [x] HITL: visual sign-off on the rail by the user.

## Blocked By

None

## Spec Reference

FS-0007 §Requirements R0.1–R0.3, R1–R4; §User Stories 1, 2, 24; §Acceptance Criteria "Attachment"; §Edge States (empty list, only notes, parent loses last child, parent archived with children remaining).

## TDD Approach

- RED: grouping test — given rows `[A(task), B(child of A), C(note child of A), D(task)]`, A is a parent with children `[B, C]`, D has none; given a filter that removes A, B and C are ungrouped.
- GREEN: extract the grouping from `orderedRows` / `renderedParents` into a pure helper and render the rail from it.
