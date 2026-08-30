# ADR-0008 — Cross-service generation chains are choreography with retries, not sagas

Status: accepted
Date: 2026-08-19
Scope: root — governs event-driven chains spanning services; Temporal's reserved use
Realized by: FS-0006 (the `plan.created` → `insight.generated` chain)
Amended by: **ADR-0011** — §6 (Temporal escalation, and the calendar write-back saga).
§§1-5 stand unchanged.

## Context

FS-0006 introduces the platform's first real multi-service async chain: plan-service publishes
`plan.created`, insights-service generates and persists insights and publishes
`insight.generated`, plan-service consumes that and materializes checklist items. Two hops,
each fire-and-forget, nothing blocking.

Two framing questions had to be answered before writing it, and both get answered wrong by
default if nobody writes them down.

**First: choreography or orchestration?** Choreography — each service reacting to facts, with
no central coordinator — is cheap and keeps services ignorant of each other. Its known failure
mode is legibility: past roughly four or five hops nobody can reconstruct the flow by reading
any single service, and debugging means grep across repos. At two hops that cost has not
arrived. Temporal is already in the stack, so the escalation path exists and does not need to
be built under pressure.

**Second: is this a saga?** It looks like one — multiple services, multiple writes, no
distributed transaction. But a saga's defining feature is **compensation**: each step carries an
inverse, and failure unwinds the completed steps. Ask what the inverses are here and none of
them are wanted:

- Item creation fails → delete the generated insights? No. They are valid, persisted, and
  reusable on retry.
- Item creation fails → delete the plan? Emphatically not. The user created it, it has their
  draft in it, and it is useful with zero items.

Nothing wants unwinding. Failure means *retry, or tell the user* — never roll back. Calling it a
saga would import compensation machinery to model inverses that must never run, and would
frame every failure as "how do we undo this" when the correct question is "how do we finish
this, or say so."

Recorded without adversarial review — locked collaboratively during the FS-0006 scoping session
and not run through `challenge-me`.

## Decision

**Cross-service generation chains are choreography — a pipeline with retries — not sagas.**

1. **Events are facts, not commands.** Name them for what happened (`insight.generated`), never
   for what should happen next (`create_items_for_plan`). The producer does not know who
   listens and must not care. This is what lets a new subscriber — an embedding worker, a
   notification service — attach later without the producer changing at all.
2. **No compensation.** Failure of a downstream step never unwinds an upstream one. Persisted
   artifacts stay persisted; the plan stays created. Recovery is retry or user-visible failure.
3. **Effectively-once on every hop:** transactional outbox on producers, inbox
   (`processed_events (event_id, consumer)`) on consumers, Redis claim as a best-effort
   efficiency layer with PostgreSQL as the authority. The consumer's business effect and its
   inbox write commit in **one transaction**, or neither does.
4. **A single state transition may use a guarded update instead of an inbox.** Where a message's
   whole effect is one status change, `UPDATE ... WHERE id = ? AND status = '<expected>'` makes
   at-least-once delivery safe by construction — a second delivery touches zero rows. Cleaner
   than a dedup protocol for that case, and not a licence to skip the inbox where the effect is
   multi-row.
5. **Propagate `correlation_id` (whole chain) and `causation_id` (immediate parent) through
   every hop, from the first hop.** It costs nothing at two steps and is the only reason a
   longer chain is debuggable later. Retrofitting it after the chain is hard to follow is
   exactly too late.
6. **Escalate to Temporal past roughly four hops.** Beyond that, choreography's legibility cost
   exceeds an orchestrator's operational cost. Temporal is otherwise reserved for the review
   workflow; the platform's first genuine saga — one with real compensation — is expected to be
   calendar write-back.

## Consequences

**Good**

- Services stay mutually ignorant. insights-service does not know plan-service consumes its
  output, and gains subscribers for free.
- No compensation machinery to build, test, or reason about for a chain that has no inverses.
- The escalation trigger is written down in advance, so the choreography-vs-orchestration
  argument does not get re-run per feature — it gets counted.
- Guarded transitions and the inbox give two idempotency tools with a stated rule for which
  applies where.

**Bad / accepted**

- **The flow lives in no single file.** Reconstructing it means reading both services' consumers
  and the routing keys. Acceptable at two hops, and the correlation ids are the mitigation.
- **Failure handling is per-hop, not central.** Each consumer decides ack / retry / DLQ for
  itself, so the policy has to be re-stated (and can drift) at every boundary.
- **A stuck chain has no coordinator to notice it.** Nothing tracks the chain as a whole, so
  each feature that cares must supply its own backstop — a sweeper over rows stuck in an
  in-flight state, with a threshold *derived from* the retry schedule rather than picked, or it
  reports failure on work still legitimately running.
- **Fact-naming makes intent implicit.** `insight.generated` does not say items should follow.
  New readers must find the consumer to learn what happens next; that is the price of the
  decoupling, paid deliberately.
- **The Temporal threshold is a judgment call, not a measurement.** Four hops is a heuristic. A
  chain may become illegible sooner if the hops are subtle, and stay fine longer if they are
  trivial.
