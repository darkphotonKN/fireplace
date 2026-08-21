---
id: I-0030
status: open
implements: FS-0006
blocked_by: [I-0029]
labels: [feature]
title: [HUMAN] FS-0006: insights failure path — DLX tiers, failure event, DLQ
---

> **HUMAN-OWNED — do not run `/develop` on this issue.**
> Flagged in FS-0006 §Ownership split as the owner's lane. An agent must not
> implement it; pick it up only if the owner hands it over explicitly.

Implements FS-0006 §Requirements

## What to Build

Two **disjoint** terminal outcomes. A message ends in exactly one; they never both fire.

**Path 1 — transient, retries exhausted.** Publish `insight.generation_failed` rather than
DLQ'ing, so plan-service can move the plan to `failed` and offer the user a retry on the page they
are already on. Nothing is broken, so the fastest recovery is the user regenerating on the spot; a
DLQ entry waiting on an operator is strictly slower for a problem that will probably work next
attempt.

**Path 2 — bugs, parse errors, unexpected exceptions.** Not retryable; DLQ. No user retry.
*Accepted cost, stated not hidden:* with no DLQ replay path, such a plan gets zero items
**permanently**, not "until it's fixed." Copy must not promise eventual recovery.

**Retry transport is TTL-tiered DLX** — `10s / 20s / 40s / 80s / 160s`, five retry queues plus the
DLQ. A bare `Nack(requeue)` redelivers the *original* message and increments nothing; `x-death` is
populated only by dead-letter round-tripping. `RetryExchange` and `DlxEventsExchange` exist as
constants and are bound for the first time here. The ladder exhausts after 310s of waiting; with
~15s of generation per attempt that is ≈7 minutes to terminal failure.

**The failure publish uses NO outbox.** insights persists nothing about the exhausted attempt — the
count lives in message metadata, the plan status is owned by plan-service — so there is no dual
write to be atomic with. **Publish before ack**: if the publish fails, do not ack; the unacked
message is the durable record. *This holds only while this service stays stateless about failures.*

**Separate delivery-count ceiling on the terminal branch**, distinct from the generation retry
count. Not-acking is reserved strictly for "the failure publish itself failed." Without its own
ceiling, a message already at max attempts that fails to publish and requeues comes back **still at
max**, re-evaluates as exhausted, and spins as fast as the broker delivers.

## Acceptance Criteria

- [ ] Exhausted transient retries publish `insight.generation_failed`; they do not DLQ.
- [ ] A malformed message DLQs; it does not publish a failure event.
- [ ] No message produces both outcomes.
- [ ] `x-death` count increases across attempts — the tiers are real, not a requeue loop.
- [ ] A failure-publish that keeps failing stops at its own ceiling instead of spinning.
- [ ] Nothing about a failed attempt is written to this service's database.
- [ ] Tests pass.

## Blocked By

I-0029.

## Spec Reference

FS-0006 §Requirements R25–R25b, R26–R30
