# FS-65B3N: Comments on a note from plan members

> Status: work-order · SPECIFICATION.md: client `## Notes` "Comments on a note from plan members"; plan-service `### Nested Items + Notes` "Authored comments on a note" → this FS · Related FS: FS-PGG2N (long-form notes: the note page these live on, and its look), FS-12ZMF (plan sharing: owners and members), FS-F6AEE (duplicate an item), FS-0001 (gRPC caller identity, not yet realized) · Related ADRs: docs/adr/0001-grpc-identity-via-metadata-rs256.md, docs/adr/0004-error-representation-rfc9457.md, docs/adr/0005-request-validation-and-contract-design.md, docs/adr/0009-consolidate-to-gateway-plan-insights.md, docs/adr/0010-three-databases-one-postgres-instance.md, docs/adr/0014-milestones-are-a-client-reading-of-items.md, docs/adr/0015-plan-access-is-a-relation.md, docs/adr/0019-generation-is-explicit-and-bounded.md

## Summary

A note gets **comments**: dated, authored entries under its body on the note page (FS-PGG2N).
They are how someone adds to a note over time (a later thought, a source, a reply on a shared
plan) without rewriting the note itself. Anyone who can reach the plan can read and add
comments; the author can edit their own (it then shows *edited*) or delete it, and the plan's
owner can delete anyone's. Comments are plain text with links made clickable, listed oldest
first with the box to add one at the bottom.

Scoped 2026-11-18 with Kranti from a real user's ask: "maybe add comment button! now only can
edit it after creating, I want to leave some new updates or sources related". Editing a note
existed; a trail of additions did not. Terms follow `client/CONTEXT.md` `### Notes` and
`services/plan-service/CONTEXT.md` `## Notes`: **note**, **note page**, **body**, **summary**,
**comment**. "Update" was the scoping word and is rejected: it collides with item update,
`UpdateChecklistItem` and `updated_at`.

Decisions and rejected alternatives carried from scoping:

- **Body + comments** on the note page. Rejected: body only (a comment is just more body text,
  and the "when did I add this" trail is lost); comments only (a journal with no single body,
  less like a proper note).
- **Authored**, not time-only: on a shared plan this is a conversation, so who wrote it matters.
- **Edit and delete: author, plus owner delete.** Rejected: author only (no moderation on a
  shared plan); append-only (typos stick forever).
- **Sources are plain text with auto-linked URLs.** Rejected: a structured link field on a
  comment (more form, little gain for "sources related").
- **Edit shows a mark, keeps no history.**
- **No comment count on the note card**: FS-PGG2N wants the card simpler, not busier.

## Requirements

### Storage and ownership (plan-service)

1. **R1.** Comments are stored in plan-service, in a new table `note_comments` in the plan
   database: `id` (UUID PK), `item_id` (UUID NOT NULL, FK → `checklist_items(id)` **ON DELETE
   CASCADE**), `author_id` (UUID NOT NULL, no FK: users live in the gateway's database,
   ADR-0010), `body` (TEXT NOT NULL), `created_at` (TIMESTAMPTZ NOT NULL DEFAULT NOW()),
   `edited_at` (TIMESTAMPTZ NULL; null until the first edit). One migration adds it, with an
   index on `(item_id, created_at, id)`. plan-service owns comments because it owns checklist
   items and answers plan access (ADR-0015). They are not in insights-service and not in the
   gateway's `notes` table.
2. **R2.** A comment belongs to exactly one note. Every comment operation addresses it as
   (plan, item, comment). The item must be in the path's plan and the comment must be on that
   item; any mismatch is answered exactly like a missing row: **not found** (ADR-0015 §5).
3. **R3.** **Only notes have comments.** Listing or adding comments on an item whose `type` is
   `task` is refused with **`ITEM_NOT_A_NOTE`** (400). A milestone is a task (ADR-0014), so this
   covers milestones too. Child notes under a milestone are notes and have comments like any
   other.
4. **R4.** **A note with comments cannot become a task.** `UpdateChecklistItem` with
   `type = task` on a note that has at least one comment is refused with
   **`NOTE_HAS_COMMENTS`** (400) and changes nothing. The check lives in plan-service's item
   update, beside "notes can't be parents". A note with no comments converts as today; task →
   note is unaffected.
