---
id: I-0034
status: open
implements: FS-0006
blocked_by: [I-0023, I-0031]
labels: [feature]
title: FS-0006 slice 10: sweeper for stuck generations
---

Implements FS-0006 §Requirements

## What to Build

The backstop for failures that never report themselves. A panic or OOM runs **no error handler at
all** — no failure publish, no DLQ decision, just redelivery — so something has to notice a plan
stuck in `generating`.

- Periodic scan in plan-service; guarded transition `generating → failed` with a `failure_class`
  marking it swept rather than reported.
- **The threshold is computed, never a literal**: the exhausted-retry window (≈7 min — the
  10/20/40/80/160 ladder plus generation time) **plus one ticker interval per hop**, plus margin —
  **10 minutes**. The relay signal is droppable, so the worst case a healthy plan can legitimately
  take is bounded by the ticker, not the signal. A threshold derived from the retry schedule alone
  would fail plans that are merely queued.
- In practice this only ever catches crashes: a well-behaved exhaustion publishes its failure at
  ≈7 minutes and gets there first.

## Acceptance Criteria

- [ ] A plan whose consumer is killed mid-generation is eventually failed by the sweeper.
- [ ] **The threshold is derived from the retry ladder and ticker interval in code, not hardcoded**
      — changing the ladder changes the threshold with no second edit.
- [ ] A failure event arriving after the sweeper already failed the plan updates zero rows; first
      writer wins and both agree on the outcome.
- [ ] The sweeper never fails a plan that is still inside its legitimate window.
- [ ] Tests pass.

## Blocked By

I-0023, I-0031.

## Spec Reference

FS-0006 §Requirements R31 · §Edge States (consumer panics mid-generation)

## TDD Approach

- RED: a plan at threshold-minus-one-second is untouched; at threshold-plus-one it is failed.
- GREEN: derived threshold + guarded update.
