---
id: I-DF4ZB-18
status: open
implements: FS-DF4ZB
blocked_by: [I-DF4ZB-1, I-DF4ZB-2]
labels: [feature]
title: "[HUMAN] FS-DF4ZB: H3, the personal-memory ingest consumer on checklist_item.completed / .uncompleted"
---

> **HUMAN-OWNED — do not run `/develop` on this issue.**
> Flagged in FS-DF4ZB §Ownership split (L15) as the owner's lane. An agent must not
> implement it; pick it up only if the owner hands it over explicitly.

Implements FS-DF4ZB §Requirements (R37–R42), §Pre-existing defects this feature depends on

## Explain before you code

Before writing code, write down, in your own words, the answer to each question below. The
positions the FS took are in §Decisions and why (L8, L10, L16); this card is your explanation of
them, not a copy. Only the questions that apply to ingest are listed.

1. **Why record the model and dimensions on every row? What does a re-embed migration look like?**

2. **What does `content_hash` buy?** (R32, R38: completing the same item again.)

3. **Where does the queue sit in ingest, and what does it protect?**

4. **Fat events versus thin events: what did this FS choose and why?**

## Owner decisions to make before starting

These are open in the FS and are **not** decided here.

- **O1. H3 vs I-0026.** H3 runs on the same insights consumer code that I-0026 (HUMAN-OWNED,
  FS-0006) lists as broken: never started, `inboxService` nil, wrong queue binding, Redis claim
  failing closed. Either H3 absorbs those four fixes, or H3 waits for I-0026. Note that I-0026's
  binding target (`plan.items_requested`) differs from H3's (`checklist_item.*`), so the queue
  topology has to serve both whichever way it goes. **Decide, and record the choice here, before
  writing code.** If H3 waits, add I-0026 to `blocked_by`.
- **O4. Personal row content.** The event carries `parent_id` but not the parent's description.
  The FS default is to embed the step alone; enriching it with the parent's title would need a
  lookup or a fatter event. **Your call; record it here.** A fatter event would reopen
  I-DF4ZB-2's proto change.

## What to Build

- **Consumer (R37).** An insights-owned consumer on `plan.events` for `checklist_item.completed`
  and `checklist_item.uncompleted`, on a queue insights declares itself.
- **Completed (R38).** Embed the item's description (I-DF4ZB-1's embedder) and upsert one
  personal row: `user_id`, `plan_id`, `source = completion`, `source_id` = item id,
  `chunk_no = 0`, `parent_id` as row metadata. Completing the same item again upserts the same row
  (unchanged text re-embeds nothing, R32).
- **Uncompleted (R39).** Delete that item's personal row. Deleting a missing row succeeds.
- **At most once (R40).** Through the inbox (`processed_events`, keyed by AMQP `MessageId`, which
  I-DF4ZB-2 now sets), in the same transaction as the row write.
- **Ordering (R41).** An event older than the last one applied for the same item never overrides
  it; a redelivered completion cannot resurrect a row a later un-completion removed.
- **Failure (R42).** Transient failure (embedding API, database) requeues; a malformed message
  (including one missing `user_id` or `MessageId` from a pre-R43 publisher) is not requeued and is
  logged. Ingest failing never affects plan-service or the user's tick.
- **Start it.** The consumer starts with the service (`config/services.go` currently discards the
  AMQP channel; see §Pre-existing defects and O1).

## Acceptance Criteria

- [ ] The decision card above is answered in the owner's words; O1 and O4 are decided and
      recorded on this issue.
- [ ] Completing an item creates one personal row; completing it again does not add a second.
- [ ] Un-completing deletes it; un-completing an item with no row succeeds.
- [ ] Replaying either event changes nothing; an older event never overrides a newer one.
- [ ] An event missing `user_id` or `MessageId` is not requeued and is logged.
- [ ] Stopping the embedding API makes ingest requeue; ticking items keeps working.
- [ ] The consumer consumes a queue insights declares, and starts with the service.
- [ ] Tests pass.

## Blocked By

- I-DF4ZB-1 — `rag_chunks`, the embedder and the repository.
- I-DF4ZB-2 — events that carry `description`, `parent_id`, `user_id` and a `MessageId`.
- Possibly I-0026, depending on O1.

No agent slice is blocked by this issue.

## Expected write set

- `services/insights-service/internal/memory/` (new, proposed: consumer + handler + tests) or
  `services/insights-service/internal/insights/amqp_consumer.go`, depending on O1
- `services/insights-service/internal/inbox/{service.go,repository.go}`
- `services/insights-service/internal/rag/repository.go` (ordering guard, if it lives there)
- `services/insights-service/config/services.go` **(hotspot)**, `cmd/server/main.go`
- `services/insights-service/migrations/00000N_*` (if the R41 ordering guard needs a column)
- `common/constants/events.go` (queue name, if centralized there)

## Spec Reference

FS-DF4ZB §Requirements R37–R42; §Decisions L10, L15, L16; §Ownership split (H3, the decision
card); §Open items O1, O4; §Pre-existing defects this feature depends on; §Edge States ("Item
completed then uncompleted quickly", "Duplicate delivery", "Event missing `user_id` or
`MessageId`", "Daily item completed every day", "Bulk nightly reset", "Embedding API down during
ingest").
