---
id: I-65B3N-3
status: done
implements: FS-65B3N
blocked_by: [I-65B3N-1]
labels: [feature]
title: "FS-65B3N slice 3: a note with comments can't become a task"
---
Implements FS-65B3N §Requirements (R4, R5 convert half, R31), §API surface (updateChecklist changed; `NOTE_HAS_COMMENTS`)

## What to Build

Converting a note that has comments to a task is refused with a code, and the client says why.

**plan-service.** In `checklistitem.service.Update`, beside the "notes can't be parents" check:
when `in.Type` is `task` and the current item is a note, check for any `note_comments` row on it
(`EXISTS`). If one exists → `FailedPrecondition` with
`commongrpc.WithCode(…, errcode.NoteHasComments, …)`, nothing written. The check and the type
write run in one transaction holding the item row `FOR UPDATE`, so a concurrent comment create
(slice 1 holds `FOR SHARE`) can't slip between them (R5). Add `NoteHasComments Code =
"NOTE_HAS_COMMENTS"` to `common/errcode`.

**Gateway.** No new code path (`CodeFor` reads the ErrorInfo detail). Add 400 ·
`NOTE_HAS_COMMENTS` to the `updateChecklist` operation's description, then `make openapi client`
and `make gates`.

**Client.** `toggleTodoType` in `client/src/components/Todo.tsx` already rolls back on failure.
When the error's `code` is `NOTE_HAS_COMMENTS`, show "This note has comments. Delete them to make
it a task." instead of the generic message. Branch on `code`, never `detail`. If the FS-PGG2N note
page offers its own Make a task by the time this lands, give it the same message.

## Acceptance Criteria

- [ ] `type: "task"` on a note with a comment → 400 `NOTE_HAS_COMMENTS`; the item is still a note.
- [ ] On a note with no comments it converts as before; task → note is unaffected.
- [ ] A concurrent add-comment and convert on one note never both succeed (a test that runs both
      against a real database, or one that proves the lock ordering).
- [ ] Make a task on such a note in the client rolls back and shows the R31 message.
- [ ] The client branches on `code` only.
- [ ] Contract regenerated; `make gates` passes.
- [ ] Tests pass.

## Blocked By

I-65B3N-1

## Spec Reference

FS-65B3N §Requirements R4, R5, R31; §API surface (updateChecklist changed, `NOTE_HAS_COMMENTS`);
§Edge States (Make a task on a note with comments; add while another converts). User story 18.

## TDD Approach

- RED: plan-service service test — update `type=task` on a note with one comment returns
  `NOTE_HAS_COMMENTS` and leaves the type.
- GREEN: the `EXISTS` check in the locked update.
