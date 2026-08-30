# ADR-0011 — Temporal is removed from Fireplace; no orchestrator is pre-selected

Status: accepted
Date: 2026-08-30
Scope: root — governs the compose stack and the escalation path for multi-hop event chains
Amends: **ADR-0008 §6** (Temporal escalation, and the calendar write-back saga). ADR-0008 stands
in every other respect — §§1-5 remain load-bearing for the plan ↔ insights chain and are not
touched here.

## Context

ADR-0008 settled that cross-service generation chains are choreography rather than sagas, and
in §6 named the escape hatch: escalate to Temporal past roughly four hops, with Temporal
otherwise reserved for the review workflow and the platform's first genuine saga — one with real
compensation — expected to be calendar write-back.

That clause rested on a premise stated in ADR-0008's own Context: *"Temporal is already in the
stack, so the escalation path exists and does not need to be built under pressure."* Two things
have since made the premise false.

**Temporal is not free to keep.** `temporal`, `temporal-ui` and `temporal-postgres` are three
containers, one of them a Postgres instance of its own, on the single VPS of ADR-0012 at
$20-40/month. ADR-0010 is collapsing ten Postgres containers to one for exactly this reason;
keeping a fourth database alive to service a workflow engine that nothing calls would undo a
meaningful fraction of that saving. Temporal was never load-bearing — it was present, which is
a different thing, and ADR-0008 read presence as availability.

**The saga it was reserved for is gone.** ADR-0009 folds calendar-service into the gateway.
Calendar write-back therefore stops being a cross-service workflow and becomes a call inside one
process, where a transaction does the job that compensation was going to do. The named future
saga does not exist any more, and no other candidate has appeared.

Verified before removing: **no Fireplace code references Temporal.** A search across `.go`,
`.mod`, `.yml`, `.yaml`, `.toml`, `Makefile` and `.md` returns exactly two hits — the compose
definitions themselves, and ADR-0008 §6. There is no worker, no client, no workflow definition.
Removal is deletion of unused infrastructure, not a migration.

Temporal remains the right tool for workflows with real compensation. Fireplace's pipeline does
not have any: ADR-0008 §2 established that failure here means retry or tell the user, never
unwind, because none of the inverses are wanted. A tool for undoing things is reserved for a
codebase that wants to undo things — which, for the author, is the game project, not this one.

Recorded without adversarial review — locked collaboratively during the consolidation scoping
session and not run through `challenge-me`.

## Decision

**Temporal is removed from Fireplace, and no orchestration engine takes its place in advance.**

1. **Delete `temporal`, `temporal-ui` and `temporal-postgres` from `docker-compose.yml`**, along
   with the `temporal-pgdata` volume. No application code changes, because none references them.
2. **ADR-0008 §6 is replaced, not merely disabled.** Its first sentence — that choreography's
   legibility cost exceeds an orchestrator's operational cost somewhere past four hops — was the
   useful half and **still holds**. What is withdrawn is the answer that followed it.
3. **The escalation path is now: past roughly four hops, orchestration becomes a fresh decision
   with its own ADR.** No tool is pre-selected. Choosing an engine before there is a chain that
   needs one is how Temporal ended up in the stack unused; the counting rule survives, the
   pre-purchased answer does not.
4. **No saga is planned.** ADR-0008's expectation that calendar write-back would be the first
   one is withdrawn along with the service. If a compensation-bearing workflow does appear, it
   is a new decision made against the situation at hand — including whether Temporal comes back.
5. **ADR-0008 §§1-5 are untouched and remain in force:** events are facts not commands, no
   compensation, effectively-once via outbox and inbox with PostgreSQL as authority, guarded
   updates where a message's whole effect is one state transition, and `correlation_id` /
   `causation_id` on every hop. These are what make the plan ↔ insights chain correct and none
   of them depended on Temporal.

## Consequences

**Good**

- Three containers and a Postgres instance leave the stack, which is a material fraction of a
  $20-40/month box and complements ADR-0010's consolidation rather than working against it.
- The escalation trigger — count the hops — survives without the cost of keeping an unused
  engine running to honor it.
- The decision to run an orchestrator gets made when there is something to orchestrate, against
  the tools and constraints of that moment, rather than being inherited from a stack choice made
  before the chain existed.

**Bad / accepted**

- **The escalation path is now a decision to make rather than a tool to reach for.** ADR-0008
  valued having it pre-built so it would not have to be chosen under pressure, and that value is
  real and is being given up. The mitigation is that the trigger is still written down and still
  countable, so the argument starts early rather than at the moment of pain.
- **If a fifth hop arrives sooner than expected, the work is larger than it would have been.**
  Standing up an orchestrator from nothing is more than pointing a worker at a running server.
  Accepted: two hops today, and paying container rent for years against a hop that may never
  come is the more likely waste.
- **ADR-0008's Context now contains a claim that is no longer true** — that Temporal is already
  in the stack. Per the append-only rule its body stays exactly as written; it records what was
  true on 2026-08-19. This ADR is the correction, and ADR-0008's header points here.
- **The game project's use of Temporal is now unrelated infrastructure.** Nothing shared, nothing
  co-located. If both projects eventually want it, they run separate deployments.
