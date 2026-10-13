---
id: I-043WP-1
status: done
implements: FS-043WP
blocked_by: []
labels: [feature]
title: "FS-043WP slice 1: checklist items carry an optional intention, end to end"
---
Implements FS-043WP §Requirements (R37–R40), §API surface

## What to Build

Checklist items gain a nullable free-text field, `intention`. It is set through the existing
item PATCH and returned on every item response. This is FS-043WP's only server change, and it
holds the text the milestone page later shows as *why it matters*. The server attaches no
meaning to it and never says "milestone" (ADR-0014, FS-043WP D5). Follow the end-to-end path
that dates (000018) and `type`/`parent_id` (000019) took.

**Storage.** A plan-service migration in the next free number. That is 000021 at the time of
writing, but FS-12ZMF is being specced in parallel and may claim a number, so take whichever is
next when this lands. Add `ALTER TABLE checklist_items ADD COLUMN intention TEXT` plus
`CHECK (intention IS NULL OR char_length(intention) <= 280)`. Down drops both.

**plan-service (R38).** `Item` gains `Intention *string`, and every SELECT column list in
`repository.go` reads it. `UpdateItemInput` gains a three-state pair like `SetParent`/`ParentID`
(`SetIntention bool`, `Intention *string`), because a COALESCE cannot clear to NULL. The service
trims the value and turns an empty result into NULL. It rejects a trimmed value over 280
characters (runes, not bytes) with `commonconstants.ErrInvalidInput`, so the handler answers
`InvalidArgument`. Create is untouched.

**Proto.** `ChecklistItem` gains `optional string intention`. `UpdateItemRequest` gains
`optional string intention`: unset leaves it alone, set trims it, empty becomes NULL. Regenerate
with `make proto-gen`.

**Gateway (R39).** `UpdateChecklistReq` gains `Intention *string json:"intention,omitempty"`:
omit leaves it, a string sets it. Document this in prose with no `maxLength`, because the limit
is a domain rule (ADR-0005). `ChecklistResp` gains `Intention *string json:"intention,omitempty"`.
Map both directions in `client.go` (`UpdateChecklist`, `checklistFromProto`). Add the field's
semantics to the `updateChecklist` operation's description in `typed_checklists.go`.
`CreateChecklistReq` stays as it is, so `intention` at create is an unknown member and gets a 422.

**Contract and client (R40).** Regenerate `services/api-gateway/openapi.yaml`
(`make -C services/api-gateway openapi`) and `client/src/api/generated/schema.d.ts`
(`make -C services/api-gateway client`). Add `intention?: string | null` to the hand-written
`ChecklistItem` in `client/src/services/api.ts`. The gates (`make -C services/api-gateway gates`)
must pass with no breaking change. This slice has no UI.

Files expected: `services/plan-service/migrations/0000NN_add_intention_to_checklist_items.{up,down}.sql`,
`common/api/proto/plan/checklist.proto` (+ generated `checklist.pb.go`),
`services/plan-service/internal/checklistitem/{model,repository,service,handler}.go` and tests,
`services/api-gateway/internal/gateway/plan/{model,client,typed_checklists}.go` and tests,
`services/api-gateway/openapi.yaml`, `client/src/api/generated/schema.d.ts`,
`client/src/services/api.ts`.

## Acceptance Criteria

- [ ] Migration up and down apply cleanly; existing items read `intention` as null.
- [ ] PATCH with `"intention": "  ship it  "` stores and returns `"ship it"`.
- [ ] PATCH with `""` or whitespace clears it, and the response omits `intention`.
- [ ] PATCH without `intention` leaves it unchanged, including alongside other fields.
- [ ] A trimmed value of 281 characters is `400 · VALIDATION_FAILED`; 280 multi-byte characters
      are accepted.
- [ ] POST create with `intention` is `422 · VALIDATION_FAILED`.
- [ ] The list, archived, upcoming, get, update, dates, archive and reorder responses all carry
      `intention` when set.
- [ ] `openapi.yaml` and `schema.d.ts` are regenerated, the contract gates pass, and the diff is
      additive only.
- [ ] No server identifier, proto name or schema name says "milestone".
- [ ] Tests pass.

## Blocked By

None.

## Spec Reference

FS-043WP §Requirements R37–R40 (the `intention` field, charter default C5). §API surface (all
rows, plus the gRPC and storage notes). §Acceptance Criteria (the PATCH, create, response,
migration and gates items). §Edge States (whitespace save, over 280, value kept when an item
stops being a milestone). §Decisions D5.

## TDD Approach

- RED: a plan-service service test. Update with intention "  x  " stores "x"; "" stores NULL;
  unset leaves the old value; 281 runes returns ErrInvalidInput.
- GREEN: the model, the repository's three-state update, and the service's trim and limit.
- Then: the proto and handler mapping. Add a gateway transport test that PATCH forwards the field
  and the response carries `intention`, and that create rejects it with 422. Regenerate and run
  the gates.
