# FS-DF4ZB: Guided daily moves with retrieval

> Status: work-order · SPECIFICATION.md: insights-service `## Guided suggestions` (daily brief, step suggestions, item-tree suggestion, feedback and dismissal filtering), insights-service `## Retrieval` (playbook corpus, personal memory, tenant-scoped retrieval, eval); api-gateway `## Insights` "Guided-suggestion and feedback endpoints"; plan-service `### Checklist Items` "Self-describing item completion events"; client `## Suggestions` "Today's moves toward milestones", "Suggested steps for a milestone", "Suggested next milestone" → this FS · Brownfield siblings: client `## Suggestions` "Daily suggestions on the daily tab" → FS-none; api-gateway `## Insights` "Suggestion endpoints proxied to insights-service" → FS-none · Related ADRs: docs/adr/0018-pgvector-in-insights-database.md, docs/adr/0014-milestones-are-a-client-reading-of-items.md, docs/adr/0007-user-edits-win.md, docs/adr/0002-contract-planes-code-first-openapi.md, docs/adr/0004-error-representation-rfc9457.md, docs/adr/0005-request-validation-and-contract-design.md, docs/adr/0008-choreography-not-saga-for-generation-chains.md, docs/adr/0010-three-databases-one-postgres-instance.md · Related FS: FS-FSFEC (milestones, the grid composer), FS-0006 (provenance FSM, deliberately not depended on), FS-0004 (the serialized insights surface)

## Summary

The daily tab's suggestion block becomes **Today's moves**: one to three moves for today, each
tied to one of the plan's milestones with a one-line reason. A move either points at a step that
already exists, which the user can tick off right in the block, or proposes a new step, which the
user can add under that milestone. A milestone card can also be asked for 3–5 suggested steps,
added one at a time, and the grid composer can propose a whole next milestone with about three
steps. Nothing is written to the plan until the user accepts.

Underneath, insights-service moves from three sequential free-text calls to one strict
structured-output call per user action, caches the day's brief, skips work by rule when nothing
changed, and grounds the prompt in retrieved examples: a shared playbook of worked milestone
breakdowns, plus the user's own completed steps from their other plans. Retrieval is built
in-house on pgvector in the insights database (ADR-0018).

Terms used here (`daily brief`, `move`, `parent item`, `step suggestion`, `item-tree suggestion`,
`suggestion outcome`, `dismissal filter`, `plan-state hash`, `rule gate`, `generation`, `RAG
chunk`, `shared row`, `playbook`, `personal memory`, `embedder`, `retriever`, `weak retrieval`)
are defined in `services/insights-service/CONTEXT.md`. Client terms (`Today's moves`, `move`,
`suggested steps`, `suggested milestone`, `milestone`, `step`) are defined in
`client/CONTEXT.md`.

## Background: what ships today

- **Client.** `client/src/components/Todo.tsx` shows a suggestions block on the **daily** tab only,
  and only when the plan has long-term items (`fetchDailyInsights`). It renders three strings from
  `getDailyInsights` (`client/src/api/insights.ts`). Adding one creates a **daily-scope, top-level**
  item (`addInsightAsTodo`), which the nightly reset then treats as daily work. The show/hide
  toggle is remembered in `localStorage` (`showDailyInsights`).
- **Gateway.** `GET /api/insights/checklist-suggestion-daily?plan_id=` is already serialized (Huma,
  FS-0004 I-0019) in `services/api-gateway/internal/insights/typed_insights.go` and returns a bare
  `string[]`. No slice ⓪ is needed for this FS: the surface it touches is governed already.
- **insights-service.** `GenerateDailySuggestions` (`internal/insights/service.go`) makes **three
  sequential** free-text calls, each told not to repeat the previous one. `PlanContext`
  (`internal/insights/model.go`) flattens items to `{Description, Scope}`, so no suggestion can
  name the item it is about, although plan-service's `ChecklistItem` proto already carries `id`,
  `parent_id`, `done`, `due_date`, `type` and `archived`. The generator (`internal/ai/generator.go`)
  uses `gpt-5-mini`, `ReasoningEffort: "minimal"`, `MaxCompletionTokens: 200`, three transport
  retries, and records no usage.
- **Events.** plan-service publishes `checklist_item.completed` / `.uncompleted` directly (not via
  the outbox) from `internal/checklistitem/publisher.go`. The payload sets `id`, `plan_id` and the
  timestamp. **`user_id` exists on both messages and is left empty**, and **no AMQP `MessageId`
  is set**, although the insights inbox dedups on `MessageId`. The bulk daily reset publishes no
  uncompleted events. No `checklist_item.deleted` is published.
- **Consumer.** insights' AMQP consumer exists but is never started and has four known defects,
  listed in I-0026 (HUMAN-OWNED, FS-0006). See §Pre-existing defects.
- **Error seam.** The gateway maps gRPC status classes to problem codes in
  `internal/apierr/problem.go`. It has no `ResourceExhausted` or `FailedPrecondition` case, and its
  only domain code (`PROFILE_NAME_EMPTY`) is picked by **substring-matching "name"** in an
  `InvalidArgument` message. Any new `InvalidArgument` whose message contains "name" would be
  misreported today.

## Decisions and why

Carried from the owner's scoping session (locked, lock preview approved, 2026-10-13). Numbering
follows that session (L1–L19). Defaults added while writing this FS, where the session was
silent, are marked **(spec default)** and listed in §Spec defaults.

**L1. The daily block becomes Today's moves.** One to three moves, each anchored to one milestone
(a top-level, long-term task) with a one-line reason. A move either points at an existing open
step (ticked inline) or proposes a new step. Rationale: the old block produced loose daily items
unconnected to the plan's structure; anchoring every suggestion to a milestone makes the AI
drive milestone completion, which is the product's unit of progress (FS-FSFEC).

**L2. Suggest steps on a milestone card.** On demand, 3–5 proposals, accepted one at a time.

**L3. Proposals only.** Nothing is written until the user accepts, and an accepted suggestion
becomes an ordinary item through plan-service's existing create/update path. This is ADR-0007's
"propose" path: regeneration may propose, never overwrite. Because no generated row is ever written
into a user-editable table, the "touched" state ADR-0007 requires has nothing to protect here; the
brief lives in insights' own table, which users never edit. No dependency on FS-0006's provenance
FSM. If FS-0006 ships, an accepted item is created through the ordinary path and is therefore
`authored`.

**L4. One structured call per generation.** One strict JSON-schema structured-output call replaces
the three sequential calls. The schema bounds the output, so the result is parseable without
heuristics and short.

**L5. Cache per (plan, user, local date), refresh capped.** Stored in the insights database. Refresh
is explicit and limited to 3 per plan per day. Over the cap the API answers RFC 9457
problem+json with its own `common/errcode` code.

**L6. A rule gate, not a classifier.** An unchanged plan-state hash serves the stored brief and
skips embedding, search and the model. Empty personal memory skips personal retrieval. The input
is structured, so a rule is the classifier; an ML classifier would add cost and a failure mode
for a decision a comparison can make exactly.

