# FS-PGG2N: Long-form notes with a short summary

> Status: work-order · SPECIFICATION.md: client `## Notes` "Long-form notes with a short summary" and "AI-written note summary on request"; plan-service `### Nested Items + Notes` "Long-form note body with a summary"; insights-service `## Features` "Note summary generation" → this FS (retrofit; the existing behaviour is client `One-line notes among a plan's items → FS-none` and plan-service `Item type (task | note) …`) · Related FS: FS-65B3N (comments on a note; the note page's comments area and the convert-with-comments rule), FS-043WP (milestone page, the pattern this follows), FS-12ZMF (plan access), FS-F6AEE (duplicate an item), FS-02JBZ (global token budget) · Related ADRs: docs/adr/0019-generation-is-explicit-and-bounded.md, docs/adr/0009-consolidate-to-gateway-plan-insights.md, docs/adr/0014-milestones-are-a-client-reading-of-items.md, docs/adr/0015-plan-access-is-a-relation.md, docs/adr/0004-error-representation-rfc9457.md, docs/adr/0005-request-validation-and-contract-design.md, docs/adr/0007-user-edits-win.md, docs/adr/0001-grpc-identity-via-metadata-rs256.md

## Summary

A note gets room to be written properly. Opening a note takes it to its own **note page**, which
fills the plan's middle card the way the milestone page does. The page shows the note's
**body** on a sheet of paper: plain text that keeps its line breaks, with live links, saved as
you type. The card in the grid stays small. It shows the note's **summary** if it has one, else
the start of the body, clamped to four lines. Leaving a long note that has no summary asks once
for one: write it yourself, have AI write it, or leave anyway. AI writes a summary only when the
user presses the button.

Notes also get a new look, picked by the owner in `/variants` (variant A, "Paper"). They stop
being italic. A warm paper tint sets them apart from tasks instead of a slant and an icon.

Terms (**note**, **note page**, **body**, **summary**, **comment**) are used as
`client/CONTEXT.md` `### Notes` and `services/plan-service/CONTEXT.md` `## Notes` define them.
This FS settles the data shape those entries left open: the body *is* the item's `description`,
and the summary is a new item field `summary`. It introduces **leave prompt** and **card text**
(R17, R4). Both are client terms, routed to `domain-model` rather than defined here.

## Background: what ships today

- A note is a checklist item with `type = 'note'`. It has one text field, `description`, so a
  note is one line. A top-level note is a **note card** in the grid. A note under a milestone is
  a child row in its card and on the milestone page. A note in the list view is a row like a
  task's, without a checkbox.
- Notes are drawn italic. Grid cards use `italic font-normal text-foreground/75` with a
  `FileText` icon where the checkbox would be (`ChecklistGrid.tsx` ~L261, ~L282). Child rows use
  `italic text-foreground/60` (~L430). Milestone page steps use `italic text-foreground/60`
  (`MilestonePage.tsx` ~L684). List rows use `italic text-muted-foreground` (`Todo.tsx` ~L1917).
  `client/src/app/layout.tsx` loads Lora's italic face for "note rows and the add-bar
  placeholder".
- Clicking a note's text renames it inline, in the grid, on the milestone page and in the list.
  The item menu (`ItemActions`) offers Rename, Duplicate, Dates, Make a task, move, indent,
  Archive and Delete.
- The plan's surrounding blocks live in `client/src/app/plan/[planId]/layout.tsx`. The middle
  slot is the Items card or the milestone page (`/plan/<planId>/m/<milestoneId>`, FS-043WP).
- Checklist routes are already Huma-typed (`internal/gateway/plan/typed_checklists.go`), and
  every one asks the access seam for owner-or-member (FS-12ZMF). The item field `intention`
  (FS-043WP R37–R40) is the pattern a new item field follows: nullable column, proto
  `optional string`, settable only through item update, trimmed, empty clears, a length rule as
  a domain error.
- insights-service reads plan items over gRPC in two shapes. `GetPlanContext` (behind
  `getChecklistSuggestion` and `getSuggestedVideos`) puts every item's `description` into the
  prompt, notes included. `guided.NewPlanTree` (behind the day plan and guided suggestions) drops
  every item whose type is not `task`, so notes never reach those prompts or the plan-state hash.
- Every model call is metered: `ai.Kind` names it, the `generations` ledger records its usage,
  and the global daily token budget refuses a call with `LLM_BUDGET_EXHAUSTED` (FS-02JBZ,
  ADR-0019). The ledger's `generations_kind_check` lists the allowed kinds.
- The gateway's own notes system (`notes` table, `internal/notes`, `NotesContainer`,
  `/api/plans/:id/notes*`) is a different, hidden thing. Nothing here touches it.

## Decisions and why

Carried from the scoping session (2026-11-18, Kranti, with feedback from a real user).

**D1. A note page for every note.** Opening a note goes to its own address and swaps only the
plan's middle card, as FS-043WP does. It applies to top-level notes and notes under a milestone.
Rejected: a modal or sheet (cramped for long writing, not linkable) and inline expand in the grid
(long notes break the grid).

**D2. The note is the body.** The text on the card is where the body starts: "it's the actual
note". There is no headline above the body. A first reading with a fixed headline line was
corrected by Kranti.

**D3. The card shows a summary.** A short note shows its own text, as today. A long one shows
its summary, or its body clamped.

**D4. Ask on leave, fall back to the clamp.** Leaving a long note with no summary offers a hand-
written or AI summary, once. Skipping, force-quitting or closing the tab leaves the clamp: "to
save tokens / limit, put the truncated version for now". A browser cannot show custom buttons on
tab close, so the clamp is the only possible outcome there anyway.

