# FS-043WP: Milestone page

> Status: work-order · SPECIFICATION.md: client `## Checklists` "Milestone page" → this FS · Related FS: FS-FSFEC (milestones, progress, completion nudge), FS-12ZMF (plan sharing: who can reach a plan, and so its milestones) · Related ADRs: docs/adr/0014-milestones-are-a-client-reading-of-items.md, docs/adr/0002-contract-planes-code-first-openapi.md, docs/adr/0005-request-validation-and-contract-design.md

## Summary

A milestone gets its own page at its own URL: a quiet room for one block of progress, with room
to think. Opened from its card in the grid, the page takes over the plan's middle card (the way
Settings does today) and shows the milestone's title, dates and progress, *why it matters*, the
*next step*, and every step with space to breathe. Back returns to the plan. Creating a milestone
stays exactly as simple as it is now: nothing on the page is asked for at create.

Terms (`milestone`, `step`, `completion nudge`, `milestone reading`) are used as
`client/CONTEXT.md` defines them. This FS introduces **milestone page**, **why it matters**
(stored as the item field `intention`) and **next step**; they are routed to `domain-model`, not
defined here.

## Background: what ships today

- The plan page (`client/src/app/plan/[planId]/page.tsx`) is one column of blocks: the plan's
  title card, the **Daily** card, the long-term **Items** card (`<Todo fixedTaskType="longterm">`),
  the Calendar, and Recommended videos. There is no `layout.tsx` under `plan/[planId]`.
- Settings is `showSettings` state inside `Todo.tsx`: it swaps the Items card's contents for the
  settings panel and puts a "Back to tasks" button above it. No URL changes.
- A milestone card (`ChecklistGrid.tsx`) has a checkbox, a title that renames inline on click
  (the title `<span onClick>` sets `editingId`), the item menu (`ItemActions`: rename, dates,
  note/task flip, archive, remove, move), a collapse chevron, a meter, a date chip
  (`ItemDateChip`), the completion nudge (FS-FSFEC R10–R13) and a "+ add step" composer.
- `lib/milestones.ts` is the one derivation of which items are milestones and how far along each
  is (`readMilestones`, `isComplete`).
- Checklist items carry no free-text field beyond `description`. Dates were added to items end
  to end in migration 000018; `type` and `parent_id` in 000019.

## Decisions and why

Carried from the scoping session (2026-10-13).

**D1. The Jira line.** Jira coordinates work *for other people*; Fireplace is about *your own
momentum*. The page is a quiet room for one milestone, not a ticket. **Banned from the page:**
assignee, priority, any status beyond done / not done, labels, estimates, comment threads.
Tone: warm and inviting, a place you want to work in.

**D2. Option B.** A roomier view of steps and dates (option A), plus **why it matters** and the
**next step**. Rejected for now: option C, B plus a dated journal, the biggest piece (a new table
and endpoints). It is parked, not dropped.

**D3. Create stays simple.** Nothing new is asked when a milestone is created. Every new field
lives on the page and is added afterwards, if wanted, the way dates already work.

**D4. Shift the middle, like Settings.** Only the middle card swaps; Back returns to the plan.
Unlike Settings it needs a real URL, so it is a route, not component state.

**D5. ADR-0014 holds.** The only server change is a generic, nullable item field `intention`.
The server still sees checklist items and never learns the word milestone: it stores `intention`
on any item and the client chooses to expose it on milestones only. This is not the "milestone
needs server-side state a checklist item cannot carry" trigger, because an item now carries it.

### Charter defaults

Settled under the charter in the scoping session. Each is folded into the requirements below,
marked *(charter default)*.

