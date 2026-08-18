# FS-0006: Guided plan creation and asynchronous initial items

> Status: draft · SPECIFICATION.md: services/plan-service/SPECIFICATION.md "## Plans" + "## Checklist Items", services/insights-service/SPECIFICATION.md "## Features", client/SPECIFICATION.md "## Plans" → this FS · Related ADRs: docs/adr/0002-contract-planes-code-first-openapi.md (§API surface obligation), docs/adr/0007-user-edits-win.md, docs/adr/0008-choreography-not-saga-for-generation-chains.md

**Cross-service FS.** plan-service, insights-service, and client all carry a thin line pointing here.

**Scoping session:** 2026-08-18/19. Decisions were locked collaboratively but **not run through
`challenge-me`** — annotated `(not challenged)` where that matters.

---

## Scoping notes (raw)

### Why this exists

Today plan creation is a vending machine: the user authors a plan themselves, then clicks
"get suggestion" and receives three suggestions. Nothing accumulates, and the assist arrives
after the plan is already shaped.

The goal is a creation flow that produces a structured plan, with checklist items derived
asynchronously from persisted insights — while remaining fully skippable for users who know
exactly what they want.

---

### Locked decisions (arrived pre-locked; constraints, not suggestions)

#### Field change

- `focus` (one line) is renamed to **`plan_draft`**: freeform, any length, a sentence or a
  page. It is the single input steering all downstream AI for that plan.
- **The reason:** `focus` was doing two jobs — identifying the plan and steering the LLM — and
  being one line meant it failed the second. Users wrote the same text into `focus` and
  `description`.
- `description` survives with a narrowed job: a short blurb for the plans-list block and for
  collaborators on shared plans. Prefilled from the draft, editable.
- A **compact derivation** of the draft feeds high-frequency calls; the **full draft** is
  reserved for low-frequency deep calls. The draft is unbounded, so putting it in every prompt
  is both expensive and context-diluting.

#### Flow

- **No preview.** Items are never shown before creation. This was scoped and then cut — it
  forced a synchronous LLM call and split the design into two divergent paths.
- Two entry paths converge on a populated `plan_draft`: **guided** (user gives a short line,
  the system drafts) and **custom / skip** (user writes the draft themselves).
- Plan creation commits **the plan row only**. No items.
- Item generation is asynchronous and never blocks creation.

#### Event chain

`plan.created` → insights-service consumes → generates and persists insights → publishes →
plan-service consumes → materializes initial checklist items.

Effectively-once throughout: transactional outbox on producers, inbox with
`processed_events (event_id, consumer)` on consumers, Redis claim as a best-effort layer, DLQ
for unexpected errors.

Items are stored in **plan-service**. Plan is the aggregate root and items are entities inside
it — the boundary was evaluated and rejected. insights-service owns the *generation*, not the
data.

#### Project invariant — user edits win

Once a user modifies AI-generated content, it is theirs. Regeneration may add or propose,
never overwrite or delete a row the user has touched. This surfaced in three features and is
recorded project-wide → **ADR-0007**, not re-decided here.

#### Deferred, already agreed

- SSE for completion notification — polling for v1, SSE is a deliberate follow-up refactor.
- Adjustment chips on generated assumptions.
- RAG / embedding-based retrieval.
- Adaptive profile from completion history.

---

### Decisions taken during this session

#### The guided step is a direct gRPC call from plan-service to insights (not the gateway)

The guided path is **not** "return a draft, user approves, then create." Generation happens
**inside** creation, in one client round trip:

```
POST /api/plans (guided)
        |
        v
 plan-service CreatePlan
        |
        +--> insights.GenerateDraft(short_line, plan_type)   <-- OUTSIDE the tx
        |        returns the full generated plan text
        |
        +--> BEGIN TX
        |      insert plans (plan_draft = generated text)
        |      insert outbox (plan.created, carries plan_draft)
        |    COMMIT
        v
   plan returned; draft visible and editable in the plan-edit surface
```

Custom mode is the same path minus the first arrow — `plan_draft` arrives in the request body.

