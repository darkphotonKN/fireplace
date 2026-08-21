# FS-0006: Guided plan creation and asynchronous initial items

> Status: work-order · SPECIFICATION.md: services/plan-service/SPECIFICATION.md "## Plans" + "## Checklist Items", services/insights-service/SPECIFICATION.md "## Features", client/SPECIFICATION.md "## Plans" → this FS · Related ADRs: docs/adr/0002-contract-planes-code-first-openapi.md, docs/adr/0004-error-representation-rfc9457.md, docs/adr/0005-request-validation-and-contract-design.md, docs/adr/0007-user-edits-win.md, docs/adr/0008-choreography-not-saga-for-generation-chains.md

**Cross-service.** plan-service, insights-service, api-gateway (contract only) and client each carry a thin line pointing here.

## Summary

Plan creation today is a vending machine: the user writes the plan themselves, then clicks
"get suggestion" and receives three suggestions. Nothing accumulates, and the assist arrives
after the plan is already shaped.

This replaces it with two creation paths that both produce a **Plan Draft** — the freeform
steering text that replaces `focus` — and then materializes a plan's first checklist items
asynchronously from generated insights. **Guided** turns a short seed line into a whole plan
draft; **Custom** lets the user write the draft themselves, at any length. Either way the plan
row commits alone, items arrive afterwards over a two-hop event chain, and nothing about the
generation blocks the user from using the plan.

Vocabulary is `services/plan-service/CONTEXT.md` and `services/insights-service/CONTEXT.md`.

---

## Requirements

### The Plan Draft field

- **R1.** `plans.focus` is renamed to `plans.plan_draft` (TEXT NOT NULL). It is freeform and
  unbounded in the database — a sentence or a page.
- **R2.** `plan_draft` is the single input from which all downstream AI for that plan derives.
  A **compact derivation** of it feeds high-frequency calls; the **full draft** is reserved for
  low-frequency deep calls. The draft is unbounded, so putting it in every prompt is both
  expensive and context-diluting.
- **R3.** `description` keeps its column and narrows its job: a short blurb for the plans list
  and for collaborators on shared plans. It is prefilled from the draft at creation and is
  independently editable thereafter.
- **R4.** The HTTP surface publishes the field as `planDraft`. `focus` is removed from
  `PlanResp`, `CreatePlanReq` and `UpdatePlanReq`, not aliased — there is one first-party
  consumer and it ships with the gateway.
- **R5.** `PlanCreatedEvent.focus` is renamed to `plan_draft`, **reusing the existing field
  number**. insights-service is its only consumer and is mid-development.
- **R6.** The migration is a rename; existing values are valid short drafts and need no
  transformation. No other backfill of `plan_draft` is required.

### Creation paths

- **R7.** Both paths are **one operation on one resource** — `POST /api/plans` — discriminated
  by a required `mode` field (`guided` | `custom`). Guided and custom differ in *how the draft is
  obtained*, not in what is created, so they are not separate resources.
- **R7a.** **Custom** (`mode=custom`) takes `planDraft` in the request body. No generation step.
- **R8.** **Guided** (`mode=guided`) takes a short `seed` line, and plan-service calls
  insights-service to generate the whole draft. The generated text is what is persisted as
  `plan_draft`; it is not returned for approval first.
- **R8a.** Per-field constraints (`mode` enum, `seed` 10–500, `planDraft` 1–20000) are **shape**
  and are rejected at the boundary with 422. The conditional rule — the field matching `mode` is
  present and the other absent — is **domain**, owned by plan-service, returning 400
  `VALIDATION_FAILED` through the existing mapping (ADR-0005). No new error code is required for
  it, and none is added.
- **R9.** Both paths complete in **one client round trip** and commit **only the plan row** —
  never items.
- **R10.** The generated draft is visible and editable in the plan-edit surface, like any other
  draft. Guided and custom plans are indistinguishable once created.
- **R11.** plan-service dials insights-service over gRPC directly. `SetupServices` stops
  discarding its `discovery.Registry` parameter, and `internal/plan/insights_client.go` follows
  the existing client pattern: one cached `*grpc.ClientConn`, never dialled per RPC, with
  gRPC status codes translated into local domain sentinels.
- **R12.** insights-service gains a **planless** generation RPC — `GenerateDraft(seed,
  plan_type) → draft`. Planless because no plan exists yet, so unlike every other method on
  that service it cannot begin by fetching Plan Context.
- **R13.** The `GenerateDraft` call happens **outside** the database transaction. Folding it
  into `ExecTx` alongside the outbox write would hold a Postgres transaction open for the
  length of a 10–15 second LLM call.
- **R14.** Guided generation failure is **fail-closed**: no plan row commits, and the client
  keeps the user's seed so retry is one action.

### The generation chain

- **R15.** `plan.items_requested` is the **sole** trigger for item generation, emitted both at
  creation (guided and custom) and on user-initiated retry. insights-service binds only this
  routing key.
- **R16.** `plan.created` remains a general-purpose lifecycle fact with no insights binding. It
  is never re-emitted for a retry.
- **R17.** Each emission of `plan.items_requested` is a new outbox row and therefore a new
  `event_id`, so retry needs no special-casing in the dedup path.
- **R18.** insights-service publishes `insight.generated` on success and
  `insight.generation_failed` when transient retries are exhausted. Both are **facts** — named
  for what happened, never for what should happen next (ADR-0008).
- **R19.** `insight.generated` carries the generated items themselves. plan-service materializes
  them without calling back, so materialization never depends on insights being reachable.
- **R19a.** The payload is `{plan_id, user_id, items[]}`, where an item is the **creatable
  subset** of a checklist item: `description`, `scope`, `type`, and an optional
  `parent_index`. It is deliberately the same shape as a checklist item minus everything the
  database owns — no `id`, `plan_id`, `status`, `done`, `created_at` or `updated_at`.
