---
id: I-PGG2N-3
status: done
implements: FS-PGG2N
blocked_by: [I-PGG2N-2]
labels: [feature]
title: "FS-PGG2N slice 3: the note page, with its body on paper and saved as you type"
---
Implements FS-PGG2N §Requirements (R16–R27, R36, R37, R49)

## What to Build

A note opens at `/plan/<planId>/n/<noteId>`, in the plan's middle card, the way the milestone
page does (FS-043WP). The page shows the body on a paper sheet. Reading mode has live links.
Writing mode is a plain textarea that saves itself. This slice reaches the page by URL only: the
links from cards and rows are slice 4. The "On the card" block is slice 5. Stopping a navigation
over a failed save, and the leave prompt, are slice 8.

**Route (R16, R18).**
- Add `client/src/app/plan/[planId]/n/[noteId]/page.tsx`, mirroring the milestone route, which
  renders a new `client/src/components/NotePage.tsx` in the layout's middle slot.
- It loads from the plan's long-term item list, the same list and cache the grid and the
  milestone page read. No new read endpoint.
- While loading it shows a quiet loading line.
- Moving between plan, milestone and note pages refetches none of the surrounding blocks.

**Header (R17, R20, R21).**
- A Back link in the milestone page's back style. The destination comes from the note's parent:
  a parent → `/plan/<planId>/m/<parentId>`, labelled with the milestone's title; none →
  `/plan/<planId>`, "Back to plan". Never history-back.
- The page's `ItemActions` offers Make a task and Delete only.
  - Make a task sends `type: "task"`. On success, go to the Back destination. A refusal (this
    includes FS-65B3N's comments rule once it exists) stays with the existing error.
  - Delete uses the existing confirmation, then goes to the Back destination.
- The tab title is `firstLine(cardText, 60)`.

**Body (R19, R23–R26, R49).**
- The order is: header, the body sheet, the meta line, then empty slots for the "On the card"
  block (slice 5) and the comments area (FS-65B3N).
- The sheet: `--note-paper`, radius 12px, padding about 28px (less below `md`), Lora about 17px,
  line-height about 1.75.
- Reading mode renders the body with line breaks kept. Only `http(s)://` URLs become links
  (`target="_blank" rel="noopener noreferrer"`). Any other scheme is plain text, and nothing is
  ever injected as HTML.
- Clicking the text (not a link), or Enter on the focused sheet, switches to an auto-growing
  `<textarea>` with the caret at the end. Blur returns to reading. No editor library.
- Autosave through `updateChecklistItem({ description })` runs 800 ms after the last keystroke,
  on blur, and on unmount (best effort). Unchanged text sends nothing.
- Empty or whitespace text is never sent (R25).
- The meta line shows "‹N› words · edited ‹relative updatedAt›", "Saving…" in flight, and
  "Couldn't save · Try again" on failure, keeping the text (R27, first half).

**States (R36, R37).**
- "This note isn't here", with a link to the plan, when the id is:
  - not in the list (unknown, deleted, archived);
  - not a note;
  - another plan's;
  - or the list answers 404, 403 or 422.
- "Couldn't load this note" with Try again on network or 5xx.
- A save answering 404 replaces the page with "This note was deleted" and Back.
- There is no live watching.

**Access (R22).** Nothing new: the page uses the checklist routes only.

Files expected:
- `client/src/app/plan/[planId]/n/[noteId]/page.tsx`
- `client/src/components/NotePage.tsx` (+ `NotePage.test.tsx`)
- possibly a small `client/src/lib/linkify.ts` (+ test)
- `client/src/app/plan/[planId]/layout.tsx` only if the middle slot needs to know the new route
  (+ `layout.test.tsx`)

## Acceptance Criteria

- [ ] `/plan/<planId>/n/<noteId>` renders the note page in the middle card, with the plan's other
      blocks around it.
- [ ] Plan → note → plan refetches none of the surrounding blocks.
- [ ] Back goes to the milestone page for a milestone's child note, else to the plan, whatever
      the history.
- [ ] The order is header, body sheet, meta line, then the block and comments slots.
- [ ] The menu offers exactly Make a task and Delete. Both go to the Back destination on success
      and stay with the error on failure.
- [ ] Reading mode keeps line breaks and links `http(s)` URLs with `noopener noreferrer`.
      `javascript:` text stays plain.
- [ ] Clicking the text enters writing mode, and blur leaves it.
- [ ] A save goes out 800 ms after the last keystroke and on blur. Unchanged text sends nothing.
- [ ] Empty or whitespace body sends no save, and a reload shows the old body.
- [ ] The meta line shows the word count and the edited time, "Saving…", and "Couldn't save · Try
      again".
- [ ] Unknown, deleted, archived, task, other-plan, 404/403/422 → "This note isn't here".
      Network or 5xx → retry.
- [ ] A save answering 404 shows "This note was deleted" with Back.
- [ ] The tab title is the note's first line, up to 60 characters.
- [ ] `npm test`, `npm run lint` and `npm run build` pass.

## Blocked By

I-PGG2N-2 (the `--note-paper` token and the card text helper).

## Spec Reference

FS-PGG2N:
- §Requirements R16–R26, R27 (first half), R36, R37, R49 (charter defaults C9–C11, C13, C14,
  C17, C18; gap defaults G1–G6, G17–G19).
- §Acceptance Criteria: the route, Back, menu, editor, meta line, not-here, deleted and tab title
  items.
- §Edge States: empty or whitespace body, `javascript:` text, failed save, deleted, archived or
  converted elsewhere, member removed, malformed ids.

## TDD Approach

- RED: a `NotePage` test. Given a list with a child note, Back points at the milestone page.
  Given a top-level note, it points at the plan.
- GREEN: the route, the load from the list, and the header.
- Then: linkify unit tests. The autosave timing with fake timers (one PATCH after 800 ms, none
  for unchanged or empty). The not-here cases. The 404-on-save state.
