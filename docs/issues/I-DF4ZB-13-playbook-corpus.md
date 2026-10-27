---
id: I-DF4ZB-13
status: open
implements: FS-DF4ZB
blocked_by: []
labels: [feature]
title: "FS-DF4ZB slice 13: the playbook corpus, about 20 worked breakdowns (owner review before ingest)"
---
Implements FS-DF4ZB §Requirements (R35)

## What to Build

About 20 playbook documents, each a worked breakdown of a goal into milestones and steps, across
both plan types (`project`, `learning`). They are the dynamic few-shot examples retrieval
injects (L17). Rules that always apply belong in the system prompt (I-DF4ZB-4), not here.

- Agent-drafted. Stored in the repo beside the service (proposed:
  `services/insights-service/playbook/`), one file per document with a stable id the ingest
  command (I-DF4ZB-12) uses as `source_id`.
- Cover a spread: roughly half `project`, half `learning`; varied domains; each breaking one goal
  into a handful of milestones with a few concrete steps each, written the way a good step reads
  in this product (short, actionable).
- **Human-in-the-loop gate: the owner reviews the drafts before ingest.** This issue is not done
  until the owner has signed off; record the sign-off (who, when) in the corpus directory's
  README or in this issue. Do not run the ingest against a shared database before then.

Agent lane (§Ownership split: "the playbook drafts (owner reviews)"). Not `[HUMAN]`; the review is
an acceptance step, not ownership.

## Acceptance Criteria

- [ ] About 20 documents exist, across both plan types.
- [ ] Each has a stable id and is a goal → milestones → steps breakdown.
- [ ] The owner has reviewed them before ingest, and the sign-off is recorded.
- [ ] They contain no real user data.

## Blocked By

None. (Loading them needs I-DF4ZB-12; drafting does not.)

## Expected write set

- `services/insights-service/playbook/*.md` (new, ~20 files)
- `services/insights-service/playbook/README.md` (format, id convention, review sign-off)

## Spec Reference

FS-DF4ZB §Requirements R35; §Decisions L17; §Acceptance Criteria ("About 20 playbook documents
exist, reviewed by the owner before ingest"); §Ownership split (agent lane, owner reviews).
