# insights-service — Specification

> Scope: AI suggestions (checklist, daily, video) + event-driven generation. Platform maps: ../../CLAUDE.md.

insights-service is the **owner** of the AI Insights + video-suggestion domain in the Fireplace platform. This document describes the **target design this service implements**. Where a piece is not yet ported (real OpenAI generator, video-finder, DLQ, read-path cache), it is marked **In Progress** — it is this service's feature that is partially implemented.

## Domain Terms

| Term | Meaning |
|------|---------|
| **Insight** | An AI-generated productivity artifact for a plan: a single suggestion, a set of daily focus items, or video recommendations. |
| **Content Generator** | The pluggable LLM seam (`ContentGenerator.Generate(prompt) → text`). Target implementation: OpenAI GPT-4o. Currently a stub. |
| **Discovery** | The subsystem that turns AI-generated search terms into concrete tutorial videos (YouTube crawler). Currently only the search-term generation half exists here. |
| **Focus** | A plan's high-level goal string, fetched from plan-service and fed into every prompt. |
| **Plan Context** | Focus + flattened checklist items fetched from plan-service; insights owns none of this data. |

## Features

### Implemented

- Single checklist suggestion — one actionable next task from focus + existing checklist.
- Daily suggestions — 3 daily focus items derived from longterm checklist items, de-duped across draws.
- Video suggestion prompt — generates search terms from focus + checklist.
- Pluggable `ContentGenerator` interface seam.
- Plan-context read path over gRPC to plan-service (focus + checklist), with ownership assertion.
- Event-driven generation trigger: consumes `plan.created`.
- Exactly-once processing: Redis best-effort lock + DB idempotency ledger, committed atomically.

### In Progress

- **Real OpenAI GPT-4o `ContentGenerator`** — currently `StubContentGenerator` returns `ErrGeneratorNotConfigured`, so generation RPCs surface `Internal`.
- **Video-finder (Discovery / YouTube crawler)** — `SuggestVideos` generates search terms then returns an empty list; the finder that resolves terms → videos is not yet ported.
- **DLQ for unexpected inbox errors** — unexpected errors are currently `Nack`-dropped + logged.
- **`generated_insights` read-path cache** — rows are written during event processing but never read.

### Future

- Serve cached insights from `generated_insights` instead of regenerating on every RPC.
- Fill `generated_insights.content` with the actual generated text (today written empty during event processing).

## gRPC Surface (`insights.InsightsService`, :7106)

| Method | Input | Output | Notes |
|--------|-------|--------|-------|
| `GenerateSuggestion` | `plan_id`, `user_id` | `suggestion` (string) | One verb-first actionable checklist item, 4–20 words. |
| `GenerateDailySuggestions` | `plan_id`, `user_id` | `suggestions` (repeated string) | 3 items derived from longterm tasks; each draw nudged away from the prior so they don't collide. |
| `SuggestVideos` | `plan_id`, `user_id` | `videos` (repeated `Video{title,url,source,type,description}`) | Generates 3 search terms; returns `[]` until video-finder is ported. |

All requests carry `user_id` for ownership assertion. Every method first calls plan-service `GetPlan` (which enforces ownership by taking `user_id`) to fetch plan context. Errors map through the shared gRPC error mapper; a malformed `plan_id`/`user_id` → `InvalidArgument`, stub generator → `Internal`.

## Event Processing (`plan.created`)

**Flow:** exchange `plan.events` (topic) → queue `insights-service.plan.created` bound on routing key `plan.created` → `Consumer.consumePlanEvents` → `service.Create`. The consumer decodes the protobuf `PlanCreatedEvent`; `msg.MessageId` is the `event_id`.

**Exactly-once design (two layers):**

1. **Redis `SetNX` (best-effort)** — `dedup:insights:<event_id>`, ~5s TTL while in-progress, upgraded to 24h after a successful commit. Deleted on failure so a crash doesn't wedge the event. Its errors are intentionally ignored; it is an efficiency lock only.
2. **DB unique `(event_id, consumer)`** — the **true authority**. The inbox insert is attempted first inside the tx; a conflict rolls the whole tx back and is reported as already-processed.

