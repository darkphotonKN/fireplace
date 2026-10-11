# FS-FSFEC: Milestones as blocks of progress

> Status: work-order · SPECIFICATION.md: client `## Checklists` "Milestones as blocks of progress", "New-milestone composer"; client `## Plans` "Plan progress across its milestones" → this FS · Brownfield sibling: client `## Checklists` "Card view of a plan's items" → FS-none (the grid as it ships today, commit 715764b) · Related ADRs: docs/adr/0014-milestones-are-a-client-reading-of-items.md, docs/adr/0013-home-composes-capped-fan-out.md · Related FS: FS-0007 (nesting, add-bar Tab), FS-0008 (paging, pinned add bar, page jumps, filter sets add type), FS-0009 (drag), FS-KSJFR (dashboard Continue cards)

## Summary

A plan's top-level tasks are its **milestones**: finishable pieces of the plan, each broken into
steps, each with its own progress. The grid view becomes the *milestone reading* of a plan: the
plan header carries a progress strip ("3 of 7 milestones", one segment per milestone filling
up), a milestone whose steps are all done offers to close itself, and the bottom bar becomes a
composer that names a new milestone and then fills it with steps while the grid traces which
card it is filling. The same strip appears on the dashboard's *Continue where you left off*
cards. Nothing changes on the server.

Terms used here (`milestone`, `step`, `progress strip`, `traced milestone`, `completion nudge`,
`checklist reading` / `milestone reading`) are defined in `client/CONTEXT.md` and used exactly as
defined there.

## Background: what ships today

- Grid view (`client/src/components/ChecklistGrid.tsx`) is the default view, **long-term list
  only**; daily and archived always render as rows (`canUseGrid` in `Todo.tsx`). Grid pages at
  6 cards (`GRID_PAGE_SIZE`), the list at 10 rows.
- A card is a top-level item; its steps are its children (one nesting level). Each card has its
  own "+ add step" composer. The meter is done/total over task children; notes never count.
- The pinned bottom add bar was built for rows. Tab nests under "the nearest top-level task
  walking up from the bottom of the page" (`findAddBarTarget`). In a grid that is whichever card
  is last in reading order, often nowhere near the input. **This mismatch triggered the
  feature.**
- Enter in the bar makes a top-level item, i.e. a new card at the end, and the view jumps to its
  page (FS-0008 R16/R17), but nothing ties the input to that card afterwards.
- Naming was split: the commit said "blocks of progress", the toggle says "Card view", code says
  `grid`, and the dashboard uses "block" for its sections.

## Decisions and why

Carried from the scope-it session (2026-10-09..11), with the reasoning that produced them.

