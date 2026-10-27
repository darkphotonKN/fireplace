# client — Context

> **Ubiquitous language for this bounded context.** Maintained by the `domain-model` skill; other
> skills read it. Vocabulary lives per-service, never at the repo root — two services may
> legitimately use the same word for different things.

## Terms

<!-- Format: | Term | Means | Does NOT mean | -->

### Home surface

| Term | Means | Does NOT mean |
| --- | --- | --- |
| **focus prompt** | The "What's your focus today?" screen: plan type, a focus line, and a start action. It is the **new-plan funnel** — using it creates a plan via `/create-plan`. | Not a picker for which existing plan to work on today. Not "focus splash" (rejected synonym — pick one name). Not the logged-out landing, which is the **product tour**. |
| **plan of the day** | *Retired, ambiguous (FS-02JBZ).* Once informal speech for the *focus prompt*; the owner later used it for what became the **Day plan**. Say "focus prompt" for the new-plan funnel and "Day plan" for the plan-page section, in speech as well as code, specs and issues. | **Not** a plan entity scoped to a day: there is no daily plan type. Not a name for the Day plan either. Recurrence no longer lives on items (`scope: daily` and `dailyReset` are retired); the only recurrence left is the **revision ladder** on learning plans. |
| **day gate** | The rule that decides whether `/` opens on the focus prompt or the dashboard: it fires on the first visit of the **browser-local calendar day**, and only when nothing was *touched today*. | Not a route and not a redirect. `/` stays one route in two states; skipping the prompt swaps the dashboard in beneath it. |
| **touched today** | Whether the user has **mutated** a plan or checklist item during the browser-local day. Read by the *day gate*, tracked as a `localStorage` date stamp. A nudge, not an audit: it is wrong on a second device or a cleared browser by design, and the cost of being wrong is one skip click. | Not viewing, scrolling or opening a plan — mutation only. Not a server-side activity record; nothing at `lastActiveAt` exists. Not an analytics figure. |
| **Today** / **Next up** | **One block with two identities.** It is *Today* when it has due or overdue work (daily-scope items stopped counting when daily scope was retired, FS-02JBZ R91), and *Next up* when nothing is due, where it lists the first unchecked items of the most recent plan. The heading follows the content so the block never claims work is due when it isn't. | Not two components, and not two blocks stacked. Anyone building a second one has misread this row. |
| **Continue where you left off** | The 2–3 most recently updated plans, surfaced as cards on the dashboard. Each card carries that plan's *progress strip*, derived from the items home already fetches (ADR-0013). | Not "active plans" — there is no active/inactive state on a plan. Recency is `updatedAt` ordering and nothing more; three dormant plans still fill the three slots. |

**"Block" on this surface means a dashboard section** (`TodayBlock`, the *Today* / *Next up*
row). It is a layout word and nothing else. It is **never** a synonym for *milestone*, even
though early grid work called milestones "blocks of progress". That phrase survives only as
a tagline, never as a name in code or specs.

### Plans and milestones