**D5. A summary is never touched automatically.** Editing the body never re-prompts or
regenerates. It may go stale; accepted. Rejected: "keep, offer refresh" and "always re-prompt".

**D6. AI is explicit and bounded (ADR-0019).** The model is called only on a press, never on
load, save, or a leave without a press. The call lives in insights-service (ADR-0009). The
gateway only routes it.

**D7. Restyle is first-class.** Notes stop being italic and look like a professional note: the
outside simpler, the inside an editor. Variant A, "Paper", was picked in `/variants`.

### Charter defaults

Approved by the owner (Kranti) under the charter and folded into the requirements, marked
*(charter default)*.

| # | Question | Default |
| --- | --- | --- |
| C1 | Data shape | The item's `description` **is** the body, the whole note. New nullable `summary` field (≤ 280 chars, trimmed, blank clears to null) on `checklist_items`, the `ChecklistItem` proto and the gateway's typed transport, set only through item update (like `intention`). Smallest change: every reader of `description` keeps working, and truncation is a render clamp, not stored derived text |
| C2 | Summary on a task | Only notes carry one. *Chosen: refused.* A non-empty summary sent for a task is a domain error |
| C3 | Card text | `summary ?? description`, clamped to 4 lines |
| C4 | Threshold | 280 characters of body, a named tunable |
| C5 | Leave prompt | In-app navigation only (Back link, other links, the sidebar), at most once per visit, when the body is longer than the threshold and there is no summary. Options: Summarize with AI · Write a summary · Leave anyway. No `beforeunload` |
| C6 | AI summary | New insights-service RPC `SummarizeNote` behind a new gateway endpoint. insights reads the note from plan-service under the plan-access check (owner or member), caps the input (~8k chars), calls the model and **returns text only**. The client saves it through item update. Only on the user's press |
| C7 | AI failure | A `problem+json` code (ADR-0004). The client stays on the page showing "Couldn't summarize" with the other two options |
| C8 | Stale summary | Never auto-touched. Clearing the field removes it |
| C9 | Editor | Plain text with line breaks, URLs rendered as links, autosave (debounced and on leave), meta line "N words · edited ‹time›". No editor library |
| C10 | Empty body | Not saved, the same rule as an empty description today |
| C11 | Route | `/plan/<planId>/n/<noteId>`, swapping the middle card like FS-043WP; inherits plan access (FS-12ZMF) |
| C12 | Ways in | The grid note card (the whole card), child note rows (grid card and milestone page), note rows in the checklist reading |
| C13 | Back | To the milestone page if the note is a milestone's child, else to the plan. Deterministic |
| C14 | Options | The card gets no new controls (its menu stays). The page's menu has Make a task and Delete |
| C15 | Convert | Note → task clears the summary. Refused when the note has comments: that rule is FS-65B3N's |
| C16 | Duplicate | Copies description and summary (FS-F6AEE) |
| C17 | Concurrent edits | Last write wins, as items behave today |
| C18 | Deleted while open | "This note was deleted" with Back |
| C19 | Look (variant A, "Paper") | Upright Lora, no italics on notes, no `FileText` icon on the card. New token `--note-paper` in both themes (light ≈ `hsl(42 70% 95%)`, dark ≈ `hsl(38 22% 17%)`), documented in `client/docs/design-guideline.md`. The note card keeps the milestone card's tile shape, radius and hover glow, filled with `--note-paper`, text at `foreground/88`. Child note rows under a milestone get a small paper chip (radius 6px, padding 2px 8px). The note page: the body on a paper sheet (radius 12, generous padding, Lora ~17px, line-height ~1.75), then an "On the card" summary block (Edit summary / Summarize with AI), then the comments area (FS-65B3N). The italic face stays in `layout.tsx` because the add-bar placeholder still uses it |
| C20 | Insights context | insights' plan-context prompts use `summary ?? first 280 chars` for notes, so token spend stays flat as notes grow |

### Gap defaults

Questions the notes and charter left open at the acceptance-criteria level, settled here as the
smallest sensible v1. Marked *(gap default)* in the requirements.

| # | Question | Default |
| --- | --- | --- |
| G1 | How the page loads | From the plan's long-term item list, the list the grid and the milestone page read. No new read endpoint (FS-043WP G2) |
| G2 | Not here / load failure | As FS-043WP R33–R34: unknown, deleted, archived, not a note, another plan's, or an unreachable plan → "This note isn't here" with a link back; network or 5xx → retryable error |
| G3 | Writing vs reading | The sheet shows the body as text with live links. Clicking the text (not a link), or Enter on the focused sheet, turns it into a plain auto-growing textarea with the caret at the end. Blur returns to reading |
| G4 | Which links | `http://` and `https://` URLs only. They open in a new tab with `rel="noopener noreferrer"`. No other scheme becomes a link |
| G5 | Autosave timing | 800 ms after the last keystroke, on blur, and before any in-app navigation. Unchanged text sends nothing |
| G6 | A failed body save | The text stays in the editor, and the meta line reads "Couldn't save · Try again". In-app leave tries once more. If that fails too, leaving stops with "Couldn't save your changes" and a Leave anyway |
| G7 | Rows: how much text | Child note rows, milestone page note steps and list rows show the card text clamped to 2 lines. One-line places (calendar, menus, tab title, accessible names) use its first line |
| G8 | Menu Rename on a note | Opens the note page with the body in writing mode. Inline rename of a multi-line body in a one-line field would flatten its line breaks |
| G9 | Clicking a note's text | Opens the note page (the way in, C12), in place of today's inline rename. Notes are edited on their page |
| G10 | Summary editing | "Edit summary" opens a textarea in the block: one paragraph (Enter saves, Shift+Enter is ignored), Esc cancels, blur saves, input stops at 280, a live "n/280" counter. Saving empty clears it. Unchanged sends nothing |
| G11 | AI with a summary already there | The result appears in the summary editor as a proposal, saved only when the user presses Save. Cancel keeps the old summary (ADR-0007). With no summary, the result is saved at once: nothing is overwritten |
| G12 | AI after a successful leave-prompt press | Save the summary, then continue to where the user was going |
| G13 | Output longer than 280 | insights cuts it at the last word boundary that fits within 279 characters and adds "…". Never an error |
| G14 | Plain or structured model output | Plain text, one paragraph, the note's own language. Whitespace collapsed |
| G15 | Insights ledger | The call is metered as kind `note_summary`. The ledger's kind check is widened to allow it |
| G16 | A member on a shared plan | Same page, same rights as the owner for body, summary and AI (FS-12ZMF owner-or-member) |
| G17 | Make a task with a summary | No extra confirmation. The server clears the summary in the same update |
| G18 | After Make a task / Delete on the page | Go to the Back destination (C13). Delete uses the existing confirmation |
| G19 | Tab title | The card text's first line, up to 60 characters |
| G20 | Calendar | A dated note shows its card text's first line, not the whole body |

