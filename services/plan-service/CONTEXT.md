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

**Description** — a short blurb shown on the plans list and to the **members** of a shared plan.
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

**Intention** — a nullable free-text field on any checklist item (`intention`, ≤ 280 chars,
trimmed, blank clears to null), set only through item update, never at create. Generic by design:
this service never says *milestone* (ADR-0014); the client reads it as a milestone's "why it
matters" (FS-043WP).

## Checklist items and the retired daily scope (FS-02JBZ)

**Scope** — an item's list, `longterm` or `daily` (`checklist_items.scope`). Since FS-02JBZ every
item is long-term in practice: `daily` survives only as an enum value in the schema and the
contract, so an old client's request is well-formed and its refusal is a domain decision. Listing
or reordering by `daily` is accepted and finds nothing.

**Daily Scope** — *retired (FS-02JBZ D12).* Formerly the short-term, repeatable list beside the
long-term one. Migration 000022 rewrites every `daily` row to `longterm`, keeping `done`, dates,
parents, intention and archived state, and places them after the existing long-term siblings.
Creating or updating an item to `daily` is refused with `DAILY_SCOPE_RETIRED`. Do not use
"daily item" for anything new; the daily surface is now insights-service's **day plan**, which
this service never sees.

**Daily Reset** — *retired (FS-02JBZ).* The bulk `DailyReset` RPC that un-completed done daily
items in plans with `daily_reset = true`, triggered nightly by the gateway. Removed with
`ToggleDailyReset` and the gateway's cron job.

**dailyReset** — *vestigial.* The `plans.daily_reset` column and field still exist, are still
accepted on create and update and still returned, and do nothing. Dropping them is later
cleanup; do not read meaning into the value.

**Done** — an item's `done`: the item is **finished**. Set and cleared only by item update; nothing
resets it on a schedule any more. One meaning only: insights-service's **day tick** ("done for
today") is a separate per-user fact stored there and never changes `done`, and `done` never
changes it. Recurrence is not a concept in this service; the only recurrence on the platform is
insights-service's revision ladder.

## Access

**Relation** — a user's relation to one plan, exactly one of **owner**, **member** or **none**
(`PlanRelation`, answered by `GetPlanAccess`). This service answers it and enforces nothing per
operation; the gateway decides what each relation may do (ADR-0015). Not a flag or a permission.

**Owner** — the user in `plans.user_id`. Exactly one per plan; never also a member (adding the
owner is `MEMBER_IS_OWNER`). Ownership wins over a stray share row.

**Member** — a user with a `plan_shares` row for the plan. Not "collaborator", "sharee",
"participant" or "editor" (rejected). The row's `role` (`edit` only in v1) does not change the
relation. **Share** names the *row* (`CreateShare`, `ListShares`); **member** names the *person*.

**None** — no relation, **including a plan that does not exist**. Never `NotFound` from this
service: a missing plan and a stranger's plan are the same answer.

**Shared plan** — from a given user's side, a plan they are a member of. Beware `ListSharedPlans`
and `/api/plans/shared`: despite the name they return owned ∪ shared, each tagged with its
relation.