**Rejected: gateway orchestration** (gateway calls insights, then calls plan-service).
Guided-vs-custom is domain logic about how a plan comes into being, and the platform's
strangler direction moves domain logic *out* of the gateway. plan-service owning the invariant
"a plan is created with a populated `plan_draft`" keeps it with the aggregate root.

**Withdrawn objection — "that's a dependency cycle."** It is not, and this was argued and
dropped during the session. `insights → plan-service` already exists
(`PlanGateway.GetPlanContext`), so plan-service dialling insights makes the graph
bidirectional — which is unremarkable. None of the three things "circular dependency" usually
means applies: (1) *compile-time* — both services import `common/api/proto/*`, neither imports
the other, so there is no Go import cycle; (2) *startup ordering* — clients resolve via Consul
and dial lazily, neither blocks the other at boot; (3) *synchronous recursion* — the guided
call is **planless** (no `plan_id` exists yet), so insights has nothing to call back for.
Recorded so it is not re-raised.

What *does* survive, and is a property of the feature rather than the wiring: if insights is
down, guided creation fails. Gateway orchestration would not have made it more available.

**Cost of the client, measured:** plan-service has **zero outbound gRPC clients today**. It
registers with Consul, but `SetupServices(workerCtx, db, amqpChannel, _ discovery.Registry, wg)`
**discards the registry**. Adding one is: name that parameter, and clone
`insights-service/internal/insights/plan_client.go` (cached `ClientConn`, status-code → domain
sentinel translation). The pattern already exists twice — insights and calendar.

#### The generation call sits OUTSIDE the transaction

Stated explicitly because the obvious-looking refactor is to fold it into `ExecTx` alongside
the outbox write, which would hold a Postgres transaction open for the length of a 10–15s LLM
call.

#### Guided generation failure is fail-closed

If `GenerateDraft` fails, **no plan row commits**. Rejected alternative: commit with the user's
short line as `plan_draft`. That silently hands the user a custom plan when they asked for a
guided one — and worse, `plan.created` then fires carrying a one-line draft, so items get
generated from the degraded input too. One failure becomes two. Failing closed costs nothing:
no state exists yet, the form still holds their input client-side, retry is one button.

#### `project` vs "development"

The original brief said "learning and development plans." **There is no `development` value.**
`internal/plan/model.go` has `PlanTypeProject = "project"` / `PlanTypeLearning = "learning"`;
the client dropdown renders Project / Learning. The only occurrence of the word anywhere is a
stale comment on migration `000003` (`plan_type TEXT NOT NULL, -- learning or development.`)
that the code never matched. Confirmed as a **typo, not a rename** — "development" means the
existing `project`. The stale comment is worth deleting; chasing the value is not.

---

### Event flow and failure handling (owner handoff, verbatim)

