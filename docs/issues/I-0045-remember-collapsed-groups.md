---
id: I-0045
status: in-progress
implements: FS-0007
blocked_by: [I-0044]
labels: [feature]
title: FS-0007 slice 3: remember collapsed groups on this device
---
Implements FS-0007 §Requirements (R6.1–R6.4), §Acceptance Criteria "Persistence", §Edge States

## What to Build

Persist which parents are collapsed in browser storage, on this device only, keyed per plan and
per list scope (daily vs long-term). Store the ids of collapsed parents; anything absent defaults
to expanded. Ids that no longer map to a Parent Item with children are ignored on render and
pruned on the next write. Storage that is unavailable, throws, or holds corrupt JSON degrades to
"everything expanded, nothing remembered" with no error surfaced. No network requests.

## Acceptance Criteria

- [x] Collapsing a parent, then reloading, shows it still collapsed.
- [x] Long-term and daily lists of the same plan keep separate collapse state.
- [x] Collapse state in plan A does not affect plan B.
- [x] No network request is made when collapsing/expanding.
- [x] Removing a parent's last child (outdent/archive/delete) removes its chevron/count; reload does not error and the stale id is pruned on next write.
- [x] With storage throwing on access, the list renders fully expanded and collapse still works for the session.
- [x] Corrupt stored JSON is treated as empty and replaced on next write.
- [x] Regression checks from I-0043 still pass; client test suite passes.

## Blocked By

I-0044

## Spec Reference

FS-0007 §Requirements R6.1–R6.4; §User Stories 7–9, 25, 26; §Acceptance Criteria "Persistence"; §Edge States (storage unavailable/corrupt, concurrent tabs, parent deleted).

## TDD Approach

- RED: storage helper round-trips collapsed ids for `(planId, scope)`; a throwing `localStorage` returns an empty set without throwing; corrupt JSON returns empty.
- GREEN: small storage module wrapping every read/write in try/catch, used by the collapse state.
