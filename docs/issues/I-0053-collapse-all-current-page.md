---
id: I-0053
status: open
implements: FS-0008
blocked_by: [I-0049]
labels: [feature]
title: FS-0008 slice 5: collapse all acts on the current page
---
Implements FS-0008 §Requirements (R23–R24)

## What to Build

Scope the header's Collapse all / Expand all toggle to the page on screen, matching the rule it
already follows for parents the type filter hides (FS-0007 R7.4).

- `collapsibleIds` becomes the parents **on the current page** that have visible children, so the
  toggle only changes those.
- Parents on other pages keep their remembered collapse state, untouched and unpruned.
- The toggle's label ("Expand all" when any of the current page's parents is collapsed, else
  "Collapse all") reflects the current page.
- The toggle is still absent when no parent on the page has children.

## Acceptance Criteria

- [ ] With parents collapsed on page 1, moving to page 2 where none are collapsed shows "Collapse all".
- [ ] "Collapse all" on page 2 leaves page 1's parents as they were; returning to page 1 proves it.
- [ ] The collapse states of off-page parents survive a remount (not pruned by the page's write).
- [ ] The toggle is absent on a page whose parents have no children.
- [ ] Regression: FS-0007's filter rule still holds — parents hidden by the type filter keep their state.
- [ ] Client test suite passes.

## Blocked By

I-0049

## Spec Reference

FS-0008 §Requirements R23–R24; §User Story 18; §Edge States (collapse all on page 2).

## TDD Approach

- RED: with 25 items, collapse all on page 2, then assert page 1's parents are still expanded and page 2's are collapsed after navigating back and forth.
- GREEN: derive `collapsibleIds` from the current page's groups rather than all rendered groups.