**L7. Prompt order.** Static system rules → retrieved snippets → plan tree → request. The plan's
own state is always injected, never retrieved: it is small, authoritative and must be complete.
The plan tree carries ids, parent ids, done and due, so the model can anchor to real items.
Static content first keeps a stable prefix (see cost rationale).

**L8. `rag_chunks` on pgvector, in fireplace_insights.** `vector(512)`; embeddings from
`text-embedding-3-small` with `dimensions=512`; model and dimensions recorded per row; `user_id
NULL` marks a shared row; `(source, source_id, chunk_no)` unique; `content_hash` makes upsert
idempotent; exact scan, no HNSW index. Infrastructure recorded in ADR-0018.

**L9. Retrieval filter.** `WHERE user_id = $1 OR user_id IS NULL`, excluding the current plan,
k = 3–5, cosine distance (`<=>`). Below a similarity threshold snippets are dropped and the
prompt forbids inventing history ("last time you…").

**L10. Personal memory ingested asynchronously.** An insights consumer on
`checklist_item.completed` / `.uncompleted` embeds and upserts, or deletes, idempotent through the
existing inbox (`processed_events`).

**L11. Dismissals filter output, never input.** A dismissed suggestion feeds a cosine post-filter
on new output. Dismissed text is never sent to the model: negative examples in a prompt prime the
very thing they forbid and cost input tokens on every call.

**L12. Model.** `gpt-5-mini`, minimal reasoning, configurable per generator. Input, cached input
and output token usage stored per generation.

**L13. "Milestone" stays out of the server.** API paths, contract types and server type names say
item / parent item (ADR-0014). Prompt text may say "milestone"; it is model-facing prose, not a
type.

**L14. The old endpoint goes.** `GET /api/insights/checklist-suggestion-daily` is removed in this FS
once the client has switched. The breaking-change gate flags the removal; that is expected.

**L15. Ownership.** H2 (Retriever) and H3 (ingest consumer) are human-owned, each with an "explain
before you code" decision card. Everything else is agent work. See §Ownership split. *Amended by
the owner (2026-10-14):* the chunker and playbook batch ingest became H4, human-owned with its own
card, and the eval became owner-gated, with the owner setting R30's final values.

**L16. Fat events.** `ChecklistItemCompletedEvent` and `ChecklistItemUncompletedEvent` gain
`description` and `parent_id` (additive). plan-service populates them. No plan-service table or RPC
change. Rationale: the consumer can embed from the event alone, with no call back into
plan-service on the ingest path, so ingest keeps working while plan-service is busy and the
consumer stays a pure function of the event stream.

**L17. Playbook corpus in v1.** About 20 worked milestone-breakdown documents, drafted by an agent
and reviewed by the owner, stored as shared rows, chunked by fixed window with overlap, loaded by a
batch ingest command. Rules that always apply go in the system prompt; examples, of which only a
few fit any one plan, are retrieved (dynamic few-shot). An eval measures recall@k across chunk
sizes and k, plus deterministic rule checks on generated output.

**L18. An accepted new step is a child of its milestone.** It counts toward the progress strip and
survives the nightly reset, unlike the daily-scope items the old block created.

**L19. Suggest next milestone, last and first to cut.** `POST /api/insights/item-tree-suggestion`
returns a title and about three steps.

### Cost rationale

On `gpt-5-mini` an output token costs about 8× an input token, so the levers are **fewer calls**
(one per action, L4; none when the gate holds, L6) and **short structured output** (a bounded
schema, one-line reasons). OpenAI's automatic prompt caching only applies to a prefix of at least
1024 tokens; the static-first order (L7) makes a cache hit possible, but the design does not
count on one. Cached input tokens are recorded (L12) so the effect can be measured, not assumed.

## Requirements

### Plan tree and prompt assembly (insights-service)

- **R1.** The plan context used by guided suggestions is a **plan tree**: every non-archived,
  long-term **task** item with its `id`, `parent_id`, `description`, `done` and `due_date`, read
  from plan-service's `ListItems`. Notes, daily-scope items and archived items are not in the tree.
  *(spec default S1)* The older generators (single suggestion, videos) keep their current input.
