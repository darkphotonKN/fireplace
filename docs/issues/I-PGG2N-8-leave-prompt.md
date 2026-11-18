---
id: I-PGG2N-8
status: done
implements: FS-PGG2N
blocked_by: [I-PGG2N-7]
labels: [feature]
title: "FS-PGG2N slice 8: leaving a long note without a summary asks for one, once"
---
Implements FS-PGG2N §Requirements (R27, R32–R35)

## What to Build

An in-app navigation guard on the note page. It does two things: it flushes the body before
leaving, and it opens the **leave prompt** when a long note has no summary.

- **The guard.**
  - Intercept in-app navigations that start while the page is mounted: the Back link, any other
    `<Link>` or anchor inside the app (the sidebar included), and programmatic pushes the page
    itself starts.
  - Next's App Router has no navigation-blocking API. Use a capture-phase click handler on
    same-origin anchors plus a wrapped `router.push` for the page's own pushes. Keep it to one
    small, tested module, such as `client/src/hooks/useLeaveGuard.ts`.
  - Do **not** intercept browser back/forward, reload or tab close.
  - Add no `beforeunload` handler (R32).
- **Flush first (R27).**
  - Any pending body save is sent before deciding.
  - If it fails, one more save is tried.
  - If that fails too, the navigation stops, and the page shows "Couldn't save your changes"
    with **Leave anyway**, which continues without saving.
- **When the prompt opens (R32, R35).** All of these hold:
  - the body's trimmed length is more than `NOTE_SUMMARY_THRESHOLD` (280);
  - the note has no summary;
  - the prompt has not opened yet in this visit (one mount for one note id).

  Otherwise the navigation simply continues. After the prompt has opened once, later navigations
  in the visit never ask again, whatever was chosen.
- **The prompt (R33).** An accessible dialog that fits 375px, with three choices:
  - **Summarize with AI.** Reuse slice 7's hook. On success, save the summary, then continue to
    the intended destination.
  - **Write a summary.** Close the prompt, drop the navigation, and open the summary editor with
    focus (slice 5's opener).
  - **Leave anyway.** Continue.

  Esc or clicking outside means staying: the navigation is dropped.
- **Failure (R34).**
  - The AI call fails → the prompt stays and shows "Couldn't summarize" (plus the
    `LLM_BUDGET_EXHAUSTED` line when it applies) with Write a summary and Leave anyway only.
  - The returned summary fails to save → "Couldn't save the summary" with the same two choices.

Files expected: `client/src/hooks/useLeaveGuard.ts` (+ test), `client/src/components/NotePage.tsx`,
a `NoteLeavePrompt.tsx` or similar (+ tests). Reuse an existing dialog primitive from
`components/ui` if one fits.

## Acceptance Criteria

- [ ] A 281-character body with no summary: clicking Back, another app link or the sidebar opens
      the prompt. A 280-character body does not.
- [ ] A note with a summary never prompts.
- [ ] The prompt opens at most once per visit. A later navigation in the same visit leaves
      without asking.
- [ ] Summarize with AI saves the summary, then lands on the intended destination.
- [ ] Write a summary stays on the page with the summary editor focused.
- [ ] Leave anyway continues. Esc or outside-click stays.
- [ ] An AI failure shows "Couldn't summarize" with only Write a summary and Leave anyway. A save
      failure shows "Couldn't save the summary".
- [ ] A pending body save flushes before leaving. A failed flush stops the navigation with
      "Couldn't save your changes" and Leave anyway.
- [ ] Browser back, reload and tab close are not intercepted, and no `beforeunload` listener is
      registered.
- [ ] `npm test`, `npm run lint` and `npm run build` pass.

## Blocked By

I-PGG2N-7 (the AI hook the prompt reuses). It also uses slice 5's summary editor opener.

## Spec Reference

FS-PGG2N:
- §Requirements R27 (second half), R32–R35 (charter defaults C4, C5, C7; gap defaults G6, G12;
  decision D4).
- §Acceptance Criteria: the leave prompt and flush items.
- §Edge States: the leave prompt rows, reading a long note, browser back or tab close, autosave
  failing again on leave.

## TDD Approach

- RED: a `useLeaveGuard` test. A click on a same-origin anchor calls the guard and does not
  navigate until it resolves. A click on an external link is untouched.
- GREEN: the capture handler and the wrapped push.
- Then: `NotePage` tests for the threshold boundary (280 vs 281), once per visit, each choice,
  each failure, and the flush-failure stop.