| # | Question | Default | Reason |
| --- | --- | --- | --- |
| C1 | Address | `/plan/<planId>/m/<milestoneId>`, under a shared plan layout | A path reads better shared and bookmarked than `?milestone=`; a separate full page loses the shift-the-middle feel |
| C2 | Way in | An explicit open button in the card header, plus "Open" in the card's item menu; clicking the title still renames | Keeps today's inline rename |
| C3 | Where the way in appears | Grid only in v1; the URL works however it is reached | Smaller v1 |
| C4 | What the page holds | Title (editable), why it matters, next step, start/due dates (editable, existing editor and chip), progress (same measure as the card), every step (check, add, rename; notes shown as notes, not counted) | Option B |
| C5 | Why it matters | Nullable item field `intention`, ≤ 280 chars, trimmed, empty clears to null, set through the existing item PATCH, edited only on the page, never on the card or at create; column on all items, exposed only for milestones; a quiet "Why does this matter?" prompt when empty | One schema; ADR-0014 |
| C6 | Next step | Derived, not stored: the first not-done, non-note step in order; checking it promotes the next; all done → the completion nudge; no steps → an invitation to add the first | No new state |
| C7 | Completion | The same completion nudge as the grid (FS-FSFEC): one behaviour, not two | Consistency |
| C8 | Not here | Unknown id, not a milestone, archived, deleted, or a plan you cannot reach → a calm "this milestone isn't here" with a link back | No crash, no blank page |
| C9 | Back | "Back to plan" in the middle card's header; browser back works because it is a route | Settings parity |
| C10 | Closing from the page | Stay on the page, shown as closed and reopenable | Don't bounce |
| C11 | Deleting from the page | Return to the plan | Nothing left to show |
| C12 | Turned into a non-milestone while open | "Not here" on the next load; no live push | No realtime in v1 |
| C13 | Shared plans | A milestone URL inherits the plan's access (FS-12ZMF owns what that is) | One access rule |

### Gap defaults

Questions the notes left open at the acceptance-criteria level, settled here as the smallest
sensible v1. Marked *(gap default)* in the requirements.

| # | Question | Default |
| --- | --- | --- |
| G1 | The plan page is one column, not side columns: what does the layout hold? | The layout holds every block except the long-term Items card: title card and Daily above, Calendar and Recommended videos below. The middle slot is the Items card or the milestone page |
| G2 | How the page loads | From the plan's long-term item list, the same list the grid reads, so progress and completeness agree with the card and the strip by construction. No new read endpoint |
| G3 | A load that fails for a reason other than "not found / no access" | A retryable error state, never "not here" |
| G4 | A milestone id that belongs to a different plan than the URL | "Not here" |
| G5 | The open control | A real link (opens in a new tab, can be copied), always visible in a muted tone on milestone cards; never on note cards or steps |
| G6 | The page's item menu | Rename, Dates, Archive, Delete. No note/task flip, no move, no indent: each would turn the page's subject into something else |
| G7 | Archiving from the page | Returns to the plan, like delete: an archived milestone is out of the plan |
| G8 | What steps can do on the page | Check, add, rename only. Steps' other actions (dates, flip, archive, remove, move, drag) stay on the grid |
| G9 | Steps added on the page | Always tasks, added at the end, composer stays open with focus (as FS-FSFEC R32) |
| G10 | Editing why it matters | Click the text or the prompt to edit; Enter or blur saves, Esc cancels; single paragraph; input stops at 280 |
| G11 | Next step while the milestone is closed | Hidden; the page shows the closed state instead |
| G12 | Next step with only note children | Treated as "no steps": the invitation to add the first step |
| G13 | Back's destination | Always the plan URL, not history-back: a bookmarked or shared link has no history to go back to |
| G14 | Browser tab title | The milestone's title, so a bookmark names the milestone |
| G15 | Page position on return | Not restored; the plan opens as on a fresh visit (view choice is already remembered per device) |

## Requirements

### Route and layout

- **R1.** A milestone page lives at `/plan/<planId>/m/<milestoneId>`. *(charter default C1)*
- **R2.** `client/src/app/plan/[planId]/layout.tsx` holds the plan's surrounding blocks: the title
  card and the Daily card above the middle slot, the Calendar and Recommended videos below it.
  The plan route (`/plan/<planId>`) fills the middle slot with the long-term Items card; the
  milestone route fills it with the milestone page, inside the same card. *(charter default C1,
  gap default G1)*