**D1. The milestone is a product concept, not a model change.** A milestone is a top-level
checklist task; its steps are its children. No server entity, no API, no migration, no contract
change. Recorded as **ADR-0014**. Rejected: view-only polish (leaves progress unnamed and
unadvertisable) and a server-side milestone entity (migration, contract and plan-service work
for no behaviour the item model can't express).

**D2. It is called a Milestone.** *One finishable piece of a plan: a single outcome you're
working toward, broken into steps, whose progress is the share of its task steps done. A plan is
the sum of its milestones.* Milestones need no dates and imply no order. Rejected "Block"
(collides with the dashboard's `TodayBlock` and its blocks) and "Chunk" (not something to be
proud to finish).

**D3. Every top-level task is a milestone, in both views.** Grid is the milestone reading, list
the checklist reading of the same plan. List view keeps its mechanics, including the add bar's
Tab-to-nest (FS-0007/0008).

**D4. The grid's big input only ever creates new milestones.** It never targets an existing
milestone for naming. Steps for an older milestone go through that card's own "+ add step".

**D5. One entry point, a pinned composer with three states** (resting, naming, filling).
Creating a parent then its children is "the most natural thing". Tab's meaning inverts against
the list: in the grid Tab takes you *out* to a new milestone, not *in*. Rejected: a "+" tile at
the grid's end (two doors, and under paging it only exists on the last page), and a tile that
turns into an inline draft card (cramped typing, a composer that differs between views).
Exception: an empty plan shows a "Start your first milestone" invitation, since a lone bar under
nothing reads as broken.

**D6. Trace.** While filling, the target milestone is visibly traced in the grid and named in
the input, because it is not spatially next to the input.

**D7. Completion is nudged, not derived.** At full progress the milestone shows a complete state
and offers to close itself; the user confirms. It never auto-closes, so a not-yet-added step
can't find its milestone already shut. Rejected: a derived, non-settable checkbox; and an
independent meter (today). A milestone with no task steps keeps a manual checkbox.

**D8. A closed milestone stays in place, faded.** The record of what you finished is the
progress. Archiving stays a separate, deliberate action. Rejected: sinking to the end;
auto-archive.

**D9. Top-level notes are not milestones.** Drawn as note cards, excluded from the strip and
the count.

**D10. Plan progress rolls up as a strip.** Headline count plus one segment per milestone, each
filled to its own progress. Rejected: count only (a big milestone at 9/10 reads as zero) and a
step-weighted percentage (big milestones dominate and milestones vanish into a number).

**D11. The strip shows in the plan header and on the dashboard's Continue cards.** The dashboard
already holds the focused plans' items (ADR-0013), so it costs no request.

**D12. Daily stays a list.** Daily items recur and reset; they are not milestones you finish.

### Charter defaults

The scoping session left open questions; the charter settled them. Each is folded into the
requirements below, marked *(charter default)*.

| # | Question | Default | Reason |
| --- | --- | --- | --- |
| C1 | User pages away while filling | Stay in filling; Enter adds to the traced milestone and the view jumps back to its page | Reuses FS-0008 R16; no new mechanism |
| C2 | Traced milestone archived / converted / deleted mid-fill | Composer drops to naming | Mirrors FS-0007 R8.11 for the list bar |
| C3 | Notes filter, or the bar's type icon set to note, in grid view | The bar creates a top-level note card, with no filling state | Honors FS-0008 R19–R21; notes take no steps |
| C4 | Keyboard shortcut to open the composer | None in v1 | Smaller v1 |
| C5 | Strip with many milestones (30+) | Equal-width segments, no cap, gaps tighten past 12 | Keeps "a row of milestones" literally true at any count |
| C6 | Does the trace survive Esc | No, Esc clears it | Resting has no target |
| C7 | Tab / Shift+Tab | Both take filling → naming; in naming both are swallowed | Same rule as FS-0007's bar: focus never leaves mid-entry |
| C8 | What "close" means on the nudge, and dismissing it | Close = mark the milestone done; dismissing hides the nudge until that milestone's progress changes | No new server state |
| C9 | Where the plan-header strip shows, and what it counts | Long-term tab, both views; long-term, non-archived top-level tasks only | Milestones are long-term; archived work is out of the plan |
| C10 | Dashboard Continue card | Strip derived from items `useHomeData` already fetches; a plan with no milestones renders as today | ADR-0013's cap holds |
| C11 | View toggle labels | "Milestones" / "Checklist" | The two readings named in `client/CONTEXT.md` |

## Requirements

### Milestones and the two readings

- **R1.** A **milestone** is a non-archived, top-level (`parentId` empty), long-term item of type
  task. Its **steps** are its children. Only task steps count toward progress; note children are
  shown and never counted.
- **R2.** The feature is client-only (ADR-0014). No endpoint, proto, plan-service or database
  change. "Milestone" is a presentation term and must not appear in API calls, request bodies or
  server code. Code identifiers may keep `grid`; every **user-facing** string says milestone.
- **R3.** The view toggle's two buttons are labelled **"Milestones"** (grid) and **"Checklist"**
  (list), as accessible name and title. The "Card view" / "List view" strings are retired. The
  pair sits in a group named "View" so the toggle's "Checklist" stays distinguishable from the
  type filter's "Checklist" tab. *(charter default C11)*
- **R4.** The grid stays long-term only. Daily and archived render as rows whatever is
  remembered; the default view (grid) and its per-device memory are unchanged.
- **R5.** A top-level **note** is drawn as a note card and is not a milestone: it has no
  checkbox, no nudge, and no segment in the strip, and it is not counted.

### Milestone progress and closing

- **R6.** A milestone's progress is done task steps over task steps, archived steps excluded.
- **R7.** A milestone with **no task steps** (no children, or only note children) has no step
  progress and keeps a manual checkbox. Its progress reads as empty until it is done and full
  after.
- **R8.** A **closed** milestone (`done = true`) stays in its grid position, faded. It is not
  moved and not archived. Its strip segment is full whatever its steps say.
- **R9.** The milestone's checkbox stays hand-settable at any progress: closing early and
  reopening are both allowed. Reopening returns its segment to its step progress (R6/R7).

### Completion nudge (grid)

- **R10.** A milestone with at least one task step, all of them done, and itself not done is
  **complete**: its meter is full, the card wears a glow, and it shows a nudge offering to
  close it.
- **R11.** The nudge's **Close** marks the milestone done through the same optimistic path as
  its checkbox; a failure reverts it and surfaces the existing error. *(charter default C8)*
- **R12.** **Dismissing** the nudge hides it until that milestone's progress (its done/total
  pair) changes. Dismissal is in memory only: no server state and no storage, so a reload or
  remount shows the nudge again. *(charter default C8)*
