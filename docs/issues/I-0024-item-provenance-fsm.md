---
id: I-0024
status: open
implements: FS-0006
blocked_by: []
labels: [enhancement]
title: FS-0006 slice 4: item provenance FSM
---

Implements FS-0006 §Requirements, §API surface

## What to Build

The structural half of ADR-0007 — a one-way state machine the database enforces.

- **Migration**: `checklist_items.status` — `authored` | `generated` | `touched`. Existing rows
  backfill to `authored`, so pre-existing data is user-owned and no regeneration can ever remove it.
- **Only legal transition: `generated → touched`.** `authored` and `touched` are absorbing.
- **Two layers of enforcement**, matching the existing two-tier-nesting idiom: a service-layer
  transition function that is the sole writer of the column, and a `BEFORE UPDATE` trigger
  rejecting illegal transitions with SQLSTATE `23514`.
- **An UPDATE that leaves `status` unchanged is NOT a transition and must pass.** The naive guard
  (`IF OLD.status = 'touched' THEN RAISE`) would reject every update to a touched row — including
  the nightly bulk `DailyReset` CTE, which sets `done=false` and never touches `status`. This is
  the difference between working and a nightly outage.
- **Parent promotion**: when an item moves `generated → touched`, its parent — if `generated` —
  moves to `touched` in the same transaction. `parent_id` is `ON DELETE CASCADE`, so without this
  a later delete of a generated parent would cascade into a touched child and destroy the user's
  work through the FK. Two-tier nesting caps depth at one hop, so no recursion.
- **"Touched" means** any user-initiated mutation — description, `done`, dates, re-parent, scope,
  type, archive — every path through `UpdateItem`, `UpdateItemDates`, `ArchiveItem`. It excludes
  daily reset and materialization, both of which are the system acting.
- **Transport + client**: `ChecklistResp` gains `status`; the client badges `generated` items and
  **drops the badge on `touched`** — the badge means "safe to regenerate away," which stops being
  true the moment the item is protected.

## Acceptance Criteria

- [ ] Every pre-existing item is `authored`.
- [ ] Editing a `generated` item makes it `touched`; editing an `authored` item leaves it `authored`.
- [ ] A raw `UPDATE` attempting `touched → generated` raises SQLSTATE `23514`.
- [ ] `DailyReset` succeeds against `touched` items — the trigger permits the no-op.
- [ ] Editing a child promotes its `generated` parent in the same transaction.
- [ ] Daily reset moves nothing to `touched`.
- [ ] Tests pass.

## Blocked By

None.

## Spec Reference

FS-0006 §Requirements R40–R45, R47 · ADR-0007 · §API surface (`ChecklistResp` delta)

## TDD Approach

- RED: transition table including the no-op row and the daily-reset row.
- GREEN: trigger + service transition function + parent promotion.