## Requirements

### The data: body and summary

- **R1.** A note's **body** is its item's `description`. No new body field is added, and every
  existing reader of `description` keeps working. *(charter default C1)*
- **R2.** Checklist items gain a nullable text field `summary`. It is writable only through the
  existing item update: omit it to leave it unchanged, or send a string to set it. plan-service
  trims the string and stores an empty result as null. A trimmed value longer than 280 characters
  (counted as characters, not bytes) is refused as a domain error. *(charter default C1)*
- **R3.** Only notes carry a summary. plan-service refuses a non-empty summary on an item that is
  a task, or that the same update makes a task, as a domain error. An empty summary sent for a
  task is accepted and changes nothing. Storage enforces the same rule: a summary is null unless
  the item is a note. *(charter default C2)*
- **R4.** The **card text** of a note is its summary if it has one, else its body. The client
  computes it in one helper, used by every place that draws a note outside its page. No derived or
  truncated text is stored. *(charter defaults C1, C3)*
- **R5.** `summary` is not accepted at create. The create body stays as it is. It is returned on
  every item response, omitted when null, like `intention`.
- **R6.** Converting a note to a task clears its summary in the same update, the way converting
  a done task to a note clears `done`. Converting a task to a note leaves the summary null. A
  note that has comments cannot be converted: that rule is FS-65B3N's, and this FS adds nothing
  to it. *(charter default C15)*
- **R7.** Duplicating a note copies its body and its summary (extends FS-F6AEE). *(charter
  default C16)*
- **R8.** A body edit never changes the summary, and nothing changes a summary automatically:
  only an explicit item update writes it. Clearing the field is how a summary is removed.
  *(charter defaults C8, D5)*
- **R9.** The change ships end to end. plan-service gets one migration: a nullable `TEXT` column
  with a 280-character check and a note-only check. Then the proto (`ChecklistItem.summary`,
  `UpdateItemRequest.summary`), plan-service read, update and duplicate paths, the gateway's
  update request and item response types, the regenerated `openapi.yaml` and the regenerated
  client types. Adding an optional field to a request and a response is non-breaking, and the
  contract gates must pass.

### The card and rows

- **R10.** A note card in the grid shows the card text clamped to 4 lines, keeping its line
  breaks. The tunable is named `NOTE_CARD_LINES = 4`. *(charter default C3)*
- **R11.** Child note rows (in a milestone card and on the milestone page) and note rows in the
  checklist reading show the card text clamped to 2 lines (`NOTE_ROW_LINES = 2`). One-line places
  use the card text's first line: the calendar's chip and title, the menu's accessible name, the
  delete confirmation, the tab title. *(gap defaults G7, G20)*
- **R12.** Clicking anywhere on a note card opens its note page, except on its menu and drag
  handle. A drag never opens. The card carries a real link named "Open note: ‹first line›", so it
  can be reached by keyboard, opened in a new tab and copied. *(charter default C12)*
- **R13.** Clicking a child note row's text, in a milestone card or on the milestone page, opens
  that note's page. So does clicking a note row's text in the checklist reading. Each is a real
  link. The inline rename these clicks start today is replaced for notes only. Tasks and
  milestones keep theirs. *(charter default C12, gap default G9)*
- **R14.** A note card gains no new controls. Its item menu keeps its entries. For a note, the
  menu's **Rename** opens the note page with the body in writing mode instead of renaming inline.
  *(charter default C14, gap default G8)*
- **R15.** The archived list renders archived notes as today. They have no note page (R36), so
  their text is not a link.

### The note page

- **R16.** A note page lives at `/plan/<planId>/n/<noteId>`. It fills the middle slot of the
  plan layout (`plan/[planId]/layout.tsx`) inside the same card, as the milestone page does.
  Moving between the plan, a milestone page and a note page swaps only the middle slot and
  refetches none of the surrounding blocks. *(charter default C11, FS-043WP R2–R3)*
- **R17.** The middle card's header carries a **Back** link in the milestone page's back style.
  A note whose parent is a milestone goes back to `/plan/<planId>/m/<parentId>`, labelled with
  the milestone's title. Any other note goes back to `/plan/<planId>`, labelled "Back to plan".
  The destination comes from the note's parent and never from history. Browser back also works.
  *(charter default C13)*
- **R18.** The page loads from the plan's long-term item list, the list the grid and the
  milestone page read. No read endpoint is added. While it loads, the middle card shows a quiet
  loading line. *(gap default G1)*