- **R19b.** Nesting is expressed by **`parent_index`**, an index into the same array, because no
  row IDs exist before the materialization transaction opens. Two-tier nesting still applies: a
  referenced parent must itself have no `parent_index`.
- **R19c.** **Array order is `sequence`.** Carrying an explicit sequence field would let the
  payload disagree with itself; order cannot.
- **R19d.** The payload carries **no dates**. AI auto-scheduling of `start_date` / `due_date` is
  an unshipped plan-service capability and is out of scope here — materialized items arrive
  unscheduled.
- **R20.** A new `insights.events` topic exchange is added to `common/constants/events.go`,
  with routing keys `insight.generated` and `insight.generation_failed` — **singular resource**,
  matching the file's existing `{resource}.{action}` convention.
- **R21.** plan-service gains a `processed_events (event_id, consumer)` inbox table and a queue
  `plan-service.insight.generated` bound to the new exchange.
- **R22.** Materialization writes **all items and the `processed_events` row in one
  transaction**. A partial insert followed by redelivery would produce a checklist with
  everything twice.
- **R23.** Effectively-once on both hops: transactional outbox on producers, inbox on consumers,
  PostgreSQL as the authority. Sentinel routing: already-processed → ack, transient → retry,
  unexpected → DLQ.
- **R23a.** The **Redis claim is retained on both hops**, scoped to **consumer dedup only**. It
  spares the database a transaction for a redelivered event while Redis is up. It is never the
  authority — `(event_id, consumer)` decides — so its whole contract is "cheap negative answer,
  no positive guarantee." plan-service gains a Redis dependency it has not had
  (`plan-service/CLAUDE.md` still says *"No Redis"* and needs updating).
- **R23b.** **The claim must fail open.** A Redis error is *not* a duplicate. Only an
  unambiguous "the key already exists" may short-circuit; an unreachable or erroring Redis falls
  through to the database, which is the authority. Implementations must distinguish
  `(acquired=false, err=nil)` — a real duplicate — from `(acquired=false, err!=nil)` — Redis
  could not answer. Collapsing the two turns a cache outage into silent event loss.
- **R23c.** **Relay drain contention is not a Redis concern.** Two relay instances draining the
  same outbox are separated by `FOR UPDATE SKIP LOCKED` in `GetUnpublished` — in-database, exact
  rather than best-effort, and free because the relay is already taking that row lock. A
  resource-keyed distributed lock (`lock:<resource>:<id>`) is the correct tool when the contended
  thing is *not* a row the worker already selects for update; that is not the case here.
  Recorded because the two locks are easy to conflate and solve different problems.
- **R24.** `correlation_id` (whole chain) and `causation_id` (immediate parent) propagate
  through both hops as **envelope** concerns, not as two more columns on every event proto.
  `correlation_id` rides AMQP's **native correlation-id property**; `causation_id` rides a
  header, since AMQP has no native equivalent.
  *Corrected during I-0022:* an earlier draft of this requirement said `commonbroker.Message`
  "exposes only `MessageId, ContentType, Body, DeliveryMode` — no headers field" and had to grow
  one. That was read off the publish-worker's call site, which sets only those four, not off the
  struct. `Message` already carried both `CorrelationId` and `Headers`, and `AmqpPublisher`
  already passed them through. The real gap was **key discipline** — nothing stopped one service
  writing `causation_id` and another reading `causationId`, and a missing header reads as an
  empty string rather than an error, so the break would have been silent.
- **R25.** Retry transport is **TTL-tiered DLX** with **exponential tiers 10s / 20s / 40s /
  80s / 160s** — five retry queues plus the DLQ — not native requeue. A bare
  `Nack(requeue)` redelivers the original message and increments nothing; `x-death` is populated
  only by dead-letter round-tripping. `RetryExchange` and `DlxEventsExchange` already exist as
  constants and are bound for the first time here.
- **R25a.** The tiers are exponential from the ticker interval (10s, R69) to a 160s ceiling,
  exhausting after **310s of waiting**. Including ~15s of generation per attempt across six
  attempts, a legitimately-retrying plan reaches terminal failure in **≈7 minutes** worst case.
  Tiers of 1m/5m/15m were considered and rejected: a 21-minute exhaustion window is correct for
  background work and wrong for something a human is watching — it would have made R27's "the
  fastest recovery is the user regenerating on the spot" false by twenty minutes.
- **R25b.** Exhaustion ends **the generation job, not the plan**. The plan row was committed at
  t=0 (R9) and stays — usable, with its draft, and with items the user may add by hand. This is
  graceful degradation of the *feature*: the capability is reduced, the plan is not.

### Failure handling

- **R26.** Two **disjoint** terminal outcomes. A message ends in exactly one; they never both
  fire.
- **R27.** *Path 1 — transient errors, retries exhausted.* insights-service publishes
  `insight.generation_failed` rather than DLQ'ing; plan-service moves the plan to `failed`; the
  user gets a retry action on the page they are already on. Nothing is broken, so the fastest
  recovery is the user regenerating on the spot — a DLQ entry waiting on an operator is strictly
  slower for a problem that will probably work next attempt.
- **R28.** The failure publish uses **no outbox**. insights-service persists nothing about the
  exhausted attempt — the count lives in message metadata and the plan status is owned by
  plan-service — so there is no dual write to be atomic with. **Publish before ack**: if the
  publish fails, do not ack, and let RabbitMQ redeliver. The unacked message is the durable
  record. *This holds only while insights-service stays stateless about failures; the moment it
  writes anything locally about a failed attempt, the dual-write problem returns and so does the
  outbox.*