- **R3.** Moving between the plan and one of its milestone pages swaps only the middle slot. The
  plan's metadata, the Daily list, the Calendar and the video suggestions are not refetched or
  remounted by that move.
- **R4.** The plan route behaves exactly as it does today, including the Items card's settings
  shift, archived view, view toggle, paging and composer.

### Way in

- **R5.** Every milestone card in the grid carries an **open** control in its header: a link to
  that milestone's page, named "Open ‹title›" for assistive tech, always visible in a muted tone
  and brighter on hover or focus. Being a link, it can be opened in a new tab or copied.
  *(charter default C2, gap default G5)*
- **R6.** A milestone card's item menu gains an **Open** entry that navigates to the page. Steps,
  note cards and list rows get neither the control nor the entry. *(charter default C2, C3)*
- **R7.** Clicking a card's title still renames it inline. *(charter default C2)*
- **R8.** The list view gets no open affordance in v1; a milestone URL reached any other way
  (typed, bookmarked, shared) works. *(charter default C3)*

### The page

- **R9.** The middle card's header carries a **Back to plan** button in the Settings back
  button's style. It navigates to `/plan/<planId>`. Browser back also works. *(charter default
  C9, gap default G13)*
- **R10.** The page loads from the plan's long-term item list (the list the grid reads) and finds
  the milestone and its steps there. It reads milestone status, steps and progress through
  `lib/milestones.ts`, so the page, the card and the strip can never disagree. No read endpoint
  is added. *(gap default G2)*
- **R11.** The page shows, in order: the milestone's **title** with its checkbox and item menu;
  its **dates**; its **progress**; **why it matters**; the **next step**, where the **completion
  nudge** stands in when it applies (R28); and **every step**. *(charter default C4)*
  *Integrator decision (I-043WP-5): the first wording put the nudge before why it matters, which
  R28 contradicts by placing the nudge in the next-step slot. R28 wins: why it matters sits above
  the next-step slot, and the nudge stays inside it.*
- **R12.** The browser tab's title is the milestone's title while the page is open. *(gap
  default G14)*
- **R13.** The page shows nothing from D1's banned list: no assignee, priority, status beyond
  done / not done, labels, estimates or comment threads.
- **R14.** While the list loads, the middle card shows a quiet loading line, never the "not here"
  state or an empty page.

### Title, closing and the item menu

- **R15.** Clicking the title renames it inline: Enter or blur saves, Esc cancels, an empty
  rename keeps the old title (as the card does).
- **R16.** The milestone's checkbox is hand-settable at any progress, as on the card (FS-FSFEC
  R9). Closing from the page keeps you on the page, shown as **closed** (title struck through,
  faded, checkbox ticked), and reopenable by unticking. *(charter default C10)*
- **R17.** The page's item menu offers **Rename**, **Dates**, **Archive** and **Delete** only.
  No note/task flip, no move up/down, no indent/outdent. *(gap default G6)*
- **R18.** **Delete** uses the same confirmation as the plan's lists; once confirmed and
  succeeded, the page navigates to the plan. **Archive** succeeds then navigates to the plan.
  A failure of either stays on the page with the existing error shown. *(charter default C11,
  gap default G7)*

### Dates and progress

