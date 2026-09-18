---
id: I-0048
status: done
implements: FS-0007
blocked_by: [I-0045, I-0047]
labels: [feature]
title: FS-0007 slice 6: adding into a collapsed parent opens it
---
Implements FS-0007 §Requirements (R8.9), §Acceptance Criteria "Add bar nesting"

## What to Build

Connect the add bar's nesting (I-0047) to collapse (I-0044/I-0045). When the add bar nests under
a collapsed parent, or a nested add lands in a collapsed parent, that parent auto-expands. The
expansion is written to remembered state like any manual expand, and the new item plays the
existing arrival animation where the user can see it.

## Acceptance Criteria

- [x] Tab nesting the add bar under a collapsed parent expands that parent.
- [x] A nested add into a parent that was collapsed after nesting expands it and shows the new item with the arrival animation.
- [x] The expansion is remembered: after reload the parent is expanded.
- [x] Nesting under an already-expanded parent does not write unnecessary state changes.
- [x] Regression checks from I-0043 still pass; client test suite passes.

## Blocked By

I-0045, I-0047

## Spec Reference

FS-0007 §Requirements R8.9; §User Stories 18; §Acceptance Criteria "Add bar nesting" ("Nesting under, or adding into, a collapsed parent expands it, and the expansion is remembered").

## TDD Approach

- RED: with parent A collapsed, pressing Tab in the add bar results in A's children being rendered and A removed from the stored collapsed set.
- GREEN: add bar nesting calls the collapse state's expand for the target.