- **R29.** *Path 2 — bugs, parse errors, unexpected exceptions.* Not retryable; goes to DLQ. The
  user is told something went wrong and needs a fix. No user retry on this path.
- **R30.** A **separate delivery-count ceiling** guards the terminal branch, distinct from the
  generation retry count. Not-acking is reserved strictly for "the failure publish itself
  failed"; without its own ceiling, a message already at max attempts that fails to publish and
  requeues comes back still at max, re-evaluates as exhausted, and spins as fast as the broker
  delivers.
- **R31.** A **sweeper** backstops plans stuck in `generating`. A panic or OOM runs no error
  handler at all — no failure publish, no DLQ decision, just redelivery — so something has to
  notice. Its threshold is **derived, never picked by hand**, from the exhausted-retry window
  (≈7 min, R25a) plus one ticker interval per hop (R69), plus margin — **10 minutes**. The signal
  (R60) is droppable, so the worst case a healthy plan can legitimately take is bounded by the
  ticker, not by the signal; a threshold derived from the retry schedule alone would fail plans
  that are merely queued. In practice the sweeper only ever catches crashes, because a
  well-behaved exhaustion publishes its failure at ≈7 minutes.
- **R32.** User-initiated retry carries a **cooldown and an attempt cap**. An LLM call sits
  behind that button; twenty clicks must not be twenty paid generations.

### Plan state

- **R33.** `plans` gains `status`, `status_changed_at`, `failure_class`, `retry_count` and
  `last_retry_at`.
- **R34.** Plan status is `generating` | `ready` | `failed`, with exactly these transitions:
  creation → `generating`; `generating → ready` (materialization, including zero items);
  `generating → failed` (failure event or sweeper); `failed → generating` and
  `ready → generating` (user retry / regeneration). Every other transition is rejected.
- **R35.** Plan status is deliberately **not monotonic** — retry re-enters `generating` on
  purpose. It is enforced by guarded conditional updates plus a database trigger, not by a
  monotonicity rule.
- **R36.** Status transitions driven by events use the guarded form
  `UPDATE plans SET status = ... WHERE id = ? AND status = '<expected>'`. A second delivery
  touches zero rows, so at-least-once delivery is safe by construction and the failure event
  needs no `processed_events` entry of its own.
- **R37.** `failure_class` travels with `failed`, because `failed` alone cannot drive the two
  things that depend on it: what the user is told, and whether a retry action is offered.
- **R38.** Any user-facing message driven off status renders **only while the plan is in that
  status**, so a duplicate delivery cannot produce a duplicate effect.
- **R39.** Existing rows backfill to `status='ready'`, `status_changed_at=NOW()`,
  `failure_class NULL`, `retry_count=0`, `last_retry_at NULL`. Nothing is in flight at migration
  time.

### Item provenance — a monotonic FSM

- **R40.** `checklist_items` gains `status`: `authored` | `generated` | `touched`.
  `authored` = created by the user. `generated` = materialized and never modified.
  `touched` = generated, then modified by the user.
- **R41.** The **only** legal transition is `generated → touched`. `authored` and `touched` are
  absorbing. Status can never move backwards.
- **R42.** Monotonicity is enforced at **two layers**, matching the existing two-tier-nesting
  idiom: a service-layer transition function that is the sole writer of the column, and a
  `BEFORE UPDATE` trigger that rejects every illegal transition with SQLSTATE `23514`.
  **An UPDATE that leaves `status` unchanged is not a transition and must pass.** The naive
  guard (`IF OLD.status = 'touched' THEN RAISE`) would reject every update to a touched row —
  including the nightly bulk `DailyReset` CTE, which sets `done=false` and never touches
  `status`.
  A convention that says "don't clear this" is not enough — ADR-0007 claims the protection is
  structural, and this is what makes that claim true against a future bug or a later feature.
- **R43.** Regeneration may add rows and may replace rows with `status='generated'`. It may
  **never** touch `authored` or `touched` rows (ADR-0007).
- **R43a.** **Touching a child promotes its parent.** When an item moves `generated → touched`,
  its parent — if `generated` — moves to `touched` in the same transaction. Two-tier nesting caps
  the depth at one hop, so no recursion is possible.
- **R43b.** *Why this rule exists:* `checklist_items.parent_id` is
  `REFERENCES checklist_items(id) ON DELETE CASCADE` (migration 000019). Without R43a, deleting a
  `generated` parent would cascade into a `touched` child and destroy the user's work — through
  the FK, silently, via the very actions (R43, R46) this FS calls ADR-0007-safe. Promoting the
  parent removes it from the deletable set, so the cascade can never reach protected data. The
  predicate stays `status='generated'` everywhere; no descendant subquery is needed anywhere.
- **R43c.** *Accepted consequence:* a parent's own text becomes un-regenerable once any of its
  children is edited, even though nobody edited the parent. That is the conservative direction,
  and ADR-0007 is a conservative rule.
- **R44.** "Touched" means any user-initiated mutation: description, `done`, dates, re-parent,
  scope, type, archive — every path through `UpdateItem`, `UpdateItemDates` and `ArchiveItem`.
  It explicitly excludes **daily reset** and **materialization**, both of which are the system
  acting rather than the user.
- **R45.** Existing rows backfill to `authored`, so pre-existing data is user-owned and no
  regeneration can ever remove it.
- **R46.** A single **"clear generated items"** action deletes exactly the `status='generated'`
  set. It is the same predicate regeneration uses, so there is one rule rather than two, and —
  given R43a — it is ADR-0007-safe by construction including through the FK cascade: any parent
  holding edited children is already `touched` and is therefore not in the set.
- **R47.** The client marks `generated` items subtly and **drops the marker once the item is
  `touched`** — the marker's only job is signalling "safe to regenerate away," which stops being
  true the moment the item is protected.