5. **R5.** R3 and R4 cannot interleave to leave a comment on a task: adding a comment and
   changing an item's type each check the other's condition inside one transaction holding a
   lock on the item row.
6. **R6.** **The author is the caller.** plan-service sets `author_id` from the caller's
   identity and never from anything the client sends. No HTTP request type carries an author,
   user or identity field. Today that identity reaches plan-service as the `user_id` the gateway
   fills from the verified token (ADR-0001 §3, the legacy channel every checklist RPC uses);
   when FS-0001's metadata interceptor lands it is read from context instead, with no change to
   this FS.
7. **R7.** Comments are never read or written by anything but the comment operations. A
   comment never changes the note's body, its summary, its `description` or its `updated_at`,
   and the AI summary (FS-PGG2N) never reads comments. No model call is involved anywhere
   (ADR-0019).

### Body rules

8. **R8.** A comment body is **plain text**. It is trimmed of surrounding whitespace before it
   is stored; inner line breaks are kept.
9. **R9.** After trimming, a body must be **1 to 2000 characters** (Unicode code points).
   Empty, whitespace-only, or over 2000 is refused with **400 `VALIDATION_FAILED`**, decided by
   plan-service. It is a domain rule, so the HTTP schema carries no `minLength`/`maxLength` on
   it (ADR-0005; the same split as `intention`, FS-043WP).
10. **R10.** The same rules apply to an edit (R16).

### Reading

11. **R11.** **List** returns all of a note's comments, **oldest first** (`created_at`, then
    `id` as the tie-break), with no pagination in v1. A note with none answers `[]`.
12. **R12.** Each listed comment carries its id, author id, **author name**, body, created time,
    edited time (null when never edited), and two booleans for the caller, `canEdit` and
    `canDelete` (R18). The client decides what to show from these and never restates the rule.
13. **R13.** **Author name** is resolved by the gateway from its users table: the display name
    if set, otherwise the account name (the FS-12ZMF R15 rule), looked up in one batch per
    list. An author with no account is answered with `authorName: null`, and the client shows
    **"Deleted user"**.

### Who may do what (gateway, ADR-0015)

14. **R14.** Every comment operation asks the gateway's access seam first. A caller with no
    relation to the plan (a stranger, or no such plan) gets **404 `NOT_FOUND`**, as on every
    plan route.
15. **R15.** **List and add:** the plan's owner and any member.
16. **R16.** **Edit:** the comment's author only, whether owner or member. The owner cannot
    edit someone else's comment. An edit replaces the body and sets `edited_at` to now. An edit
    whose trimmed body equals the stored body is a no-op: it answers the comment unchanged and
    does not set `edited_at`. No previous versions are kept.
17. **R17.** **Delete:** the comment's author, or the plan's owner. A delete is permanent.
18. **R18.** Anyone else with access who tries to edit or delete gets **403 `FORBIDDEN`**. The
    gateway decides R15–R17 from plan-service's relation answer plus the comment's `author_id`,
    in one policy function that also computes `canEdit` and `canDelete` for R12, so the flags and
    the enforcement cannot disagree.
19. **R19.** plan-service keeps a defensive backstop, as owner-scoped mutations do (ADR-0015
    §2): its update scopes by `author_id = caller`, and its delete by `author_id = caller` or
    the plan's owner = caller, answering `PermissionDenied` otherwise (which the gateway maps to
    403 `FORBIDDEN`).

### Lifecycle

20. **R20.** Deleting a note deletes its comments (FK cascade). Deleting a plan deletes them
    through its items. Archiving or unarchiving a note leaves them untouched.
21. **R21.** **Duplicating a note does not copy its comments** (FS-F6AEE copies item fields
    only): the copy starts with none.
22. **R22.** A member removed from a plan keeps their comments there, still shown under their
    name; they can no longer reach the plan (404). The owner may delete those comments.
23. **R23.** A user who deletes their account (`user.deleted`) loses the plans they own and so
    the comments on them (cascade). Their comments on other people's plans stay, shown as
    **"Deleted user"**. Nobody can edit those; the plan owner can delete them.

### On screen (client, on the FS-PGG2N note page)

24. **R24.** The **comments section** sits on the note page below the body and the summary.
    Every note page has one, top-level or under a milestone. It follows FS-PGG2N's note look
    (paper surface, upright Lora) and the design guideline's tokens.