> Scope of this briefing: the event flow and failure handling for generating a plan's initial
> checklist items on the guided creation path. Nothing else. Not the creation UI, not the draft
> field changes, not the insights schema.
>
> **The flow**
>
> Two-hop choreography. Nothing blocks; each hop is fire-and-forget.
>
> 1. Plan is created. Plan row commits with the draft only — no items. Outbox row written in
>    the same transaction. `plan.created` published.
> 2. insights-service consumes it, generates insights via LLM, persists them, and publishes a
>    fact: insights were generated for plan X.
> 3. plan-service consumes that and materializes the initial checklist items in a single
>    transaction.
>
> **Why choreography and not orchestration**
>
> At two hops, choreography is well within its competence — the readability problem starts
> around four or five steps. If this chain grows past that, Temporal is already in the stack
> and is the answer.
>
> This is deliberately **not a saga**. A saga's defining feature is compensation, and there is
> nothing here to compensate. Item creation fails → don't delete the insights (still valid and
> reusable), don't delete the plan (user made it, it's usable). Failure means retry or tell the
> user, never unwind. It's a pipeline with retries.
>
> **Why generation is async and reads are sync**
>
> Generation takes 10-15 seconds, so it can't sit behind a synchronous call. Reading
> already-persisted insights is a different job and should be a synchronous gRPC call. Both,
> for different purposes.
>
> **Event naming matters here**
>
> The second event is a **fact, not a command**. insights-service does not know plan-service is
> listening and does not care. Name it after what happened (`insights.generated`), not after
> what should happen next (`create_items_for_plan`). Same wire traffic, completely different
> coupling — and it's what lets an embedding worker or notification service subscribe later
> without insights-service changing at all.
>
> Propagate `correlation_id` (whole chain) and `causation_id` (immediate parent) through both
> hops. Costs nothing at two steps and is the only reason a longer chain is debuggable later.
>
> **Delivery guarantees**
>
> Effectively-once on both hops:
>
> - Transactional outbox on producers
> - Inbox with `processed_events (event_id, consumer)` on consumers
> - Redis claim as a best-effort layer, PostgreSQL as authority
> - Sentinel routing: already-processed → ack, transient → retry, unexpected → DLQ
>
> Item materialization must be **one transaction** covering all items plus the
> `processed_events` write. A partial insert followed by redelivery produces a checklist with
> everything twice.
>
> **Failure handling — the part that needs care**
>
> Two **disjoint** outcomes. A message ends in exactly one; they never both fire.
>
> *Path 1 — transient errors, retries exhausted*
>
> Timeouts, rate limits, temporary unavailability. Retried with backoff and jitter. When
> attempts are exhausted:
>
> - insights-service publishes a **failure event** rather than DLQ'ing
> - plan-service consumes it and moves the plan to a **failed** state
> - User gets a **retry action** on the page they're already on
>
> Rationale: nothing is broken, so the fastest recovery is the user regenerating on the spot. A
> DLQ entry waiting on an operator is strictly slower for a problem that will probably just work
> on the next attempt.
>
> **No outbox needed for this failure publish.** The outbox exists to make a local write and a
> publish atomic. insights-service persists nothing about the exhausted attempt — the count
> lives in message metadata, the plan status is owned by plan-service — so there is no dual
> write and nothing to be atomic with. **Publish before ack**: if the publish fails, don't ack,
> let RabbitMQ redeliver. The unacked message is the durable record.
>
> **Condition:** the moment insights-service starts writing anything locally about failed
> attempts, the dual-write problem returns and so does the outbox. Keep it stateless about
> failures.
>
> *Path 2 — bugs, parse errors, unexpected exceptions*
>
> Nothing retryable. Goes to DLQ. Copy tells the user something went wrong and a fix is needed.
> No user retry on this path for now.
>
> **Known accepted cost, state it rather than hide it:** with no DLQ replay path, a plan that
> hits a bug gets zero items **permanently** — not "until it's fixed." If the copy promises
> eventual recovery, either soften the copy or accept that shipping the fix leaves those plans
> stranded. Fine as an edge case; shouldn't be invisible.
>
> **Idempotency on the failure path**
>
> The failure event does **not** need `processed_events`. Use a guarded status transition:
>
> ```sql
> UPDATE plans SET status = 'failed'
> WHERE id = ? AND status = 'generating'
> ```
>
> Second delivery touches zero rows. A conditional update is cleaner than a dedup protocol for a
> single state transition — and it means at-least-once delivery of the failure event is safe by
> construction. Same principle for any user-facing message driven off status: render it only
> while the plan is in that state, so a duplicate delivery can't produce a duplicate effect.
>
> **Two things to get right**
>
> *1. Delivery-count ceiling on the terminal branch, separate from the generation retry count.*
>
> If the attempt counter travels in message metadata, a native requeue preserves it unchanged.
> So a message that reaches max attempts, fails to publish the failure event, and gets requeued
> comes back **already at max**. It re-evaluates as exhausted, tries to publish again, fails
> again, requeues again — a hot loop with no backoff, spinning as fast as the broker delivers.
>
> Safe as long as not-acking is reserved strictly for "the failure publish itself failed," which
> is rare and usually self-resolving. Needs a **separate ceiling on that branch** so a
> persistently broken publish path can't spin forever.
>
> *2. A sweeper as backstop.*
>
> Explicit failure events cover well-behaved failures. A panic or OOM runs **no error handler at
> all** — no failure publish, no DLQ decision, just redelivery. Something has to notice a plan
> stuck in `generating` with no resolution.
>
> Threshold must be **derived from the retry schedule, not picked**. With TTL-tiered retry at
> 1m/5m/15m, a 5-minute sweeper fires while retries are legitimately in flight and reports
> failure on work that's still running. The threshold sits beyond the exhausted-retry window by
> construction.
>
> Fast path for known failures, sweeper for the ones that kill the process before it can report.
>
> **Retry budget**
>
> An LLM call sits behind the user's retry button. Needs a **cooldown and an attempt cap** —
> otherwise twenty clicks is twenty paid generations.
>
> **Schema implication**
>
> `status = failed` alone isn't enough. The **failure class** needs to travel with it, since it
> drives both the copy and whether a retry action is offered.
>
> **Already settled, don't reopen**
>
> - RabbitMQ durability (durable queues, persistent delivery, publisher confirms) — handled
> - Items live in plan-service. Plan is the aggregate root, items are entities within it. The
>   service boundary was evaluated and rejected — insights-service owns generation, not the data
> - Preview was scoped and cut
> - Temporal is reserved for the review workflow; the first real saga is calendar write-back

