---
id: I-65B3N-1
status: done
implements: FS-65B3N
blocked_by: []
labels: [feature]
title: "FS-65B3N slice 1: list and add comments on a note, plan-service to gateway"
---
Implements FS-65B3N §Requirements (R1–R3, R5 create half, R6–R9, R11–R15, R18 flags, R20–R21), §API surface (new resource: note comment; listNoteComments, createNoteComment; ListNoteComments, CreateNoteComment RPCs; `NOT_A_NOTE`)

## What to Build

Comments exist and can be read and added through the gateway, end to end. No UI yet (the note
page is FS-PGG2N's; slice 4 mounts the section).

**plan-service.**
- One migration (next free number in `services/plan-service/migrations/`; FS-PGG2N may also add
  one, so take whatever is free when you land) creating `note_comments` per R1: `id`, `item_id`
  FK → `checklist_items(id)` ON DELETE CASCADE, `author_id` (no FK), `body`, `created_at`,
  `edited_at` NULL, index `(item_id, created_at, id)`. Down drops it.
- Proto (`common/api/proto/plan/checklist.proto`): `NoteComment` message and, on
  `ChecklistService`, `ListNoteComments` and `CreateNoteComment` (shapes in §API surface).
  Regenerate with `make proto-gen`; never hand-edit the `.pb.go`.
- A comment module (suggested: `internal/notecomment`, with a narrow interface over the item
  lookup it needs) and handlers on the checklist gRPC server:
  - Both RPCs scope the item by `plan_id` (`GetInPlan`); item not in plan → `NotFound` (R2).
  - Item `type = task` → `FailedPrecondition` with `commongrpc.WithCode(…, errcode.NotANote, …)`
    (R3).
  - Create trims the body; empty or over 2000 code points (`utf8.RuneCountInString`) →
    `InvalidArgument` (R8–R9, as `normalizeIntention` does). `author_id` = the request's
    `user_id`, the caller as the gateway fills it (R6). The type check and the insert run in
    one transaction holding a lock on the item row (`FOR SHARE`), the create half of R5.
  - List: oldest first, `created_at` then `id`; `[]` when none (R11).
- Add `NotANote Code = "NOT_A_NOTE"` to `common/errcode` with a comment in the house style
  (additive).

**Gateway.**
- Extend `ChecklistsClient` (or a sibling narrow interface) and `client.go` with the two calls.
- New typed file (suggested `internal/gateway/plan/typed_note_comments.go`), registered in
  `config/api.go` beside the checklist operations:
  - `GET /api/plans/{id}/checklists/{checklist_id}/comments` and `POST …/comments` with a strict
    `NoteCommentReq { body }` (no length bounds in the schema, R9), owner-or-member via the
    access seam first (R14–R15).
  - `NoteCommentResp` per §API surface. `authorName` from the existing `People.PeopleByIDs` in
    one batch (display name, else account name); a missing account → `null` (R13).
  - One policy function, `commentRights(relation, caller, authorID) (canEdit, canDelete bool)`:
    edit = caller is author; delete = caller is author or relation is owner (R16–R18). Slice 2
    enforces with the same function.
- `make openapi client` in `services/api-gateway`; `make gates`.

## Acceptance Criteria

- [ ] Migration up creates the table, FK cascade and index; down drops it.
- [ ] Owner and member can each add a comment on a note → 201 with trimmed body, `authorId` =
      caller, `editedAt: null`, the caller's name.
- [ ] Sending `authorId` (or any unknown member) in the body → 422 `VALIDATION_FAILED`; missing
      or non-string `body` → 422.
- [ ] `""`, `"  "`, and 2001 code points after trimming → 400 `VALIDATION_FAILED`, nothing stored;
      exactly 2000 (with multi-byte characters) is stored.
- [ ] List and add on a task (including a milestone) → 400 `NOT_A_NOTE`.
- [ ] Item id not in the path plan → 404 `NOT_FOUND`; stranger → 404 on both routes;
      unauthenticated → 401.
- [ ] List is oldest first (ties by id) and `[]` for a note with none.
- [ ] `authorName` is the display name, else account name, and `null` for an author whose account
      is gone; names come from one batch lookup.
- [ ] `canEdit` true only on the caller's own; `canDelete` true on own, and on all for the owner.
- [ ] Deleting the note removes its comments; duplicating it (FS-F6AEE) copies none; archiving
      leaves them; adding a comment does not change the item's `description` or `updated_at`.
- [ ] At least one 400 asserts `Content-Type: application/problem+json`.
- [ ] `openapi.yaml`, `client/src/api/generated/` and the proto Go are regenerated and committed
      together; `make gates` passes.
- [ ] Tests pass.

## Blocked By

None

## Spec Reference

FS-65B3N §Requirements R1–R3, R5 (create), R6–R9, R11–R15, R18 (flags), R20–R21; §API surface
(listNoteComments, createNoteComment, `NoteCommentResp`, `NoteCommentReq`, ListNoteComments,
CreateNoteComment); §Edge States (empty, whitespace, over-length, 2000 exactly, comment on a task,
deleted author, duplicate, archive, cascade, stranger). User stories 1–6, 13–17, 19–20, 25–27.

## TDD Approach

- RED: plan-service service test — create on a note stores the trimmed body with author = caller;
  create on a task returns `NOT_A_NOTE`.
- GREEN: migration, repository insert/list, service rules, handler.
- RED: gateway typed test — member POSTs → 201 with `authorName`; stranger → 404; `authorId` in
  body → 422.
- GREEN: typed operations, People batch, `commentRights`.
