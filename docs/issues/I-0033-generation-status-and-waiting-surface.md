---
id: I-0033
status: open
implements: FS-0006
blocked_by: [I-0023, I-0032]
labels: [feature]
title: FS-0006 slice 9: generation status endpoint and waiting surface
---

Implements FS-0006 §Requirements, §API surface

## What to Build

**`GET /api/plans/{id}/generation`** → `{status, failureClass?, itemCount, requestedAt}`.

A **dedicated lightweight endpoint**, deliberately not a full plan GET: `plan_draft` runs to 20,000
characters and cannot change while generating, so polling the plan would re-ship it every few
seconds to learn one enum. SSE later pushes this same payload, so the client swaps transport
without touching its state handling — polling remains the degradation path.

**Client:**
- Waiting state after creation; the plan is usable immediately and generation never gates it.
- Polling **2s for the first 30s, then 5s to 2 minutes, then 10s**. The happy path resolves in
  ~20s, so the later tiers exist for the retrying case, not the normal one.
- Stops when status leaves `generating`, or at the **sweeper threshold (10 min)** — client give-up
  and server sweeper agree rather than disagreeing visibly.
- **Pauses on tab blur; immediate poll on focus.**
- **Zero items reads as ready-and-empty, never as an error.**
- Failure copy driven by `failure_class`, and the retry affordance offered **only** for classes
  that can be retried. Copy rendered **only while the plan is in that status**, so a duplicate
  delivery cannot produce a duplicate effect.
- Plans-list entries gain a status indicator (name, plan type, `description`, status — never the
  draft).

## HITL

**Failure copy per class is not specified in the FS.** Decide the wording with the owner before
building, and remember the constraint from §Edge States: with no DLQ replay path, bug-class copy
**must not promise eventual recovery**, because that plan gets zero items permanently.

## Acceptance Criteria

- [ ] The endpoint returns the small payload and never the draft.
- [ ] Polling follows 2s/5s/10s and stops on both conditions.
- [ ] Blur pauses; focus resumes with an immediate poll.
- [ ] Zero items renders ready-and-empty.
- [ ] Retry is offered only for retryable classes; bug-class copy promises no recovery.
- [ ] Tests pass; gates pass.

## Blocked By

I-0023, I-0032.

## Spec Reference

FS-0006 §Requirements R27 (client half), R37–R38, R51–R55 · §API surface (`getPlanGeneration`)
