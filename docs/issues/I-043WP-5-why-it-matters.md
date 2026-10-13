---
id: I-043WP-5
status: done
implements: FS-043WP
blocked_by: [I-043WP-1, I-043WP-2]
labels: [feature]
title: "FS-043WP slice 5: write down why a milestone matters, on its page"
---
Implements FS-043WP §Requirements (R22–R25)

## What to Build

This slice adds the page's *why it matters*. It is the item's `intention` from slice 1, shown
and edited only on the milestone page.

**Show (R22).** When it's empty, the page shows a quiet prompt, "Why does this matter?". When
it's set, the page shows the text. Either way it sits between the progress and the next step
(R11).

**Edit (R23, G10).** Clicking the text or the prompt edits it in place. Enter or blur saves and
Esc cancels. Input stops at 280 characters (`maxLength`). The text is a single paragraph, so
Enter saves instead of starting a new line. Saving sends the trimmed text as `intention` through
`updateChecklistItem`. Saving empty text clears it, and saving unchanged text sends nothing.

**Failure (R24).** The save is optimistic. If it fails, the previous text comes back and the
existing error shows.

**Nowhere else (R25).** No card, row, composer or create path shows it or asks for it. Nothing
changes in `ChecklistGrid`, `Todo` or the composers.

Files expected: `client/src/components/MilestonePage.tsx`, or a small
`client/src/components/MilestoneIntention.tsx` that it renders, + test.

## Acceptance Criteria

- [ ] An empty why-it-matters shows "Why does this matter?".
- [ ] Clicking it starts editing. Enter or blur saves the trimmed text, and Esc cancels.
- [ ] The input stops at 280 characters.
- [ ] Saving empty text clears it and brings the prompt back. Saving unchanged text sends no
      request.
- [ ] A failed save restores the old text and shows the existing error.
- [ ] Why it matters appears on no card, row or create form.
- [ ] Tests pass.

## Blocked By

- I-043WP-1: the `intention` field must exist on the wire and in the generated types.
- I-043WP-2: needs `MilestonePage`.

## Spec Reference

FS-043WP §Requirements R22–R25 (why it matters, charter default C5, gap default G10).
§Acceptance Criteria (the why-it-matters items). §Edge States (saved as whitespace, pasted past
280, save fails). §Out of Scope (never on the card, never at create).

## TDD Approach

- RED: a test that clicking "Why does this matter?", typing and pressing Enter calls
  `updateChecklistItem` with `{ intention: "…" }` and shows the text.
- GREEN: the editor.
- Then: Esc cancels, saving empty clears it, and a rejected save restores the old text.
