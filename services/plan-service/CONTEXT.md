# CONTEXT.md — Plan (ubiquitous language)

> The shared vocabulary for this bounded context. One term, one meaning — used
> identically in conversation, code, and specs. Populated by the `domain-model` skill.

## Plan

**Plan Draft** — the plan's freeform steering text (`plan_draft`), any length from a sentence
to a page. It is the single input from which all downstream AI for that plan is derived.
Replaces **Focus**, which was one line and therefore failed at the steering half of its job.
Not a description and not a summary: it is an *instruction to the system*, authored by the user
or generated for them, and it stays editable for the plan's whole life.

**Focus** — *retired.* The former one-line goal field (`focus`). Superseded by **Plan Draft**;
existing values are valid short drafts. Do not introduce the word in new code or prose — it
survives only in migration history and in insights-service's prompt-context vocabulary.

**Description** — a short blurb shown on the plans list and to collaborators on a shared plan.
Prefilled from the Plan Draft, then independently editable. It describes the plan *to people*;
the Plan Draft steers the plan *for the system*. Confusing the two is what produced the original
duplication.

**Guided Creation** — the creation path where the user supplies a small amount of information
and the system generates the whole Plan Draft for them. Synchronous: generation happens inside
the create call, and the generated text is what gets saved.

**Custom Creation** — the creation path where the user writes the Plan Draft themselves, at
whatever length they choose. No generation step. Formerly called "skip".

**Materialization** — turning generated insights into real `checklist_items` rows owned by this
service. plan-service materializes; insights-service generates. The distinction is the service
boundary: insights owns the *generation*, never the *data*.

**Initial Items** — the checklist items materialized from the first generation pass triggered
by a plan's creation. Distinguished from items the user adds later and from items produced by a
subsequent regeneration.

**Generating** — the plan status meaning "a generation pass is in flight and items are expected
to arrive." Entered at creation, left by materialization or by failure. The status a sweeper
looks for when deciding whether a plan is stuck.

**Failure Class** — the *kind* of failure attached to a failed plan, carried alongside the
status because `failed` alone cannot drive the two things that depend on it: what the user is
told, and whether a retry action is offered at all.

**Seed** — the short line a user supplies on the Guided Creation path, from which the whole
Plan Draft is generated. Not a draft and never persisted as one: the seed is discarded once the
draft it produced is saved.

**Authored** — an item created directly by the user. Terminal: an authored item is never
generated, and editing it does not change what it is. Regeneration can never touch it.

**Generated** — an item created by Materialization and not yet modified by anyone. The only
status regeneration is allowed to replace, and the only one the client badges.

**Touched** — a generated row a user has modified, and which regeneration may therefore neither
overwrite nor delete (ADR-0007). Being touched is recorded, never inferred; what counts as
touching is stated per surface.

**Fact Event** — an event named for what happened, not for what should happen next
(`insight.generated`, never `create_items_for_plan`). Producers of fact events do not know who
consumes them (ADR-0008).

**Inbox** — this service's `processed_events (event_id, consumer)` ledger, written in the same
transaction as the business effect it guards, making redelivery a no-op. The DB constraint is
the authority; the Redis claim in front of it is an efficiency layer only.

**Item Status** — the one-way state machine carrying item provenance: `authored` and `generated`
on creation, with `generated → touched` as the single legal transition. Enforced by a service-layer
transition function and a `BEFORE UPDATE` trigger, so ADR-0007's protection is structural rather
than conventional.