| Term | Means | Does NOT mean |
| --- | --- | --- |
| **milestone** | One finishable piece of a plan: a **top-level task item** together with its steps. Its progress is the share of its task steps that are done. A plan's milestones are all of its top-level tasks, in both views. A milestone's completion is the item's `done`. Closing one leaves it in place, faded. It is not moved, and it is not archived. | Not a dated or sequenced checkpoint. A milestone has no date of its own and implies no order; the item may still carry optional start/due dates. Not a server entity: plan-service only ever sees a checklist item (ADR-0014). Not a top-level **note**, which is drawn as a note card and is never a milestone. Not a daily item: daily scope is retired (FS-02JBZ), and a former daily item that was top-level is now a long-term top-level task, so it reads as a milestone like any other. Not "block" or "card" (rejected names; "card" is only how the grid draws one). |
| **step** | A milestone's child item: one level of nesting under a top-level task. Only **task** steps count toward progress. Note children are shown but never counted. | Not a separate type: a step is a checklist item with a parent. Not ordered progress: steps can be done in any order. A milestone with no task steps, including one whose children are all notes, has no step progress and keeps a **manual tick** instead. |
| **progress strip** | A plan's progress across its milestones: one segment per milestone, each filled to that milestone's own progress, headlined **"N of M milestones"** (N = closed). A no-step milestone's segment is empty until ticked and full after. Shown in the plan header on the long-term tab in both readings (milestone and checklist), and on *Continue where you left off* cards. | Not a step-weighted percentage, so a big milestone can't swamp small ones. Not an average. Not a bare count either. Notes are excluded from the segments and from M. |
| **traced milestone** | The milestone the grid's composer is currently **filling** with steps. It is highlighted in the grid and named in the input, because it may sit anywhere in a row, far from the input. | Not the selected or focused card. Not where the list view's Tab-to-nest lands (`findAddBarTarget`). The grid composer never targets an existing milestone; it only traces the one it just created. |
| **completion nudge** | The **complete** state a milestone enters when all its task steps are done: full meter, glow, and an offer to close it. The user confirms, and only then is `done` set. | Not auto-close: a step added later must never find its milestone already shut. Not a derived checkbox: `done` is still set by a person. |
| **checklist reading** / **milestone reading** | The two readings of the **same plan**. The *checklist reading* is the list view: rows, Tab nests under a task (FS-0007/0008). The *milestone reading* is the grid view: cards, progress strip, and a composer that creates new milestones (in the grid, Tab moves *out* to a new milestone). Code calls the milestone reading `grid` (and the checklist reading `list`); the toggle, a group named "View", labels them "Milestones" and "Checklist" (FS-FSFEC). | Not two models or two data sets. Switching views changes the presentation, never the plan. Only long-term lists get a milestone reading; archived always renders as the checklist reading. (Daily, which also did, is retired: FS-02JBZ.) |
| **milestone page** | One milestone opened at its own address, `/plan/<planId>/m/<milestoneId>`, filling the plan's **middle card** (where the long-term Items card sits) while every other plan block stays put. Holds title, dates, progress, *why it matters*, the *next step* and every step. Reached by the "Open" link on a milestone card; Back returns to the plan (FS-043WP). | Not a modal, drawer or separate screen: only the middle slot swaps. Not its own permission: it inherits the plan's access (FS-12ZMF D1). Not "milestone detail" or "milestone view" in prose (`MilestoneDetail` is only the data shape it reads). Never opened from a note card or a step. |
| **why it matters** | A milestone's one-line intention, ≤ 280 chars, edited and shown **only** on the milestone page; empty, it shows as a quiet "Why does this matter?" prompt. Stored as the generic item field `intention`. | Not a description, note or plan draft. Never on the card, never asked at create. Not a server word: plan-service stores `intention` on any item and never says milestone (ADR-0014). Use "why it matters" in UI and specs, `intention` only for the field. |
| **next step** | The first live, not-done **task** step of a milestone, in stored order (`nextStep()`). Derived on every read, never stored: ticking it lets the following one through. All done → the *completion nudge* takes its slot; no task steps (none, or only notes) → an invitation to add the first. | Not a field, a pin or a user choice. Never a note. Hidden while the milestone is closed. Not "current step" or "focus" (rejected; *Focus* is a retired plan word). |
| **shared plan** | A plan the caller is a **member** of (its `relation` is `member`), shown in the sidebar, My Plans and home alongside owned plans with a small lowercase **shared** mark (`SharedMark`). The member works its items, notes and calendar as the owner does; the plan's name, deletion and people stay the owner's (the daily-reset toggle that was also owner-only is gone, FS-02JBZ). | Not a plan the caller owns and has shared out: owned plans carry no mark in v1. Not a role or a read-only copy. Not "collaborator" or "sharee" (rejected; the person is a **member**, a plan-service term). |

### Suggestions (FS-DF4ZB)

| Term | Means | Does NOT mean |
| --- | --- | --- |
| **Today's moves** | *Superseded by the **Day plan** section (FS-02JBZ A5); its show/hide toggle, Add, dismiss and quiet retry carry over.* The block on a plan's daily tab that shows up to three **moves** for today, each tied to one milestone. Replaces the old daily-suggestions block. Drawn once per browser-local day and kept; the user can refresh a capped number of times. | Not daily items: accepting a move never creates a daily-scope item. Not a to-do list the server keeps — moves are proposals until acted on. |
| **move** | *Superseded by **entry** (FS-02JBZ).* One suggestion in *Today's moves*: a milestone, a one-line reason, and either an existing open step to tick off right there or a new step to add under that milestone. | Not an item. Nothing exists on the server until the user ticks or adds. The API calls the milestone a *parent item*; the client translates. |
| **suggested steps** | 3–5 proposed steps for one milestone, asked for from that milestone's card and added one at a time. | Not auto-filled: a step lands only when the user adds it. |
| **suggested milestone** | A proposed new milestone with about three steps, asked for from the grid composer and created only on accept. | Not a server entity: on accept it becomes a top-level task and its children (ADR-0014). |

### Day plan (FS-02JBZ)

The UI name and the domain term are the same, **Day plan**, with no translation step; so are
*entry*, *day tick* and *revision*. The one translation is the anchor: the client names an entry's
top-level ancestor as its **milestone**, where the API says item / parent item (ADR-0014).