- **R19.** The page shows, in order:
  1. the header (Back, and the page's menu);
  2. the **body** on a paper sheet;
  3. the meta line;
  4. the **"On the card"** block;
  5. the comments area, which FS-65B3N fills. Until FS-65B3N ships, nothing renders there.
  *(charter default C19)*
- **R20.** The page's item menu offers **Make a task** and **Delete** only. Make a task converts
  the note (R6). On success it goes to the Back destination, and a refusal (FS-65B3N's comments
  rule, or any other) stays on the page with the existing error. Delete asks for the existing
  confirmation. On success it goes to the Back destination, and a failure stays with the error.
  *(charter default C14, gap defaults G17, G18)*
- **R21.** The browser tab's title is the card text's first line, up to 60 characters, while the
  page is open. *(gap default G19)*
- **R22.** Any plan member opens, edits and summarizes a note exactly as the owner does. The page
  reads and writes only through the plan's existing checklist routes and the summary endpoint,
  so whoever those serve, it serves. This FS adds no access rule. *(charter default C11, gap
  default G16)*

### The body editor

- **R23.** The body is plain text. Line breaks are kept as typed, and there is no formatting,
  toolbar or markdown. In reading mode, `http://` and `https://` URLs in the body render as links
  that open in a new tab with `rel="noopener noreferrer"`. No other scheme becomes a link.
  Clicking the text (not a link), or Enter on the focused sheet, switches to writing mode: a
  plain auto-growing textarea with the caret at the end. Blur returns to reading. No editor
  library is added. *(charter default C9, gap defaults G3, G4)*
- **R24.** The body saves itself through the item update (`description`): 800 ms after the last
  keystroke, on blur, and before any in-app navigation away. Saving unchanged text sends
  nothing. While a save is in flight the meta line shows "Saving…". *(charter default C9, gap
  default G5)*
- **R25.** A body that is empty or only whitespace is not saved. The stored body stays as it
  was, and leaving or reloading shows it again. *(charter default C10)*
- **R26.** The meta line reads "‹N› words · edited ‹relative time›". N counts the
  whitespace-separated words of the current text. The time is the item's `updatedAt`. *(charter
  default C9)*
- **R27.** A failed save keeps the text in the editor and shows "Couldn't save · Try again" in
  the meta line. Before an in-app navigation away, one more save is tried. If it fails, the
  navigation stops and the page shows "Couldn't save your changes" with a **Leave anyway** that
  continues without saving. *(gap default G6)*

### The "On the card" block

- **R28.** Below the meta line, an **"On the card"** block shows what the card shows. With a
  summary, it shows the summary. With none, it shows "No summary yet. The card shows the start of
  the note." It offers **Edit summary** (or **Write a summary** when there is none) and
  **Summarize with AI**. *(charter default C19)*
- **R29.** Editing the summary opens a textarea in the block. It is one paragraph: Enter saves
  and Shift+Enter is ignored. Blur saves and Esc cancels. Input stops at 280 characters, and an
  "n/280" counter shows the length. Saving sends the trimmed text through the item update, and
  saving empty clears the summary. Saving unchanged text sends nothing. The save is optimistic
  and reverts on failure with the existing error. *(charter defaults C1, C8, gap default G10)*
- **R30.** Pressing **Summarize with AI** calls the summary endpoint (R38) for this note. While
  it runs, the button reads "Summarizing…" and is disabled, so a second press cannot start a
  second call. With no summary, the returned text is saved through the item update at once. With
  a summary, the returned text opens in the summary editor as a proposal, saved only when the
  user presses Save; Cancel keeps the old one. *(charter default C6, gap default G11, ADR-0007)*
- **R31.** A failed AI call leaves the summary unchanged. The block shows "Couldn't summarize"
  and keeps Edit/Write a summary. When the code is `LLM_BUDGET_EXHAUSTED`, it adds "AI summaries
  are paused until tomorrow". The client switches on `code`, never on `detail`. *(charter default
  C7, ADR-0004)*

### The leave prompt

- **R32.** The **leave prompt** opens when an in-app navigation away from the note page starts
  (its Back link, any other link in the app, the sidebar) and all of these hold:
  - the body's trimmed length is more than `NOTE_SUMMARY_THRESHOLD = 280` characters, judged on
    the current text after the pending save has flushed;
  - the note has no summary;
  - the prompt has not opened yet in this visit. A visit is one mount of the page for one note.

  Browser back and forward, reload, closing the tab and leaving the site are not intercepted, and
  no `beforeunload` handler is added: the card falls back to the clamp. *(charter defaults C4,
  C5, D4)*
- **R33.** The prompt offers three choices:
  - **Summarize with AI.** Runs R30 for a note with no summary. On success the summary is saved
    and the navigation continues to where the user was going.
  - **Write a summary.** Closes the prompt, drops the navigation and opens the summary editor
    with focus.
  - **Leave anyway.** Continues the navigation.

  Esc or clicking outside the prompt means staying: the navigation is dropped. *(charter default
  C5, gap default G12)*
- **R34.** If the AI call fails, the prompt stays open and shows "Couldn't summarize" (with R31's
  budget line when it applies) and the other two choices only. If saving the returned summary
  fails, the prompt shows "Couldn't save the summary" with the same two choices. *(charter
  default C7)*
- **R35.** The prompt opens at most once per visit, whatever was chosen. A later navigation in
  the same visit leaves without asking. *(charter default C5)*

### Not here, deleted, and failures

- **R36.** The page shows a calm **"This note isn't here"** state with a link to the plan when
  the id is not in the plan's long-term list (unknown, deleted, archived), is in it but is not a
  note, belongs to another plan, or cannot be loaded because the plan cannot be reached (the
  list answers 404, 403 or 422). Any other load failure (network, 5xx, 503) shows "Couldn't load
  this note" with Try again. *(gap default G2)*