- **R19.** The milestone's start and due dates are shown with the existing `ItemDateChip`.
  With no dates, a quiet "Add dates" control stands in its place. Both open the existing date
  picker (the item menu's Dates editor); setting and clearing go through the existing dates
  endpoint. *(charter default C4)*
- **R20.** Progress is done task steps over task steps, archived steps excluded, as on the card
  (FS-FSFEC R6): a meter and "done/total". A milestone with no task steps shows no meter.

### Completion

- **R21.** When every task step is done and the milestone is not, the page shows the same
  completion nudge as the grid, with the same Close and dismiss behaviour (FS-FSFEC R10–R13).
  The grid and the page render one shared nudge, not two copies. Dismissal is in memory and
  lasts while that progress pair holds and the page stays mounted. *(charter default C7)*

### Why it matters

- **R22.** **Why it matters** is the item's `intention`. Empty, it shows as a quiet prompt, "Why
  does this matter?". Set, it shows as text. *(charter default C5)*
- **R23.** Clicking the text or the prompt edits it in place. Enter or blur saves, Esc cancels,
  and input stops at 280 characters. It is a single paragraph: Enter saves rather than adding a
  line. Saving sends the trimmed text through the item PATCH; saving empty text clears it.
  Saving unchanged text sends nothing. *(charter default C5, gap default G10)*
- **R24.** The save is optimistic; a failure restores the previous text and shows the existing
  error.
- **R25.** Why it matters is edited and shown only on the milestone page. It never appears on a
  card, a list row, the composer or anywhere at create. *(charter default C5)*

### Next step

- **R26.** The **next step** is derived, never stored: the first non-archived task step that is
  not done, in the steps' stored order. Notes are never the next step. *(charter default C6)*
- **R27.** It is shown on its own near the top, with its checkbox. Ticking it marks that step
  done, and the next open step takes its place. In the full step list the same step is marked as
  next. *(charter default C6)*
- **R28.** With no open task step:
  - every task step done and the milestone not closed → the completion nudge stands in (R21);
  - no task steps at all (none, or only notes) → an invitation to add the first step, which
    opens the page's step composer;
  - the milestone closed → no next step; the closed state speaks (R16). *(charter default C6,
  gap defaults G11, G12)*

### Steps

- **R29.** Every non-archived step is listed in stored order with room to breathe: task steps
  with a checkbox, notes drawn as notes (no checkbox, not counted). *(charter default C4)*
- **R30.** A task step can be ticked and unticked, and any step renamed inline (click its text;
  Enter or blur saves, Esc cancels, empty keeps the old text). Steps carry no other actions on
  the page. *(gap default G8)*
- **R31.** An always-visible "+ add step" opens a composer. Enter with text adds a **task** step
  at the end of this milestone; the composer stays open, clears and keeps focus. Esc closes it.
  Enter is ignored while a create is in flight. A failed create keeps the text and shows the
  existing error. *(gap default G9)*
- **R32.** Every mutation on the page (tick, rename, add, dates, why it matters, close) is
  optimistic where today's lists are optimistic, reverts on failure, and shows the existing
  error text.

### Not here

- **R33.** The page shows a calm **"This milestone isn't here"** state with a link back to the
  plan when the id:
  - is not in the plan's long-term list: unknown, deleted, archived, or a daily item;
  - is in it but is not a milestone: a note, or a step (it has a parent);
  - belongs to a different plan than the URL; or
  - cannot be loaded because the plan cannot be reached (the list answers 404 or 403) or the
    plan id in the URL is not a well-formed uuid (422). A malformed milestone id is simply not
    in the list.
  No crash, no blank page. *(charter default C8, gap default G4)*
- **R34.** Any other load failure (network, 5xx, 503) shows a retryable error ("Couldn't load
  this milestone" with Try again), not "not here". *(gap default G3)*
- **R35.** The page does not watch for changes made elsewhere. A milestone archived, deleted,
  converted to a note or nested in another tab shows "not here" on its next load. *(charter
  default C12)*

### Access

- **R36.** A milestone URL inherits the plan's access: the page reads only through the plan's
  existing checklist routes, so whoever those routes serve, and with whatever edit rights, the
  page serves the same. This FS adds no access rule; FS-12ZMF owns them. *(charter default C13)*

### The `intention` field

- **R37.** Checklist items gain a nullable text field `intention`. The column lives on every item;
  the server attaches no meaning to it and never uses the word milestone (ADR-0014, D5).
- **R38.** It is writable only through the existing item update (PATCH): omit it to leave it
  unchanged; send a string to set it. plan-service trims the string, stores an empty result as
  null, and rejects a trimmed value longer than 280 characters (counted as characters, not bytes)
  as a domain error. *(charter default C5)*
- **R39.** It is not accepted at create (the create body stays as it is), and it is returned on
  every item response, omitted when null, like `parentId` and the dates.
- **R40.** The change ships end to end: a plan-service migration (nullable `TEXT` column with a
  length check), the proto (`ChecklistItem.intention`, `UpdateItemRequest.intention`),
  plan-service read and update paths, the gateway's update request and item response types, the
  regenerated `openapi.yaml`, and the regenerated client types. Adding an optional field to a
  request and a response is non-breaking; the contract gates must pass.

## User Stories

1. As a planner, I want a milestone to have a page of its own, so that I have room to think about
   one block of progress.
2. As a planner, I want that page to have a URL, so that I can bookmark it and come back to it.
3. As a planner sharing a plan, I want to send a milestone's link to a plan member, so that we
   look at the same thing.
4. As a planner, I want to open a milestone from its card with one obvious control, so that I
   don't have to hunt for the way in.
5. As a planner, I want clicking a card's title to keep renaming it, so that what I already know
   still works.
6. As a planner, I want to open a milestone in a new tab, so that I can keep the plan in view.
7. As a planner, I want the page to replace only the middle of the plan, so that my daily list
   and the plan's title stay where they are.
8. As a planner, I want a Back to plan button like Settings has, so that leaving feels the same.
9. As a planner, I want browser back to work, so that the page behaves like a page.
10. As a planner, I want to write down why a milestone matters, so that I remember the point of
    it when motivation dips.
11. As a planner, I want an empty why-it-matters to be a gentle question rather than a required
    field, so that the page invites without nagging.
12. As a planner, I want creating a milestone to ask nothing new, so that getting started stays
    quick.
13. As a planner, I want the page to show me the one next step, so that I know what to do now.
14. As a planner, I want ticking the next step to bring up the one after, so that I can keep
    moving without scanning the list.
15. As a planner with an empty milestone, I want an invitation to add the first step, so that the
    page doesn't feel empty and broken.
16. As a planner, I want to see every step with room to breathe, so that I can think about them.
17. As a planner, I want to tick, add and rename steps on the page, so that I can work there
    rather than going back to the card.
18. As a planner, I want my notes among the steps shown as notes and not counted, so that they
    don't read as unfinished work.
19. As a planner, I want to rename the milestone on its page, so that its name can grow with it.
20. As a planner, I want to see and edit the milestone's dates on its page, so that timing is
    part of thinking about it.
21. As a planner, I want the page's progress to match the card's, so that the two never disagree.
22. As a planner, I want the page to offer to close a milestone whose steps are all done, the
    same way the grid does, so that finishing is noticed wherever I am.
23. As a planner, I want closing on the page to keep me on the page, so that I can enjoy what I
    finished and reopen it if I was wrong.
24. As a planner, I want deleting or archiving from the page to take me back to the plan, so that
    I'm never left on a page about nothing.
25. As a planner following an old or wrong link, I want a calm "not here" with a way back, so
    that a dead link isn't a crash.
26. As a planner on a flaky connection, I want a failed load to offer a retry, so that a blip
    isn't mistaken for a missing milestone.
27. As a planner, I want the tab to show the milestone's name, so that my bookmarks and tabs say
    what they are.
28. As a planner, I want the page free of assignees, priorities, labels and estimates, so that it
    stays about my momentum and not a ticket.
29. As a keyboard and screen-reader user, I want the open control, the next step and the why-it-
    matters prompt named in words, so that the page works without seeing it.
30. As a developer, I want the server to see only a generic item field, so that milestones stay a
    client reading of items (ADR-0014).
31. As a developer, I want the page to read the same item list the grid does, so that progress
    and completeness are computed once.

## Acceptance Criteria

- [ ] `/plan/<planId>/m/<milestoneId>` renders the milestone page inside the plan's middle card,
      with the title card and Daily above and Calendar and videos below.
- [ ] Navigating plan → milestone → plan refetches neither the plan, the Daily list, the Calendar
      nor the video suggestions.
- [ ] The plan route behaves as before: settings shift, archived view, toggle, paging, composer.
- [ ] Every milestone card in the grid shows an open link named "Open ‹title›"; note cards,
      steps and list rows show none.
- [ ] A milestone card's item menu has Open; steps' and note cards' menus do not.
- [ ] Clicking a card's title still renames it.
- [ ] Back to plan navigates to `/plan/<planId>`; browser back returns to the previous page.
- [ ] The tab title is the milestone's title on its page.
- [ ] The page shows title + checkbox + menu, dates (or "Add dates"), progress, why it matters,
      next step and every step, in that order (nudge when it applies).
- [ ] The page has no assignee, priority, labels, estimates, comment thread or extra status.
- [ ] Renaming the title on the page saves; empty keeps the old title.
- [ ] Closing on the page keeps the page, shows it closed, and unticking reopens it.
- [ ] The page's item menu offers exactly Rename, Dates, Archive, Delete.
- [ ] Delete (after confirmation) and Archive navigate to the plan; a failure stays with the error.
- [ ] Dates show in the chip; "Add dates" shows when there are none; both open the date picker;
      setting and clearing persist.
- [ ] Progress reads the same done/total as the card for the same milestone; a no-step milestone
      shows no meter.
- [ ] All task steps done and not closed → the same nudge as the grid; Close closes, dismiss
      hides until progress changes; the grid and the page use one nudge component.
- [ ] Empty why-it-matters shows "Why does this matter?"; editing, Enter/blur saves, Esc cancels;
      the input stops at 280; saving empty clears it; a failed save restores the old text.
- [ ] Why it matters appears nowhere but the milestone page.
- [ ] Next step is the first open task step in order; ticking it promotes the next; it is marked
      in the full list.
- [ ] No task steps → an invitation that opens the step composer; all done → the nudge; closed →
      no next step.
- [ ] Steps tick, untick and rename on the page; notes show as notes and are not counted.
- [ ] "+ add step" adds task steps at the end, keeps focus, ignores Enter mid-create, keeps text
      on failure.
- [ ] Unknown, deleted, archived, daily, note, step, other-plan, malformed and unreachable ids
      show "This milestone isn't here" with a link to the plan.
- [ ] A network or 5xx failure shows a retryable error, not "not here".
- [ ] While loading, the middle card shows a quiet loading line.
- [ ] PATCH with `intention` sets it (trimmed); an empty or whitespace string clears it to null;
      omitting it leaves it unchanged; over 280 characters is `400 · VALIDATION_FAILED`.
- [ ] Create rejects an `intention` member (`422 · VALIDATION_FAILED`).
- [ ] Every item response carries `intention` when set and omits it when null.
- [ ] Migration up and down both apply cleanly; existing items read `intention` as null.
- [ ] `openapi.yaml` and the client's generated types are regenerated; contract gates pass with
      no breaking change.
- [ ] No API name, request body or server identifier says "milestone".
- [ ] Tests pass.

## Edge States

| Situation | Behavior |
| --- | --- |
| Unknown id, or deleted | "Not here" with a link to the plan (R33). |
| Archived milestone (or archived from another tab) | "Not here" (R33, R35). |
| Id is a note, a step or a daily item | "Not here" (R33). |
| Id belongs to another plan than the URL | "Not here" (R33). |
| Milestone id is not a uuid | Not in the list; "not here" (R33). |
| Plan id is not a uuid | The list answers 422; "not here" (R33). |
| Plan unreachable (no access, deleted plan) | List answers 404/403; "not here" (R33, R36). |
| Network failure, 5xx, 503 | Retryable error, not "not here" (R34). |
| Loading | Quiet loading line (R14). |
| Milestone with no steps | No meter; next step reads as an invitation to add the first (R20, R28). |
| Milestone with only note children | Notes listed as notes; no meter; invitation to add the first step (R28). |
| All task steps done, not closed | Nudge in the next-step slot (R21, R28). |
| Milestone closed with steps open | Closed state; no next step; steps still listed and tickable (R16, R28). |
| Nudge dismissed, then a step unticked and re-ticked | Nudge returns (R21). |
| Nudge dismissed, then Back and reopen | Nudge returns: dismissal is per mount (R21). |
| Next step ticked, ticking fails | Step reverts and stays the next step; existing error (R32). |
| Next step renamed in the list | The next-step slot shows the new text (one item, two places). |
| Why it matters saved as whitespace | Cleared to null; prompt returns (R23, R38). |
| Why it matters pasted past 280 | Input stops at 280 (R23); a direct API call over 280 is `400 · VALIDATION_FAILED` (R38). |
| Why it matters save fails | Previous text restored, existing error (R24). |
| Delete or archive from the page | Navigate to the plan (R18). |
| Delete or archive fails | Stay on the page, existing error (R18). |
| Milestone converted to a note or nested in another tab | "Not here" on next load; no live push (R35). |
| Milestone renamed in another tab | Old title until the next load (R35). |
| Dates edited on the page | Saved through the dates endpoint; the Calendar block is not refreshed by it, as today when dates change in the grid. |
| Back after paging the grid to page 3 | The plan opens as on a fresh visit; page position is not restored (G15). |
| Milestone URL opened by a plan member (FS-12ZMF) | Same page, same edit rights as the plan's routes give them (R36). |
| Two levels of steps | Cannot exist (one nesting level, FS-0007); nothing to render. |
| Item with `intention` set, later turned into a step or note | Value kept on the item, not shown anywhere (R25, R37). |

## API surface

One field added to two existing operations; established patterns otherwise (FS-0004). No new
operation, no new path.

| Op | Method + Path | Query/Params | Request body | Response | Errors |
|----|---------------|--------------|--------------|----------|--------|
| updateChecklist | PATCH `/api/plans/{id}/checklists/{checklist_id}` | unchanged | `UpdateChecklistItemRequest` gains optional `intention: string`. Omit = leave unchanged; a string sets it after trimming; empty after trim clears it to null. No `maxLength` in the schema: the 280 limit is a domain rule (ADR-0005). | `ChecklistItemResponse` with `intention` | unchanged, plus trimmed `intention` over 280 characters → `400 · VALIDATION_FAILED` |
| every op returning `ChecklistItemResponse` (list, archived, upcoming, get, create, update, dates, archive, reorder) | unchanged | unchanged | unchanged | `ChecklistItemResponse` gains optional `intention: string`, omitted when null | unchanged |
| createChecklist | POST `/api/plans/{id}/checklists` | unchanged | unchanged: `intention` is **not** a member, so sending it is an unknown member | unchanged | sending `intention` → `422 · VALIDATION_FAILED` (strict body, ADR-0005) |

gRPC (plan-service, `common/api/proto/plan/checklist.proto`): `ChecklistItem` gains
`optional string intention`; `UpdateItemRequest` gains `optional string intention` (unset =
leave unchanged, set = trim, empty → NULL). A trimmed value over 280 characters is
`InvalidArgument`, which the gateway already maps to `400 · VALIDATION_FAILED`.

Storage: `checklist_items.intention TEXT NULL` with `CHECK (intention IS NULL OR
char_length(intention) <= 280)`, in the next free plan-service migration number (000021 at the
time of writing; FS-12ZMF may claim a number in parallel, so take whichever is next when the
slice lands). Down drops the constraint and the column.

## Out of Scope

- **A journal** or dated log per milestone (option C). Parked, not dropped.
- **Anything from D1's banned list**: assignee, priority, status workflow, labels, estimates,
  comment threads.
- **An open affordance in the list view**, and a strip segment that opens its milestone.
- **Why it matters on the card**, on hover or anywhere outside the page.
- **Step actions beyond check, add and rename** on the page (dates, flip, archive, remove, move,
  drag); they stay on the grid.
- **Live updates** from other tabs or other plan members.
- **Restoring the grid's page or scroll position** on Back.
- **Refreshing the Calendar block** after a date change on the page.
- **Access rules.** Who can reach a plan, and so its milestone URLs, is FS-12ZMF's.
- **Card colour or emoji "presence"** for milestones.
- **AI help** to formulate a milestone's steps or its why (fireplace-ai, `feat/ai`).
- **Making `intention` settable at create**, or using it in insights, search or events.
- **Renaming code identifiers** from `grid` to `milestone`.