- **R2.** The prompt is assembled in this order, and only this order: (1) static system rules,
  byte-identical across calls of the same generator; (2) retrieved snippets, if any pass R27;
  (3) the plan tree, serialized compactly with ids; (4) the request (which kind of output, and for
  step suggestions the target parent item's id).
- **R3.** The plan's own state is always injected and never retrieved. Retrieval never returns a
  row from the plan being suggested for (R26).
- **R4.** Retrieved snippets are labelled by origin: playbook examples as worked examples,
  personal rows as things this user finished on other plans.
- **R5.** When no snippet survives the threshold (weak retrieval), the snippet block is omitted
  entirely and the system rules forbid referring to the user's past work.
- **R6.** Prompt text may say "milestone". No API path, contract type, proto message or server
  type name does (ADR-0014). Server code calls the anchor a **parent item**.
- **R7.** Every generation is **one** strict JSON-schema structured-output call. The schema fixes
  the field set and bounds array lengths (moves 1–3, step suggestions 3–5, item-tree steps 1–5).
  No free-text parsing, no follow-up call to repair output. Transport retries on a failed call
  (as today) are not extra generations.
- **R8.** The model is configurable **per generator** (daily brief, step suggestions, item tree),
  defaulting to `gpt-5-mini` with minimal reasoning. The output-token ceiling is configurable per
  generator and sized to the schema.
- **R9.** Every generation records: kind (`daily_brief` | `step_suggestions` | `item_tree`), plan id,
  user id, model, input tokens, cached input tokens, output tokens, and time. A failed call records
  what the API reported, if anything.

### Validation of generated output (insights-service)

- **R10.** Each generated move is checked before it is stored: its `parent_item_id` is an open,
  top-level task in the tree; an `existing_step` names a `step_item_id` that is an open child of
  that parent; a `new_step` carries non-empty text that does not duplicate (case- and
  whitespace-insensitive) an existing child of that parent; `reason` is non-empty.
- **R11.** Within one output, no two moves name the same `step_item_id`, and no two texts are
  duplicates.
- **R12.** Output failing R10/R11 is dropped item by item, never repaired and never regenerated.
  A brief may therefore hold fewer moves than the model returned, down to zero.
- **R13.** The same checks apply to step suggestions (target parent valid, no duplicates of
  existing children or of each other) and to an item-tree suggestion (non-empty title not
  duplicating an existing top-level task, steps non-empty and distinct).

### Daily brief (insights-service + gateway)

- **R14.** A daily brief belongs to one (plan, user, date), where `date` is the user's local
  calendar date sent by the client. At most one brief is stored per key.
- **R15.** **Read.** `GET daily-brief` returns the stored brief for the key. If none exists, the
  rule gate runs (R18); if the gate does not serve one, a brief is generated, stored and returned.
  A read never counts against the refresh cap. *(spec default S2)*
- **R16.** **Refresh.** `POST daily-brief/refresh` redraws the brief for the key. A refresh that
  reaches the model counts against the cap; a refresh served by the rule gate does not. With no
  stored brief for the key, a refresh behaves as a read but counts if it reaches the model.
  *(spec default S2)*
- **R17.** **Cap.** At most **3** counted refreshes per (plan, user, date). A refresh that would be
  the fourth is refused with `429 · BRIEF_REFRESH_LIMIT_REACHED` and the stored brief is left
  untouched. The check and the increment are one atomic step, so concurrent refreshes cannot
  overspend it. A failed generation does not count.
- **R18.** **Rule gate.** Before any embedding, retrieval or model call:
  - if the plan has no parent items (no open, top-level, long-term task), store and return an
    empty brief with no generation;
  - if the plan-state hash (R19) equals the hash stored with the user's most recent brief for this
    plan, **and** none of that brief's moves has been dismissed, serve that brief's moves (copied
    forward to today's key on a read) with no generation; *(spec default S3)*
  - if the user has no personal-memory rows, retrieval reads shared rows only.
- **R19.** The plan-state hash is a digest over the plan tree of R1 (each item's id, parent id,
  description, done, due date), ordered by id, so two trees the model would see identically hash
  identically.
- **R20.** The response carries `refreshesRemaining` (0–3) so the client can show or retire the
  refresh action without a failed request. *(spec default S4)*
- **R21.** The date must lie within one day of the server's UTC date (inclusive), which covers
  every real time zone. Outside it the request is refused with `400 · BRIEF_DATE_OUT_OF_RANGE`.
  This bounds cap evasion by date-hopping to three dates. *(spec default S5)*
- **R22.** Concurrent first reads for one key produce one stored brief; every caller receives it.
  A duplicate generation lost to the race is still recorded as usage (R9).

### Step and item-tree suggestions (insights-service + gateway)

- **R23.** `POST step-suggestions` with a plan and a parent item returns 3–5 step suggestions for
  that parent (fewer after R13 and the dismissal filter, down to zero). The parent must be an open,
  top-level, long-term task of the plan; an item that exists in the plan but fails that is refused
  with `400 · PARENT_ITEM_INVALID`; an item not in the plan is `404 · NOT_FOUND`.
- **R24.** `POST item-tree-suggestion` with a plan returns one proposed parent item: a title, a
  one-line reason and 1–5 (aim about 3) steps. It works on a plan with no parent items.
- **R25.** Step and item-tree suggestions are not cached: each request is one generation. Each
  suggestion is stored with an id so feedback can reference it. *(spec default S6; see open item O2)*

### Retrieval (insights-service; the Retriever is H2, human-owned)

- **R26.** Retrieval reads only shared rows and the requesting user's own rows
  (`user_id = $user OR user_id IS NULL`), never a row whose plan is the plan being suggested for,
  and only rows whose model and dimensions equal the embedder's.
- **R27.** It returns at most k rows (k between 3 and 5, configurable) ordered by cosine distance,
  and drops every row below a configurable similarity floor. Fewer than k, including zero, is a
  valid result and is never padded.
- **R28.** **Tenant isolation is tested:** an automated test shows user A never retrieves user B's
  rows, for any query, including when B's rows are the nearest.
- **R29.** Retrieval is a seam (an interface owned by its consumer), so the brief, step and
  item-tree flows can be built and tested against a stub before H2 lands.
- **R30.** k, the similarity floor and the playbook chunk size are set **by the owner** from the eval (R44), with
  the chosen values and the run that chose them noted beside the configuration.

### Storage: `rag_chunks` and the embedder (insights-service)

- **R31.** `rag_chunks` lives in `fireplace_insights` on pgvector (ADR-0018). Each row: `id`;
  `user_id` (NULL = shared); `plan_id` (NULL for shared rows; set for personal rows, used by R26);
  `source` (`playbook` | `completion`); `source_id`; `chunk_no`; `content`; `content_hash`;
  `embedding vector(512)`; `embedding_model`; `embedding_dims`; timestamps.

**Infrastructure (ADR-0018, owner-accepted).** pgvector is **not** a trusted extension, so it
cannot be created by the owner role that migrations run as (ADR-0010 §2). R31a–R31d are how the
database gets it without giving any app or migration role superuser.

- **R31a. Image.** The shared Postgres instance image is pinned to
  `pgvector/pgvector:0.8.7-pg17` (an exact tag, never a floating one). The `postgres-test`
  service in `docker-compose.override.yml` changes to the same image in the same change, because
  the test tier mirrors production (ADR-0010 §5). The backup image stays `postgres:17-alpine`.
- **R31b. Bootstrap.** A superuser step in `scripts/db/init-databases.sh`, **outside** the
  `SET ROLE <owner>` block, runs `CREATE EXTENSION IF NOT EXISTS vector` in `fireplace_insights`
  **only**; `fireplace_plans` and `fireplace_gateway` never get it. That step creates the
  extension and nothing else. Migrations assume the extension exists and never try to create it.
  The app (runtime) and migration (owner) roles never connect as superuser, and the runtime role
  gains no `CREATE`.
- **R31c. Existing volumes.** `init-databases.sh` only runs on an empty data directory, so every
  volume that already exists (production, local dev, test) needs a **documented one-off**
  `CREATE EXTENSION vector;` in `fireplace_insights`, run by hand as the instance superuser before
  the `rag_chunks` migration is deployed to it. Without it the migration fails with
  `type "vector" does not exist`. The procedure is written down beside the other database
  operations runbooks, with the exact command per environment.
- **R31d. Backup and restore.** `deploy/backup/backup.sh` dumps `fireplace_insights` with
  `pg_dump --exclude-extension=vector`, so the dump (taken as the owner role) carries the
  `rag_chunks` table and data but no `CREATE EXTENSION` it could not replay.
  `deploy/backup/restore.sh` states and checks its precondition: the target `fireplace_insights`
  already has the `vector` extension (from R31b or R31c), and the restore refuses with a clear
  message otherwise. The existing restore drill stays green.
- **R32.** `(source, source_id, chunk_no)` is unique. An upsert whose `content_hash` matches the
  stored row does not re-embed and does not rewrite the row.
- **R33.** No ANN index (HNSW/IVFFlat) in v1. Queries are exact scans. Adding one is a later
  decision with its own recall check (ANN plus a filter can return fewer than k).
- **R34.** One **embedder** serves ingest and query. It calls `text-embedding-3-small` with
  `dimensions=512` and stamps model and dimensions on every row it produces, so a mismatch is
  impossible by construction and a future model change is a re-embed migration rather than a
  silent mix.

### Playbook corpus (insights-service; the chunker and batch ingest are H4, human-owned)

- **R35.** About 20 playbook documents, each a worked breakdown of a goal into milestones and steps,
  across both plan types (`project`, `learning`). Agent-drafted, **owner-reviewed before
  ingest**. They live in the repo beside the service.
- **R36.** A batch ingest command (and make target) chunks each document by fixed window with
  overlap, embeds, and upserts shared rows. Re-running with no document changes embeds nothing
  (R32). A document removed from the corpus has its rows deleted on the next run. *(spec default S7)*

### Personal memory (insights-service ingest consumer; H3, human-owned)

- **R37.** An insights-owned consumer receives `checklist_item.completed` and
  `checklist_item.uncompleted` from `plan.events`, on a queue insights declares itself.
- **R38.** On completed: embed the item's description and upsert one personal row (`user_id`,
  `plan_id`, `source = completion`, `source_id` = item id, `chunk_no = 0`), with `parent_id` kept
  as row metadata. Completing the same item again upserts the same row.
- **R39.** On uncompleted: delete that item's personal row. Deleting a row that does not exist
  succeeds.
- **R40.** Each event is applied at most once through the inbox (`processed_events`, keyed by the
  AMQP `MessageId`), in the same transaction as the row write.
- **R41.** An event older than the last one applied for the same item never overrides it, so a
  redelivered completion cannot resurrect a row a later un-completion removed.
- **R42.** A transient failure (embedding API, database) requeues; a malformed message is not
  requeued. Ingest failing never affects plan-service or the user's tick.

### Self-describing completion events (plan-service + common proto)

- **R43.** `ChecklistItemCompletedEvent` and `ChecklistItemUncompletedEvent` gain `description` and
  `parent_id` (optional), additively. plan-service populates them, populates the existing
  `user_id` (the user who flipped `done`), and sets a fresh UUID `MessageId` on every publish.
  Generated code is regenerated, never hand-edited. No plan-service table, migration or RPC
  changes.

### Eval (insights-service; owner-gated)

- **R44.** A manual make target runs the eval against the live OpenAI API and a pgvector database.
  It is **not** part of CI. It reports recall@k for the playbook over a labelled query set, across
  at least three chunk sizes and k ∈ {3, 5}.
- **R45.** The same target runs deterministic rule checks on generated output over fixture plans:
  anchors valid, kinds valid, counts within bounds, no duplicates, no new step duplicating an
  existing child (R10–R13), and reports pass rates.

### Feedback and the dismissal filter (insights-service + gateway)

- **R46.** `POST suggestions/{id}/feedback` records `accepted` or `dismissed` for a suggestion (a
  move, a step suggestion or an item-tree suggestion) owned by the caller. Recording is
  idempotent; a later outcome replaces an earlier one. *(spec default S8)* A suggestion that does
  not exist or is not the caller's is `404 · NOT_FOUND`.
- **R47.** Accepting is the client's job, through plan-service's ordinary endpoints (create a child
  item, create a top-level task and its children, or tick an existing step); feedback only records
  it. insights-service never writes plan data. *(spec default S9)*
- **R48.** A dismissed suggestion's text is embedded once, at dismissal, and kept with the
  suggestion (not in `rag_chunks`).
- **R49.** New output for a plan is post-filtered: a new-step move, step suggestion or item-tree
  title whose cosine similarity to any suggestion the user dismissed on that plan is at or above a
  configurable threshold is dropped. An `existing_step` move whose step the user dismissed as a
  move on that plan is dropped by id. Dismissed text never enters a prompt. *(spec default S10)*
- **R50.** When the user has no dismissals on the plan, the filter does nothing and embeds nothing
  (rule gate).

### Gateway and contract

- **R51.** The five operations in §API surface are typed Huma handlers; `openapi.yaml` and the
  generated client are regenerated, never hand-edited. Request types list writable fields only;
  identity comes from the JWT and is forwarded as `user_id` over gRPC.
- **R52.** Domain codes cross gRPC as structured status detail (`ErrorInfo.reason` = the errcode),
  not message text, and the gateway's single error seam reads that detail before the status-class
  fallback. The seam maps `ResourceExhausted → 429` and `FailedPrecondition → 400`. Existing
  mappings, including `Unavailable → 503`, are unchanged.
- **R53.** New codes in `common/errcode`: `BRIEF_REFRESH_LIMIT_REACHED`, `BRIEF_DATE_OUT_OF_RANGE`,
  `PARENT_ITEM_INVALID`. No others.
- **R54.** A generation that fails (model unreachable or timing out after transport retries, or
  output that is not valid against the schema) surfaces as `503 · SERVICE_UNAVAILABLE`: retrying
  is correct. It stores nothing and spends no refresh.
- **R55.** `GET /api/insights/checklist-suggestion-daily` and the `GenerateDailySuggestions` RPC are
  removed once the client no longer calls them. The breaking-gate finding is recorded in the
  allowlist per `docs/agents/contract.md`, citing this FS.
- **R56.** JSON bodies and new query parameters use camelCase like the rest of the contract
  (`planId`, `parentItemId`, `stepItemId`). *(spec default S11)*

### Client: Today's moves

- **R57.** On a plan's **daily** tab, the suggestions block is replaced by **Today's moves**. It
  keeps the existing show/hide toggle and its remembered preference.
- **R58.** It reads the brief for the browser-local date. Each move shows its milestone's name,
  the move's text and its reason.
- **R59.** Moves are resolved against the plan's loaded items at render time. A move whose
  milestone is gone, archived, closed or no longer top-level is hidden; an `existing_step` move
  whose step is gone or archived is hidden.
- **R60.** An `existing_step` move has a checkbox. Ticking it marks that step done through the
  existing item update (optimistic, reverted on failure), then records `accepted`. A ticked move
  stays in the block, shown done, for the rest of the day. The step's milestone progress moves
  with it (FS-FSFEC R20).
- **R61.** A `new_step` move has an **Add** action. Adding creates a long-term task **as a child of
  the move's milestone** through the existing create endpoint, then records `accepted`; the move
  then shows as added. A failed create keeps the move and shows the existing error.
- **R62.** Every move can be dismissed. Dismissing hides it and records `dismissed`.
- **R63.** A **refresh** action shows the refreshes left. At zero it is retired. A refresh that
  comes back unchanged (gate served) says nothing has changed since these moves were drawn.
- **R64.** A plan with no milestones shows an invitation to start one instead of moves. A plan
  with milestones and an empty brief says there is nothing to suggest right now.
- **R65.** Loading shows a skeleton of the block; a failed read shows a quiet retry, never an
  error banner over the checklist.

### Client: suggested steps

- **R66.** In the grid (milestone reading), every open milestone card offers **Suggest steps**.
  Closed milestones do not. *(spec default S12)*
- **R67.** Asking shows 3–5 suggested steps under the card. Each has **Add** (creates a task child
  of that milestone, records `accepted`) and dismiss (records `dismissed`). Adding one leaves the
  rest in place; the panel closes when the user closes it or the list is empty.
- **R68.** Closing the panel records nothing; only an explicit dismiss feeds the filter.

### Client: suggested milestone (last slice; first to cut)

- **R69.** The grid composer's resting state offers **Suggest a milestone**. *(spec default S13)*
- **R70.** The proposal shows a title, reason and its steps. **Accept** creates a top-level task
  with that title and then its steps as children through the existing create endpoint, then
  records `accepted`; **Dismiss** records `dismissed`. If a step create fails, what was created
  stays, and the existing error is shown.

## User Stories

1. As a planner, I want today's suggestions tied to my milestones, so that what I do today moves
   a milestone forward.
2. As a planner, I want each move to say why, in one line, so that I can judge it at a glance.
3. As a planner, I want a move that points at a step I already have, so that I'm nudged to finish
   work instead of adding more.
4. As a planner, I want to tick that step right in the block, so that doing it takes one click.
5. As a planner, I want a move that proposes a new step, so that a stuck milestone gets unstuck.
6. As a planner, I want an added move to land under its milestone, so that it counts toward the
   progress strip.
7. As a planner, I want an added move to survive the nightly reset, so that it isn't treated as
   daily busywork.
8. As a planner, I want no more than three moves, so that the day stays focused.
9. As a planner, I want today's moves to stay put when I reload, so that the plan for my day
   doesn't shuffle under me.
10. As a planner, I want to redraw my moves a few times a day, so that a bad draw isn't final.
11. As a planner, I want to know how many redraws I have left, so that the cap doesn't surprise
    me.
12. As a planner who changed nothing, I want a redraw to tell me nothing changed, so that I know
    why the moves are the same.
13. As a planner, I want to dismiss a move, so that it goes away.
14. As a planner, I want dismissed ideas not to come back, so that I don't keep saying no.
15. As a planner, I want to ask a milestone for suggested steps, so that I can break it down when
    I'm not sure how.
16. As a planner, I want to add suggested steps one at a time, so that I keep only the good ones.
17. As a planner, I want the AI to propose a next milestone, so that I can see what the plan is
    missing.
18. As a planner, I want nothing written to my plan until I accept, so that my plan stays mine.
19. As a planner, I want suggestions informed by how I broke down similar goals before, so that
    they fit how I work.
20. As a planner, I want suggestions never to claim history I don't have, so that I can trust
    them.
21. As a planner on a new account, I want suggestions to still be good, so that the feature works
    before I have any history.
22. As a planner, I want my history never shown to another user, so that my work stays private.
23. As a planner with an empty plan, I want to be invited to start a milestone, so that the block
    isn't blank.
24. As the owner, I want one model call per action and none when nothing changed, so that the
    feature stays cheap.
25. As the owner, I want token usage stored per generation, so that I can see what it costs.
26. As the owner, I want the model configurable per generator, so that I can trade cost for
    quality where it matters.
27. As the owner, I want an eval that measures retrieval recall across chunk sizes and k, so that
    the numbers I pick are measured, not guessed.
28. As the owner, I want rule checks on generated output, so that invalid anchors and duplicates
    are caught.
29. As the owner, I want to build the retriever, the ingest consumer and the chunker myself, so that I can
    explain every RAG decision in an interview.
30. As a developer, I want the server to say "parent item", so that milestones stay a client
    reading (ADR-0014).
31. As a client developer, I want a distinct error code for the refresh cap, so that I can switch
    on it rather than parse text.
32. As an operator, I want ingest failures not to affect ticking items, so that memory is never on
    the user's critical path.

## Acceptance Criteria

### Server: brief, gate, cap

- [ ] `GenerateDailySuggestions`' three sequential calls are gone; a brief costs at most one
      structured-output call.
- [ ] The plan tree sent to the model carries ids, parent ids, done and due for long-term,
      non-archived tasks; notes, daily and archived items are absent.
- [ ] The prompt order is system rules → snippets → plan tree → request, and the system rules
      are byte-identical across two calls.
- [ ] A second read for the same (plan, user, date) makes no model call and returns the same moves.
- [ ] A read on a new day with an unchanged plan and no dismissals makes no model call.
- [ ] A refresh with an unchanged plan and no dismissals makes no model call and does not lower
      `refreshesRemaining`.
- [ ] Three counted refreshes succeed; the fourth returns `429 · BRIEF_REFRESH_LIMIT_REACHED` and
      leaves the stored brief unchanged.
- [ ] Concurrent refreshes cannot exceed the cap.
- [ ] A plan with no open top-level long-term task returns an empty brief with no model call.
- [ ] A date more than one day from the server's UTC date returns `400 · BRIEF_DATE_OUT_OF_RANGE`.
- [ ] Moves with an invalid parent, a non-child or done step, a duplicate, or an empty reason are
      dropped; nothing is repaired or regenerated.
- [ ] A failed generation returns `503 · SERVICE_UNAVAILABLE`, stores nothing and spends no refresh.
- [ ] Every generation stores model, input, cached input and output tokens.
- [ ] The model is set per generator; the default is `gpt-5-mini` with minimal reasoning.

### Server: suggestions and feedback

- [ ] Step suggestions return 3–5 items for a valid parent; a top-level note, closed task, child
      item or daily item as parent returns `400 · PARENT_ITEM_INVALID`; an item outside the plan
      returns `404 · NOT_FOUND`.
- [ ] Item-tree suggestion returns a title, reason and 1–5 steps, including for an empty plan.
- [ ] Feedback on the caller's suggestion returns 204, idempotently; on another user's returns 404.
- [ ] After dismissing a suggestion, a near-identical one is filtered out of later output on that
      plan; the dismissed text never appears in any prompt.
- [ ] With no dismissals on a plan, no output embedding call is made.
- [ ] No API path, contract type, proto message or server type name contains "milestone".
- [ ] `GET /api/insights/checklist-suggestion-daily` and `GenerateDailySuggestions` are removed;
      the breaking gate's finding is allowlisted citing FS-DF4ZB; `make -C services/api-gateway
      gates` is otherwise green.
- [ ] Domain codes reach the gateway as `ErrorInfo.reason`; an `InvalidArgument` from insights
      whose message contains "name" is not reported as `PROFILE_NAME_EMPTY`.

### Retrieval, storage, ingest, eval

- [ ] `rag_chunks` exists in `fireplace_insights` with the columns of R31, the unique key of R32,
      and no ANN index.
- [ ] The shared instance and `postgres-test` both run `pgvector/pgvector:0.8.7-pg17`; the backup
      image is still `postgres:17-alpine`.
- [ ] On a fresh volume, `init-databases.sh` creates `vector` in `fireplace_insights` as superuser,
      outside the `SET ROLE` block; `fireplace_plans` and `fireplace_gateway` have no `vector`
      extension.
- [ ] No migration contains `CREATE EXTENSION vector`; app and migration roles connect as their
      own (non-superuser) roles; the runtime role has no `CREATE`.
- [ ] The one-off `CREATE EXTENSION vector` procedure for existing volumes is documented per
      environment (prod, local dev, test) and has been run on local dev and test before the
      `rag_chunks` migration lands there.
- [ ] `backup.sh` dumps `fireplace_insights` with `--exclude-extension=vector`; `restore.sh`
      refuses a target without the extension.
- [ ] **A restore of `fireplace_insights` containing `rag_chunks` rows has been performed** into a
      target that has the extension, and the restored rows are present and queryable with `<=>`.
      The restore drill is green.
- [ ] Re-ingesting an unchanged playbook embeds nothing; editing one document re-embeds only its
      chunks; removing one deletes its rows.
- [ ] Every row records `text-embedding-3-small` and 512; retrieval ignores rows with other
      values.
- [ ] Retrieval never returns a row from the current plan or another user (tenant-isolation
      test passes).
- [ ] Retrieval returns fewer than k when fewer clear the floor, and zero when none do; with
      zero, the prompt has no snippet block and forbids claims about past work.
- [ ] A user with no personal rows triggers no personal retrieval.
- [ ] Completed/uncompleted events carry `description`, `parent_id`, a populated `user_id` and a
      `MessageId`; plan-service has no migration or RPC change.
- [ ] Completing an item creates one personal row; completing it again does not add a second;
      un-completing deletes it; replaying either event changes nothing; an older event never
      overrides a newer one.
- [ ] Stopping the embedding API makes ingest requeue; ticking items keeps working.
- [ ] The eval target prints recall@k for ≥3 chunk sizes × k ∈ {3, 5} and rule-check pass rates;
      it is not run in CI. The owner has read the results and set k, floor and chunk size, noted
      beside their configuration.
- [ ] About 20 playbook documents exist, reviewed by the owner before ingest.

### Client

- [ ] The daily tab shows "Today's moves" in place of the old suggestions block, with the same
      show/hide toggle.
- [ ] Each move names its milestone, its text and its reason; there are never more than three.
- [ ] Ticking an existing-step move marks that step done, moves its milestone's progress, and
      reverts on failure.
- [ ] Adding a new-step move creates a long-term child of that milestone that survives the nightly
      reset; no daily-scope item is created.
- [ ] Moves pointing at deleted, archived or closed items are not shown.
- [ ] Dismissing hides a move and records it.
- [ ] Refresh shows the remaining count and retires at zero; an unchanged redraw says so.
- [ ] No milestones → invitation; milestones but no moves → "nothing to suggest right now".
- [ ] Open milestone cards offer "Suggest steps"; closed ones do not; added steps land as children
      and the rest stay listed.
- [ ] The grid composer offers "Suggest a milestone"; accepting creates the task and its steps.
- [ ] The client no longer calls `checklist-suggestion-daily`.
- [ ] Tests pass.

## Edge States

| Situation | Behavior |
| --- | --- |
| Plan has no long-term tasks | Empty brief, no model call (R18); client shows the start-a-milestone invitation (R64). |
| Every milestone closed | No parent items: empty brief (R18). |
| Milestone with no steps | Valid anchor; only `new_step` moves possible for it. |
| Model returns only invalid moves | Brief stored with zero moves (R12); "nothing to suggest right now". |
| Dismissal filter removes every move | Same as above; no regeneration (R12, R49). |
| User ticks a move's step elsewhere (list, grid, home) | The move renders done from live items (R59–R60). |
| Move's milestone archived or deleted after drawing | Move hidden (R59). Brief not redrawn automatically. |
| Plan edited after the brief was drawn | Brief kept for the day; the next refresh is not gated (hash differs) and counts. |
| Refresh with nothing changed | Gate serves the stored moves, no cap spent, client says unchanged (R16, R63). |
| Refresh after dismissing a move, nothing else changed | Gate bypassed (R18); counts; dismissed move filtered (R49). |
| Fourth counted refresh | `429 · BRIEF_REFRESH_LIMIT_REACHED`; stored brief stays (R17). |
| Two tabs refresh at once at count 2 | One succeeds, one gets 429 (R17). |
| Two first reads at once | One brief stored, both receive it (R22). |
| Local midnight passes with the page open | Next read sends the new date and gets (or makes) that day's brief. |
| Client clock wrong by more than a day | `400 · BRIEF_DATE_OUT_OF_RANGE` (R21); client shows the quiet retry. |
| Model or OpenAI down | `503 · SERVICE_UNAVAILABLE`; nothing stored, no refresh spent (R54). |
| `rag_chunks` empty (playbook not ingested) | Weak retrieval: no snippets, prompt forbids history claims (R5). Generation still works. |
| Existing volume never given the extension | `rag_chunks` migration fails with `type "vector" does not exist`; run the R31c one-off as superuser, then migrate. |
| Restore into a target without the extension | `restore.sh` refuses before restoring (R31d). |
| New user, no personal memory | Shared rows only (R18). |
| Personal row from a plan later deleted | Still retrievable for that user; see open item O3. |
| Item completed then uncompleted quickly, delivered out of order | Older event does not override newer (R41). |
| Duplicate delivery of a completion | Inbox makes it a no-op (R40). |
| Event missing `user_id` or `MessageId` (old publisher during rollout) | Treated as malformed: not requeued, logged (R42). Personal memory starts from the deploy of R43. |
| Daily item completed every day | One row per item id, upserted; unchanged text re-embeds nothing (R32, R38). |
| Bulk nightly reset | Publishes nothing, so personal rows for daily items stay (they were real completions). |
| Embedding API down during ingest | Requeue (R42); ticking unaffected. |
| Embedding API down during a request that needs retrieval | Weak-retrieval path; generation proceeds without snippets. *(spec default S14)* |
| Feedback for an unknown or foreign suggestion id | `404 · NOT_FOUND` (R46). |
| Accept create fails | Move or suggestion stays, existing error shown; no `accepted` recorded (R61, R67). |
| Item-tree accept partly fails | Created items stay, error shown (R70). |
| Shared plan, viewed by a non-owner | Whatever plan-service's `GetPlan` allows today; brief and memory are per user. |
| Step suggestions on a parent that already has many open steps | Allowed; suggestions must not duplicate them (R13). |

## API surface

All operations are JWT-protected, routed to insights-service over gRPC, and carry the caller's
identity from the token, never the body. Shape failures (malformed uuid or date, unknown member,
bad enum) are `422 · VALIDATION_FAILED` at the boundary; domain refusals are decided by
insights-service (ADR-0005). Every operation also returns `401 · UNAUTHENTICATED`,
`403 · FORBIDDEN` (plan not the caller's, from plan-service), `503 · SERVICE_UNAVAILABLE`
(insights or the model unreachable, R54) and `500 · INTERNAL_ERROR`, as the existing insights
operations do; the Errors column lists only what is specific to each.

| Op | Method + Path | Query/Params | Request body | Response | Errors |
|----|---------------|--------------|--------------|----------|--------|
| getDailyBrief | `GET /api/insights/daily-brief` | `planId` uuid, required · `date` `YYYY-MM-DD`, required | — | `200` **DailyBrief** | `400 · BRIEF_DATE_OUT_OF_RANGE` · `404 · NOT_FOUND` (plan) |
| refreshDailyBrief | `POST /api/insights/daily-brief/refresh` | — | `planId` uuid, required · `date` `YYYY-MM-DD`, required | `200` **DailyBrief** | `400 · BRIEF_DATE_OUT_OF_RANGE` · `404 · NOT_FOUND` (plan) · `429 · BRIEF_REFRESH_LIMIT_REACHED` |
| suggestSteps | `POST /api/insights/step-suggestions` | — | `planId` uuid, required · `parentItemId` uuid, required | `200` **StepSuggestions** | `400 · PARENT_ITEM_INVALID` · `404 · NOT_FOUND` (plan, or item not in plan) |
| recordSuggestionFeedback | `POST /api/insights/suggestions/{id}/feedback` | `id` uuid (path) | `outcome` enum `accepted` \| `dismissed`, required | `204` | `404 · NOT_FOUND` (unknown or not the caller's) |
| suggestItemTree | `POST /api/insights/item-tree-suggestion` | — | `planId` uuid, required | `200` **ItemTreeSuggestion** | `404 · NOT_FOUND` (plan) |
| ~~getDailyChecklistSuggestion~~ | ~~`GET /api/insights/checklist-suggestion-daily`~~ | — | — | **Removed** (R55) | breaking gate flags it; allowlisted citing this FS |

**DailyBrief**

| Field | Type | Notes |
|---|---|---|
| `planId` | uuid | |
| `date` | `YYYY-MM-DD` | the key's date, as sent |
| `generatedAt` | timestamp | when the moves were drawn (earlier than today when the gate carried them forward) |
| `refreshesRemaining` | int 0–3 | |
| `moves` | **Move**[] 0–3 | never null |

**Move**

| Field | Type | Notes |
|---|---|---|
| `id` | uuid | suggestion id, used for feedback |
| `parentItemId` | uuid | the anchoring parent item |
| `kind` | enum `existing_step` \| `new_step` | |
| `stepItemId` | uuid, optional | present iff `kind = existing_step` |
| `text` | string | for `existing_step`, the step's description when drawn; for `new_step`, the proposed description |
| `reason` | string | one line |

**StepSuggestions**: `parentItemId` uuid · `suggestions` **StepSuggestion**[] 0–5, never null.
**StepSuggestion**: `id` uuid · `text` string · `reason` string.

**ItemTreeSuggestion**: `id` uuid · `title` string · `reason` string · `steps` { `text` string }[] 1–5.

**New error codes** (`common/errcode`, R53): `BRIEF_REFRESH_LIMIT_REACHED` (429),
`BRIEF_DATE_OUT_OF_RANGE` (400), `PARENT_ITEM_INVALID` (400).

**Plane 2 (gRPC), additive except the removal:** `insights.InsightsService` gains `GetDailyBrief`,
`RefreshDailyBrief`, `SuggestSteps`, `RecordSuggestionFeedback`, `SuggestItemTree` and loses
`GenerateDailySuggestions`. `events.ChecklistItemCompletedEvent` and
`ChecklistItemUncompletedEvent` gain `description` and `optional parent_id`.

## Out of Scope

- **The add-bar single suggestion** (`GET /api/insights/checklist-suggestion`) and its generator:
  unchanged.
- **Video suggestions**, **Notes** and the gateway's `internal/ai`: unchanged.
- **FS-0006** and its provenance FSM: not depended on and not changed.
- **Fixing I-0026's consumer defects inside an agent slice.** They are human-owned; see
  §Pre-existing defects and open item O1.
- **An ANN index** on `rag_chunks`. Exact scan until volume says otherwise (R33).
- **Purging personal rows on `user.deleted` or plan deletion.** Open item O3.
- **Embedding a step's parent title into its personal row.** Open item O4.
- **A cap on step and item-tree suggestions.** Open item O2.
- **Auto-scheduling, due dates on moves, or moves on the dashboard home.**
- **Suggested steps in the list view.** The grid is the milestone reading; list view is unchanged.
- **A UI for token usage.** Usage is stored; reading it is a query.
- **Running the eval in CI.**

## Spec defaults

Where the locked session was silent, this FS chose. Each is cheap to change before the slice that
implements it.

| # | Question | Default | Reason |
| --- | --- | --- | --- |
| S1 | What is in the plan tree? | Long-term, non-archived tasks only | Moves anchor to milestones and steps; notes and daily items cost tokens and anchor nothing |
| S2 | Does a read generate? Does it count? | A read generates on a miss and never counts; only refreshes that reach the model count | The first brief of the day must just appear; the cap guards explicit redraws |
| S3 | What does the gate compare? | Plan-state hash with the most recent brief, bypassed if any of its moves was dismissed | Otherwise dismissing everything then refreshing would serve the dismissed moves back |
| S4 | How does the client know the cap? | `refreshesRemaining` in the brief | No failed request needed to discover it |
| S5 | Can the client-supplied date be trusted? | Accept server UTC date ±1 day | Covers every time zone; bounds cap evasion to three dates |
| S6 | Cache step and item-tree suggestions? | No | On demand by nature; open item O2 covers cost |
| S7 | Playbook document removed | Its rows are deleted on the next ingest | The corpus in git is the source of truth |
| S8 | Feedback changed after the fact | Last outcome wins, idempotent, 204 | No extra error code for a rare, harmless case |
| S9 | Who writes the accepted item? | The client, through existing item endpoints | insights owns no plan data; keeps "accepted = ordinary item" literal |
| S10 | Filter scope | Same plan, same user, no expiry; existing-step moves by id | A dismissal is about this plan; ids are exact where they exist |
| S11 | JSON casing | camelCase bodies and new query params | The contract's convention (`planId`, `parentId`); the thread's snake_case was shorthand, and legacy insights `plan_id` stays on legacy operations only |
| S12 | Suggest steps on closed milestones | Not offered | A closed milestone is finished |
| S13 | Where "Suggest a milestone" lives | Grid composer's resting state | The composer is already where new milestones start (FS-FSFEC) |
| S14 | Embedding API down mid-request | Proceed without snippets | Retrieval improves output; it must not gate it |

## Open items for the owner

Not decided on the owner's behalf. None blocks the agent slices.

- **O1. H3 vs I-0026.** H3 runs on the same insights consumer code that I-0026 (HUMAN-OWNED, FS-0006)
  lists as broken: never started, `inboxService` nil, wrong queue binding, Redis claim failing
  closed. Either H3 absorbs those four fixes, or H3 waits for I-0026. The owner chooses. Note that
  I-0026's binding target (`plan.items_requested`) differs from H3's (`checklist_item.*`), so the
  queue topology has to serve both whichever way it goes.
- **O2. Cost cap for step and item-tree suggestions.** Uncapped in v1 by default; usage rows make
  spend visible. A per-plan daily cap like R17 would need its own code.
- **O3. Deletion of personal memory.** Nothing removes a user's rows when the user is deleted
  (insights does not consume `user.deleted`) or when a plan is deleted (`plan.deleted` is
  stubbed), and no `checklist_item.deleted` is published. Personal data outlives its source.
  Recommended as a follow-up FS before real users.
- **O4. Personal row content.** The event carries `parent_id` but not the parent's description.
  Embedding the step alone is the default; enriching it with the parent's title would need a
  lookup or a fatter event. H3's call.

## Ownership split for `/spec-to-issues`

**Owner-lane work is filed as issues, marked `[HUMAN]`, and excluded from `/develop`**, with the
do-not-develop banner in the body, as FS-0006 does.

**H2: Retriever. [HUMAN]** R26–R29, and R30's mechanism: the query text for each generation
kind, the similarity floor and its fallback, k, and the tenant-isolation test. Builds on the
agent's embedder and repository (R31–R34). Agent work codes against the R29 seam until it lands.

**H3: Ingest consumer. [HUMAN]** R37–R42. **Blocked by** the fat-event slice (R43) and the storage
slice (R31–R34). Depends on open item O1.

**H4: Chunker and playbook batch ingest. [HUMAN]** R36, plus R32's re-ingest behavior for playbook
rows: comparing `content_hash` so an unchanged chunk is not re-embedded or rewritten, and deleting
the rows of a document removed from the corpus. The storage slice (agent) still provides R32's
unique key and the repository's upsert keyed on `(source, source_id, chunk_no)`; H4 owns the
chunker (fixed window with overlap), the batch command and its make target, and what a re-run
does. **Blocked by** the storage slice (R31–R34). Uses the playbook drafts (R35, agent-drafted,
owner-reviewed) as input.

**Each of H2, H3 and H4 opens with an "explain before you code" decision card.** Before writing
code, the owner writes down, in their own words, the answer to each question that applies to their
slice. The positions this FS took are in §Decisions and why; the card is the owner's explanation
of them, not a copy.

| Question | H2 | H3 | H4 |
|---|:-:|:-:|:-:|
| Why 512 dimensions and not 1536? | ✓ | ✓ | |
| Why record the model and dimensions on every row? What does a re-embed migration look like? | ✓ | ✓ | |
| Why a `user_id` column filter, rather than a namespace per tenant or a table per tenant? | ✓ | ✓ | |
| Why no HNSW yet? Why can an ANN index plus a filter return fewer than k? | ✓ | | |
| Cosine versus dot product on normalized vectors: what differs, and what doesn't? | ✓ | | |
| How is k chosen? | ✓ | | |
| What is the weak-retrieval threshold for, and what happens below it? | ✓ | | |
| What does `content_hash` buy? | | ✓ | |
| Where does the queue sit in ingest, and what does it protect? | | ✓ | |
| Fat events versus thin events: what did this FS choose and why? | | ✓ | |
| How are chunk size and overlap chosen? | | | ✓ |
| Why are steps not chunked (one row is one chunk) while playbook documents are? | | | ✓ |
| What does `content_hash` buy on re-ingest? | | | ✓ |
| Why is this ingest batch and synchronous while personal memory is asynchronous? | | ✓ | ✓ |

**Owner-gated: the eval (R44–R45).** The eval target is agent-buildable, but an agent implements
it **only when the owner explicitly says so**, never as part of a parallel or charter wave, and
`spec-to-issues` files it with that gate stated in the body. **The judgement is the owner's:** the
owner reads the eval results and sets the final k, similarity floor and chunk size (R30's final
values). An agent that builds the eval reports its numbers; it does not choose or commit those
values.

**Agent lane:** pgvector infrastructure per ADR-0018 (R31a–R31d; the hand-run superuser one-off on
prod is the owner's to execute), the `rag_chunks` migration, the embedder, the repository
(including the keyed upsert H4 builds on), the playbook drafts (owner reviews), the fat-event
proto and plan-service publisher change, the plan tree, prompt assembly, the rule gate, the
structured generators, brief storage and the cap, usage recording, output validation, the
dismissal filter, the error seam extension and codes, the five gateway operations, the old
endpoint's removal, and all client work. Not in the agent lane: H2, H3, H4, and the eval unless
the owner releases it.

**Suggested order** (for slicing; `spec-to-issues` decides):

1. Storage and embedder (R31–R34, including the R31a–R31d infrastructure: image pin, superuser
   bootstrap, one-off extension runbook for existing volumes, backup/restore with a performed
   restore), per ADR-0018.
2. Fat events (R43).
3. Plan tree, structured brief generation, gate, cache, cap, usage, error seam, `getDailyBrief` and
   `refreshDailyBrief` (R1–R22, R51–R54, R56), against a stub retriever.
4. Client Today's moves (R57–R65), then removal of the old endpoint (R55).
5. Feedback and dismissal filter (R46–R50).
6. Playbook drafts (R35), then **H4** Chunker and playbook batch ingest (R36, R32 re-ingest).
7. **H2** Retriever.
8. **Owner-gated:** Eval (R44–R45), which needs 6 and 7. Built only on the owner's explicit
   word; the owner sets R30's final values from its results.
9. Step suggestions, server and client (R23, R25, R66–R68).
10. **H3** Ingest consumer.
11. Item-tree suggestion, server and client (R24, R69–R70). First to cut.

## Pre-existing defects this feature depends on

H3 consumes events through the insights consumer that I-0026 records as broken. On this branch:

1. The consumer is never started (`SetupServices` discards the AMQP channel).
2. `inboxService` is never injected, so the first inbox write panics.
3. The queue binding is wrong: it consumes plan-service's own queue and binds a queue it never
   declares.
4. The Redis claim fails closed: a Redis outage acks and drops every event.

These are recorded, not fixed, here. Whether H3 absorbs them or waits for I-0026 is open item O1.

Two further gaps, in the agent lane via R43: plan-service leaves `user_id` empty on completion
events and sets no `MessageId`, which the inbox needs as its key.

## Design notes: alternatives rejected, and why

- **Three sequential calls, each told to avoid the last** (today). Rejected: three times the
  output cost, and de-duplication by instruction rather than by schema.
- **Retrieving the plan's own state.** Rejected: plan state is small, must be complete and is
  authoritative; retrieval would return a partial and possibly stale view.
- **An ML classifier for "should we regenerate?".** Rejected: the input is structured, so a hash
  comparison decides exactly, at zero cost.
- **Sending dismissed suggestions to the model as negative examples.** Rejected: it costs input
  tokens on every call and primes the model toward what it should avoid. A post-filter is exact.
- **insights writing accepted items itself.** Rejected: insights owns no plan data, and
  plan-service's ordinary path already enforces every item rule.
- **Accepted new steps as daily items** (today's behavior). Rejected: they vanish into the nightly
  reset and never count toward a milestone.
- **1536 dimensions, a namespace or table per tenant, HNSW from day one.** Rejected for v1; the
  reasoning is the decision card's subject (§Ownership split) and ADR-0018's.
- **Thin events with a lookup back into plan-service.** Rejected: it couples ingest to
  plan-service availability and adds a call per completion (L16).
- **Message-text matching for new domain codes.** Rejected: the existing "name" substring rule
  would misfire on any insights message containing "name"; a structured detail cannot.