### Plan types

- **R48.** Learning and project plans share **one code path** and swap prompts. The data model
  is already identical — `scope`, `type`, `parent_id`, `sequence`, dates — and
  prerequisites-versus-dependencies is a content difference the LLM expresses through nesting and
  sequence, both of which two-tier nesting already carries.
- **R49.** The stale comment on migration `000003` (`-- learning or development.`) is deleted.
  There has never been a `development` value; the live values are `project` and `learning`.

### Client surface

- **R50.** The creation form offers guided and custom paths. Custom is fully skippable — a user
  who knows what they want types their draft and creates.
- **R51.** After creation the user lands on the plan with the draft in place and a visible
  "items are being generated" state. The plan is usable immediately; generation never gates it.
- **R52.** Completion is detected by **polling**: every 2s for the first 30s, then every 5s to
  2 minutes, then every 10s. Polling stops when status leaves `generating`, or at the sweeper
  threshold (10 min, R31) — the client give-up and the server sweeper agree rather than
  disagreeing visibly. The happy path resolves in ~20s (signal-driven pickup plus one
  generation), so the later tiers exist for the retrying case, not the normal one.
- **R53.** Polling pauses on tab blur and resumes with an immediate poll on focus.
- **R54.** The status read is a **dedicated lightweight endpoint**, not a full plan GET. SSE
  later pushes the same payload, so the client swaps transport without touching its state
  handling. Polling remains the degradation path when SSE lands.
- **R55.** A plans-list entry shows name, plan type, `description` and a status indicator. It
  never shows `plan_draft`, which is too large for an inline block and lives in plan-edit. The
  list's current `focus` render is removed — that duplication is what the field split exists to
  fix.
- **R56.** Editing `plan_draft` after items exist triggers **nothing automatically**. The draft
  saves, and the plan offers "draft changed — regenerate items?" as an opt-in. Auto-regeneration
  would burn a paid call on every save and delete work the user never asked to lose.

### Validation

- **R57.** `planDraft`: non-empty, at most 20,000 characters. Shape-validated at the boundary
  → 422.
- **R58.** `seed`: 10 to 500 characters, **rejected before any LLM call is made**, so a seed of
  "asdf" never costs a generation.
- **R59.** Domain rules stay with the owning service and return 400 with a specific code; the
  gateway never restates a downstream rule (ADR-0005).

### Outbox relay signalling

> **Shared code.** This changes `common/worker`, which has no spec of its own — a behaviour
> change there belongs to the FS of the feature driving it (root CLAUDE.md). plan-service is the
> only constructor today (`config/services.go:40`); insights-service becomes the second when it
> gains an outbox for `insight.generated`.

- **R60.** The relay gains an **in-process signal channel** alongside its existing ticker.
  Producers notify it after committing outbox rows, so pickup is immediate on the happy path
  instead of waiting out the interval.
- **R61.** **The ticker is still the correctness guarantee.** It runs unconditionally — not as a
  fallback armed by a failed signal. The signal is purely a latency optimization and must always
  be safely droppable.
- **R62.** *Rationale, recorded so it is not "simplified" later:* the signal is silently absent in
  cases undetectable at the send site — process death between commit and notify, a full channel,
  an outbox row written by another process or by hand, or a new code path that forgets to notify.
  None of these produces an error. The relay therefore cannot know a signal was owed and never
  arrived, and can never skip polling on that basis. If the signal ever becomes the delivery
  mechanism rather than a hint, a dropped signal strands a row — reintroducing the dual-write
  problem the outbox exists to eliminate.
- **R63.** The channel is `chan struct{}` with **buffer 1**. *Not unbuffered:* an unbuffered send
  succeeds only if a receiver is blocked at that exact instant, so a notify arriving while the
  relay is mid-drain would be dropped — the case where the signal matters most. *Not larger:* the
  signal carries no information; ten notifies mean what one means — go look. Buffer 1 is a
  one-slot latch meaning "work arrived while you weren't looking," and one bit is all the state
  there is.
- **R64.** Notify is **non-blocking** — `select` with a `default` that drops. It never blocks the
  request path and never returns an error. A full channel means a wake-up is already pending, so
  dropping is correct behaviour, not a failure.
- **R65.** The relay loop selects over **signal, ticker, and context cancellation**. All wake
  paths call the same drain, and the drain does not know or care why it woke.
- **R66.** Notify fires **after commit**, never inside the transaction. Inside, the relay could
  wake and query before the commit is visible, find nothing, and sleep again — leaving the row for
  the ticker and losing the optimization exactly when it was requested.
- **R67.** Notify call sites, exhaustively:
  (a) after every commit that wrote outbox rows — today plan creation in plan-service and the
  insights consumer after writing its own outbox row, plus any future outbox write;
  (b) on relay **startup**, before entering the loop, since rows may have been written while the
  service was down;
  (c) optionally after a drain returns a **full batch**, signalling a backlog larger than the
  query limit.
  Nothing else fires it. In particular the failure-event publish (**R28**) writes no outbox row
  and therefore has **no** notify call site — adding one for symmetry would reintroduce the dual
  write R28 exists to avoid.
- **R68.** The outbox write and the notify must be **hard to separate** — one repository helper
  doing both, rather than two things a caller has to remember to pair. A new write path that
  forgets to notify degrades silently to ticker latency, with no error anywhere.
- **R69.** The polling interval is **10 seconds**, replacing `time.Minute * 2`
  (`plan-service/config/services.go:40`). This is a *loosening* relative to the ~2s the relay
  would need if the ticker were the only pickup path — the signal takes the ticker off the
  critical path, so it only has to catch rare misses. Two minutes was defensible only while
  nothing user-facing waited on the outbox.
