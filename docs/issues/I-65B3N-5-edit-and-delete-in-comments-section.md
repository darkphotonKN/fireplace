---
id: I-65B3N-5
status: done
implements: FS-65B3N
blocked_by: [I-65B3N-2, I-65B3N-4]
labels: [feature]
title: "FS-65B3N slice 5: edit and delete in the note page's comments section"
---
Implements FS-65B3N §Requirements (R16–R17, R29–R30), §API surface (updateNoteComment, deleteNoteComment)

## What to Build

Edit and delete controls in `NoteComments`, shown only where the server says the caller may.

- API wrappers `updateNoteComment(planId, itemId, commentId, body)` and
  `deleteNoteComment(planId, itemId, commentId)` beside slice 4's, each recording the
  touched-today stamp on success.
- Per comment: Edit when `canEdit`, Delete when `canDelete` (never computed client-side). Shown
  on hover and keyboard focus on pointer devices, always on touch (R29).
- Edit: the body becomes a textarea prefilled with it, Save and Cancel; Escape cancels,
  Ctrl/Cmd+Enter saves; Save follows R28's enable rule. On success replace the comment with the
  response (shows " · edited").
- Delete: `DeleteConfirm` first, then remove from the list on 204.
- Errors by `code` (R30): `NOT_FOUND` → remove the comment with "This comment was deleted.";
  `FORBIDDEN` → reload the list; others → inline error, edit text kept.

## Acceptance Criteria

- [ ] Edit/Delete render only where `canEdit` / `canDelete` are true (test both flags both ways).
- [ ] Edit saves in place and shows " · edited"; Escape cancels and restores the body.
- [ ] Save disabled for empty / over 2000.
- [ ] Delete asks to confirm; confirming removes the comment; cancelling keeps it.
- [ ] `NOT_FOUND` on save or delete removes the comment with "This comment was deleted.";
      `FORBIDDEN` reloads the list.
- [ ] Controls are reachable by keyboard and visible on touch.
- [ ] `npm test`, `npm run lint` and `npm run build` pass in `client/`.

## Blocked By

I-65B3N-2, I-65B3N-4

## Spec Reference

FS-65B3N §Requirements R16–R17, R29–R30; §API surface (updateNoteComment, deleteNoteComment);
§Edge States (edit to same text, comment deleted while editing, author deleted). User stories
7–12.

## TDD Approach

- RED: component test — a comment with `canEdit: true` edits to new text and shows " · edited";
  one with both flags false shows no controls.
- GREEN: controls, edit mode, delete confirm, code-based error handling.
