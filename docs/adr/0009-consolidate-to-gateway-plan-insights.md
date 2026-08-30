# ADR-0009 — Fireplace runs three services: gateway, plan-service, insights-service

Status: accepted
Date: 2026-08-30
Scope: root — governs which domains run as their own process, and which do not
Related: ADR-0008 (the plan ↔ insights chain this preserves), ADR-0010 (the database
layout that follows from it), ADR-0012 (the hosting posture that motivated it)

## Context

Fireplace extracted five services plus an orchestrator from the original gateway monolith.
That extraction was correct as a direction and wrong as a destination: it was carried out
because a strangler migration was underway, not because each domain had produced a reason to
be a process. The reasons were never written down per service, so nothing ever failed the
test that was never applied.

Applying it now, retroactively. A domain earns its own process when it has at least one of:

- **independent lifecycle** — it ships on its own cadence
- **independent scaling profile** — its load curve does not track the rest
- **independent failure mode** — it can be down while the rest is up, usefully
- **a different language or runtime**
- **it is read independently** — someone reasons about it without the rest in view

Scored honestly:

- **insights-service passes on four counts.** It calls an external API it does not control;
  it is slow and bursty where the rest is fast and steady; it is expensive per call in a way
  no other domain is; and it is where a Python runtime would live if one ever arrives. It is
  the first thing in this platform that will genuinely need its own box.
- **plan-service passes on the aggregate.** It owns plans, checklist items and the outbox, and
  it is the other participant in every event flow. Folding it into the gateway would put the
  producer and the consumer of the core chain in one process, which is the one thing that must
  not happen (see Decision 4).
- **auth-service and calendar-service pass on none of them.** They ship with the gateway, scale
  with the gateway, are useless while the gateway is down, are Go like everything else, and are
  read alongside the gateway because that is the only thing that calls them. What they cost is
  concrete: a migration lineage, a config surface, a deploy, a health check, a log destination,
  and a network hop on the hot path of every authenticated request.
- **example-service and orchestrator-service are scaffolds**, verified rather than assumed.
  Both expose a single `Ping` RPC returning `"pong: <msg>"`. Both declare an AMQP topology with
  no bindings behind a no-op consumer. orchestrator-service has no database *by design* and its
  own spec describes it as "Scaffold / reference. No product features yet." There is no domain
  in either one to fold.

The counter-argument, stated so it is not lost: three services is still a distributed system,
and the cheapest architecture would be one process. That is true and rejected on the insights
boundary alone — an LLM call that takes twenty seconds and costs real money does not belong in
the same process as a request that must return in fifty milliseconds. Once insights is out,
plan-service must also be out, because the whole effectively-once guarantee rests on producer
and consumer being unable to share a transaction. Three is the floor, not a compromise.

Recorded without adversarial review — locked collaboratively during the consolidation scoping
session and not run through `challenge-me`.

## Decision

**Fireplace runs three services: api-gateway, plan-service, insights-service.**

1. **auth-service and calendar-service fold back into api-gateway.** Their domains become
   packages inside the gateway, not processes beside it.
2. **example-service and orchestrator-service are deleted, not folded.** Folding a `Ping` RPC
   moves nothing. Their directories, `go.work` entries, compose entries and databases go with
   them. The scaffold pattern they demonstrated is already captured in
   `GO_MCP_PROJECT_TEMPLATE.md` and `setup-go-mcp-project.sh`, which is where a new service
   should be copied from.
3. **insights-service is the AI service in fact, if not yet in name.** Every model call belongs
   here. It is not renamed as part of this decision; renaming is cosmetic and can happen when
   the discipline is actually true (see Consequences).
4. **plan-service and insights-service remain separate processes with separate database
   connections, and must stay that way.** This is not a preference. ADR-0008's effectively-once
   guarantee is built on the producer and the consumer being physically unable to share a
   transaction. Consolidation does not weaken it, because neither of these two moved.
5. **JWT verification stays shared, in `common/auth`.** Auth folding into the gateway does not
   make identity a gateway-private concern; plan-service and insights-service still verify
   tokens themselves under ADR-0001. Do not duplicate the verifier into the gateway.
6. **The `Makefile`'s `SERVICES` list becomes `api-gateway insights-service plan-service`.**
   It currently reads `api-gateway auth-service calendar-service example-service plan-service`
   — omitting insights-service, which means the platform's other event-flow participant has
   been outside `build-all`, `clean-builds` and `check-builds` this whole time. Fix this
   *first*, before anything moves, so the build audit covers insights-service during the
   migration rather than after it.

## Consequences

**Good**

- Four fewer processes to deploy, configure, health-check and read logs from, in exchange for
  no lost capability.
- Auth stops being a network hop on the hot path of every authenticated request.
- The two services that remain are the two that were always going to remain, and the reason is
  now written down per service instead of assumed.
- The boundary test exists as a reusable artifact. The next "should this be a service?"
  argument gets scored rather than re-litigated.

**Bad / accepted**

- **The reversal is real and worth naming.** Five services were extracted, three are folded
  back. That is the substance of this decision rather than an embarrassment to bury: the
  extraction taught us which boundaries were load-bearing, and that information was not
  available before doing it. The mistake would be leaving the other four out to avoid admitting
  the first pass overshot.
- **api-gateway still calls OpenAI directly, and this ADR does not fix it.**
  `services/api-gateway/internal/ai` imports `go-openai` and is wired at
  `services/api-gateway/config/routes.go:75` via `ai.NewNotesGenerator()` for the notes domain.
  So Decision 3 states a direction, not today's truth. Moving notes generation into
  insights-service is deliberately **out of scope here** and is expected to happen later; until
  it does, "nothing else calls an LLM directly" is aspirational and should not be quoted as if
  it were enforced.
- **insights-service has no outbox yet.** It has an inbox (`processed_events`) and no outbox
  table. The second hop's transactional outbox is in active development. Nothing in this
  consolidation touches it, and nothing in this consolidation should be allowed to constrain
  its design.
- **The gateway gets larger.** Absorbing auth and calendar makes it the biggest thing in the
  repo by some margin, and internal package boundaries now carry weight that process boundaries
  used to carry for free. Nothing enforces them but review.
- **Re-extracting auth or calendar later means redoing this work.** Accepted knowingly: the
  extraction is mechanical if the packages stay clean, and paying for a boundary in advance of
  needing it is what produced this ADR.
