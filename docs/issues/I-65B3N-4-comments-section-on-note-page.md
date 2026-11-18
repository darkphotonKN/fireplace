---
id: I-65B3N-4
status: done
implements: FS-65B3N
blocked_by: [I-65B3N-1, I-PGG2N-3]
labels: [feature]
title: "FS-65B3N slice 4: comments section on the note page, read and add"
---
Implements FS-65B3N §Requirements (R12–R13, R24–R28, R30 partial, R32–R33), §API surface (listNoteComments, createNoteComment)

## What to Build

Mount a comments section into the FS-PGG2N note page, below the body and summary, for every
note (top-level and under a milestone). This slice reads and adds; slice 5 adds edit and
delete.

- API wrappers in `client/src/api/` (suggested `noteComments.ts`) over the generated client:
  `listNoteComments(planId, itemId)`, `createNoteComment(planId, itemId, body)`. Types from
  `components["schemas"]`, none hand-written. A create records the touched-today stamp like the
  other mutations in `checklists.ts`.
- A `NoteComments` component (suggested `client/src/components/NoteComments.tsx`) the note page
  renders:
  - Loads on open, independently of the body (R33). Failure → an inline "Couldn't load comments"
    with Retry; the body stays editable.
  - Each comment (R25): round initial avatar (`rounded-full`; first letter of the shown name,
    uppercased; muted for "Deleted user"), "name · relative time", " · edited" when `editedAt`,
    body with `white-space: pre-wrap`. `authorName: null` → "Deleted user" (R13).
  - Relative time per R26 (pure helper, suggested `client/src/lib/relativeTime.ts`), with the
    full date-time as `title` and accessible text.
  - Auto-link per R27 (pure helper, suggested `client/src/lib/autolink.ts`) returning text and
    link segments rendered as React nodes, never `dangerouslySetInnerHTML`.
  - Oldest first; the "Add a comment…" growing textarea and Comment button at the bottom (R28):
    enabled for 1–2000 trimmed code points (`[...s].length`), Ctrl/Cmd+Enter submits, Enter is a
    newline, counter past 1800; on success append, clear, keep focus; on failure keep the text
    and show the error.
  - `NOT_A_NOTE` reloads the note page; branch on `code` only (R30, ADR-0004).
- Look per FS-PGG2N's note look (paper, upright Lora) and `docs/design-guideline.md` tokens.
- Nothing on the note card or child note row (R32).

## Acceptance Criteria

- [ ] The section renders below body and summary on a top-level note's page and a child note's
      page.
- [ ] Comments show avatar, "name · relative time", " · edited" when edited, and line breaks;
      `authorName: null` shows "Deleted user" with a muted avatar.
- [ ] `relativeTime` unit tests cover every R26 band and the other-year date.
- [ ] `autolink` unit tests: http/https linked, trailing `.,;:!?` and an unbalanced `)` excluded,
      `<b>x</b>` stays literal text, `javascript:` not linked; links have `target="_blank"` and
      `rel="noopener noreferrer nofollow"`.
- [ ] Comment button disabled for empty / whitespace / over 2000; counter appears past 1800;
      Ctrl/Cmd+Enter submits and Enter inserts a newline.
- [ ] A successful add appears last and clears the box; a failed add keeps the text with an
      error.
- [ ] A failed load shows Retry and the body is still editable.
- [ ] No comment hint on the card or child row.
- [ ] `npm test`, `npm run lint` and `npm run build` pass in `client/`.

## Blocked By

I-65B3N-1, and the FS-PGG2N note page issue (the integrator links its id)

## Spec Reference

FS-65B3N §Requirements R12–R13, R24–R28, R30 (`NOT_A_NOTE`), R32–R33; §API surface
(listNoteComments, createNoteComment); §Edge States (no comments, empty/over-length body, HTML in
body, 2000 exactly, deleted author, load failure, note deleted while open). User stories 1–6,
14–17, 21–24.

## TDD Approach

- RED: `autolink` and `relativeTime` pure-function tests.
- RED: `NoteComments` component test with a mocked API — renders two comments oldest first,
  "Deleted user" for a null name, adds a third at the bottom.
- GREEN: helpers, component, mount in the note page.
