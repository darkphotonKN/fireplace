---
id: I-0054
status: done
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

- [x] Two items with the same `sequence` come back in the same order on repeated reads.
- [x] Given equal `sequence`, the older `created_at` comes first.
- [x] Given equal `sequence` and equal `created_at`, order is by `id` and is stable.
- [x] Existing ordering is unchanged where sequences differ.
- [x] The date-window query keeps its date keys first, with the tiebreak after.
- [x] plan-service tests pass.

Ticked by `services/plan-service/internal/checklistitem/repository_order_test.go`, the first
test file this service has ever had. Ordering is a property of the database, so these run
against the local `fireplace_plans` Postgres and skip under `-short`, matching the Makefile's
`test-unit` / `test-integration` split. No CI workflow runs Go tests today (only
`contract.yml`), so nothing downstream starts demanding a database.

Two of the four had a true red-then-green. The `id` tiebreak was already in place by the time
its test was written, so it was proved by removing `, id ASC` and watching it fail. The
date-window test **passed on arrival with two rows** — that query is not a plain table scan and
a wrong order had an even chance of looking right — so it was rebuilt with five rows seeded in
a scrambled order, which failed properly and now passes. The "unchanged where sequences differ"
row is a regression guard, green from the start and honestly labelled as one in the test.

Out of slice: `GetByUserID` (repository.go) orders by `created_at DESC` alone and never touches
`sequence`, so R10 does not reach it. It is non-total too, but that is a different query with a
different key and no caller in this feature.

## Blocked By

None.

## Spec Reference

FS-0009 §Requirements R10; §Out of Scope (the global counter this guards against, which is
deliberately NOT fixed here); §Background.

## TDD Approach

- RED: insert two items with the same `sequence` (and the same `created_at`), list them
  repeatedly, assert the order is identical every time — fails while the order is undefined.
- GREEN: extend the `ORDER BY` with `created_at`, then `id`.