- **R13.** A milestone never closes itself. Adding a step or unticking one ends the complete
  state; the nudge disappears with it.
- **R14.** The nudge is a milestone-reading behaviour: it appears in grid view only. List view
  is unchanged.

### Progress strip: plan header

- **R15.** On the long-term tab, in **both** views, the items section opens with the progress
  strip and its headline **"N of M milestones"**, where N counts closed milestones and M all
  milestones. *(charter default C9)*
- **R16.** The strip is computed over the whole long-term list: every milestone per R1,
  independent of paging, the type filter and collapse state. *(charter default C9)*
- **R17.** One segment per milestone, in list order (the order the grid draws them). Each is
  filled to that milestone's own progress per R6–R8. No averaging, no step weighting.
- **R18.** Segments are equal width with no cap on their number. Past 12 segments the gaps
  between them tighten. The strip never wraps and never scrolls horizontally. *(charter
  default C5)*
- **R19.** A plan with no milestones shows no strip and no headline. There is no "0 of 0".
- **R20.** The strip reads the same optimistic state the items do: ticking a step moves its
  segment immediately, and a reverted tick reverts the segment.
- **R21.** The headline is the strip's text equivalent. Each segment carries its milestone's
  name and progress as a title; the fills themselves are decorative.
- **R22.** No strip on the daily tab, the archived view or settings.

### Progress strip: dashboard

- **R23.** A *Continue where you left off* card for a plan with at least one milestone shows that
  plan's progress strip and milestone count in place of its done/total. *(charter default C10)*
- **R24.** The card's strip is derived from the items `useHomeData` already fetched for that
  plan, filtered per R1 (long-term scope, non-archived, top-level, task). No request is added;
  home still issues `1 + min(planCount, 3)` requests (ADR-0013). *(charter default C10)*
- **R25.** Items ticked from *Today* during the visit count toward the card's strip as they
  already do toward its done/total (FS-KSJFR single-derivation rule).
- **R26.** A plan with **no milestones** renders exactly as today, with done/total. A plan whose
  items are loading or failed keeps today's empty progress slot, never an empty strip or a
  zero. *(charter default C10)*
- **R27.** The strip sits in the card's existing footer row, so it settling in changes no card
  height (FS-KSJFR R22).
- **R28.** On the dashboard the strip is drawn in neutral tokens; coral stays reserved for the
  Today block (FS-KSJFR R49).

### New-milestone composer (grid view only)

- **R29.** In grid view the pinned bottom bar becomes the **new-milestone composer**, with three
  states: **resting**, **naming**, **filling**. In list view the add bar is unchanged (FS-0007
  R8, FS-0008 R15–R22).
- **R30.** **Resting** is a quiet "+ New milestone" bar. It is a button: click, Enter or Space
  opens naming with focus in the input. There is no global keyboard shortcut to open it.
  *(charter default C4)*
- **R31.** **Naming** shows the big input, placeholder "Name a new milestone…". Enter with text
  creates a top-level task at the end of the list and, on success, moves to filling with that
  milestone traced; the view jumps to the page holding it (FS-0008 R17). Enter on empty text
  does nothing. A failed create stays in naming with the text kept and the existing error shown.
