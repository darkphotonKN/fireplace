# CONTEXT.md — Insights (ubiquitous language)

> The shared vocabulary for this bounded context. One term, one meaning — used
> identically in conversation, code, and specs. Populated by the `domain-model` skill.

## Insights

**Insight** — an AI-generated productivity artifact for a plan: a single suggestion, a set of
daily focus items, or video recommendations. Generated and persisted here; the *data* derived
from it downstream is not this service's.

**Content Generator** — the pluggable LLM seam (`ContentGenerator.Generate(prompt) → text`).
One generator per system prompt, so a prompt change is a wiring change rather than an edit to
shared code.

**Discovery** — the subsystem that turns generated search terms into concrete tutorial videos.

**Plan Context** — plan draft plus a flattened view of the plan's checklist items, fetched from
plan-service over gRPC to build a prompt. This service owns none of it.

**Planless Generation** — a generation call that takes raw user input and a plan type, with no
`plan_id`, because no plan exists yet. Every other method here begins by fetching Plan Context;
a planless call cannot, and that is what makes it a distinct shape rather than another
parameterization. Backs Guided Creation in plan-service.

**Generation Outcome** — the result of an event-driven generation pass, published as a fact
regardless of which way it went: insights were generated, or generation exhausted its retries.
Both are outcomes; only one is success.

**Exhausted Retries** — a transient failure (timeout, rate limit, temporary unavailability)
that survived the full backoff schedule. Distinct from a **Poison Message**: exhausted retries
publish a failure fact so the user can retry from the page they are on, while a poison message
goes to the DLQ with no user-facing retry.

**Poison Message** — a message that cannot be processed by any number of retries: malformed
body, parse error, unexpected exception. DLQ'd, never requeued.

**Stateless About Failures** — the standing constraint that this service persists nothing local
about failed attempts. The attempt count lives in message metadata and the plan status is owned
by plan-service, so a failure publish has no local write to be atomic with and therefore needs
no outbox. The moment anything about a failure is written here, the dual-write problem returns
and so does the outbox.

**Inbox** — the `processed_events (event_id, consumer)` ledger, written in the same transaction
as the generated insight so the dedup mark and the business effect commit together or not at
all. The DB unique constraint is the authority; the Redis claim is best-effort.
