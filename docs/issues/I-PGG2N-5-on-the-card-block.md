---
id: I-PGG2N-5
status: done
implements: FS-PGG2N
blocked_by: [I-PGG2N-3]
labels: [feature]
title: "FS-PGG2N slice 5: write or edit a note's summary on its page"
---
Implements FS-PGG2N §Requirements (R28, R29)

## What to Build

Add the **"On the card"** block below the meta line on `NotePage`. It shows what the card
shows and lets the user write the summary by hand. The AI button is slice 7. Leave it out here,
or render it disabled behind a flag, whichever keeps slice 7's diff small.

- **What it shows (R28).** With a summary, the summary. With none, "No summary yet. The card
  shows the start of the note." The action reads **Edit summary**, or **Write a summary** when
  there is none.
- **Editing (R29).**
  - A textarea inside the block. It is one paragraph: Enter saves and Shift+Enter is ignored.
    Blur saves and Esc cancels.
  - Input stops at 280 characters, with a live "n/280" counter.
  - Saving sends the trimmed text as `summary` through `updateChecklistItem`. Saving empty clears
    it, and unchanged text sends nothing.
  - The save is optimistic: a failure restores the previous summary and shows the existing error.
    A 404 shows "This note was deleted" (R37, reusing slice 3's state).
- **Expose an opener.** Export a way to open the editor with focus, and optionally with prefilled
  text. Slices 7 and 8 use it for the AI proposal and for the leave prompt's Write a summary.

Files expected: `client/src/components/NotePage.tsx`, or a new
`client/src/components/NoteSummaryBlock.tsx` (+ test).

## Acceptance Criteria

- [ ] With no summary the block shows the "No summary yet…" line and Write a summary. With one it
      shows the summary and Edit summary.
- [ ] Enter or blur saves the trimmed summary. Esc cancels. Shift+Enter adds no line.
- [ ] Input stops at 280, and the counter reads n/280.
- [ ] Saving empty clears the summary, and the block returns to "No summary yet…".
- [ ] Saving unchanged text sends no request.
- [ ] A failed save restores the previous summary with the existing error. A 404 shows "This note
      was deleted".
- [ ] Going back to the plan, the card shows the new summary.
- [ ] `npm test`, `npm run lint` and `npm run build` pass.

## Blocked By

I-PGG2N-3 (the note page).

## Spec Reference

FS-PGG2N:
- §Requirements R28, R29 (charter defaults C1, C8; gap default G10), R37 (the deleted state on
  save).
- §Acceptance Criteria: the Edit summary item.
- §Edge States: summary cleared by hand.

## TDD Approach

- RED: a block test. Typing "  gist " and pressing Enter sends `{ summary: "gist" }` once.
- GREEN: the editor and the save.
- Then: Esc, blur, empty clears, unchanged sends nothing, the 280 cap and counter, and failure
  revert.