- **R32.** **Filling** labels the input "↳ steps for ‹title›" with placeholder "Add a step…".
  Enter with text creates a task step under the traced milestone; the composer stays filling,
  the input clears and focus stays. Steps made here are always tasks: the type control is not
  offered while filling.
- **R33.** The **traced milestone** is visibly marked in the grid (ring and ✦) and named in the
  input's accessible label ("Add a step to ‹title›"). At most one milestone is traced, and it is
  only ever the one the composer just created, never an existing one picked by position.
- **R34.** A step added to a collapsed traced milestone opens it, so the step is seen arriving.
- **R35.** In filling, **Tab** and **Shift+Tab** both go to naming: the trace clears and typed
  text is kept. In naming both are swallowed, with no state change and focus kept in the input.
  *(charter default C7)*
- **R36.** **Esc** from naming or filling goes to resting. The trace clears and typed text is
  discarded, as the per-card step composer does today. *(charter default C6)*
- **R37.** Paging away while filling keeps filling and the trace. Enter adds the step to the
  traced milestone and the view jumps back to the page holding it (FS-0008 R16). *(charter
  default C1)*
- **R38.** If the traced milestone stops being a milestone that can take steps (archived,
  deleted, converted to a note, nested under another item, or hidden by the type filter), the
  composer drops to naming with typed text kept (FS-0007 R8.11). *(charter default C2)*
- **R39.** Under the **Notes** filter, or with the bar's type icon set to note, the composer
  creates **top-level note cards**. There is no filling state: after Enter it stays in naming
  with the input cleared and focus kept. Resting then reads "+ New note" and naming's
  placeholder "Jot down a note…". The filter sets the type only when chosen, and the icon wins
  afterwards (FS-0008 R19–R21). The type icon is offered in naming only. *(charter default C3)*
- **R40.** Leaving the input (blur, clicking a card, ticking a step) does not change the
  composer's state or the trace.
- **R41.** Switching to list view ends filling and clears the trace. The list's add bar starts at
  top level (FS-0007 R8.10) with any typed text kept. Coming back to grid opens naming if text
  remains, otherwise resting.
- **R42.** Composer state is not persisted: every load starts at resting.
- **R43.** While a create is in flight, Enter is ignored and the input stays focused (read-only,
  not disabled), as the add bar does today.
- **R44.** The FS-0007 "Tab to nest" tip never fires from the grid composer: Tab means something
  else here.
- **R45.** Each card's own "+ add step" composer stays as it is, for adding steps to older
  milestones. Using it does not change the big composer's state or trace.

### Empty plan

- **R46.** In grid view, when the long-term list holds no items at all and the filter is All or
  Checklist, the grid shows a single **"Start your first milestone"** invitation in place of the
  empty-list text, and the resting bar is not shown beneath it. Activating it opens the composer
  in naming with focus. Once a milestone exists, the normal grid and resting bar return.
- **R47.** When items exist but the filter hides them all, today's filter-empty text stays
  ("No notes yet." / "No tasks yet.") with the composer below it.

## User Stories

1. As a planner, I want my plan's top-level tasks called milestones, so that the cards mean
   something I can be proud of finishing.
2. As a planner, I want to see how far along the whole plan is at a glance, so that I know where
   I stand without opening every card.
3. As a planner, I want each milestone's own progress visible in the plan's strip, so that one
   big milestone at 9/10 doesn't read as nothing done.
4. As a planner, I want milestones counted equally in the headline, so that a big milestone
   doesn't swamp the small ones.
5. As a planner, I want to name a new milestone and then type its steps straight after, so that
   creating a parent and its children is one motion.
6. As a planner, I want the card I'm filling to be marked in the grid, so that I can see where my
   steps are landing even when it's far from the input.
7. As a planner, I want the input to name the milestone I'm filling, so that I never add a step
   to the wrong one.
8. As a planner, I want Tab to take me to a fresh milestone, so that I can outline milestone
   after milestone without the mouse.
9. As a planner, I want Esc to put the composer away, so that the page goes quiet when I'm done.
10. As a planner who paged back to check something, I want my next step to still land in the
    milestone I was filling, and the view to follow it, so that paging doesn't lose my place.
