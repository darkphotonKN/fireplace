---
id: I-DF4ZB-14
status: open
implements: FS-DF4ZB
blocked_by: [I-DF4ZB-1, I-DF4ZB-4]
labels: [feature]
title: "[HUMAN] FS-DF4ZB: H2, the Retriever (tenant-scoped pgvector retrieval behind the R29 seam)"
---

> **HUMAN-OWNED — do not run `/develop` on this issue.**
> Flagged in FS-DF4ZB §Ownership split (L15) as the owner's lane. An agent must not
> implement it; pick it up only if the owner hands it over explicitly.

Implements FS-DF4ZB §Requirements (R26–R30)

## Explain before you code

Before writing code, write down, in your own words, the answer to each question below. The
positions the FS took are in §Decisions and why (L8, L9) and ADR-0018; this card is your
explanation of them, not a copy. Only the questions that apply to the retriever are listed.

1. **Why 512 dimensions and not 1536?**

2. **Why record the model and dimensions on every row? What does a re-embed migration look like?**
   (R26 filters on them.)

3. **Why a `user_id` column filter, rather than a namespace per tenant or a table per tenant?**

4. **Why no HNSW yet? Why can an ANN index plus a filter return fewer than k?**

5. **Cosine versus dot product on normalized vectors: what differs, and what doesn't?**

6. **How are k and the chunk size chosen?**

7. **What is the weak-retrieval threshold for, and what happens below it?**

## What to Build

The real implementation of the retrieval seam that I-DF4ZB-4 defined (R29) and that the brief,
step and item-tree flows already call through a stub. Builds on I-DF4ZB-1's embedder and
`rag_chunks` repository.

- **Query text per generation kind.** What text is embedded as the query for a daily brief, a step
  suggestion (target parent item) and an item-tree suggestion.
- **Tenant-scoped filter (R26).** `user_id = $user OR user_id IS NULL`; never a row whose
  `plan_id` is the plan being suggested for; only rows whose `embedding_model` and
  `embedding_dims` equal the embedder's. Honour the shared-only scope the rule gate passes
  (I-DF4ZB-5) when the user has no personal rows.
- **k and the floor (R27).** At most k rows (k in 3–5, configurable) by cosine distance (`<=>`);
  drop every row below a configurable similarity floor; fewer than k, including zero, is valid
  and never padded. Exact scan (R33).
- **Embedding API down mid-request (S14).** Return zero snippets so generation proceeds on the
  weak-retrieval path; do not fail the request.
- **Tenant-isolation test (R28).** An automated test showing user A never retrieves user B's
  rows, for any query, including when B's rows are the nearest.
- **Wire it.** Replace the stub in `config/services.go` with the real retriever.
- **Values from the eval (R30).** Ship with provisional k, floor and chunk size. Once I-DF4ZB-15
  has run, set the chosen values and note them, with the run that chose them, beside the
  configuration. This issue closes after that note is in.

## Acceptance Criteria

- [ ] The decision card above is answered in the owner's words.
- [ ] Retrieval never returns a row from the current plan or another user (tenant-isolation test
      passes, including when the other user's rows are nearest).
- [ ] Rows with a different `embedding_model` or `embedding_dims` are ignored.
- [ ] Retrieval returns fewer than k when fewer clear the floor, and zero when none do; with
      zero, the prompt has no snippet block and forbids claims about past work.
- [ ] A user with no personal rows triggers no personal retrieval.
- [ ] With the embedding API down, a brief still generates, without snippets.
- [ ] Chosen k, floor and chunk size are noted beside their configuration, citing the eval run.
- [ ] Tests pass.

## Blocked By

- I-DF4ZB-1 — the embedder and `rag_chunks` repository.
- I-DF4ZB-4 — the R29 seam this implements (consumer-owned interface plus stub).

R30's final values depend on I-DF4ZB-15, which itself needs this retriever. That is not a
`blocked_by` (it would be a cycle): ship provisional values, run the eval, then set them.

## Expected write set

- `services/insights-service/internal/retrieval/` (new, proposed: the pgvector retriever + the
  tenant-isolation integration test)
- `services/insights-service/internal/rag/repository.go` (the similarity query, if it lives with
  the repository)
- `services/insights-service/config/services.go` **(hotspot)**: swap stub for real
- `services/insights-service/.env.example` (k, floor)

## Spec Reference

FS-DF4ZB §Requirements R26–R30 (plus R5 and S14 as the retriever's behaviour at zero); §Decisions
L8, L9, L15; §Ownership split (H2, the decision card); §Edge States ("`rag_chunks` empty", "New
user, no personal memory", "Embedding API down during a request that needs retrieval"); ADR-0018.
