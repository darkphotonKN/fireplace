---
id: I-PGG2N-7
status: done
implements: FS-PGG2N
blocked_by: [I-PGG2N-5, I-PGG2N-6]
labels: [feature]
title: "FS-PGG2N slice 7: Summarize with AI on the note page"
---
Implements FS-PGG2N §Requirements (R30, R31, R44)

## What to Build

The "On the card" block gains **Summarize with AI**. It calls `summarizeNote` (slice 6), and
the client saves the result through the item update. The model runs only on this press (R44).

- **The press (R30).** The button reads "Summarizing…" and is disabled while the call runs, so a
  second press starts no second call.
  - No summary yet → save the returned text as `summary` at once.
  - A summary already exists → open the summary editor (slice 5's opener) prefilled with the
    returned text as a proposal. Save keeps it, and Cancel or Esc keeps the old summary
    (ADR-0007).
- **Failure (R31).** The summary is unchanged. The block shows "Couldn't summarize" and keeps
  Write/Edit a summary. With `code === "LLM_BUDGET_EXHAUSTED"`, add "AI summaries are paused
  until tomorrow". Switch on `code` only, never on `detail`.
  - A 404 shows "This note was deleted".
  - A failed save of the returned text reverts, with the existing error.
- **Shared logic.** Put the press-and-save logic in one function or hook, for example
  `useSummarizeNote`. Slice 8's leave prompt reuses it.

Files expected: `client/src/components/NotePage.tsx` / `NoteSummaryBlock.tsx`, a hook such as
`client/src/hooks/useSummarizeNote.ts` or one inside the component, and tests.

## Acceptance Criteria

- [ ] With no summary, Summarize with AI saves the returned text, and the block and card show it.
- [ ] With a summary, the result appears in the editor as a proposal. Save keeps it, and Cancel
      keeps the old one.
- [ ] The button is disabled with "Summarizing…" while in flight, and one press makes one call.
- [ ] A failure shows "Couldn't summarize" and leaves the summary unchanged.
      `LLM_BUDGET_EXHAUSTED` adds the paused line.
- [ ] A 404 shows "This note was deleted".
- [ ] No model call is made on page load, autosave or summary edits (no `summarizeNote` request
      without a press).
- [ ] `npm test`, `npm run lint` and `npm run build` pass.

## Blocked By

- I-PGG2N-5 (the block and the summary editor).
- I-PGG2N-6 (the endpoint and the client function).

## Spec Reference

FS-PGG2N:
- §Requirements R30, R31, R44 (charter defaults C6, C7; gap default G11; ADR-0007, ADR-0019).
- §Acceptance Criteria: the Summarize with AI and failure items.
- §Edge States: AI pressed twice, AI with a summary already there, budget spent.

## TDD Approach

- RED: a block test with a mocked `summarizeNote`. A press with no summary sends one PATCH with
  the returned text.
- GREEN: the hook and the button.
- Then: the proposal path with an existing summary, the double press, failure codes, and that no
  request fires on mount.
