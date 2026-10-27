# CONTEXT.md — Insights (ubiquitous language)

> The shared vocabulary for this bounded context. One term, one meaning — used
> identically in conversation, code, and specs. Populated by the `domain-model` skill.

## Insights

**Insight** — an AI-generated productivity artifact for a plan: a single suggestion, a set of
daily focus items, or video recommendations. Generated and persisted here; the *data* derived
from it downstream is not this service's.

**Content Generator** — the pluggable LLM seam (`ContentGenerator.Generate(ctx, call, prompt) → text`, where `call` names the ledger kind, plan and user the usage is recorded under).
One generator per system prompt, so a prompt change is a wiring change rather than an edit to
shared code.

**Discovery** — the subsystem that turns generated search terms into concrete tutorial videos.

**Plan Context** — plan draft plus the plan's checklist items, fetched from plan-service over
gRPC to build a prompt. This service owns none of it. Older generators read it flattened to
description and scope; guided suggestions (FS-DF4ZB) read it as a **plan tree**: each item's id,
parent id, type, done and due date, so a suggestion can point at a real item.

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

## Guided suggestions (FS-DF4ZB)

**Daily Brief** — the set of 0–3 **moves** drawn for one user on one plan for one local date. Stored
here, served from storage for the rest of that date, and redrawn only by an explicit refresh
(capped per plan per date) or when the **rule gate** says the plan changed. *Retired by the
**Day Plan** (FS-02JBZ A2, R92):* its HTTP surface, RPCs and generation path are removed. Its
tables stay; its plan tree, plan-state hash, refresh counter, retrieval and ledger are the day
plan's machinery. New prose says day plan.

**Move** — one proposal inside a daily brief, anchored to one **parent item** (a top-level,
long-term task in the plan) with a one-line reason. Two kinds: `existing_step` points at an
open child item already under that parent; `new_step` proposes text for a child that does not
exist yet. A move is a proposal and writes nothing to plan-service. The client calls this
anchor a *milestone*; this service and the API never do (ADR-0014). Superseded for new work by
the day plan's **Entry**, whose `existing_item` kind is not limited to children.

**Parent Item** — the anchor of a move or of a step suggestion: a non-archived, open, top-level
task item in the plan's long-term scope. The API word for what the client calls a milestone.

**Step Suggestion** — one of 3–5 proposed child items for a named parent item, drawn on demand.

**Item-Tree Suggestion** — a proposed new parent item: a title plus about three child steps,
drawn on demand for a plan.

**Suggestion Outcome** — `accepted` or `dismissed`, recorded per suggestion by the client after
it has acted. Accepting is done by the client through plan-service's ordinary item endpoints;
the outcome only records it. A dismissed suggestion feeds the **dismissal filter**.

**Dismissal Filter** — a cosine post-filter that drops a newly generated suggestion when it is
too close to one the user dismissed on the same plan. Dismissed text is never sent to the model.

**Plan-State Hash** — a digest of the plan tree as a prompt would see it. Equal hashes mean the
model would see the same plan, which is what lets the **rule gate** skip a generation.

**Rule Gate** — the deterministic checks that run before any embedding, search or model call
and can short-circuit them: an unchanged plan-state hash serves the stored brief, or the stored
`ai` day plan (whose hash also covers plan type, focus, plan date, due revisions and today's
ticks; a stored rules plan never satisfies it); empty personal memory skips personal retrieval; a
plan with no parent items skips generation. Rules, not a learned classifier, because the input is
structured. Not the **rules plan**, which is a fallback draw, not a check.

**Generation** — one structured-output model call and its recorded token usage (input, cached
input, output) and model name. One user action costs at most one generation.

## Retrieval (FS-DF4ZB)

**RAG Chunk** — one embedded passage in `rag_chunks`, carrying its source, source id, chunk
number, content hash, embedding model and dimensions. A row with no `user_id` is a **shared
row**, readable by every user; a row with a `user_id` belongs to that user alone.

**Playbook** — the shared corpus of worked examples of breaking a goal into finishable pieces
and steps. Stored as shared rows; used as dynamic few-shot examples. Rules that always apply
live in the system prompt instead; the playbook holds examples, of which only a few are
relevant to any one plan.

**Personal Memory** — a user's own completed items from their other plans, embedded as rows
owned by that user. Written by the ingest consumer on completion, removed on un-completion.
Never retrieved for the plan being suggested for: plan state is injected, never retrieved.

**Embedder** — the single component that turns text into vectors for both ingest and query, so
the two cannot disagree about model or dimensions. Retrieval reads only rows whose model and
dimensions match the embedder's.

**Retriever** — the query side: given a user, a plan to exclude and query text, return up to k
chunks above the similarity floor, shared and the user's own only.

**Weak Retrieval** — every candidate fell below the similarity floor. Snippets are dropped
rather than padded, and the prompt is told not to claim history the user does not have.

## Day plan (FS-02JBZ)