---

### Plan state model

The handoff's guarded transition **requires a stored `status` column** — a derived-from-
timestamps state cannot be the target of a conditional `UPDATE`, so idempotency-by-construction
only works with real stored status. An earlier two-timestamp proposal from this session is
superseded by that and recorded only so it is not re-proposed.

Timestamps do not disappear, because the sweeper needs to know *when* `generating` started.

| Column              | Purpose                                                                  |
| ------------------- | ------------------------------------------------------------------------ |
| `plan_draft`        | renamed from `focus`; TEXT NOT NULL, unbounded                           |
| `status`            | the guarded-transition target; drives copy and whether retry is offered  |
| `status_changed_at` | sweeper input — "stuck in `generating` past the exhausted-retry window"  |
| `failure_class`     | the handoff's requirement: `failed` alone isn't enough                   |
| `retry_count`       | the retry budget's attempt cap                                           |
| `last_retry_at`     | the retry budget's cooldown                                              |

**plan-service can observe when items materialize; it cannot observe that insights failed**
except by being told (failure event) or by elapsed time (sweeper). That asymmetry is why both
mechanisms exist.

---

### Missing infrastructure this FS must create (verified against the code, 2026-08-19)

Not objections — inputs the work orders have to carry.

1. **No `insights.events` exchange.** `common/constants/events.go` declares auth / plan /
   example / orchestrator only. Note the file's own convention is `{resource}.{action}`
   **singular** (`user.created`, `plan.created`), so the constant should be
   **`insight.generated`**, not `insights.generated`, and the failure event named to match
   (e.g. `insight.generation_failed`).
2. **plan-service has no `processed_events` table.** It has an outbox (migration `000020`) but
   no inbox. Needs a new migration, plus a queue — `plan-service.insight.generated` under the
   `{consumer-service}.{routing-key}` convention — bound to the new exchange.
3. **`RetryExchange` and `DlxEventsExchange` are declared but never used** — no binding, no TTL
   tiers, no DLQ anywhere in the repo. The 1m/5m/15m tiering is the mechanism that makes the
   attempt counter work at all: a bare `Nack(requeue)` redelivers the *original* message and
   increments nothing; `x-death` is populated only by DLX round-tripping. **TTL-tiered DLX is
   the retry transport, not native requeue.**
4. **`correlation_id` / `causation_id` have nowhere to go.** `PlanCreatedEvent` carries
   `id, user_id, name, focus, plan_type, created_at`, and `commonbroker.Message` exposes only
   `MessageId, ContentType, Body, DeliveryMode` — no headers field. Either extend the broker
   `Message` struct (headers, transport-level — preferred, since these are envelope concerns
   not domain fields) or add the fields to every event proto.
5. **`PlanCreatedEvent.focus` is a published wire field.** Renaming it to `plan_draft` is a
   contract change. insights is its only consumer and is mid-development, so a straight rename
   is safe now — but the field number must be **reused deliberately**, not bumped.

### Pre-existing defects in the chain (coordination points, not work here)

Owned by the parallel insights-service effort. Recorded because this FS's chain does not
function until they are fixed, and because the insights spec currently overstates them.