- **R70.** **Observability:** track rows found by ticker-triggered drains versus signal-triggered
  ones. Near-zero on the ticker path is healthy; a rising count means notifies are being missed
  somewhere. Nothing else surfaces that failure.

---

## User Stories

1. As a user with a rough idea, I want to type one line and get a whole plan drafted for me, so
   I don't stare at an empty form.
2. As a user who knows exactly what I want, I want to skip generation entirely and write my own
   draft, so the assist never slows me down.
3. As a user, I want my plan created immediately rather than waiting on an AI call for items, so
   I can start working while the system catches up.
4. As a user, I want the generated draft to be editable afterwards, so a near-miss is fixable
   rather than a restart.
5. As a user, I want to write a draft as long as I like, so a complex plan isn't squeezed into
   one line.
6. As a user, I want a short description on the plans list, so I can recognise a plan without
   reading its whole draft.
7. As a user, I want the description prefilled from my draft, so I don't write the same thing
   twice.
8. As a user, I want to see that items are on their way, so an empty checklist doesn't read as a
   failure.
9. As a user, I want the page to update on its own when items arrive, so I don't refresh to
   check.
10. As a user, I want polling to stop when I switch tabs, so a background tab isn't hammering the
    server on my behalf.
11. As a user whose generation failed transiently, I want a retry button on the page I'm already
    on, so recovery is one click and not a support request.
12. As a user, I want to be told when a failure needs a fix rather than a retry, so I'm not
    clicking a button that cannot work.
13. As a user, I want a cooldown on retry, so I can't accidentally spend a fortune by
    double-clicking.
14. As a user, I want a plan that generated zero items to be usable, so an unhelpful generation
    isn't a dead plan.
15. As a user, I want to see which items came from AI, so I know what I'm reviewing.
16. As a user, I want that marker to disappear once I've edited an item, so the badge means
    something.
17. As a user, I want to reject a whole generation in one action, so declining isn't more work
    than accepting.
18. As a user, I want my edits to survive regeneration, so improving a generated item isn't
    punished by losing it.
19. As a user, I want my hand-written items to be untouchable by any regeneration, so the feature
    can never eat my own work.
20. As a user, I want changing my draft to be my decision to act on, not an automatic
    regeneration, so editing a typo doesn't rewrite my checklist.
21. As a user, I want a learning plan to be sequenced like a curriculum and a project plan to
    read like tasks, so the assist fits what I'm actually doing.
22. As a user, I want to delete a plan mid-generation without anything breaking, so changing my
    mind is safe.
23. As a returning user, I want my existing plans untouched by this change, so nothing I already
    built is disturbed.
24. As a collaborator on a shared plan, I want a readable description, so I understand the plan
    without the owner's raw draft.
25. As an operator, I want a plan stuck in `generating` to be detected without a user reporting
    it, so a crashed consumer surfaces on its own.
26. As an operator, I want transient failures to reach the user and bugs to reach the DLQ, so my
    queue isn't a support channel.
27. As an operator, I want a redelivered event to be a no-op, so at-least-once delivery never
    duplicates a checklist.
28. As an operator, I want `correlation_id` on every hop, so I can trace one plan's chain across
    two services.
29. As an operator, I want a broken failure-publish path to stop rather than spin, so one bad
    branch can't saturate the broker.
30. As a developer, I want insights-service to publish facts and know nothing about who consumes
    them, so a notification or embedding worker can subscribe later without touching it.
31. As a developer, I want one generation trigger for both creation and retry, so retry isn't a
    second code path.
32. As a developer, I want provenance enforced by the database, so a future feature can't quietly
    break ADR-0007.

---

## Acceptance Criteria

**Field change**

- [ ] `plans.focus` is renamed to `plan_draft`; every existing value survives unchanged.
- [ ] `focus` appears nowhere in `PlanResp`, `CreatePlanReq` or `UpdatePlanReq`; `planDraft` does.
- [ ] `PlanCreatedEvent`'s draft field is renamed with its field number reused.
- [ ] The plans list renders `description`, not the draft.
- [ ] Migration `000003`'s `-- learning or development.` comment is gone.

**Creation**

- [ ] `mode=custom` with a valid `planDraft` returns 201 and commits exactly one `plans` row and
      zero `checklist_items`.
- [ ] `mode=guided` with a valid `seed` returns 201, and the persisted `plan_draft` is the
      generated text, not the seed.
- [ ] `mode=guided` carrying `planDraft`, or `mode=custom` carrying `seed`, is rejected 400
      `VALIDATION_FAILED` with no plan row and no LLM call.
- [ ] When `GenerateDraft` fails, creation returns 503 `GENERATION_FAILED` and **no `plans` row
      exists**.
- [ ] `SetupServices` no longer discards its registry parameter.
- [ ] The insights client dials once and reuses the connection across calls.
- [ ] Wrapping `GenerateDraft` inside `ExecTx` is caught by review; the call is demonstrably
      outside the transaction.

**Chain**

- [ ] Creating a plan writes exactly one `plan.items_requested` outbox row in the same
      transaction as the plan row.
- [ ] Retrying writes another `plan.items_requested` row with a different `event_id`.
- [ ] `plan.created` is never emitted twice for one plan.
- [ ] Delivering the same `insight.generated` twice creates the items once.
- [ ] With **Redis stopped**, events are still processed exactly once and none are dropped —
      the claim fails open and the inbox decides.
- [ ] A Redis error and a genuine duplicate produce different outcomes, not the same one.
- [ ] A materialization that fails partway leaves zero items and no `processed_events` row.
- [ ] `correlation_id` and `causation_id` survive both hops.
- [ ] Retries traverse the DLX tiers; `x-death` count increases across attempts.

**Failure**

