---
id: I-65B3N-2
status: done
implements: FS-65B3N
blocked_by: [I-65B3N-1]
labels: [feature]
title: "FS-65B3N slice 2: edit and delete comments, author and owner rights"
---
Implements FS-65B3N §Requirements (R2, R9–R10, R16–R19), §API surface (updateNoteComment, deleteNoteComment; GetNoteComment, UpdateNoteComment, DeleteNoteComment RPCs)

## What to Build

The author edits their comment (marked edited, no history); the author or the plan owner
deletes it; everyone else is refused 403.

**plan-service.**
- Proto: `GetNoteComment`, `UpdateNoteComment`, `DeleteNoteComment` on `ChecklistService`
  (§API surface). `make proto-gen`.
- All three scope plan → item → comment; any mismatch → `NotFound` (R2).
- Update: trim + 1–2000 code points (R9–R10). Same trimmed body as stored → return it unchanged,
  `edited_at` untouched (R16). Otherwise set body and `edited_at = now()`. Backstop: scoped by
  `author_id = user_id`; anyone else → `PermissionDenied` (R19).
- Delete: hard delete. Backstop: `author_id = user_id` or the plan's `user_id = user_id`;
  otherwise `PermissionDenied` (R19).

**Gateway.**
- `PATCH /api/plans/{id}/checklists/{checklist_id}/comments/{comment_id}` (strict
  `NoteCommentReq`) and `DELETE …/comments/{comment_id}` → 204.
- Flow: access seam (owner-or-member; stranger → 404) → `GetNoteComment` for `author_id` →
  slice 1's `commentRights` → 403 `FORBIDDEN` if not allowed → the write. The flags and the
  enforcement come from the one function (R18).
- Response of PATCH is `NoteCommentResp` with `authorName`, `canEdit`, `canDelete` filled as on
  list.
- `make openapi client`; `make gates`.

## Acceptance Criteria

- [ ] Author (owner or member) edits → 200, new body, `editedAt` set.
- [ ] Editing to the same trimmed text → 200, `editedAt` still null.
- [ ] Edit to empty or over 2000 after trim → 400 `VALIDATION_FAILED`, unchanged.
- [ ] Member edits another's → 403 `FORBIDDEN`; owner edits a member's → 403 `FORBIDDEN`.
- [ ] Author deletes own → 204; owner deletes a member's → 204; member deletes another's or the
      owner's → 403 `FORBIDDEN`.
- [ ] A comment of a deleted account: nobody can edit (403); the owner can delete (204).
- [ ] Comment id on a different note, or already deleted → 404 `NOT_FOUND`; stranger → 404;
      unauthenticated → 401.
- [ ] plan-service backstop: calling Update/Delete RPCs directly as a non-author (non-owner for
      delete) → `PermissionDenied`.
- [ ] An edit leaves the item's `description`, body, summary and `updated_at` unchanged.
- [ ] At least one 403 asserts `Content-Type: application/problem+json`.
- [ ] Contract and client regenerated together; `make gates` passes.
- [ ] Tests pass.

## Blocked By

I-65B3N-1

## Spec Reference

FS-65B3N §Requirements R2, R9–R10, R16–R19, R22–R23; §API surface (updateNoteComment,
deleteNoteComment, GetNoteComment, UpdateNoteComment, DeleteNoteComment); §Edge States (edit to
same text, author left, author deleted, comment deleted elsewhere). User stories 7–12, 13–14, 27.

## TDD Approach

- RED: gateway test table over (relation, caller is author) × (edit, delete) → expected
  status, through `commentRights`.
- GREEN: Get/Update/Delete RPCs and the two typed operations.
