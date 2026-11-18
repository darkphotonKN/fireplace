---
id: I-PGG2N-2
status: done
implements: FS-PGG2N
blocked_by: [I-PGG2N-1]
labels: [feature]
title: "FS-PGG2N slice 2: notes look like paper and show their card text"
---
Implements FS-PGG2N §Requirements (R4, R10, R11, R46–R48, R50)

## What to Build

Notes stop being italic and take on variant A, "Paper". Every place that draws a note outside
its page shows the **card text** (`summary ?? description`), clamped. This slice changes looks
only. Clicks still behave as today: the ways into the note page are slice 4.

**Card text helper (R4).** A small client module, say `client/src/lib/noteText.ts`. Don't name
it `notes.ts`: `client/src/api/notes.ts` is the hidden gateway notes system. It exports:
- `cardText(item)`, which is `summary ?? description`;
- `firstLine(text, max?)`;
- the tunables `NOTE_CARD_LINES = 4`, `NOTE_ROW_LINES = 2` and `NOTE_SUMMARY_THRESHOLD = 280`.

Every caller below uses it.

**Token (R46).**
- Add `--note-paper` to both themes in `client/src/app/globals.css`: light ≈ `42 70% 95%`, dark
  ≈ `38 22% 17%`, in the HSL-channel format the other tokens use.
- Expose it to Tailwind the way the other tokens are, if that is the pattern.
- Document it in `client/docs/design-guideline.md` (Color Palette & Tokens) with its usage rule:
  the surface of a note, and nothing else.

**Note card (R10, R47), `ChecklistGrid.tsx`.**
- Keep the milestone card's tile shape, radius, padding and hover glow, filled with
  `--note-paper`.
- Text in upright Lora at `foreground/88`, the card text clamped to 4 lines with line breaks kept
  (`whitespace-pre-line` plus `line-clamp`), and `break-words`.
- Remove the `FileText` icon from the card and drop `italic`.

**Rows (R11, R48).**
- Child note rows in a milestone card (`ChecklistGrid.tsx`) and note steps on the milestone page
  (`MilestonePage.tsx`) set the card text upright in a paper chip: `--note-paper` fill, radius
  6px, padding 2px 8px, clamped to 2 lines.
- Note rows in the checklist reading (`Todo.tsx`) get the same chip and clamp.
- No note is italic anywhere.

**One-line places (R11, G20).** The calendar (`components/calendar/Calendar.tsx`) shows a dated
note's `firstLine(cardText)` in its chip and title. Accessible names that embed a note's text,
such as `ItemActions`' "Actions for …" and the delete confirmation, use the first line.

**Leave alone (R50).**
- Tasks, milestones, the composer and the add-bar keep their look.
- The add-bar placeholder stays italic, so Lora's italic face stays in
  `client/src/app/layout.tsx`. Only its comment changes: the placeholder is now its only user.
- `FileText` stays in the composer's type toggle and the menu's Make a note entry.

Check the result in the browser in both themes and at 375px wide.

Files expected:
- `client/src/lib/noteText.ts` (+ test)
- `client/src/app/globals.css`, `client/docs/design-guideline.md`, `client/src/app/layout.tsx`
  (comment only)
- `client/src/components/{ChecklistGrid,MilestonePage,Todo,ItemActions}.tsx`,
  `client/src/components/calendar/Calendar.tsx`, and their tests

## Acceptance Criteria

- [ ] `cardText` returns the summary when set, else the description. `firstLine` takes the
      first line, cut to a maximum.
- [ ] `--note-paper` exists in light and dark in `globals.css` and is documented in
      `design-guideline.md`.
- [ ] A note card has the `--note-paper` fill, the milestone tile shape and hover glow, upright
      text at `foreground/88`, and no `FileText` icon.
- [ ] A note card shows the card text clamped to 4 lines with line breaks kept.
- [ ] Child note rows (grid card, milestone page) and list note rows show upright text in a paper
      chip (radius 6px, padding 2px 8px), clamped to 2 lines.
- [ ] No element drawing a note's text carries `italic`. The add-bar placeholder is still italic.
- [ ] Tasks and milestones render exactly as before (existing tests unchanged).
- [ ] A dated note in the calendar shows its card text's first line.
- [ ] A long unbroken URL wraps and never widens a card at 375px (FS-49CQ2 R23).
- [ ] `npm test`, `npm run lint` and `npm run build` pass.

## Blocked By

I-PGG2N-1 (the `summary` field on the client's item type).

## Spec Reference

FS-PGG2N:
- §Requirements R4, R10, R11, R46–R48, R50 (charter defaults C3, C19; gap defaults G7, G20).
- §Acceptance Criteria: the card, row, token, italics and calendar items.
- §Edge States: long URL, narrow screen.

## TDD Approach

- RED: a `noteText` unit test for `cardText` and `firstLine`. Then a `ChecklistGrid` test: a note
  card with a summary renders the summary, not the description, with no `FileText` icon and no
  `italic` class.
- GREEN: the helper and the class changes.
- Then: the row, list and calendar tests, and a visual check in both themes.
