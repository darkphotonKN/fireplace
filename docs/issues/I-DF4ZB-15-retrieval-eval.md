---
id: I-DF4ZB-15
status: open
implements: FS-DF4ZB
blocked_by: [I-DF4ZB-12, I-DF4ZB-13, I-DF4ZB-14]
labels: [feature]
title: "FS-DF4ZB slice 15: the eval target, recall@k across chunk sizes and rule checks on generated output"
---

> **OWNER-GATED — run `/develop` only when the owner explicitly says so, never as part of a
> parallel or charter wave.** The owner owns the judgement: they read the results and set the
> final k, similarity floor and chunk size (R30).

Implements FS-DF4ZB §Requirements (R44, R45)

## What to Build

A **manual** make target (never run in CI) that measures what the retrieval and generation
numbers should be.

- **Recall@k (R44).** Over a labelled query set (queries → the playbook documents that should be
  retrieved), against the live OpenAI embeddings API and a pgvector database (the test instance),
  re-ingest the playbook at each of at least three chunk sizes (I-DF4ZB-12's chunker is
  parameterized) and report recall@k for k ∈ {3, 5} through H2's retriever (I-DF4ZB-14). Also
  report what the similarity floor would cut at each setting, so R30 can pick a floor.
- **Rule checks (R45).** Over fixture plans, run the daily-brief generator and report pass rates
  for: anchors valid, kinds valid, counts within bounds, no duplicates, no new step duplicating an
  existing child (R10–R13). Structure the checks so the step-suggestion and item-tree generators
  (I-DF4ZB-16, -19) can be added to the same report when they land.
- Output is a readable table on stdout. Hand the numbers to H2's R30 step; this slice does not set
  the retriever's configuration itself.

## Acceptance Criteria

- [ ] The eval target prints recall@k for ≥3 chunk sizes × k ∈ {3, 5}.
- [ ] It prints rule-check pass rates over fixture plans for the daily brief.
- [ ] It is not wired into CI or any default test target.
- [ ] It uses the test database, never production.
- [ ] Tests pass (unit tests for the recall@k and rule-check arithmetic; the live run itself is
      manual).

## Blocked By

- I-DF4ZB-12 — the parameterized chunker and ingest.
- I-DF4ZB-13 — the reviewed corpus being measured.
- I-DF4ZB-14 (H2) — the retriever being measured.

(The daily-brief generator from I-DF4ZB-4 is reached transitively through I-DF4ZB-14.)

## Expected write set

- `services/insights-service/cmd/eval/main.go` (new)
- `services/insights-service/internal/eval/` (new: recall@k, rule checks + tests)
- `services/insights-service/eval/queries.*` (labelled query set), `eval/fixtures/` (plan trees)
- `services/insights-service/Makefile` (`eval` target)

## Spec Reference

FS-DF4ZB §Requirements R44, R45 (feeds R30); §Decisions L17; §Acceptance Criteria ("The eval
target prints recall@k…"); §Out of Scope ("Running the eval in CI").

## TDD Approach

- RED: a recall@k unit test over a hand-built result set with known relevant ids.
- GREEN: the metric.
- Then: rule-check tests over fixture outputs with planted invalid anchors and duplicates.
