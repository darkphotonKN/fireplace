---
id: I-0022
status: open
implements: FS-0006
blocked_by: []
labels: [enhancement]
title: FS-0006 slice 2: event vocabulary and correlation envelope
---

Implements FS-0006 §Requirements

## What to Build

The shared contracts both lanes need before either chain hop can be written.

- **`common/constants/events.go`**: `InsightsEventsExchange = "insights.events"`;
  routing keys `insight.generated`, `insight.generation_failed`, `plan.items_requested`;
  queue constant `plan-service.insight.generated`. Note the file's own convention is
  `{resource}.{action}` **singular** — `insight`, not `insights`.
- **`common/broker`**: `Message` gains a headers field so `correlation_id` (whole chain) and
  `causation_id` (immediate parent) travel at the envelope level. They are envelope concerns,
  not domain fields, so no event proto grows two columns for them.
- **`common/api/proto`**: the `insight.generated` payload — `{plan_id, user_id, items[]}` where
  an item is the **creatable subset** of a checklist item: `description`, `scope`, `type`,
  optional `parent_index`. No `id`, `plan_id`, `status`, `done` or timestamps — the database
  owns those. Nesting is by **index into the same array**, because no row IDs exist before the
  materialization transaction opens. **Array order is `sequence`** — an explicit sequence field
  could disagree with the order; order cannot. **No dates**: AI scheduling is an unshipped
  plan-service capability.

## Acceptance Criteria

- [ ] Constants follow the existing singular `{resource}.{action}` convention.
- [ ] A message published with correlation/causation headers round-trips them intact.
- [ ] The payload proto has no field the database owns.
- [ ] A payload whose `parent_index` points at an item that itself has a `parent_index` is
      rejectable — two-tier nesting still applies.
- [ ] Tests pass.

## Blocked By

None.

## Spec Reference

FS-0006 §Requirements R15–R17, R19a–d, R20, R24

## TDD Approach

- RED: round-trip test asserting headers survive publish → consume.
- GREEN: headers field on `commonbroker.Message` plus the constants and proto.
