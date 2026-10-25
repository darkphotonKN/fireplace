---
id: I-DF4ZB-12
status: open
implements: FS-DF4ZB
blocked_by: [I-DF4ZB-1]
labels: [feature]
title: "[HUMAN] FS-DF4ZB: H4, playbook chunker and batch ingest command"
---

> **HUMAN-OWNED — do not run `/develop` on this issue.**
> Flagged in FS-DF4ZB §Ownership split (H4) as the owner's lane. An agent must not
> implement it; pick it up only if the owner hands it over explicitly.

Implements FS-DF4ZB §Requirements (R36, R32)

## Explain before you code

Before writing code, write down, in your own words, the answer to each question below. The
positions the FS took are in §Decisions and why (L8, L10, L17); this card is your explanation of
them, not a copy.

1. **How are chunk size and overlap chosen?**

2. **Why aren't steps chunked (one personal row is one chunk) while playbook documents are?**

3. **What does `content_hash` buy on re-ingest?**

4. **Why is this ingest batch and synchronous while personal memory is asynchronous?**

## What to Build

A batch command (and make target) that loads the playbook corpus into `rag_chunks` as shared rows.

- **Chunker.** Fixed window with overlap; window and overlap are parameters (the eval,
  I-DF4ZB-15, sweeps at least three chunk sizes, so the size must not be a constant baked into
  the command).
- **Ingest.** For each document in the corpus directory: chunk, embed through I-DF4ZB-1's
  embedder, upsert shared rows (`user_id NULL`, `plan_id NULL`, `source = playbook`,
  `source_id` = the document's stable id, `chunk_no`).
- **Idempotent (R32).** Re-running with no document changes embeds nothing. Editing one document
  re-embeds only its changed chunks; if its chunk count shrank, its surplus rows are removed.
  The `content_hash` compare-and-skip lives **here**: I-DF4ZB-1's repository upsert is a plain
  keyed upsert that stores `content_hash` but never compares it.
- **Removal (S7).** A document no longer in the corpus has its rows deleted on the next run.
- Prints a summary (documents, chunks embedded, skipped, deleted).
- Developed and tested against fixture documents; the real corpus is I-DF4ZB-13.

## Acceptance Criteria

- [ ] Re-ingesting an unchanged corpus embeds nothing.
- [ ] Editing one document re-embeds only its chunks.
- [ ] Removing a document deletes its rows.
- [ ] All rows are shared (`user_id` and `plan_id` NULL) with `source = playbook`.
- [ ] Window and overlap are configurable.
- [ ] A make target runs it.
- [ ] Tests pass.

## Blocked By

I-DF4ZB-1 — `rag_chunks`, the embedder and the repository.

## Expected write set

- `services/insights-service/cmd/ingest-playbook/main.go` (new)
- `services/insights-service/internal/rag/chunker.go` (+ test)
- `services/insights-service/internal/playbook/` (new, proposed: ingest orchestration + test)
- `services/insights-service/internal/rag/testdata/` (fixture documents)
- `services/insights-service/Makefile` (ingest target)

## Spec Reference

FS-DF4ZB §Requirements R36, R32; §Decisions L17; §Spec defaults S7; §Acceptance Criteria
("Re-ingesting an unchanged playbook embeds nothing; editing one document re-embeds only its
chunks; removing one deletes its rows").

## TDD Approach

- RED: a chunker table test (length, overlap, last-window handling, empty doc).
- GREEN: the chunker.
- Then: an ingest test with a counting fake embedder over three fixture docs: second run 0 calls;
  edit one → only its chunks; remove one → its rows gone.
