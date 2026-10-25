---
id: I-DF4ZB-11
status: open
implements: FS-DF4ZB
blocked_by: [I-DF4ZB-10]
labels: [feature]
title: "FS-DF4ZB slice 11: the dismissal filter on brief output"
---
Implements FS-DF4ZB §Requirements (R49, R50)

## What to Build

Dismissed ideas stop coming back, without ever entering a prompt (L11).

- **Post-filter (R49).** After validation (R10–R12) and before storing, drop:
  - a `new_step` move whose text has cosine similarity at or above a configurable threshold to
    any suggestion this user dismissed **on this plan**;
  - an `existing_step` move whose `step_item_id` this user dismissed as a move on this plan (by
    id, no embedding).
  Same plan, same user, no expiry (S10). Dropped is dropped: no regeneration (R12).
- **Rule gate (R50).** When the user has no dismissals on the plan, the filter does nothing and
  makes no embedding call.
- **Never in the prompt.** Dismissed text is read only by the filter.
- Build the filter as a small unit the step-suggestion (I-DF4ZB-16) and item-tree (I-DF4ZB-19)
  flows reuse for their own output (step text, item-tree title).

No contract change.

## Acceptance Criteria

- [ ] After dismissing a suggestion, a near-identical new-step move is filtered out of later
      briefs on that plan.
- [ ] An existing-step move whose step was dismissed as a move on the plan is dropped by id.
- [ ] A dismissal on another plan does not filter this plan's output.
- [ ] With no dismissals on the plan, no output embedding call is made.
- [ ] The dismissed text never appears in any prompt (assert on the assembled prompt).
- [ ] If the filter removes every move, the brief is stored with zero moves and nothing is
      regenerated.
- [ ] The threshold is configurable.
- [ ] Tests pass.

## Blocked By

I-DF4ZB-10 — dismissals and their stored embeddings.

## Expected write set

- `services/insights-service/internal/suggestion/` (filter + test)
- `services/insights-service/internal/guided/` (call the filter in the brief flow)
- `services/insights-service/config/services.go` **(hotspot)**, `.env.example` (threshold)

## Spec Reference

FS-DF4ZB §Requirements R49, R50; §Decisions L11; §Spec defaults S10; §Edge States ("Dismissal
filter removes every move"); §Design notes ("Sending dismissed suggestions to the model as
negative examples").

## TDD Approach

- RED: a filter table test with a fake embedder returning fixed vectors: near-duplicate dropped,
  distinct kept, dismissed step id dropped, other plan's dismissal ignored.
- GREEN: the filter.
- Then: with zero dismissals the fake embedder records no calls; a brief-flow test asserting the
  prompt contains no dismissed text.