- [ ] Exhausted transient retries move the plan to `failed` with a `failure_class`, and the UI
      offers retry.
- [ ] A malformed message reaches the DLQ and the plan's copy does not promise recovery.
- [ ] Delivering `insight.generation_failed` twice leaves one `failed` plan and updates zero rows
      the second time.
- [ ] A failure-publish that keeps failing stops at its own ceiling instead of spinning.
- [ ] A plan whose consumer is killed mid-generation is picked up by the sweeper.
- [ ] The sweeper threshold is computed from the retry schedule, not a literal.
- [ ] Retry beyond the attempt cap, or inside the cooldown, is refused without an LLM call.

**Provenance**

- [ ] Materialized items are `generated`; user-created items are `authored`.
- [ ] Editing a `generated` item moves it to `touched`; editing an `authored` item leaves it
      `authored`.
- [ ] A direct `UPDATE` attempting `touched → generated` is rejected by the trigger with
      SQLSTATE `23514`.
- [ ] Daily reset does not move any item to `touched`.
- [ ] Regeneration replaces `generated` rows and leaves `touched` and `authored` rows byte-identical.
- [ ] Editing a child of a `generated` parent promotes the parent to `touched` in the same
      transaction.
- [ ] Clearing generated items on a plan whose generated parent has a touched child deletes
      neither — the FK cascade never reaches protected data.
- [ ] `DailyReset` succeeds against `touched` items; the status trigger permits the no-op.
- [ ] "Clear generated items" removes every `generated` row and nothing else.
- [ ] Every pre-existing item backfills to `authored`.

**Outbox relay**

- [ ] Notify against a full channel does not block.
- [ ] A drain with no pending rows is a no-op.
- [ ] **An outbox row written with no notify at all is still published on the next tick** — the
      test that proves the signal is an optimization and not a dependency.
- [ ] The ticker fires regardless of signal activity.
- [ ] Notify happens after commit: a relay woken by the signal always finds the row.
- [ ] Shutdown drains in-flight work before returning.

**Client**

- [ ] A plan in `generating` shows a waiting state and polls on the 2s/5s/10s schedule.
- [ ] Polling stops on leaving `generating` and at the sweeper threshold.
- [ ] Blurring the tab pauses polling; focusing resumes with an immediate poll.
- [ ] A plan that finishes with zero items reads as ready-and-empty, never as an error.
- [ ] Editing the draft on a plan with items triggers no generation and offers the opt-in.
- [ ] `seed` under 10 or over 500 characters is rejected with 422 and no LLM call is made.

---

## Edge States

| Scenario | Behaviour |
| --- | --- |
| Insights returns nothing usable | Plan reaches `ready` with zero items — a valid end state, not a failure. Showing an error for a non-error trains people to ignore errors. |
| Plan deleted while generation is in flight | The materialization consumer checks the plan exists inside its transaction; if it is gone, **ack and drop, log at info**. Terminal, not an error, not DLQ — the user deleted it deliberately. |
| Plan deleted, insights still generating | insights-service's write has no FK to `plans` (separate database), so it succeeds and leaves an orphan `generated_insights` row. Accepted: it is a cache, and the retention scan handles it. |
| `insight.generated` redelivered | Inbox insert conflicts, the transaction rolls back, the consumer acks. Items exist once. |
| `insight.generation_failed` redelivered | Guarded update matches zero rows; status unchanged, no second effect. |
| Failure event arrives after the sweeper already failed the plan | Guarded update matches zero rows — the plan is already `failed`. First writer wins, and both agree on the outcome. |
| Failure publish itself fails repeatedly | Terminal-branch ceiling stops the loop. Without it the message returns already at max attempts and spins with no backoff. |
| Consumer panics mid-generation | No handler runs, so no failure event and no DLQ decision. The sweeper is the only thing that notices. |
| Bug-class failure with no DLQ replay | **Accepted cost, stated not hidden:** that plan gets zero items *permanently*, not "until it's fixed." Copy must not promise eventual recovery. |
| User retries during cooldown | Refused before any LLM call, with the remaining cooldown surfaced. |
| User retries past the attempt cap | Refused; the plan stays `failed` and the retry affordance is withdrawn rather than left dead. |
| Regeneration on a plan where every item is touched | Adds new items only; nothing is replaced. A legitimate no-delete outcome. |
| User edits an item while materialization is committing | Materialization writes in one transaction; the edit either precedes it (item is `authored`, untouched by materialization) or follows it (`generated → touched`). No interleaved state is observable. |
| Insights unreachable at guided creation | 503; no plan row; the client keeps the seed. |
| Insights unreachable at retry | Retry accepted (it is asynchronous), and the chain fails normally into `failed` — the retry path does not need insights to be reachable at the moment of the click. |
| Draft edited while items are generating | Allowed. The in-flight generation used the draft as it was at request time; the opt-in regenerate offer appears once the plan leaves `generating`. |
| Existing plans at migration time | `status='ready'`, all items `authored`. No chain runs for them and no regeneration can touch their items. |
| Empty `{}` PATCH on a plan | Valid no-op — every `UpdatePlanReq` field carries `omitempty` (contract-patterns §5). |

---

## API surface

Plane 1 (client ↔ api-gateway). `/api/plans` is already serialized under FS-0004, so **no
slice ⓪ is required** — changes flow through the typed Huma handlers and `openapi.yaml`
regenerates. Errors are RFC 9457 `problem+json` with a `code` from `common/errcode` (ADR-0004).
Error rows below follow the **existing** `apierr.StatusFor` / `CodeFor` mapping, which already
maps gRPC `Unavailable → 503 SERVICE_UNAVAILABLE` correctly (contract-patterns §9) — this FS
does not restate or generalise it.

