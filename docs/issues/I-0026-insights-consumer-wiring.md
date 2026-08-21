---
id: I-0026
status: open
implements: FS-0006
blocked_by: []
labels: [bug]
title: [HUMAN] FS-0006: start the insights consumer and fix its inbox path
---

> **HUMAN-OWNED — do not run `/develop` on this issue.**
> Flagged in FS-0006 §Ownership split as the owner's lane. An agent must not
> implement it; pick it up only if the owner hands it over explicitly.

Implements FS-0006 §Pre-existing defects this chain depends on

## What to Build

Four defects that stop the chain functioning. Recorded during scoping; none is new work this
feature invented.

1. **The consumer is never started.** `NewConsumer` and `SetupAMQPInfrastructure` are not called
   from `cmd/server/main.go` or `config/services.go`. Nothing consumes plan events today.
2. **`inboxService` is never injected.** `insights.NewService(...)` omits it, so
   `Service.inboxService` is a nil interface and `s.inboxService.CreateTx` panics on the first
   event — the exactly-once path is unreachable rather than merely untested.
3. **The queue binding is wrong.** `amqp_consumer.go` declares and consumes
   `plan-service.events` but binds `insights-service.plan.created`. The bound queue is never
   declared; the consumed queue is plan-service's own. Wired as-is, insights would compete with
   plan-service's `user.deleted` consumer for the same messages. It should bind
   `plan.items_requested` (I-0022), not `plan.created`.
4. **The Redis claim fails closed.** `acquired, _ := s.cache.SetNX(...)` discards the error, so an
   unreachable Redis yields `acquired=false`, which is reported as `ErrEventAlreadyProcessed`,
   which the consumer **acks and drops**. Every event is silently discarded for the whole outage —
   under a comment saying the intent was best-effort. Distinguish `(false, nil)` — a real
   duplicate — from `(false, err)` — Redis could not answer, fall through to the DB, which is the
   authority.

## Acceptance Criteria

- [ ] The consumer starts with the service and consumes a queue it declares itself.
- [ ] It binds `plan.items_requested`, not `plan.created`.
- [ ] `inboxService` is injected; the inbox write and the business write share one transaction.
- [ ] **With Redis stopped, events are still processed and none are dropped.**
- [ ] A Redis error and a genuine duplicate produce different outcomes.
- [ ] Tests pass.

## Blocked By

None (the binding target lands with I-0022, but the wiring can be written against it).

## Spec Reference

FS-0006 §Pre-existing defects, §Requirements R23a–R23b
