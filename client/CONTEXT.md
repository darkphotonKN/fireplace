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
| **plan of the day** | Informal speech for the *focus prompt*. Use "focus prompt" in code, specs and issues. | **Not** a plan entity scoped to a day. There is no daily-plan type; recurrence lives on items via `scope: daily` and a plan's `dailyReset`. |
| **day gate** | The rule that decides whether `/` opens on the focus prompt or the dashboard: it fires on the first visit of the **browser-local calendar day**, and only when nothing was *touched today*. | Not a route and not a redirect. `/` stays one route in two states; skipping the prompt swaps the dashboard in beneath it. |
| **touched today** | Whether the user has **mutated** a plan or checklist item during the browser-local day. Read by the *day gate*, tracked as a `localStorage` date stamp. A nudge, not an audit: it is wrong on a second device or a cleared browser by design, and the cost of being wrong is one skip click. | Not viewing, scrolling or opening a plan — mutation only. Not a server-side activity record; nothing at `lastActiveAt` exists. Not an analytics figure. |
| **Today** / **Next up** | **One block with two identities.** It is *Today* when it has due, overdue or daily-scope work, and *Next up* when nothing is due, where it lists the first unchecked items of the most recent plan. The heading follows the content so the block never claims work is due when it isn't. | Not two components, and not two blocks stacked. Anyone building a second one has misread this row. |
| **Continue where you left off** | The 2–3 most recently updated plans, surfaced as cards on the dashboard. Each card carries that plan's *progress strip*, derived from the items home already fetches (ADR-0013). | Not "active plans" — there is no active/inactive state on a plan. Recency is `updatedAt` ordering and nothing more; three dormant plans still fill the three slots. |

**"Block" on this surface means a dashboard section** (`TodayBlock`, the *Today* / *Next up*
row). It is a layout word and nothing else. It is **never** a synonym for *milestone*, even
though early grid work called milestones "blocks of progress". That phrase survives only as
a tagline, never as a name in code or specs.

### Plans and milestones

| Term | Means | Does NOT mean |
| --- | --- | --- |
| **milestone** | One finishable piece of a plan: a **top-level task item** together with its steps. Its progress is the share of its task steps that are done. A plan's milestones are all of its top-level tasks, in both views. A milestone's completion is the item's `done`. Closing one leaves it in place, faded. It is not moved, and it is not archived. | Not a dated or sequenced checkpoint. A milestone has no date of its own and implies no order; the item may still carry optional start/due dates. Not a server entity: plan-service only ever sees a checklist item (ADR-0014). Not a top-level **note**, which is drawn as a note card and is never a milestone. Not a daily item: daily stays a list. Not "block" or "card" (rejected names; "card" is only how the grid draws one). |
| **step** | A milestone's child item: one level of nesting under a top-level task. Only **task** steps count toward progress. Note children are shown but never counted. | Not a separate type: a step is a checklist item with a parent. Not ordered progress: steps can be done in any order. A milestone with no task steps, including one whose children are all notes, has no step progress and keeps a **manual tick** instead. |
| **progress strip** | A plan's progress across its milestones: one segment per milestone, each filled to that milestone's own progress, headlined **"N of M milestones"** (N = closed). A no-step milestone's segment is empty until ticked and full after. Shown in the plan header on the long-term tab in both readings (milestone and checklist), and on *Continue where you left off* cards. | Not a step-weighted percentage, so a big milestone can't swamp small ones. Not an average. Not a bare count either. Notes are excluded from the segments and from M. |
| **traced milestone** | The milestone the grid's composer is currently **filling** with steps. It is highlighted in the grid and named in the input, because it may sit anywhere in a row, far from the input. | Not the selected or focused card. Not where the list view's Tab-to-nest lands (`findAddBarTarget`). The grid composer never targets an existing milestone; it only traces the one it just created. |
| **completion nudge** | The **complete** state a milestone enters when all its task steps are done: full meter, glow, and an offer to close it. The user confirms, and only then is `done` set. | Not auto-close: a step added later must never find its milestone already shut. Not a derived checkbox: `done` is still set by a person. |
| **checklist reading** / **milestone reading** | The two readings of the **same plan**. The *checklist reading* is the list view: rows, Tab nests under a task (FS-0007/0008). The *milestone reading* is the grid view: cards, progress strip, and a composer that creates new milestones (in the grid, Tab moves *out* to a new milestone). Code calls the milestone reading `grid` (and the checklist reading `list`); the toggle, a group named "View", labels them "Milestones" and "Checklist" (FS-FSFEC). | Not two models or two data sets. Switching views changes the presentation, never the plan. Only long-term lists get a milestone reading; daily and archived always render as the checklist reading. |
| **milestone page** | One milestone opened at its own address, `/plan/<planId>/m/<milestoneId>`, filling the plan's **middle card** (where the long-term Items card sits) while every other plan block stays put. Holds title, dates, progress, *why it matters*, the *next step* and every step. Reached by the "Open" link on a milestone card; Back returns to the plan (FS-043WP). | Not a modal, drawer or separate screen: only the middle slot swaps. Not its own permission: it inherits the plan's access (FS-12ZMF D1). Not "milestone detail" or "milestone view" in prose (`MilestoneDetail` is only the data shape it reads). Never opened from a note card or a step. |
| **why it matters** | A milestone's one-line intention, ≤ 280 chars, edited and shown **only** on the milestone page; empty, it shows as a quiet "Why does this matter?" prompt. Stored as the generic item field `intention`. | Not a description, note or plan draft. Never on the card, never asked at create. Not a server word: plan-service stores `intention` on any item and never says milestone (ADR-0014). Use "why it matters" in UI and specs, `intention` only for the field. |
| **next step** | The first live, not-done **task** step of a milestone, in stored order (`nextStep()`). Derived on every read, never stored: ticking it lets the following one through. All done → the *completion nudge* takes its slot; no task steps (none, or only notes) → an invitation to add the first. | Not a field, a pin or a user choice. Never a note. Hidden while the milestone is closed. Not "current step" or "focus" (rejected; *Focus* is a retired plan word). |
| **shared plan** | A plan the caller is a **member** of (its `relation` is `member`), shown in the sidebar, My Plans and home alongside owned plans with a small lowercase **shared** mark (`SharedMark`). The member works its items, notes and calendar as the owner does; the plan's name, daily reset, deletion and people stay the owner's. | Not a plan the caller owns and has shared out: owned plans carry no mark in v1. Not a role or a read-only copy. Not "collaborator" or "sharee" (rejected; the person is a **member**, a plan-service term). |

## Boundaries

The client owns **presentation vocabulary only** — the terms above name screens, blocks and
interaction rules that exist nowhere on the server. `day gate`, `touched today` and
`Next up` are client-side inventions and must not leak into API names or server code. The
same holds for **milestone**, **step**, **progress strip**, **traced milestone** and
**completion nudge**, and FS-043WP's **milestone page**, **why it matters** and **next step**: a
milestone is a client reading of a top-level checklist item, not a server entity (ADR-0014). The
one server trace is the generic item field `intention`, which the server stores without knowing
what it is for.

**Plan**, **checklist item**, **scope** (`daily` | `longterm`), **dailyReset**, **archived**,
**done**, **intention**, and **relation** (`owner` | `member`, with **owner** and **member**)
are owned by **plan-service** and reach the client through the generated
contract. The client renders and re-labels them; it does not redefine them. Where a screen
needs a word the server does not have — "recently updated", "due today" — that word is a
client term, derived from server fields (`updatedAt`, `dueDate`) and defined here.