- **R37.** While the page is open, a body save, a summary save or an AI call that answers 404
  replaces the page with **"This note was deleted"** and the Back link. The page does not watch
  for changes otherwise: a note archived, converted or edited elsewhere shows on its next load.
  Concurrent edits are last write wins, as items are today. *(charter defaults C17, C18)*

### The AI summary (insights-service and the gateway)

- **R38.** A new gateway operation `summarizeNote` (`POST /api/insights/note-summary`, body
  `planId`, `itemId`) asks the access seam for owner-or-member first, as every insights operation
  does. It then forwards to insights-service's new `SummarizeNote` RPC with the caller's
  identity (ADR-0001), and answers `{ summary }`. It writes nothing. *(charter default C6)*
- **R39.** `SummarizeNote` asks plan-service for the caller's relation to the plan and refuses
  `none` as NotFound (ADR-0015: callers enforce). It then reads the item with `GetItem` in that
  plan. A missing item, another plan's item or an archived one is NotFound. An item that is not a
  note is refused with `ITEM_NOT_A_NOTE`. *(charter default C6)*
- **R40.** The model sees only the body: the first `NOTE_SUMMARY_INPUT_CAP = 8000` characters of
  `description`. Not the summary, not comments, not other items. It is asked for one plain
  paragraph of at most 280 characters in the note's own language. *(charter default C6, gap
  default G14)*
- **R41.** The call is one metered model call of kind `note_summary`. It checks the global daily
  token budget first and refuses with `LLM_BUDGET_EXHAUSTED` when it is spent. It records its
  usage in the `generations` ledger like every generator (FS-02JBZ, ADR-0019). There is no
  per-note cap in v1: the global budget bounds it. *(gap default G15)*
- **R42.** The output is trimmed and its whitespace collapsed to single spaces. Output longer
  than 280 characters is cut at the last word boundary that fits in 279 and given a trailing "…".
  Empty output, a provider error or a timeout is refused with `NOTE_SUMMARY_FAILED`. *(charter
  default C7, gap default G13)*
- **R43.** insights-service returns the text and stores nothing. It writes nothing to
  plan-service and publishes no event. Saving is the client's item update (R30). *(charter
  default C6)*
- **R44.** No model call happens on a page load, a read, an autosave, a leave without a press, or
  a job. The only trigger is a press of Summarize with AI, on the page or in the leave prompt.
  *(D6, ADR-0019)*

### Insights context

- **R45.** Wherever an insights prompt includes a note's text, it uses the note's summary, or
  else the first 280 characters of its body. Today that means `GetPlanContext`'s checklist lines
  behind `getChecklistSuggestion` and `getSuggestedVideos`. The day plan and guided suggestions
  already leave notes out of their tree and hash (`guided.NewPlanTree`). That stays, so a long
  note changes neither their prompts nor their regeneration. *(charter default C20)*

### The look (variant A, "Paper")

- **R46.** A new token `--note-paper` is defined in both themes in `client/src/app/globals.css`:
  light ≈ `hsl(42 70% 95%)`, dark ≈ `hsl(38 22% 17%)`. It is documented in
  `client/docs/design-guideline.md` with its usage rule: the surface of a note and nothing else.
  *(charter default C19)*
- **R47.** A note card keeps the milestone card's tile shape, radius, padding and hover glow. Its
  fill is `--note-paper` and its text upright Lora at `foreground/88`. It has no `FileText` icon
  and no italics. *(charter default C19)*
- **R48.** Child note rows under a milestone, in a card and on the milestone page, set their text
  upright in a paper chip: `--note-paper` fill, radius 6px, padding 2px 8px. Note rows in the
  checklist reading use the same upright text and chip. No note is italic anywhere. *(charter
  default C19)*
- **R49.** On the note page the body sits on a sheet of `--note-paper`: radius 12px, generous
  padding (about 28px, less below `md`), Lora about 17px, line-height about 1.75. The "On the
  card" block and the comments area sit below it on the card's own surface. *(charter default
  C19)*
- **R50.** Tasks, milestones, the composer and the add-bar are not restyled. The add-bar
  placeholder stays italic, so Lora's italic face stays loaded in `client/src/app/layout.tsx`.
  Only its comment changes, to say that the placeholder is now its only user. The `FileText`
  icon stays where it marks the note type in the composer and the menu. *(charter default C19)*

## User Stories

1. As a planner, I want a note to have a page of its own, so that I have room to write more than
   one line.
2. As a planner, I want that page to have an address, so that I can bookmark a note and send it
   to a plan member.
3. As a planner, I want to open a note by clicking its card anywhere, so that getting in takes no
   hunting.
4. As a planner, I want to open a note under a milestone from the milestone's card or its page,
   so that every note works the same way.
5. As a planner using the checklist view, I want to open a note from its row, so that the list
   isn't a dead end for notes.
6. As a planner, I want the text I typed on the card to be the start of the note, so that I keep
   writing where I left off.
7. As a planner, I want my line breaks kept, so that a note can have paragraphs and lists.
8. As a planner, I want links in my note to be clickable, so that sources I paste are one click
   away.
9. As a planner, I want my writing saved as I type, so that I never lose a note by navigating
   away.
10. As a planner, I want to see how long the note is and when it was edited, so that I know what
    I'm looking at.
11. As a planner, I want the card to stay small however long the note gets, so that the grid
    stays readable.
12. As a planner, I want a short summary on the card, so that I can tell what a long note is
    about without opening it.
