---
id: I-0031
status: open
implements: FS-0006
blocked_by: [I-0022, I-0023, I-0024]
labels: [feature]
title: FS-0006 slice 7: plan-service inbox and materialization
---

Implements FS-0006 §Requirements

## What to Build

plan-service's side of the return hop. It consumes a fact and turns it into rows; it calls nothing
back.

- **`processed_events (event_id, consumer)` migration** — plan-service has an outbox but no inbox.
- Queue `plan-service.insight.generated` declared and bound to `insights.events`, with the DLX
  retry tiers for this hop.
- **Materialization in ONE transaction**: all items plus the `processed_events` row. A partial
  insert followed by redelivery produces a checklist with everything twice — this is the whole
  reason the boundary is where it is.
- Resolve `parent_index` → real `parent_id` within the transaction; **array order is `sequence`**;
  every materialized item is `status='generated'` and is **not** a "touch".
- Transition the plan `generating → ready` in the same transaction — **including when zero items
  arrive.** A plan with no items is a valid end state, not a failure.
- **Redis claim, fail-open** (R23b): only an unambiguous "key exists" may short-circuit. A Redis
  error is *not* a duplicate — fall through to the DB, which is the authority. This gives
  plan-service a Redis dependency it has not had; update `plan-service/CLAUDE.md`.
- **Plan deleted mid-flight**: check the plan exists inside the transaction; if gone, **ack and
  drop, log at info**. Terminal, not an error, not DLQ — the user deleted it deliberately.

## Acceptance Criteria

- [ ] Delivering the same `insight.generated` twice creates the items once.
- [ ] A materialization that fails partway leaves zero items **and** no `processed_events` row.
- [ ] With Redis stopped, events still process exactly once and none are dropped.
- [ ] Zero items → plan reaches `ready`, never `failed`.
- [ ] A deleted plan's event is acked and dropped without an error log.
- [ ] Nesting round-trips: `parent_index` produces the right `parent_id`, two-tier still enforced.
- [ ] `correlation_id` / `causation_id` are read from the envelope and logged.
- [ ] Tests pass.

## Blocked By

I-0022, I-0023, I-0024. **Not** blocked by I-0029/I-0030 — testable against a hand-published
message; only the end-to-end chain needs the producer.

## Spec Reference

FS-0006 §Requirements R19a–d (consumer half), R21–R23c, R25, R34

## TDD Approach

- RED: publish the same event twice; assert one set of items and one ledger row.
- GREEN: inbox insert first inside `ExecTx`, then the item writes.
