---
id: I-0029
status: open
implements: FS-0006
blocked_by: [I-0022, I-0026]
labels: [feature]
title: [HUMAN] FS-0006: insights outbox, publish worker, and insight.generated
---

> **HUMAN-OWNED — do not run `/develop` on this issue.**
> Flagged in FS-0006 §Ownership split as the owner's lane. An agent must not
> implement it; pick it up only if the owner hands it over explicitly.

Implements FS-0006 §Requirements

## What to Build

The second hop's producer side. insights-service has `generated_insights` and `processed_events`
but **no outbox table** and publishes nothing today.

- Outbox table + repository in insights-service, mirroring plan-service's.
- A `commonworker.PublishWorker` instance for this service.
- **`insight.generated`** published on success, carrying the payload from I-0022 — the items
  themselves, so plan-service materializes without calling back and never depends on insights
  being reachable.
- Populate `generated_insights.content`, which is written empty today.
- The insight write and the outbox row commit in **one transaction** — that is what the outbox is
  for.

## Acceptance Criteria

- [ ] The insight row and its outbox row commit together or not at all.
- [ ] The published payload matches I-0022's proto exactly — no field the database owns.
- [ ] `parent_index` values are consistent with array order.
- [ ] `generated_insights.content` is no longer empty.
- [ ] Tests pass.

## Blocked By

I-0022 (payload proto), I-0026 (consumer must run before it can produce).

## Spec Reference

FS-0006 §Requirements R18, R19–R19d, R23
