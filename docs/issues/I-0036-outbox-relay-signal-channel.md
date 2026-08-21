---
id: I-0036
status: open
implements: FS-0006
blocked_by: []
labels: [enhancement]
title: [HUMAN] FS-0006: outbox relay signal channel
---

> **HUMAN-OWNED — do not run `/develop` on this issue.**
> Flagged in FS-0006 §Ownership split as the owner's lane. An agent must not
> implement it; pick it up only if the owner hands it over explicitly.

Implements FS-0006 §Requirements

## What to Build

The relay gains an **in-process signal channel** alongside its existing ticker, so pickup is
immediate on the happy path instead of waiting out the interval. Changes `common/worker`, which has
no spec of its own — a behaviour change there belongs to the FS driving it.

**The ticker remains the correctness guarantee.** It runs unconditionally — not as a fallback armed
by a failed signal. The signal is purely a latency optimization and must always be safely
droppable, because it is silently absent in cases undetectable at the send site: process death
between commit and notify, a full channel, a row written by another process or by hand, or a new
code path that forgets to notify. None produces an error, so the relay can never know a signal was
owed and never arrived. If the signal becomes the delivery mechanism rather than a hint, a dropped
signal strands a row — reintroducing the dual-write problem the outbox exists to eliminate.

- **`chan struct{}`, buffer 1.** *Not unbuffered:* an unbuffered send succeeds only if a receiver
  is blocked at that instant, so a notify arriving mid-drain would be dropped — the case where the
  signal matters most. *Not larger:* the signal carries no information; ten notifies mean what one
  means. Buffer 1 is a one-slot latch meaning "work arrived while you weren't looking."
- **Notify is non-blocking** — `select` with a `default` that drops. Never blocks the request path,
  never returns an error. A full channel means a wake-up is already pending.
- **The loop selects over signal, ticker, and context cancellation.** All wake paths call the same
  drain; the drain does not know why it woke.
- **Notify fires after commit**, never inside the transaction — inside, the relay could wake, query
  before the commit is visible, find nothing, and sleep again.
- **Call sites, exhaustively:** (a) after every commit that wrote outbox rows; (b) relay startup,
  before entering the loop; (c) optionally after a full-batch drain. Nothing else. In particular
  the failure-event publish (I-0030) writes no outbox row and has **no** notify call site — adding
  one for symmetry would reintroduce the dual write.
- **Make the write and the notify hard to separate** — one repository helper doing both, not two
  things a caller must remember to pair. A path that forgets degrades silently to ticker latency.
- **Interval → 10s**, replacing `time.Minute * 2` (`plan-service/config/services.go:40`). A
  *loosening* against the ~2s a ticker-only relay would need.
- **Observability:** count rows found by ticker-triggered drains versus signal-triggered ones.
  Near-zero on the ticker path is healthy; a rising count means notifies are being missed.

## Acceptance Criteria

- [ ] Notify against a full channel does not block.
- [ ] A drain with no pending rows is a no-op.
- [ ] **An outbox row written with no notify at all is still published on the next tick** — the
      test that proves the signal is an optimization and not a dependency.
- [ ] The ticker fires regardless of signal activity.
- [ ] Notify happens after commit: a relay woken by the signal always finds the row.
- [ ] Shutdown drains in-flight work before returning.
- [ ] Tests pass.

## Blocked By

None.

## Spec Reference

FS-0006 §Requirements R60–R70
