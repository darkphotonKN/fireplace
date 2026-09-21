---
id: I-0054
status: open
implements: FS-0009
blocked_by: []
labels: [feature]
title: "FS-0009 slice 1: a list never flaps when two items share a sequence"
---
Implements FS-0009 §Requirements (R10), §Edge States

## What to Build

Make the order of checklist items total, so two items with the same `sequence` always come back
in the same order. Today they do not, and ties are already reachable: `service.Create` assigns
`repo.CountItems() + 1`, and `CountItems` is `SELECT COUNT(id) FROM checklist_items` over the
whole table — two creates in flight read the same count and store the same number.

In `services/plan-service/internal/checklistitem/repository.go`:

- Both list queries currently end `ORDER BY sequence ASC`. Extend the tiebreak so the order is
  total — `sequence`, then `created_at`, then `id`. `id` is the last resort that guarantees
  totality: two rows can share a sequence *and* a timestamp.
- `ListInDateWindow` already orders by `COALESCE(start_date, due_date) ASC, sequence ASC`; the
  same tiebreak applies after its existing keys.

This ships on its own and changes no contract, no proto, and no client. It is listed first
because every later slice asserts on order, and an assertion against a flapping list is worth
nothing.

## Acceptance Criteria

- [ ] Two items with the same `sequence` come back in the same order on repeated reads.
- [ ] Given equal `sequence`, the older `created_at` comes first.
- [ ] Given equal `sequence` and equal `created_at`, order is by `id` and is stable.
- [ ] Existing ordering is unchanged where sequences differ.
- [ ] The date-window query keeps its date keys first, with the tiebreak after.
- [ ] plan-service tests pass.

## Blocked By

None.

## Spec Reference

FS-0009 §Requirements R10; §Out of Scope (the global counter this guards against, which is
deliberately NOT fixed here); §Background.

## TDD Approach

- RED: insert two items with the same `sequence` (and the same `created_at`), list them
  repeatedly, assert the order is identical every time — fails while the order is undefined.
- GREEN: extend the `ORDER BY` with `created_at`, then `id`.