25. **R25.** Each comment shows: a round **initial avatar** (the first letter of the shown name,
    uppercased; muted for "Deleted user"), then a meta line **"name · relative time"**, plus
    **" · edited"** when `editedAt` is set, then the body with its line breaks kept.
26. **R26.** **Relative time:** "just now" under a minute, "N min ago" under an hour, "N h ago"
    under a day, "yesterday", "N days ago" under a week, then the date ("Nov 3"; with the year
    when not this year). The full date and time is the element's tooltip and its accessible
    text.
27. **R27.** **URLs are auto-linked on render.** `http://` and `https://` URLs in a body become
    links that open in a new tab (`rel="noopener noreferrer nofollow"`). Trailing sentence
    punctuation (`.,;:!?`) and an unbalanced closing `)` are not part of the link. Everything
    else renders as text: a body is never interpreted as HTML or markdown.
28. **R28.** **Adding:** at the bottom of the list, a growing text box with the placeholder
    **"Add a comment…"** and a **Comment** button. The button is enabled only when the trimmed
    text is 1–2000 characters. Ctrl/Cmd+Enter submits; Enter makes a new line. A character
    count appears once the text passes 1800. On success the new comment appears last, the box
    clears and keeps focus. On failure the text stays in the box with an inline error.
29. **R29.** **Edit and delete** controls appear per comment only where `canEdit` / `canDelete`
    is true: shown on hover and keyboard focus on pointer devices, always shown on touch. Edit
    turns the body into a text box prefilled with it, with Save and Cancel; Escape cancels,
    Ctrl/Cmd+Enter saves; Save follows R28's enable rule. Delete asks to confirm first (the
    existing `DeleteConfirm`), then removes the comment from the list.
30. **R30.** The section switches on `code`, never on `detail` (ADR-0004). `NOT_FOUND` on an
    edit or delete means the comment is gone: it is removed from the list with a short "This
    comment was deleted." `FORBIDDEN` reloads the list (the flags were stale). `ITEM_NOT_A_NOTE`
    reloads the note page (the item became a task elsewhere).
31. **R31.** **Make a task** on a note refused with `NOTE_HAS_COMMENTS` leaves the note as it is
    and tells the user: "This note has comments. Delete them to make it a task." No other
    convert path changes.
32. **R32.** **No comment count on the note card**, and nothing about comments on the card or a
    child note row.
33. **R33.** Comments load when the note page opens, separately from the body: a slow or
    failed comment load never blocks writing the body.

## User Stories

1. As a note's writer, I want to add a comment under my note, so that I can record a later
   thought without rewriting what I first wrote.
2. As a note's writer, I want each comment dated, so that I can see when I added what.
3. As a note's writer, I want to paste a link into a comment and have it clickable, so that I
   can keep sources next to the note they support.
4. As a member of a shared plan, I want to comment on a note the owner wrote, so that we can
   discuss it where it lives.
5. As an owner of a shared plan, I want to see who wrote each comment, so that the conversation
   is readable.
6. As a reader, I want comments oldest first with the add box at the bottom, so that the page
   reads top to bottom like a conversation.
7. As an author, I want to fix a typo in my comment, so that a mistake doesn't stick forever.
8. As a reader, I want to see that a comment was edited, so that I know it changed after it was
   written.
9. As an author, I want to delete my own comment, so that I can take back something I didn't
   mean to post.
10. As a plan owner, I want to delete anyone's comment on my plan, so that I can keep a shared
    plan tidy.
11. As a plan owner, I want not to be able to edit someone else's words, so that a comment shown
    under their name is always what they wrote.
12. As a member, I want edit and delete offered only where I am allowed, so that I'm never shown
    a control that fails.
13. As a member, I want my comments to keep my name after I leave a plan, so that the history
    stays readable for those still on it.
14. As a reader, I want a deleted account's comments to read "Deleted user", so that nothing
    breaks or shows a blank where a name was.
15. As a note's writer, I want comments separate from my note's body and summary, so that adding
    to a note never changes its card.
16. As a note's writer, I want my card to stay simple with no comment count, so that the grid
    stays calm.
17. As a user, I want comments on notes under a milestone too, so that every note works the same.
18. As a user, I want to be told why a note with comments won't turn into a task, so that I know
    what to do instead of seeing nothing happen.
