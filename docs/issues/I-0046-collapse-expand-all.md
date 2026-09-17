---
id: I-0046
status: done
implements: FS-0007
blocked_by: [I-0045]
labels: [feature]
title: FS-0007 slice 4: collapse / expand all
---
Implements FS-0007 §Requirements (R7.1–R7.4), §Acceptance Criteria "Collapse / expand all"

## What to Build

A single quiet toggle in the checklist list header, rendered only when at least one Parent Item
has children. Label is "Expand all" if any rendered parent is collapsed, otherwise "Collapse all".
It overwrites per-parent state (not a temporary lens): every rendered parent collapses or expands,
and that result is what gets remembered (I-0045). Parents hidden by the active type filter keep
their stored state.

## Acceptance Criteria

- [x] Toggle is absent when no parent has children; present otherwise.
- [x] Label is "Expand all" when any rendered parent is collapsed, else "Collapse all".
- [x] "Collapse all" collapses every rendered parent; "Expand all" expands every rendered parent; the result survives reload.
- [x] Parents hidden by the active type filter keep their stored state after using the toggle.
- [x] Regression checks from I-0043 still pass; client test suite passes.

## Blocked By

I-0045

## Spec Reference

FS-0007 §Requirements R7.1–R7.4; §User Stories 10, 11; §Acceptance Criteria "Collapse / expand all"; §Edge States (type filter changes while collapsed).

## TDD Approach

- RED: with one of two parents collapsed, the header reads "Expand all"; activating it expands both and the stored set is empty.
- GREEN: header toggle driven by the collapse state's rendered-parent set.