13. As a planner, I want to write that summary myself, so that it says exactly what I mean.
14. As a planner, I want AI to write the summary when I ask, so that I don't have to.
15. As a planner, I want AI never to run unless I press the button, so that nothing spends tokens
    behind my back.
16. As a planner, I want a gentle reminder when I leave a long note with no summary, so that the
    card doesn't end up with a cut-off sentence.
17. As a planner, I want that reminder only once per visit and only inside the app, so that it
    never nags or traps me.
18. As a planner, I want to leave anyway, so that the reminder is never a wall.
19. As a planner, I want my summary never changed for me, so that what I wrote stays mine.
20. As a planner, I want AI not to overwrite a summary I already have without my say, so that I
    can compare and keep mine.
21. As a planner, I want a failed AI summary to keep me on the page with my other options, so
    that a provider outage doesn't lose me anything.
22. As a planner, I want Back to take me to the milestone a note belongs to, so that I return to
    where I came from.
23. As a planner, I want to turn a note into a task or delete it from its page, so that I don't
    have to go back to the grid to do it.
24. As a planner, I want duplicating a note to copy its summary, so that the copy looks the same
    on its card.
25. As a planner on a shared plan, I want a member to read, edit and summarize notes as I do, so
    that we can work on them together.
26. As a planner, I want a calm message when a note was deleted while I had it open, so that I'm
    not left typing into nothing.
27. As a planner, I want notes to look like paper rather than slanted grey text, so that they read
    as notes and stay easy to read.
28. As a planner, I want a dated note in the calendar to show its first line, so that the calendar
    doesn't fill with a whole essay.
29. As a keyboard or screen-reader user, I want note cards and rows to be named links, so that I
    can open a note without a mouse.
30. As the owner, I want long notes not to raise the tokens every suggestion spends, so that the
    budget holds as notes grow.
31. As a developer, I want the body to stay in `description`, so that every existing reader keeps
    working without a migration of data.

## Acceptance Criteria

- [ ] PATCH with `summary` on a note sets it (trimmed). An empty or whitespace string clears it
      to null. Omitting it leaves it unchanged. Over 280 characters is `400 · VALIDATION_FAILED`.
- [ ] PATCH with a non-empty `summary` on a task, or with `type: "task"` and a non-empty
      `summary`, is `400 · VALIDATION_FAILED`. An empty `summary` on a task succeeds and changes
      nothing.
- [ ] PATCH `type: "task"` on a note with a summary returns the item with no `summary`.
- [ ] Create rejects a `summary` member (`422 · VALIDATION_FAILED`).
- [ ] Every item response carries `summary` when set and omits it when null.
- [ ] Duplicating a note copies its `description` and `summary`.
- [ ] Migration up and down apply cleanly. Existing items read `summary` as null, and the
      database refuses a summary on a task and one over 280 characters.
- [ ] `openapi.yaml` and the client's generated types are regenerated, and `make gates` passes
      with no breaking change.
- [ ] A note card shows `summary ?? description`, clamped to 4 lines with line breaks kept. Child
      note rows, milestone page note steps and list note rows clamp to 2.
- [ ] Clicking a note card anywhere but its menu or handle opens `/plan/<planId>/n/<noteId>`. A
      drag does not open it. The card is a link named "Open note: ‹first line›".
- [ ] Clicking a child note row's text (grid card, milestone page) or a list note row's text
      opens its note page. Tasks and milestones still rename inline.
- [ ] A note's menu Rename opens its note page in writing mode. The card has no new controls.
- [ ] The note page renders in the plan's middle card. Moving plan → note → plan, or milestone →
      note → milestone, refetches none of the surrounding blocks.
- [ ] Back goes to the milestone page for a milestone's child note, else to the plan, whatever
      the history.
- [ ] The page shows header, body sheet, meta line, "On the card" block, then the comments slot,
      in that order.
- [ ] The page's menu offers exactly Make a task and Delete. Both go to the Back destination on
      success and stay with the error on failure.
- [ ] Reading mode renders line breaks and `http(s)` links (new tab, `noopener noreferrer`). A
      `javascript:` or other scheme is shown as plain text.
- [ ] Clicking the text enters writing mode. Typing autosaves 800 ms after the last keystroke and
      on blur. Unchanged text sends nothing.
- [ ] Clearing the body to empty or whitespace sends no save, and the old body returns on reload.
- [ ] The meta line shows "N words · edited ‹time›", "Saving…" in flight, and "Couldn't save ·
      Try again" on failure.
- [ ] A pending save flushes before in-app navigation. A failed flush stops the navigation with
      "Couldn't save your changes" and Leave anyway.
- [ ] Edit summary edits in place: Enter or blur saves, Esc cancels, input stops at 280 with a
      counter, empty clears. A failed save reverts.
- [ ] Summarize with AI with no summary saves the result. With a summary, it shows the result as
      a proposal that only Save keeps. The button is disabled while the call runs.
- [ ] A failed AI call shows "Couldn't summarize" and keeps the summary unchanged.
      `LLM_BUDGET_EXHAUSTED` adds the paused line.
- [ ] The leave prompt opens on an in-app navigation (Back, a link, the sidebar) only when the
      body is over 280 characters and has no summary, and at most once per visit.
- [ ] Leave prompt: Summarize with AI saves then continues to the destination. Write a summary
      stays and focuses the summary editor. Leave anyway continues. Esc stays.
- [ ] Leave prompt AI failure shows "Couldn't summarize" with Write a summary and Leave anyway
      only.
- [ ] Browser back, reload and tab close are not intercepted, and no `beforeunload` handler
      exists.
- [ ] Unknown, deleted, archived, task, other-plan and unreachable ids show "This note isn't
      here". Network or 5xx shows a retry.