19. As a user, I want a duplicated note to start with no comments, so that the copy is a fresh
    note and the conversation stays with the original.
20. As a user, I want comments to go when I delete the note, so that nothing is left orphaned.
21. As a user, I want a clear limit on comment length with a count near it, so that I don't lose
    a long comment to a refusal.
22. As a user, I want a failed comment to stay in the box, so that I can retry without retyping.
23. As a user, I want to submit with Ctrl/Cmd+Enter and still write line breaks, so that typing a
    comment is quick and I can still lay it out.
24. As a user, I want the body still writable when comments fail to load, so that a comment
    outage doesn't stop me writing my note.
25. As a stranger who guesses a plan's id, I want nothing to reveal that the plan or its
    comments exist, so that private plans stay private.
26. As a developer, I want the author taken from identity and never the request, so that nobody
    can post under someone else's name.
27. As a developer, I want the rights decided in one place in the gateway, so that the flags the
    client sees and the enforcement can't drift.

## Acceptance Criteria

### plan-service

- [ ] A migration creates `note_comments` with the columns, FK cascade and index of R1; its down
      migration drops it.
- [ ] Create on a note stores the trimmed body with `author_id` = the caller and `edited_at`
      null; the response carries both times.
- [ ] Create or list on a task → `FailedPrecondition` with domain code `ITEM_NOT_A_NOTE`.
- [ ] Create with `""`, `"   "` or a body over 2000 code points after trimming → InvalidArgument,
      and nothing is stored; exactly 2000 code points (multi-byte characters included) is stored.
- [ ] List answers oldest first, ties broken by id, and `[]` for a note with none.
- [ ] Any comment RPC whose item is not in the given plan, or whose comment is not on the given
      item → `NotFound`.
- [ ] Update by the author replaces the body and sets `edited_at`; the same trimmed body leaves
      `edited_at` unset; update by anyone else → `PermissionDenied` (R19 backstop).
- [ ] Delete by the author or the plan owner removes the row; by anyone else →
      `PermissionDenied`.
- [ ] `UpdateChecklistItem` `type=task` on a note with a comment → `FailedPrecondition` with
      `NOTE_HAS_COMMENTS`, and the item is unchanged; on a note with none it converts as before.
- [ ] Concurrent create-comment and convert-to-task on the same note never both succeed (R5).
- [ ] Deleting the note deletes its comments; duplicating it copies none; archiving leaves them.
- [ ] Creating, editing and deleting a comment leave the item's `description`, body, summary and
      `updated_at` unchanged.
- [ ] `ITEM_NOT_A_NOTE` and `NOTE_HAS_COMMENTS` are added to `common/errcode` (additive), and the
      proto Go is regenerated, not hand-edited.

### Gateway

- [ ] The four operations in §API surface are registered as typed (Huma) endpoints; `make
      openapi client` regenerates `openapi.yaml` and the TS client, and `make gates` passes.
- [ ] A stranger gets 404 `NOT_FOUND` on all four; an unauthenticated caller 401.
- [ ] Owner and member can list and add; the response's `authorName` is the display name, else
      the account name, and null for a missing account, resolved in one batch per list.
- [ ] `canEdit` is true only for the caller's own comments; `canDelete` is true for the caller's
      own and, for the plan owner, for all.
- [ ] A member editing or deleting another's comment → 403 `FORBIDDEN`; the owner editing
      another's → 403 `FORBIDDEN`; the owner deleting another's → 204.
- [ ] A request body with an unknown member (e.g. `authorId`), a missing `body`, or a non-string
      `body` → 422 `VALIDATION_FAILED`; an empty-after-trim or over-length body → 400
      `VALIDATION_FAILED`.
- [ ] `updateChecklist` with `type: "task"` on a note with comments → 400 `NOTE_HAS_COMMENTS`.
- [ ] Problem responses carry `application/problem+json` (asserted on at least one 400 and one
      403).

### Client

- [ ] The note page shows the comments section below the body and summary, for top-level notes
      and notes under a milestone.
- [ ] Each comment shows the initial avatar, "name · relative time", " · edited" when edited,
      and the body with line breaks; a missing author shows "Deleted user".
- [ ] Relative times follow R26, with the full date and time as tooltip and accessible text.
- [ ] `http(s)` URLs render as new-tab links with `rel="noopener noreferrer nofollow"`; trailing
      punctuation is excluded; `<b>x</b>` in a body renders as those literal characters.
