---
id: I-0023
status: open
implements: FS-0006
blocked_by: [I-0021]
labels: [enhancement]
title: FS-0006 slice 3: plan status FSM
---

Implements FS-0006 §Requirements, §API surface

## What to Build

Plans gain an observable generation lifecycle.

- **Migration**: `status`, `status_changed_at`, `failure_class`, `retry_count`, `last_retry_at`.
  Existing rows backfill to `status='ready'`, `status_changed_at=NOW()`, `failure_class NULL`,
  `retry_count=0`, `last_retry_at NULL` — nothing is in flight at migration time.
- **Transitions** (`generating` | `ready` | `failed`), and only these: creation → `generating`;
  `generating → ready`; `generating → failed`; `failed → generating`; `ready → generating`.
  Everything else is rejected.
- **Not monotonic, deliberately** — retry re-enters `generating`. Enforced by guarded conditional
  updates plus a DB trigger, not by a monotonicity rule.
- **Guarded-update helper**: `UPDATE plans SET status = ... WHERE id = ? AND status = '<expected>'`.
  A second delivery touches zero rows, which is what makes at-least-once delivery of the failure
  event safe without an inbox entry of its own.
- **Transport**: `PlanResp` gains `status` and `failureClass?`.

## Acceptance Criteria

- [ ] Every existing plan is `ready` after migration.
- [ ] Each legal transition succeeds; a table-driven test rejects every illegal one.
- [ ] A guarded update against the wrong expected status affects zero rows and does not error.
- [ ] `failure_class` is only ever set alongside `failed`.
- [ ] `openapi.yaml` and the client regenerate; gates pass.
- [ ] Tests pass.

## Blocked By

I-0021 (shares the `plans` table and the transport type).

## Spec Reference

FS-0006 §Requirements R33–R39 · §API surface (`PlanResp` delta)

## TDD Approach

- RED: case-per-row transition table, every illegal pair expecting rejection.
- GREEN: transition function + trigger + guarded-update helper.