11. As a planner, I want the composer to let go of a milestone I just archived or deleted, so
    that I can't type steps into something that's gone.
12. As a planner with the Notes filter on, I want the bar to make a note card, so that what I
    create matches what I'm looking at.
13. As a planner, I want to add steps to an older milestone from its own card, so that the big
    composer doesn't have to guess which one I mean.
14. As a planner, I want a milestone with every step done to offer to close itself, so that
    finishing is a moment I notice.
15. As a planner, I want closing to wait for my say-so, so that a step I haven't written yet
    can't find its milestone already shut.
16. As a planner, I want to dismiss the nudge, so that a milestone I'm not ready to close stops
    asking.
17. As a planner, I want a closed milestone to stay where it was, faded, so that the record of
    what I finished is the progress.
18. As a planner, I want a milestone with no steps to keep a plain checkbox, so that small
    milestones are still finishable.
19. As a planner, I want my notes kept out of the milestone count, so that notes I keep don't
    read as unfinished work.
20. As a planner, I want the progress strip in both views, so that switching to the checklist
    reading doesn't hide where the plan stands.
21. As a returning user, I want each Continue card on home to show its plan's strip, so that I
    can pick the plan that's closest to done.
22. As a returning user, I want home to stay as fast as it is, so that the strip costs no extra
    request.
23. As a planner with an empty plan, I want an invitation to start my first milestone, so that
    the plan doesn't look broken.
24. As a planner with thirty milestones, I want the strip still to fit on one line, so that a
    large plan stays readable.
25. As a planner using the checklist reading, I want my add bar to behave as it always has, so
    that Tab-to-nest still works there.
26. As a keyboard user, I want Tab in the composer never to throw focus out of the input, so that
    I don't lose what I'm typing.
27. As a screen-reader user, I want the strip's headline and the input's label to say what they
    mean in words, so that the trace and the progress aren't visual only.
28. As a developer, I want milestones to stay a client reading of items, so that no migration or
    contract change rides on a presentation idea.

## Acceptance Criteria

- [ ] The view toggle reads "Milestones" / "Checklist"; "Card view" appears nowhere.
- [ ] Daily and archived still render as rows; the grid's default and memory are unchanged.
- [ ] On the long-term tab, both views show "N of M milestones" and a strip with one segment
      per milestone, in list order.
- [ ] Top-level notes are not segments and are not counted.
- [ ] The strip counts across all pages and ignores the type filter and collapse.
- [ ] A milestone at 3/4 steps fills its segment 75%; a closed milestone fills it fully; a
      no-step milestone is empty until ticked.
- [ ] A plan with no milestones shows no strip and no headline.
- [ ] At 30 milestones the strip is one line, equal-width segments, no horizontal scroll; gaps
      are tighter than at 12.
- [ ] Ticking a step moves its segment immediately; a failed tick moves it back.
- [ ] No strip on daily, archived or settings.
- [ ] A milestone whose task steps are all done, and which is not done, shows a full meter, a
      glow and a Close nudge.
- [ ] Close marks the milestone done; it stays in place, faded, and its segment is full.
- [ ] Dismissing the nudge hides it; ticking or adding a step on that milestone and returning to
      full progress shows it again.
- [ ] A milestone never becomes done without a person setting it.
- [ ] No nudge appears in list view.
- [ ] Continue cards for plans with milestones show the strip and milestone count; plans without
      milestones show done/total as today.
- [ ] Home still issues `1 + min(planCount, 3)` requests.
- [ ] A Continue card's strip counts items ticked from Today in this visit.
- [ ] A Continue card's height does not change as its strip settles in.
- [ ] The dashboard strip uses neutral tokens; coral appears only in the Today block.
- [ ] In grid view, the bottom bar rests as "+ New milestone" and opens to naming on click,
      Enter or Space.
- [ ] Naming + Enter creates a top-level task, jumps to its page, and switches to filling with
      that card traced (ring, ✦) and named in the input.
- [ ] Filling + Enter adds a task step to the traced milestone; the composer stays filling with
      focus kept.
- [ ] Tab and Shift+Tab in filling go to naming with typed text kept; in naming they do nothing
      and focus stays in the input.