- [ ] The "Add a comment…" box sits below the last comment; Comment is disabled for empty or
      over-2000 text; Ctrl/Cmd+Enter submits, Enter adds a line; the count shows past 1800.
- [ ] A new comment appears last and the box clears; a failed add keeps the text and shows an
      error.
- [ ] Edit and Delete appear only where `canEdit` / `canDelete`; edit saves in place and shows
      " · edited"; delete confirms first.
- [ ] `NOT_FOUND` on edit/delete removes the comment with "This comment was deleted.";
      `FORBIDDEN` reloads the list; the client branches on `code` only.
- [ ] Make a task on a note with comments shows "This note has comments. Delete them to make it a
      task." and the note stays a note.
- [ ] The note card and child note row show nothing about comments.
- [ ] A failed comment load shows a retry in the section and the body stays editable.
- [ ] Request and response types come from the generated client; none is hand-written.

## Edge States

| Situation | Behavior |
| --- | --- |
| Note has no comments | Section shows only the "Add a comment…" box; list answers `[]` |
| Body empty or whitespace only | Button disabled; if sent anyway, 400 `VALIDATION_FAILED` and nothing stored |
| Body over 2000 after trimming | Button disabled with the count shown; if sent, 400 `VALIDATION_FAILED` |
| Exactly 2000 code points, emoji included | Accepted; client and server both count code points |
| Body is HTML or markdown | Shown as literal text; only `http(s)` URLs become links |
| Edit to the same text | No-op: no " · edited" mark, `edited_at` stays null |
| Comment on a task (API, or a stale page) | 400 `ITEM_NOT_A_NOTE`; the page reloads the item |
| Make a task on a note with comments | 400 `NOTE_HAS_COMMENTS`; note unchanged; message per R31 |
| Add a comment while another member converts the note | One wins (R5): the comment lands and the convert is refused, or the convert lands and the add gets `ITEM_NOT_A_NOTE` |
| Another member deletes a comment you are editing | Save gets 404 `NOT_FOUND`; comment removed with "This comment was deleted." |
| Note deleted by another member while its page is open | Add, edit and delete get 404; the note page's own "gone" handling applies (FS-PGG2N) |
| Caller removed from the plan while on the page | Next comment call gets 404, as on every plan route |
| Two members add at the same moment | Both stored; order is `created_at`, then `id`. No live refresh: others' new comments show on the next load |
| Author left the plan | Comments stay under their name; `canEdit` is false for everyone else; owner may delete |
| Author's account deleted | `authorName: null` → "Deleted user", muted avatar; nobody can edit; owner may delete |
| Owner's account deleted | Their plans cascade, comments with them (R23) |
| Note duplicated | Copy has no comments |
| Note archived / unarchived | Comments untouched |
| Note deleted, or plan deleted | Comments deleted by cascade |
| Many comments on one note | All returned, no pagination in v1 |
| Users table lookup fails while listing | The list fails with the lookup's status (503 when unreachable); no partial names |
| plan-service unreachable | 503 `SERVICE_UNAVAILABLE`; the section shows a retry and the body stays usable |
| Unauthenticated | 401 `UNAUTHENTICATED` |
| Stranger, or a guessed plan id | 404 `NOT_FOUND`, identical to a missing plan |

## API surface

### HTTP (gateway, new resource: note comment)

All four are typed (Huma) operations under the checklist item route, owner-or-member access
first (R14), strict request bodies (ADR-0005).

**`NoteCommentResp`**

| Field | Type | Notes |
| --- | --- | --- |
| `id` | uuid | |
| `authorId` | uuid | |
| `authorName` | string \| null | Display name, else account name; null when the account no longer exists (client shows "Deleted user") |
| `body` | string | Trimmed plain text, line breaks kept |
| `createdAt` | date-time | |
| `editedAt` | date-time \| null | Null until first edited |
| `canEdit` | boolean | For the caller: true only on their own comment |
| `canDelete` | boolean | For the caller: own comment, or any comment when the caller owns the plan |

**`NoteCommentReq`** (create and edit): `body` string, required. No length bounds in the
schema (R9 is a domain rule). No author or identity field (R6).