- [ ] A save or AI call answering 404 while open shows "This note was deleted" with Back.
- [ ] `POST /api/insights/note-summary` returns `{ summary }` of at most 280 characters for an
      owner or member, and writes nothing to plan-service.
- [ ] It answers `404 · NOT_FOUND` for a missing, archived or other-plan item, `400 ·
      ITEM_NOT_A_NOTE` for a task, `429 · LLM_BUDGET_EXHAUSTED` when the budget is spent, and
      `503 · NOTE_SUMMARY_FAILED` on a model failure, timeout or empty output.
- [ ] `SummarizeNote` refuses a caller whose relation is `none`, sends the model at most 8000
      characters of body, and records one `note_summary` generation in the ledger.
- [ ] No model call happens on page load, any GET, autosave, or a leave without a press.
- [ ] `getChecklistSuggestion` and `getSuggestedVideos` prompts carry a note's summary, or else
      its first 280 body characters, never the full body.
- [ ] `--note-paper` exists in both themes in `globals.css` and is documented in
      `design-guideline.md`.
- [ ] Note cards: `--note-paper` fill, milestone tile shape and glow, upright text at
      `foreground/88`, no `FileText` icon. Child and list note rows: upright text in a paper chip
      (radius 6px, padding 2px 8px).
- [ ] No note text is italic anywhere. The add-bar placeholder is still italic, and tasks and
      milestones look as before.
- [ ] A dated note in the calendar shows its card text's first line.
- [ ] The tab title is the note's first line (≤ 60 characters) on its page.
- [ ] Tests pass.

## Edge States

| Situation | Behavior |
| --- | --- |
| Short note, no summary | Card shows the body. Leaving never prompts (R32). |
| Long note, no summary, user only reads it | Leaving in-app prompts once in that visit: the rule is length and no summary (R32). |
| Long note with a summary | Card shows the summary. Leaving never prompts. |
| Body edited back under 280 while a summary exists | Card still shows the summary (R4, R8). |
| Body edited after a summary was written | Summary unchanged and possibly stale. No prompt, no regeneration (R8). |
| Summary cleared by hand | Card shows the body clamped. The next in-app leave of a long body may prompt, if not yet this visit. |
| Body cleared to empty | Not saved. The stored body stays (R25). |
| Body only whitespace | Treated as empty (R25). |
| Very long body (past 8000 characters) | Saved whole. AI reads the first 8000 (R40). The card clamps. |
| Body with an unbroken long URL | Wraps on the sheet and the card (break-words) and never widens the card (FS-49CQ2 R23). |
| `javascript:` or `mailto:` text in the body | Plain text, not a link (R23). |
| Autosave fails | Text kept. "Couldn't save · Try again" (R27). |
| Autosave fails again on leave | Navigation stops with Leave anyway (R27). |
| Leave prompt opened, user picks Write a summary, then leaves without writing | No second prompt this visit (R35). |
| Leave prompt, AI succeeds, saving the summary fails | Prompt stays: "Couldn't save the summary", Write a summary, Leave anyway (R34). |
| Leave prompt, AI fails (provider, timeout, empty output) | "Couldn't summarize" with two choices (R34). |
| AI with the budget spent | `429 · LLM_BUDGET_EXHAUSTED`. "Couldn't summarize" plus the paused line (R31). |
| AI output over 280 characters | Cut at a word boundary with "…" (R42). |
| AI pressed twice quickly | One call: the button is disabled in flight (R30). |
| AI pressed with a summary already there | Proposal in the editor. Save keeps it, Cancel keeps the old (R30). |
| Browser back, reload, tab close on a long note with no summary | Not intercepted. The card shows the clamp (R32). |
| Note deleted by a member while open | Next save or AI call answers 404: "This note was deleted" with Back (R37). |
| Note archived elsewhere while open | Saves may still succeed. "Not here" on the next load (R36, R37). |
| Note converted to a task elsewhere while open | Body saves still succeed (last write wins). A summary save answers `400 · VALIDATION_FAILED` with the existing error. "Not here" on the next load. |
| Two members edit the same body | Last write wins, as items today (R37). |
| Member removed from the plan while the page is open | Saves answer 403 or 404: 404 shows "deleted", 403 shows "not here" (R36, R37). |
| Id is a task, a milestone, archived, unknown or another plan's | "This note isn't here" (R36). |
| Plan id not a uuid | List answers 422: "not here" (R36). |
| Network or 5xx on load | "Couldn't load this note" with Try again (R36). |
| Note under a milestone | Back goes to that milestone's page (R17). |
| Note outdented from a milestone in another tab | Back still goes to the milestone the page loaded with, until the next load. |
| Make a task on a note with a summary | Summary cleared by the server. No confirmation (R6, G17). |
| Make a task on a note with comments | Refused by FS-65B3N's rule. The page stays with the error (R20). |
| Make a task, later Make a note | Summary stays null. Nothing is restored (R6). |
| Duplicate a note with a summary | Copy has the same body and summary (R7). |
| Summary sent at create | `422 · VALIDATION_FAILED` (R5). |
| Summary sent for a task by a direct API call | `400 · VALIDATION_FAILED` (R3). |
| Archived note in the archived list | Not a link. No note page (R15). |
| Dated note in the calendar | First line of the card text (R11). |
| Member on a shared plan | Same page, edits and AI as the owner (R22). |
| Day plan / guided suggestions with long notes | Unaffected: notes are not in their tree or hash (R45). |
| Narrow screen (below `md`) | Sheet padding shrinks. The prompt is a dialog that fits 375px. |

## API surface