- `insights-service/internal/insights/amqp_consumer.go` **declares and consumes
  `plan-service.events`** but **binds `insights-service.plan.created`**. The bound queue is
  never declared; the consumed queue is plan-service's own. Wired as-is, insights would compete
  with plan-service's `user.deleted` consumer for the same messages.
- `insights.NewService(...)` does not pass `inboxService`, so `Service.inboxService` is a nil
  interface and `s.inboxService.CreateTx` panics on the first event.
- `NewConsumer` and `SetupAMQPInfrastructure` are **never called** from `cmd/server/main.go` or
  `config/services.go`. Nothing consumes `plan.created` today. (The insights spec was updated
  during this session to mark the trigger and exactly-once lines unchecked, which now matches.)

### Chore, out of band

`Makefile` line 5: `SERVICES = api-gateway auth-service calendar-service example-service
plan-service` — **insights-service and orchestrator-service are missing**, so both are excluded
from `build-all`, `clean-builds`, and `check-builds`. insights-service is a consumer in this
flow. Worth fixing before this lands; it is not a slice of this FS.

---

### Ownership split for `/spec-to-issues`

**Flagged as the owner's — do not assign these:**

- The publish legs on both hops: outbox drain, publish-worker, `insight.generated` publishing.
- All insights-service work: the planless `GenerateDraft` RPC, consumer wiring, inbox
  injection, queue-binding fix, populating `generated_insights.content`, the failure-event
  publisher, DLQ.

**In this FS's lane:** the `plan_draft` field change end-to-end (migration, proto, gRPC,
generated HTTP contract, client), guided/custom creation paths, plan-service's insights client,
plan-service's inbox + materialization transaction, plan status model, the sweeper, the retry
budget, and the client waiting/failed/retry surface.

Issues and docs are safe to write while insights work proceeds in parallel — no file conflict
is possible between the two lanes.

---

### Open questions — explicitly NOT settled

- **Is a plan with zero items a valid end state?** If insights legitimately returns nothing,
  does the consumer mark the plan complete-but-empty, or is empty treated as failure?
- **Does a stalled/failed plan offer regeneration, and does regenerating re-emit
  `plan.created`?** insights dedupes on `event_id`, so a retry needs a *new* event id and
  therefore probably a distinct routing key (e.g. `plan.items_requested`) rather than a second
  `plan.created`.
- **Provenance on generated items** — what marks an item as AI-generated, and what the UI does
  with it. Interacts with ADR-0007: the invariant tells us a touched row is protected, but not
  how "touched" is recorded.
- **Bulk rejection.** A generation the user dislikes means deleting items one at a time, and
  deleting is more effort than accepting. What mitigates that?
- **User edits `plan_draft` after items exist** — ADR-0007 constrains this; it does not fully
  answer it.
- **Learning vs project plans** — do they branch structurally (sequenced curriculum with
  prerequisites vs. tasks with dependencies and a definition of done), or swap prompts on one
  code path?
- **Polling** — interval, backoff, give-up threshold, behaviour on tab blur. And: polling must
  remain the degradation path when SSE lands — does that shape the API now?
- **Plans-list entry** — `plan_draft` is too large for an inline block. What does a list entry
  show, and what does clicking in reveal? (`description` is the intended blurb, but the current
  list renders `description` *and* `focus`.)
- **Validation on `plan_draft`** — is there a minimum before generation is worth attempting?
- **Migration backfill** — existing `focus` values are already valid short drafts; confirm
  nothing else needs backfilling.
- **Plan deleted while generation is in flight.**

---

### Out of scope

- SSE implementation
- Adjustment chips, RAG / embedding retrieval, adaptive profile
- Calendar-derived plan seeding
- Changes to the insights schema (that is the insights persistence scope)
- Model selection per call, beyond noting which calls are high- vs low-frequency

### API surface — deferred to `/write-a-spec`

Recorded here as an input, not as the section itself. This FS changes the `POST /api/plans` request body (`focus` →
`plan_draft`, plus the guided-mode discriminator) and adds at least one plan-status read path
for polling. `/api/plans` is already serialized under FS-0004, so **no slice ⓪ is required** —
the change flows through the typed Huma handler and `openapi.yaml` regenerates. The legacy
`/api/insights/*` gin routes are untouched by this FS and remain unserialized.