| Op | Method + Path | Query/Params | Request body | Response | Errors |
| --- | --- | --- | --- | --- | --- |
| listNoteComments | GET `/api/plans/{id}/checklists/{checklist_id}/comments` | path `id`, `checklist_id` (uuid) | — | 200 `NoteCommentResp[]`, oldest first; `[]` when none | 401 · `UNAUTHENTICATED`; 404 · `NOT_FOUND` (no relation, no plan, item not in plan); 400 · `ITEM_NOT_A_NOTE`; 422 · `VALIDATION_FAILED` (bad uuid); 503 |
| createNoteComment | POST `/api/plans/{id}/checklists/{checklist_id}/comments` | same | `NoteCommentReq` | 201 `NoteCommentResp` | 401; 404 · `NOT_FOUND`; 400 · `ITEM_NOT_A_NOTE`; 400 · `VALIDATION_FAILED` (empty after trim, over 2000); 422 · `VALIDATION_FAILED` (missing or non-string `body`, unknown member); 503 |
| updateNoteComment | PATCH `/api/plans/{id}/checklists/{checklist_id}/comments/{comment_id}` | plus `comment_id` (uuid) | `NoteCommentReq` | 200 `NoteCommentResp` (`editedAt` set unless a no-op) | 401; 404 · `NOT_FOUND` (also comment not on this note); 403 · `FORBIDDEN` (not the author); 400 · `VALIDATION_FAILED`; 422 · `VALIDATION_FAILED`; 503 |
| deleteNoteComment | DELETE `/api/plans/{id}/checklists/{checklist_id}/comments/{comment_id}` | same | — | 204 | 401; 404 · `NOT_FOUND`; 403 · `FORBIDDEN` (neither author nor plan owner); 503 |
| updateChecklist (changed) | PATCH `/api/plans/{id}/checklists/{checklist_id}` | unchanged | unchanged | unchanged | unchanged, plus `type: "task"` on a note with comments → 400 · `NOTE_HAS_COMMENTS` |

New codes in `common/errcode` (additive, ADR-0004): `ITEM_NOT_A_NOTE` and `NOTE_HAS_COMMENTS`, both
400, decided by plan-service and sent as an ErrorInfo detail on `FailedPrecondition`
(`commongrpc.WithCode`), so the gateway needs no mapping of its own.

### gRPC (plan-service `ChecklistService`, `common/api/proto/plan/checklist.proto`)

`NoteComment { id, item_id, author_id, body, created_at (Timestamp), optional edited_at (Timestamp) }`

| RPC | Request | Response | Notes |
| --- | --- | --- | --- |
| ListNoteComments | `plan_id, item_id, user_id` | `ListNoteCommentsResponse { repeated NoteComment comments }` | Oldest first; `ITEM_NOT_A_NOTE` on a task |
| GetNoteComment | `plan_id, item_id, comment_id, user_id` | `NoteComment` | Lets the gateway read `author_id` before deciding R16–R17 |
| CreateNoteComment | `plan_id, item_id, user_id, body` | `NoteComment` | `author_id` = caller (R6); R5 lock; R8–R9 |
| UpdateNoteComment | `plan_id, item_id, comment_id, user_id, body` | `NoteComment` | Author-scoped backstop (R19); R16 no-op rule |
| DeleteNoteComment | `plan_id, item_id, comment_id, user_id` | `google.protobuf.Empty` | Author-or-owner backstop (R19) |

`user_id` is the caller, filled by the gateway from the verified token, as on every checklist
RPC today (R6).

## Out of Scope

- Notifications of any kind (new comment, reply, mention), and @mentions.
- Replies or threads, reactions, pinning, resolving.
- Rich text, markdown, attachments or images in a comment; a structured link field or link
  previews.
- Edit history or showing previous versions; an "edited at" time beyond the mark.
- Pagination, "load more", or live updates (polling, push) of comments.
- A comment count or any comment hint on the note card or child row.
- Comments on tasks or milestones.
- Copying comments on duplicate.
- Comments feeding the AI summary or any model call.
- Searching or filtering by comments.
- Events for comment changes (no outbox rows, no AMQP).
- The gateway's hidden notes system (`internal/notes`, the `notes` table, `NotesContainer`,
  `/api/plans/:id/notes*`): untouched.
- Moving identity to gRPC metadata: that is FS-0001's work (R6 rides on whatever it delivers).
- The note page itself, its body editor, summary and note look: FS-PGG2N.