**Day Plan** — up to six **entries** in up to three **groups**, drawn for one user on one plan for
one **plan date**, only when the user asks (**Plan my day**). Stored per (plan, user, date) beside
the daily brief, reusing its plan-state hash, validation, refresh counter and usage ledger. Its
`status` is `planned` or `not_planned`; a planned day plan with zero entries ("nothing to plan")
is a different answer from `not_planned` ("nobody asked yet"). A read returns what is stored or
`not_planned`, and never generates, never runs a rules draw and never copies another date's plan
forward (ADR-0019). A day plan is **not a Plan**: it belongs to one, and plan-service never sees
it. On a shared plan each member has their own. Not the brief renamed: the brief is 0–3 moves on
read, the day plan is ≤ 6 grouped entries on request.

**Plan my day** — the one operation that draws or redraws a day plan (`planMyDay`). The first
successful AI draw for a key is the **free draw** and does not count; every later successful AI
draw counts against the **refresh counter** (at most 3 counted draws per (plan, user, date); the
retired brief's refresh shared it while it existed). A draw that falls back to a rules plan, or an
empty plan with no candidates, neither counts nor uses up the free draw. "Draw" is the verb for
producing a day plan, AI or rules; "redraw" is any draw after the first that day.

**Plan Date** — the user's local calendar date (`YYYY-MM-DD`) sent by the client, and the date in
a day plan's key, its day ticks and its revision due dates. It must lie within one day of the
server's UTC date. Not the **budget day**, which is the server's UTC day: "today" in this section
means the plan date unless the token budget is the subject.

**Entry** — one thing to do on the plan date: either an existing item of the plan
(`existing_item`, any open long-term task, top-level or child) or a proposed new child of a parent
item (`new_step`). Carries a one-line reason, a rough minutes estimate when the model drew it, and
"revision N" when the server stamped one. Not an item, and not a **move**: a move's
`existing_step` only covered children, an entry's `existing_item` covers any task. An entry's id
doubles as a suggestion id for FS-DF4ZB's feedback.

**Binding** — what turns a `new_step` entry into an `existing_item` one: the user adds the step
through plan-service's ordinary create, then ticks the entry with the new item's id; the server
checks it is a task child of the entry's parent and rebinds the entry to it, keeping its text and
reason. Only bound or `existing_item` entries are tickable.

**Group** — a short label (1–40 characters) over related entries, drawn by the model by topic
affinity, or by rules (a "Revisions" group first, then one per top-level item). A label, not a
parent item: a group may span items under different parents, and has no id.

**Day Tick** — "done for today" on an entry, stored per (user, plan, item, plan date) with a
reference to its entry. Independent of the item's `done`, which plan-service owns and which means
the item is **finished**: a day tick never completes an item and completing an item never ticks or
unticks. Idempotent. Unticking on the same date removes the tick and whatever it did to the
revision ladder, leaving no trace. Ticked entries are **pinned**: they keep their place, text,
group and tick through every redraw that date, and the model fills only the remaining slots.

**Revision** — a due return of a topic on a **learning** plan, labelled "revision N" (1–3). Due
when its due date is on or before the plan date and it is not yet completed; it stays due
(overdue) until ticked, and is offered **even when the item is done**, but not when it is archived
or deleted. Decided and stamped on entries by the server, never by the model: the model only places
a due revision in the day. Not a version, edit or regeneration of anything.

**Revision Ladder** — the fixed rule that schedules revisions: an item's **first day tick ever**
(for this user, on this plan) schedules revision 1 at +1 day; ticking revision 1 schedules
revision 2 at +3 days from that tick; revision 2 schedules revision 3 at +7 days; revision 3 ends
the ladder. Chained, so a slipped revision pushes the later rungs. Never on project plans. The
**revisions switch** (`revisionsEnabled`, per user, plan and item, on by default) turns it off,
cancelling the open rung; turning it back on schedules nothing until the item's next day tick. The
ladder is the only recurrence left on the platform now that daily scope is retired.

**Rules Plan** — a day plan drawn deterministically, `source = rules`, when Plan my day's
generation fails (budget spent, provider unreachable or timing out, schema-invalid output).
Ranking: due revisions, then overdue, then due soonest, then the next open step of the most
recently updated parent item. Same shape as an AI plan, no minutes, reasons that name their rule.
Records nothing in the ledger, and never satisfies the rule gate. The client calls it a *basic
plan*; FS prose also says "rule-based plan". Not the **rule gate**, which decides whether to draw
at all.

**Usage Ledger** — one row per generation (kind, model, input / cached input / output tokens).
FS-DF4ZB introduced it; FS-02JBZ makes it load-bearing: every model call this service makes is
recorded, including the single checklist suggestion (`checklist_suggestion`) and video search
terms (`video_terms`). Embedding calls are not generations and are not recorded.

**Token Budget** — `LLM_DAILY_TOKEN_BUDGET`: the platform-wide cap on input + output tokens summed
from the usage ledger for the current **budget day** (server UTC). At or over it, no model call is
made and the generator fails with `LLM_BUDGET_EXHAUSTED` (429). A circuit breaker, not an invoice:
concurrent calls may overshoot slightly. Distinct from a generator's **output ceiling** (one call)
and the **refresh counter** (one key): the budget is the one bound across every generator and user.

**Vocabulary boundary.** Server names say item, parent item and top-level item, never
*milestone* (ADR-0014); only prompt prose may use the word. *Day plan*, *entry*, *day tick* and
*revision* are the same words in the UI (no translation); the client's *basic plan* is this
service's rules plan, and its *done for today* / *milestone done* contrast is day tick versus the
item's `done`.