`inbox.CreateTx` (ledger write) and `repo.CreateTx` (`generated_insights` write) run in **one atomic tx** (`ExecTx`), so the dedup mark and the business effect commit together or not at all.

**ack/nack policy** (`errorHandler`):

| Condition | Action | Rationale |
|-----------|--------|-----------|
| `ErrEventAlreadyProcessed` (duplicate) | `Ack` + drop | Already handled; no retry. |
| `commonconstants.ErrTransient` | `Nack(requeue)` | Transient DB/infra fault; retry. |
| `ErrUnexpectedError` | `Nack(no-requeue)` + log | Poison/unexpected; **TODO: real DLQ**. |
| default / unknown | `Nack(no-requeue)` + log | Safety net. |
| Unmarshal / UUID-parse failure (pre-service) | `Nack(no-requeue)` | Poison message. |

insights-service **publishes no events**.

## Data Model

### `generated_insights` (migration 000001)

| Column | Type | Notes |
|--------|------|-------|
| `id` | UUID PK | `gen_random_uuid()` |
| `plan_id` | UUID NOT NULL | |
| `user_id` | UUID NOT NULL | |
| `insight_type` | TEXT NOT NULL | CHECK in (`suggestion`, `daily`, `video`) |
| `content` | TEXT NOT NULL | written empty today; will hold generated text |
| `created_at` | TIMESTAMPTZ | default `NOW()` |
| `updated_at` | TIMESTAMPTZ | default `NOW()`, maintained by trigger |

Index: `(plan_id, insight_type, created_at DESC)` for "latest insights for a plan of a given type." Written during event processing, **not yet read**.

### `processed_events` (migration 000002)

| Column | Type | Notes |
|--------|------|-------|
| `event_id` | UUID NOT NULL | part of PK |
| `consumer` | TEXT NOT NULL | default `'insights'`, part of PK |
| `created_at` | TIMESTAMPTZ | default `NOW()` |

PK `(event_id, consumer)` enforces dedup. Append-only ledger — never updated (no `updated_at`/trigger). Index on `created_at` for retention scans.

## Business Rules

- **Prompt: single suggestion** — one concrete task, verb-first, specific enough for one sitting, relevant to plan focus, **4–20 words**, no trailing punctuation or commentary.
- **Daily suggestions** — 3 items, biased toward breaking down **longterm** checklist items; each subsequent draw is instructed not to closely repeat the previous, so the set is de-duped.
- **Video suggestions** — prompt asks for exactly 3 relevant search terms from focus + checklist; terms feed the video-finder (once ported).
- **Ownership** — every request carries `user_id`; plan context is fetched via plan-service `GetPlan`, which enforces ownership, so generation is gated on the caller owning the plan. `AssertPlanOwnership` is also available for a standalone check.
- **Idempotency / exactly-once** — DB `(event_id, consumer)` is authority; Redis is best-effort; ledger + business write are atomic.

## Edge Cases

- **Duplicate redelivery** — Redis `SetNX` fails fast → `ErrEventAlreadyProcessed` → `Ack`+drop; even if Redis is unavailable, the DB unique constraint rolls the tx back and reports the same.
- **Transient vs poison** — transient DB errors → `Nack(requeue)`; malformed body / bad UUIDs / unexpected errors → `Nack(no-requeue)` (future DLQ).
- **Generator not configured** — with the stub, all generation RPCs return `Internal` (`ErrGeneratorNotConfigured`); event processing still writes ledger + `generated_insights` (with empty content).
- **plan-service unreachable** — plan-client dial/discovery failures surface as server faults (`Internal`); mapped status codes (`NotFound`, `PermissionDenied`, `InvalidArgument`, `Unauthenticated`) translate to the corresponding domain sentinels.

## Owned elsewhere

- **Plan context (focus + checklist)** → owned by **plan-service** (gRPC `GetPlan` / `ListItems` / `AssertPlanOwnership`, port 7103). insights-service owns none of this data.
- **HTTP `/api/insights` endpoints** → exposed by **api-gateway**, which authenticates the user and proxies to this service's gRPC surface via Consul.