| Op | Method + Path | Query/Params | Request body | Response | Errors |
|----|---------------|--------------|--------------|----------|--------|
| `createPlan` *(changed)* | POST `/api/plans` | — | `name` (req), `planType` (req), `mode` (req, `guided`\|`custom`), `planDraft` (opt, 1–20000), `seed` (opt, 10–500), `description` (opt) — `focus` **removed** | 201 `PlanResp` | 400 `VALIDATION_FAILED`, 401, 422, 503 `SERVICE_UNAVAILABLE` \| `GENERATION_FAILED` |
| `getPlanGeneration` *(new)* | GET `/api/plans/{id}/generation` | — | — | 200 `{status, failureClass?, itemCount, requestedAt}` | 401, 403, 404, 503 |
| `retryPlanGeneration` *(new)* | POST `/api/plans/{id}/generation/retry` | — | — | 202 (no body) | 401, 403, 404, 409, 429 `GENERATION_COOLDOWN`, 503 |
| `updatePlan` *(changed)* | PATCH `/api/plans/{id}` | — | `focus` → `planDraft`, all optional, all `omitempty` | 200 `PlanResp` | as today |
| `listPlans` / `getPlan` *(changed)* | unchanged paths | unchanged | — | `PlanResp` gains `planDraft`, `status`, `failureClass?`; loses `focus` | as today |
| `clearGeneratedItems` *(new)* | DELETE `/api/plans/{id}/checklists/generated` | — | — | 204 | 401, 403, 404, 503 |
| `listChecklists` *(changed)* | unchanged path | unchanged | — | `ChecklistResp` gains `status` (`authored`\|`generated`\|`touched`) | as today |

**New error codes** (added because a real failure needs distinguishing, not speculatively —
`docs/agents/contract.md`):

| Code | Status | Meaning |
|---|---|---|
| `GENERATION_FAILED` | **503** | insights was reachable but could not generate; no plan was created. Client keeps the seed and offers retry. Shares 503 with `SERVICE_UNAVAILABLE` (insights *unreachable*) — both retryable, different copy. Status is the coarse signal, the code is the precise one (ADR-0004). |
| `GENERATION_COOLDOWN` | 429 | Retry requested inside the cooldown or past the attempt cap. |
| `NOT_IMPLEMENTED` | 501 | **Interim, not permanent.** `mode=guided` while the `GenerateDraft` RPC has not landed. Already in `common/errcode` and wired through `apierr.StatusFor`/`CodeFor`; its own comment calls it "a DELIVERY statement, not a failure — the operation is published in the contract with its success shape declared, but its data path has not landed yet." `mode=custom` is unaffected. Falling through to 500 or 503 here would tell the client something broke when nothing did. |

**502 was considered and rejected.** `apierr.httpForCode` has no 502 branch, so it would widen
the seam for a single case. **500 was also rejected**, and for a stronger reason: per
`errcode.go`'s own rationale, 500 means "your request broke us — do not retry," which is false
here. A failed generation is exactly the case where retry is correct.

**No code is added for the `mode` discriminator.** A body whose `mode` and payload disagree is a
first-party client bug, not a user-facing state, and `VALIDATION_FAILED` already carries it.
`docs/agents/contract.md`: domain codes are added when a real failure needs distinguishing,
never speculatively.

`retryPlanGeneration` returns **409** when the plan is already `generating` — the request is
well-formed but the state forbids it, and it must not silently enqueue a second chain.

Plane 2 (gRPC) additions: `insights.InsightsService.GenerateDraft(seed, plan_type) → draft`, and
`PlanCreatedEvent.focus → plan_draft` with the field number reused. Plane 2 governance
(`buf lint` / `buf breaking`) is **not yet wired** in this repo — noted so the gap is explicit.

---

## Out of Scope

- **SSE** — polling for v1; SSE is a deliberate follow-up refactor, and R54 is what keeps the
  API from having to change when it lands.
- Adjustment chips on generated assumptions.
- RAG / embedding-based retrieval.
- Adaptive profile from completion history.
- Calendar-derived plan seeding.
- Changes to the insights schema — that is the insights persistence scope.
  **Where a generated item set is stored inside insights-service is owner-lane and runs in
  parallel** — deliberately not covered here, and not a gap in this plan. Noted only so nobody
  re-derives it: `generated_insights.insight_type` is currently
  `CHECK (insight_type IN ('suggestion','daily','video'))`, so an initial item set has no type yet
  and today's code writes those rows as `'suggestion'`. It has **no bearing on this FS's lane** —
  plan-service consumes the `insight.generated` *event* and never reads that table. The event is
  the entire contract between the two lanes; the storage behind it is insights' business.
- Model selection per call, beyond noting which calls are high- versus low-frequency (R2).
- DLQ **replay** tooling. The DLQ receives messages; nothing reads it back. R29's accepted cost
  depends on this staying out of scope.
- Bulk multi-select item deletion. R46 covers the generation case; general selection is its own
  feature.

---

## Ownership split for `/spec-to-issues`

**Owner-lane work is filed as issues, marked `[HUMAN]`, and excluded from `/develop`** — not
left unfiled. Filing keeps the dependency graph complete: `blocked_by` cannot reference an issue
that does not exist, and untracked work is invisible work. Each carries a do-not-develop banner in
its body, which survives a tracker migration in a way a custom label or frontmatter field would
not (`docs/agents/README.md` carries over only title / body / labels / blocked_by).

**Flagged as the owner's — file, mark `[HUMAN]`, never assign to an agent:**

- The publish legs on both hops: outbox drain, publish-worker, and `insight.generated` /
  `insight.generation_failed` publishing.
- All insights-service work: `GenerateDraft`, consumer wiring, inbox injection, the queue-binding
  fix, populating `generated_insights.content`, the failure-event publisher, DLQ.
