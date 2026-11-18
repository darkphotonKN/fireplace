---
id: I-PGG2N-1
status: done
implements: FS-PGG2N
blocked_by: []
labels: [feature]
title: "FS-PGG2N slice 1: notes carry an optional summary, end to end"
---
Implements FS-PGG2N §Requirements (R1–R9), §API surface

## What to Build

Checklist items gain a nullable `summary`, which only notes may carry. It is set through the
existing item PATCH and returned on every item response. The body stays `description`, with no
new body field (R1). Follow the path `intention` took (I-043WP-1), plus one rule `intention`
doesn't have: a summary belongs to notes only.

**Storage (R9).** One plan-service migration in the next free number. FS-65B3N adds a
plan-service migration in parallel, so take whichever number is next when this lands. It adds
`summary TEXT NULL`, `CHECK (summary IS NULL OR char_length(summary) <= 280)` and
`CHECK (type = 'note' OR summary IS NULL)`, the twin of `chk_note_not_done`. Down drops both
constraints and the column.

**plan-service.**
- `Item` gains `Summary *string`, and every SELECT column list reads it.
- `UpdateItemInput` gains `SetSummary bool` and `Summary *string`, three-state like
  `SetIntention`.
- The service trims the value and turns an empty result into NULL. A trimmed value over 280
  runes is refused with `ErrInvalidInput` (R2).
- A non-empty summary is refused with `ErrInvalidInput` when the item is a task and the update
  doesn't make it a note, or when the update makes it a task (R3). An empty summary on a task
  is accepted and changes nothing.
- An update that sets `type = task` on a note also sets the summary to NULL in the same write,
  the way task→note forces `done = false` (R6).
- `Duplicate` copies `summary` with `description` (R7).
- Create is untouched (R5).

**Proto.** `ChecklistItem` gains `optional string summary = 15`. `UpdateItemRequest` gains
`optional string summary = 12`. Regenerate with `make proto-gen`.

**Gateway.**
- `UpdateChecklistReq` gains `Summary *string json:"summary,omitempty"`, documented in prose with
  no `maxLength` (ADR-0005). `ChecklistResp` gains `Summary *string json:"summary,omitempty"`.
- Map both directions in `client.go`.
- Describe the field's semantics in the `updateChecklist` and `duplicateChecklist` operation
  descriptions.
- `CreateChecklistReq` is unchanged, so `summary` at create is an unknown member and answers 422.

**Contract and client.** Regenerate `openapi.yaml` and `client/src/api/generated/schema.d.ts`
(`make openapi client` in `services/api-gateway`). Add `summary?: string | null` to the
hand-written `ChecklistItem` in `client/src/services/api.ts`, and let `updateChecklistItem` in
`client/src/api/checklists.ts` send it. `make gates` must pass with no breaking change. This slice
has no UI.

Files expected:
- `services/plan-service/migrations/0000NN_add_summary_to_checklist_items.{up,down}.sql`
- `common/api/proto/plan/checklist.proto` and the generated `checklist.pb.go`
- `services/plan-service/internal/checklistitem/{model,repository,service,handler}.go` and tests
- `services/api-gateway/internal/gateway/plan/{model,client,typed_checklists}.go` and tests
- `services/api-gateway/openapi.yaml`, `client/src/api/generated/schema.d.ts`
- `client/src/services/api.ts`, `client/src/api/checklists.ts`

## Acceptance Criteria

- [ ] Migration up and down apply cleanly. Existing items read `summary` as null.
- [ ] The database refuses a summary on a task and one over 280 characters.
- [ ] PATCH a note with `"summary": "  gist  "` stores and returns `"gist"`.
- [ ] PATCH with `""` or whitespace clears it, and the response omits `summary`.
- [ ] PATCH without `summary` leaves it unchanged, including alongside other fields.
- [ ] A trimmed 281-character summary is `400 · VALIDATION_FAILED`. 280 multi-byte characters are
      accepted.
- [ ] A non-empty `summary` on a task, or with `type: "task"` in the same body, is
      `400 · VALIDATION_FAILED`. An empty `summary` on a task succeeds and changes nothing.
- [ ] `type: "note"` plus a `summary` on a task in one body succeeds.
- [ ] PATCH `type: "task"` on a note with a summary returns no `summary`, and the column is NULL.
- [ ] Duplicating a note copies its `summary`.
- [ ] POST create with `summary` is `422 · VALIDATION_FAILED`.
- [ ] Every item response (list, archived, upcoming, get, create, update, duplicate, dates,
      archive, reorder) carries `summary` when set.
- [ ] `openapi.yaml` and `schema.d.ts` are regenerated, gates pass, and the diff is additive only.
- [ ] Tests pass.

## Blocked By

None.

## Spec Reference

FS-PGG2N:
- §Requirements R1–R9 (charter defaults C1, C2, C8, C15, C16).
- §API surface: the checklist rows, plus the plan gRPC and storage notes.
- §Acceptance Criteria: the PATCH, create, response, duplicate, migration and gates items.
- §Edge States: summary at create, summary on a task, Make a task, duplicate.

## TDD Approach

- RED: a plan-service service test. A note updated with "  x  " stores "x", "" stores NULL, and
  unset keeps the old value. 281 runes is ErrInvalidInput. A non-empty summary on a task is
  ErrInvalidInput. type=task on a note clears it.
- GREEN: the model, the repository's three-state update, and the service rules.
- Then: the repository duplicate test copies summary. Add a gateway transport test (PATCH
  forwards, the response carries it, create rejects it with 422). Regenerate and run the gates.