| Term | Means | Does NOT mean |
| --- | --- | --- |
| **Day plan** | The section on a plan's page, where the Daily card was, that shows the day's **entries** in up to three labelled groups (at most six entries), made only when the user presses **Plan my day**. Absorbs *Today's moves*. Per user and per browser-local day; on a shared plan each member has their own. | Not *plan of the day* (retired) and not a plan type or a Plan: it is a section of one plan. Not a daily list: daily scope is retired. Not made on page load: opening the page only reads it, and an unplanned day shows **Plan my day**. |
| **Plan my day** / **Plan again** | The section's two actions. *Plan my day* asks for the first day plan of the day; *Plan again* redraws, shows how many redraws are left (`refreshesRemaining`) and retires at 0. The first plan of the day is free; entries already ticked stay put through a redraw. | Not a refresh of the page. Not unlimited: a fourth counted redraw is refused (`BRIEF_REFRESH_LIMIT_REACHED`), and the plan on screen stays. |
| **entry** | One row of the day plan: an existing item to work on today, or a proposed step with **Add** that becomes tickable once added. Shows its milestone's name, its text, a reason, maybe a minutes estimate, maybe "Revision N". | Not an item. Not a move (moves are retired with Today's moves). Dismissing an entry hides it for the session only. |
| **done for today** / **milestone done** | The owner's contrast between the two checks a piece of work can get. *Done for today* is the entry's checkbox, the **day tick**: "I worked on this today". *Milestone done* is the item's own `done`: "this is finished". After a day tick, an inline "Also mark '<item>' done?" offers the second; it disappears after 8 seconds or on the next action. Ticking the item in the plan view sets `done` as before and never touches the day tick. | Not one state shown twice: the two never set each other. When the entry is a step rather than a milestone, the second check is that step's `done`, and the inline ask names the item, never says "milestone". Not a modal. |
| **day tick** | The entry checkbox, meaning *done for today*. Unticking the same day undoes it completely, including any revision it scheduled. A done item's entry stays tickable. | Not the item's `done`. Not available on home's roll-up (read-only rows). |
| **revision** | On learning plans, a topic that comes back for review: first ticked today → "Revision 1" tomorrow, then "Revision 2" three days after that, "Revision 3" a week after that, then it stops. Done items still come back. A small **Don't revise** switch on an entry turns it off for that item. | Not on project plans. Not chosen by the AI: timing is a fixed rule. Not a version or edit of the item. |
| **your daily progression** | The framing of the section's "i" tooltip (same pattern as the plan-focus tooltip): plan your day, tick what you did today, and what is left and what is due for revision come back on their own. The section's purpose in one phrase. | Not a streak, score or analytics figure. Not a separate screen. |
| **basic plan** | A day plan made by rules instead of the AI (budget spent or AI unavailable), labelled as such with a one-line explanation, still offering Plan my day. The server calls it a *rules plan*. On home, the **basic** mark also covers rule picks the client makes itself for a plan with no day plan today. | Not an error state. Same rows, same ticks (on the plan page). |
| **day plan roll-up** | The home block, above Today, showing up to two entries from each of the three focused plans' day plans (six from a single plan), or basic picks where a plan has none today. Read-only, each row links to its plan. | Never generates (never calls Plan my day); never ticks; no Add. Not a cross-plan AI day (parked). |

## Boundaries

The client owns **presentation vocabulary only** — the terms above name screens, blocks and
interaction rules that exist nowhere on the server. `day gate`, `touched today` and
`Next up` are client-side inventions and must not leak into API names or server code. The
same holds for **milestone**, **step**, **progress strip**, **traced milestone** and
**completion nudge**, FS-043WP's **milestone page**, **why it matters** and **next step**,
FS-DF4ZB's **Today's moves**, **suggested steps** and **suggested milestone**, and FS-02JBZ's
**milestone done**, **your daily progression**, **basic plan** and **day plan roll-up**: a
milestone is a client reading of a top-level checklist item, not a server entity (ADR-0014). The
one server trace is the generic item field `intention`, which the server stores without knowing
what it is for.

**Plan**, **checklist item**, **scope** (`daily` | `longterm`; `daily` is a retired value since
FS-02JBZ, still in the contract but refused on create and update), **dailyReset** (vestigial:
still accepted and returned, no effect), **archived**, **done**, **intention**, and **relation**
(`owner` | `member`, with **owner** and **member**) are owned by **plan-service** and reach the
client through the generated contract. **Day plan**, **entry**, **day tick** ("done for today")
and **revision** are owned by **insights-service** and keep the same names on screen; its
**rules plan** is shown as a *basic plan*. The client renders and re-labels them; it does not
redefine them. Where a screen needs a word the server does not have — "recently updated", "due
today" — that word is a client term, derived from server fields (`updatedAt`, `dueDate`) and
defined here.
