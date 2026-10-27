---
id: I-DF4ZB-2
status: open
implements: FS-DF4ZB
blocked_by: []
labels: [feature]
title: "FS-DF4ZB slice 2: self-describing completion events (description, parent_id, user_id, MessageId)"
---
Implements FS-DF4ZB §Requirements (R43), §API surface (plane 2, events)

## What to Build

Make `checklist_item.completed` / `.uncompleted` carry everything H3's consumer needs, so ingest
never calls back into plan-service (L16).

- `events.ChecklistItemCompletedEvent` and `ChecklistItemUncompletedEvent` in
  `common/api/proto/events/plan.proto` gain `description` and `optional parent_id`, additively
  (new field numbers, nothing renumbered).
- Regenerate with `make -C common` (protoc). Generated `*.pb.go` is never hand-edited.
- plan-service's publisher (`services/plan-service/internal/checklistitem/publisher.go`)
  populates `description`, `parent_id` (when the item has one), and the existing `user_id`
  (the user who flipped `done`, which is empty today), and sets a fresh UUID AMQP `MessageId` on
  every publish.
- No plan-service table, migration or RPC change. The bulk daily reset still publishes nothing.

## Acceptance Criteria

- [ ] Completed and uncompleted events carry `description`, `parent_id` (when set), a populated
      `user_id` and a non-empty, per-publish-unique `MessageId`.
- [ ] A top-level item's event has no `parent_id`.
- [ ] The proto change is additive (no renumbered or removed field).
- [ ] plan-service has no migration and no RPC change.
- [ ] Generated code was regenerated, not edited.
- [ ] Tests pass.

## Blocked By

None.

## Expected write set

- `common/api/proto/events/plan.proto` and regenerated `common/api/proto/events/plan.pb.go`
- `services/plan-service/internal/checklistitem/publisher.go` (+ test)
- `services/plan-service/internal/checklistitem/service.go` (only to pass description, parent and
  acting user to the publisher)

Watch: `make -C common` regenerates **every** proto package in one run, including
`common/api/proto/insights` **(hotspot)**. If this slice runs alongside I-DF4ZB-4 (or any slice
editing `insights.proto`), commit only the files this slice meant to change and regenerate after
rebasing.

## Spec Reference

FS-DF4ZB §Requirements R43; §Decisions L16; §Pre-existing defects ("Two further gaps"); §API
surface (plane 2: events gain `description` and `optional parent_id`).

## TDD Approach

- RED: a publisher test with a capturing channel asserting the published message's `MessageId`
  is a UUID and the decoded event carries `description`, `parent_id` and `user_id`.
- GREEN: proto fields, regenerate, populate.
- Then: two publishes produce different `MessageId`s; a top-level item omits `parent_id`.