- **R60–R70, the relay signal channel** — being done by hand as a Go concurrency exercise.

**In this FS's lane:** the `plan_draft` change end-to-end (migration, proto, gRPC, HTTP contract,
client), both creation paths, plan-service's insights client, plan-service's inbox and
materialization transaction, both FSMs, the sweeper, the retry budget, and the client
waiting/failed/retry surface.

Issues and docs are safe to write while insights work proceeds in parallel — no file conflict is
possible between the two lanes.

---

## Design notes — alternatives rejected, and why

Kept because each of these was argued and would otherwise be re-argued.

**Preview of items before creation.** Scoped and cut. It forced a synchronous LLM call and split
the design into two divergent paths. R46's "clear generated items" is the mitigation that
replaced it; a staging tray was reconsidered during this session and rejected as the same preview
arriving by a different door.

**Gateway orchestration of guided creation** (gateway calls insights, then calls plan-service).
Rejected: guided-versus-custom is domain logic about how a plan comes into being, and the
platform's strangler direction moves domain logic *out* of the gateway. plan-service owning "a
plan is created with a populated `plan_draft`" keeps the invariant with the aggregate root.

**"plan-service dialling insights is a dependency cycle."** Raised during this session and
**withdrawn as unfounded** — recorded so it is not raised again. `insights → plan-service`
already exists via `PlanGateway.GetPlanContext`, making the graph bidirectional, which is
unremarkable. None of the three things "circular dependency" means applies: *compile-time* — both
import `common/api/proto/*`, neither imports the other, so there is no Go import cycle;
*startup ordering* — clients resolve through Consul and dial lazily, so neither blocks the other
at boot; *synchronous recursion* — the guided call is planless, so insights has nothing to call
back for. What survives is a property of the feature, not the wiring: if insights is down,
guided creation fails, and gateway orchestration would not have made it more available.

**Two creation endpoints** (`POST /api/plans` plus `POST /api/plans/guided`). Rejected: guided
and custom create the *same resource* and differ only in how the draft is obtained. Two endpoints
would have made the conditional-required rule enforceable as shape (422 rather than 400), which
was the only argument for it — but it buys that by splitting one user-facing concept across two
operations and putting a verb in a path. The discriminated body costs one small domain rule that
the existing mapping already handles, and adds no error code.

**Degrading instead of failing closed on guided generation.** Rejected: committing the plan with
the user's seed as `plan_draft` silently hands them a custom plan when they asked for a guided
one, and `plan.items_requested` then fires carrying a one-line draft, so items get generated from
the degraded input too. One failure becomes two.

**insights binding both `plan.created` and `plan.items_requested`.** Rejected: it makes insights
know that "a plan was created" *implies* "items are wanted" — plan lifecycle semantics leaking
into the generation service. **Re-emitting `plan.created` on retry** was also rejected: it would
give every future `plan.created` subscriber a false creation signal. The single-trigger design
was nearly free to adopt because insights' binding is broken today and being rewritten regardless.

**Two nullable timestamps for provenance** (`generated_at` / `user_edited_at`). Rejected in favour
of the FSM: timestamps carry the same information, but nothing *prevents* `user_edited_at` being
cleared by a bug or a future "reset" feature. ADR-0007 claims the protection is structural, and
only a one-way state machine with a database trigger makes that claim true. **Deriving "touched"
by diffing against a stored original** was also rejected — it contradicts "recorded, never
inferred", false-positives on whitespace, and cannot see a date set or a re-parent.

**Structural branching by plan type.** Rejected for now: two materializers and two schemas
maintained before either need is proven. Revisit when one plan type genuinely needs a column the
other does not — a real prerequisite DAG for learning plans would be that trigger.

**Naming: "generation" covers two different things.** Noted and deliberately not renamed. The
word names both the *synchronous draft call* (`GenerateDraft`, `GENERATION_FAILED`, guided mode
only, ~15s, inside the create request) and the *asynchronous item chain*
(`/api/plans/{id}/generation`, both modes, minutes, after the response). Renaming the resource to
`item-generation` and the code to `DRAFT_GENERATION_FAILED` was proposed and passed over. Anyone
reading a ticket should know the URL refers only to the second.

**A derived plan state from timestamps** (an earlier proposal in this session). Superseded: a
derived state cannot be the target of a conditional `UPDATE`, so idempotency-by-construction
(R36) only works with stored status.

## Pre-existing defects this chain depends on

Owned by the parallel insights-service effort, recorded because the chain does not function until
they are fixed.

- `insights-service/internal/insights/amqp_consumer.go` declares and consumes
  `plan-service.events` but binds `insights-service.plan.created`. The bound queue is never
  declared; the consumed queue is plan-service's own. Wired as-is, insights would compete with
  plan-service's `user.deleted` consumer.
- `insights.NewService(...)` does not pass `inboxService`, so `Service.inboxService` is a nil
  interface and `s.inboxService.CreateTx` panics on the first event.
- `NewConsumer` and `SetupAMQPInfrastructure` are never called from `cmd/server/main.go` or
  `config/services.go`.
- **The Redis claim fails closed** (`internal/insights/service.go`). `acquired, _ :=
  s.cache.SetNX(...)` discards the error, so an unreachable Redis yields `acquired=false`, which
  is reported as `ErrEventAlreadyProcessed`, which the consumer **acks and drops**. Every event
  is silently discarded for the duration of a Redis outage. The comment above it states the
  intent was best-effort; the code does the opposite. R23b is the fix.

## Chore, out of band

`Makefile` line 5 omits **insights-service** and **orchestrator-service** from `SERVICES`, so
both are excluded from `build-all`, `clean-builds` and `check-builds`. insights-service is a
consumer in this flow. Not a slice of this FS.