One new operation. One field added to existing checklist operations, otherwise established
patterns (FS-0004, FS-043WP's `intention`).

| Op | Method + Path | Query/Params | Request body | Response | Errors |
|----|---------------|--------------|--------------|----------|--------|
| updateChecklist | PATCH `/api/plans/{id}/checklists/{checklist_id}` | unchanged | `UpdateChecklistItemRequest` gains optional `summary: string`. Omit = leave unchanged. A string sets it after trimming. Empty after trim clears it to null. No `maxLength` in the schema: the 280 limit and the note-only rule are domain rules (ADR-0005). | `ChecklistItemResponse` with `summary` | unchanged, plus: trimmed `summary` over 280 characters → `400 · VALIDATION_FAILED`; non-empty `summary` on a task, or with `type: "task"` in the same body → `400 · VALIDATION_FAILED` |
| every op returning `ChecklistItemResponse` (list, archived, upcoming, get, create, update, duplicate, dates, archive, reorder) | unchanged | unchanged | unchanged | `ChecklistItemResponse` gains optional `summary: string`, omitted when null | unchanged |
| createChecklist | POST `/api/plans/{id}/checklists` | unchanged | unchanged: `summary` is **not** a member | unchanged | sending `summary` → `422 · VALIDATION_FAILED` (strict body, ADR-0005) |
| duplicateChecklist | unchanged | unchanged | unchanged | copies carry the source's `summary` | unchanged |
| getCalendar | GET `/api/plans/{id}/calendar` | unchanged | — | `CalendarItemResponse` gains required `type: "task" \| "note"` and optional `summary: string`, omitted when null, so a dated note's chip shows the card text's first line (R11, G20). Additive | unchanged |
| summarizeNote | POST `/api/insights/note-summary` | none | `SummarizeNoteRequest`: `planId: uuid` (required), `itemId: uuid` (required). Identity comes from the JWT, never the body. | `200` `{ summary: string }`: one paragraph, 1–280 characters. Nothing is stored | `401 · UNAUTHENTICATED`; the access seam's existing answers when the caller is neither owner nor member (FS-12ZMF); `404 · NOT_FOUND` item missing, archived or in another plan; `400 · ITEM_NOT_A_NOTE` item is a task; `422 · VALIDATION_FAILED` malformed or missing ids, unknown members; `429 · LLM_BUDGET_EXHAUSTED` daily token budget spent; `503 · NOTE_SUMMARY_FAILED` model error, timeout or empty output (retrying is correct); `503 · SERVICE_UNAVAILABLE` insights-service unreachable |

New `common/errcode` codes:
- `ITEM_NOT_A_NOTE` (400). Decided by insights-service and sent as an ErrorInfo detail on
  `FailedPrecondition`.
- `NOTE_SUMMARY_FAILED` (503). Decided by insights-service and sent as an ErrorInfo detail on
  `Unavailable`.

The client tells the two 503s apart by `code` only.

gRPC:
- plan-service (`common/api/proto/plan/checklist.proto`):
  - `ChecklistItem` gains `optional string summary = 15`.
  - `UpdateItemRequest` gains `optional string summary = 12`. Unset leaves it. Set is trimmed,
    and empty clears to NULL.
  - A trimmed value over 280 characters, or a non-empty value for an item that is or becomes a
    task, is `InvalidArgument`, which the gateway maps to `400 · VALIDATION_FAILED`.
  - `type` → `task` clears `summary` in the same write.
- insights-service (`common/api/proto/insights/insights.proto`):
  - New `rpc SummarizeNote(SummarizeNoteRequest) returns (SummarizeNoteResponse)`.
  - `SummarizeNoteRequest { string plan_id = 1; string user_id = 2; string item_id = 3; }`.
  - `SummarizeNoteResponse { string summary = 1; }`.
  - Refusals:
    - relation `none`, or an item that is missing, archived or in another plan → `NotFound`;
    - not a note → `FailedPrecondition` with `ITEM_NOT_A_NOTE`;
    - budget spent → `ResourceExhausted` with `LLM_BUDGET_EXHAUSTED`;
    - model failure, timeout or empty output → `Unavailable` with `NOTE_SUMMARY_FAILED`.

Storage:
- plan-service, one migration:
  - `checklist_items.summary TEXT NULL`;
  - `CHECK (summary IS NULL OR char_length(summary) <= 280)`;
  - `CHECK (type = 'note' OR summary IS NULL)`, the twin of `chk_note_not_done`.
  - Take the next free number when the slice lands. FS-65B3N adds a plan-service migration in
    parallel.
  - Down drops both constraints and the column.
- insights-service: the `generations_kind_check` constraint is widened to allow `note_summary`
  (R41). No table is added. Down restores the previous list.

## Out of Scope

- **Comments** on a note: the comments area, its endpoints and the convert-with-comments refusal.
  These are FS-65B3N's. This FS only leaves the slot and defers to its rule.
- **Rich text**: markdown, headings, lists, bold, an editor library, a toolbar.
- **A headline or title** above the body (rejected, D2).
- **A `beforeunload` prompt**, or intercepting browser back, reload or tab close.
- **Auto-generated or refreshed summaries**: on load, on save, on leave without a press, by job,
  or when the body changes.
- **A per-note AI cap** or a summary history. The global budget is the bound in v1.
- **AI reading comments**, the plan, or other items to write a summary.
- **Live updates** from other tabs or members, merge or conflict UI. Last write wins.
- **A body length limit.**
- **Summaries on tasks or milestones.**
- **Showing on the card** that a note has comments (FS-65B3N's question).
- **The gateway's hidden notes system** (`notes` table, `internal/notes`, `NotesContainer`,
  `/api/plans/:id/notes*`): not touched, not folded in, not retired here.
- **Restyling tasks, milestones, the composer or the add-bar.**
- **Search over note bodies.**
- **Events** for summary or body changes.