- [ ] Esc from naming or filling goes to resting and clears the trace.
- [ ] Paging away while filling, then Enter, adds the step to the traced milestone and jumps
      back to its page.
- [ ] Archiving, deleting or converting the traced milestone drops the composer to naming with
      text kept.
- [ ] Under the Notes filter (or with the type icon on note) the composer creates top-level note
      cards and never enters filling.
- [ ] Switching to list view mid-fill clears the trace and leaves the list's add bar at top
      level with text kept.
- [ ] The "Tab to nest" tip never appears from the grid composer.
- [ ] A card's own "+ add step" still adds steps to that card and does not disturb the composer.
- [ ] An empty long-term plan in grid view shows "Start your first milestone" and no resting bar;
      activating it opens naming.
- [ ] List view's add bar, Tab-to-nest and page jumps behave exactly as FS-0007/0008 specify.
- [ ] No API call, request body or server file mentions "milestone".
- [ ] Tests pass.

## Edge States

| Situation | Behavior |
| --- | --- |
| Empty long-term list, grid view, filter All/Checklist | "Start your first milestone" invitation; no resting bar (R46). |
| Items exist but the filter hides them all | Today's filter-empty text, composer below (R47). |
| Only notes at top level | No strip, no headline (R19); notes draw as note cards (R5). |
| Milestone whose children are all notes | No step progress; manual checkbox; segment empty until ticked (R7). Card still says "N notes". |
| All steps done, milestone not closed | Segment full, not counted in N until closed; nudge offered (R10). |
| Milestone closed early with steps undone | Allowed (R9); faded, segment full, counted. |
| Closed milestone reopened | Segment returns to its step progress; nudge returns if every step is done and it was not dismissed at that progress. |
| Step added to a complete milestone | Progress drops below full, complete state and nudge end (R13). |
| Nudge dismissed, then reload | Nudge shows again; dismissal is in memory only (R12). |
| Close fails | Milestone reverts to not done, existing error shown (R11). |
| 30+ milestones | Equal-width segments, tighter gaps past 12, one line (R18). |
| Traced milestone collapsed | A step add opens it (R34). |
| New milestone lands on another page | View jumps to it so the trace is visible (R31). |
| Paged away while filling | Stays filling; Enter jumps back to the traced milestone's page (R37). |
| Traced milestone archived, deleted, converted, nested or filtered out | Composer drops to naming, text kept (R38). |
| Type icon set to task under the Notes filter, then a milestone is named | It is created but hidden by the filter, so R38 applies at once: composer stays in naming. |
| Create fails (naming or filling) | State unchanged, text kept, existing error shown (R31, R32). |
| Enter while a create is in flight | Ignored, focus kept (R43). |
| Switch to list mid-fill | Trace clears; list bar at top level with text kept (R41). |
| Leave the long-term tab or reload mid-fill | Composer state is gone; next visit starts resting (R42). |
| Dashboard plan items loading or failed | Today's empty progress slot; no strip, no zero (R26). |
| Dashboard plan with only daily items | No milestones, so done/total as today (R24, R26). |
| Toggle's "Checklist" vs the filter's "Checklist" | Distinguished by the toggle's "View" group (R3). |

## Out of Scope

- **Any server change.** No endpoint, proto, plan-service or database change, and so no API
  surface section. Milestones stay a client reading of items (ADR-0014).
- **List view behaviour.** The list's add bar (FS-0007/0008 Tab-to-nest), its paging and its
  rows are unchanged. No nudge in list view.
- **Daily and archived as milestones.** They stay rows.
- **Persisted nudge dismissal**, or any new server or storage state for closing.
- **A keyboard shortcut to open the composer** (C4). N was floated; not in v1.
- **Navigating from a strip segment to its milestone.** Segments are not links in v1.
- **The grid composer targeting an existing milestone.** Older milestones use their card's
  "+ add step" (D4).
- **Dates or ordering on milestones.** Items keep their optional start/due dates; milestones
  imply neither.
- **Renaming code identifiers** from `grid` to `milestone`.
- **AI breakdown of a milestone** ("suggest steps" in the filling state). Belongs to the
  fireplace-ai work on `feat/ai`.
- **Product tour and landing copy** rewritten around milestones.
