# FS-KSJFR: Dashboard home and the day-gated focus prompt

> Status: work-order · SPECIFICATION.md: client `## Home` entries → this FS · Related ADRs: docs/adr/0013-home-composes-capped-fan-out.md · Related FS: FS-0003 (logged-out tour owns the same route), FS-0007 (nesting this reads)

## Summary

The signed-in home becomes a place that tells you where your work stands: what's due today
across the plans you're actually working on, which plans to pick back up, and everything else
one quiet line each. The focus prompt — "What's your focus today?" — is kept, but it stops
being the whole page: it becomes a **day gate** that opens `/` once a day when you haven't
touched anything yet, and skipping it reveals the dashboard beneath rather than throwing you
at the plans list.

Terms in this spec (`day gate`, `touched today`, `Today` / `Next up`, `focus prompt`,
`Continue where you left off`) are defined in `client/CONTEXT.md` and used here exactly as
defined.

## Background: what ships today, and what's wrong with it

`client/src/app/page.tsx` renders `LandingTour` when logged out and a `Dashboard()` component
when logged in. That component is a welcome card stacked above the focus prompt, and nothing
else. Three problems:

1. **The welcome card is not a header convention, it is copy-paste.** The same
   `backdrop-blur-sm rounded-2xl p-8 shadow-lg bg-white/5 dark:bg-gray-900/10` wrapper around
   the same `text-4xl font-bold` heading appears on `/myplans` ("My Plans") and
   `/plan/[planId]`. Home wears the chrome of a content page while having no content, which is
   why it reads as a third list page.
2. **The page is not informative.** Every fact the user might want on arrival — what's due,
   what's in progress, what they were last doing — is available through endpoints that already
   ship, and none of it is shown.
3. **The prompt is unconditional.** It is the new-plan funnel, and it asks every single visit,
   including mid-project, when starting something new is the last thing the user wants.

It is also **visual drift**: raw `rgb(247,111,83)` literals and `bg-white/5` /
`dark:bg-gray-900/10` utilities, both of which `client/docs/design-guideline.md` forbids in new
or changed components. Rebuilding `/` retires that drift on this route.

**On the word "dashboard".** `client/docs/design-guideline.md` is the visual authority and
states the product should feel like "the hearth of your productivity, **not a cold dashboard**".
That is not in tension with this feature, and this paragraph exists so nobody resolves it the
wrong way: what is being built is an *informative home* in warm minimalism — prose-like density,
serif headings, generous spacing, one coral moment. It is not a metrics console. There are no
charts, no gauges, no KPI tiles anywhere in this spec.

## Decisions and why

Carried from scoping, with the reasoning that produced them.

**D1 — The day gate is a state of `/`, not a second route.** `/` stays the single signed-in
home; when the gate fires it paints the focus prompt full-screen, and skipping swaps the
dashboard in underneath with no navigation. Rejected: moving the prompt to `/focus` behind a
redirect (redirect flash, a second guarded route, a back-button trap), and overlaying the
prompt on a live dashboard (pays for the dashboard's fetch on visits where the prompt is the
only thing touched).

**D2 — The gate fires on the first visit of the day, and only if nothing was touched today.**
A plain daily cadence was rejected on a specific ground: the focus prompt *creates a plan*, so
a daily ask nudges plan sprawl — a user three weeks into a project would be asked every morning
to start something new. The idle condition makes it fire when starting something genuinely is
the sensible next move.

**D3 — "Touched today" is a `localStorage` date stamp.** Read synchronously so the gate decides
before paint. Known wrong on a second device or a cleared browser, where the prompt shows when
it shouldn't and costs one skip click — accepted, because it is a nudge and not an audit.
Rejected: deriving it from item `updatedAt` (true across devices, but the gate then cannot
decide until the fetch lands, so home opens on a spinner or flashes the wrong state), and a
server-side `lastActiveAt` (truthful and cheap to read, but a new field plus a write on every
mutation path — server work inside a client slice). The server-side version is the honest
upgrade if multi-device turns out to matter.

**D4 — Three blocks: Today, Continue where you left off, Your other plans.** An AI-nudge block
was cut: `getDailyInsights` would put an LLM call on every home load.

**D5 — Today falls back to Next up.** `dueDate` is optional and sparse, so a strict due-date
block would be permanently empty for users who never set dates. The heading follows the
content, so the block never claims work is due when it isn't.

**D6 — Continue and Your-other-plans are disjoint.** Every plan appears exactly once. Letting
the list stay a complete index with the cards as emphasis was rejected: on a four-plan account
it looks like a rendering bug.

**D7 — Capped fan-out over the three most recently updated plans.** Recorded as **ADR-0013**,
including the consequence this spec must surface rather than hide: a due item in a
long-dormant plan will not appear on home.

**D8 — The greeting lives on the focus prompt only.** The dashboard opens on a status line,
which is warm by being useful. No surface greets the user by name twice, and the prompt is the
day's opening ritual, so that is where the warmth belongs.

**D9 — Today is interactive: tick inline**, optimistically. This is the interaction that makes
home a place you *use* rather than a table of contents. **Refined during spec-writing:** a
ticked row does **not** disappear. It stays in place, marked done and dimmed, for the rest of
the visit, and only the count decrements. A vanishing row collapses the list under the user's
cursor and makes the next tick a mis-click; and if rows vanished, finishing the last one would
yank in unrelated rows from another plan mid-interaction.

**D10 — Home does not wear the shared header card.** That block stays the convention on
`/myplans` and `/plan/[planId]` — they are siblings and should rhyme. Home is the only page
whose job is orientation rather than content.

**D11 — No momentum tile, no graphs.** `GET /api/analytics/user/{userId}` is a documented 501
stub (`services/api-gateway/internal/useranalytics/typed_analytics.go`): `tasksCompleted`,
`completionRate`, `currentStreak` and `activePlansCount` are declared in the contract and
returned by nothing. Streaks are not a design choice deferred; they are a data path that does
not exist.

**D12 — The gate's skip reveals the dashboard**, so the existing "Skip to plans" copy is wrong.

**D13 — Zero plans gets an invitation, not three empty blocks.** A brand-new account trips the
gate, skips, and would otherwise land on the coldest screen in the product.

**D14 — Any item that qualifies can appear in Today, at any depth**, carrying its parent's text
as context. Leaves-only was rejected because it silently drops a dated parent with no signal;
parents-with-counts was rejected because it leaves nothing real to tick, which kills D9.

## Requirements

### The day gate

- **R1** — `/` renders the logged-out product tour unchanged when unauthenticated. FS-0003 and
  the existing no-wait-on-`isLoading` behavior are untouched by this feature.
- **R2** — When authenticated, `/` evaluates the day gate on mount and renders **either** the
  focus prompt **or** the dashboard. It is one route in two states; no redirect, no second URL.
- **R3** — The gate fires when **both** hold: no gate has fired yet during the current
  browser-local calendar day, **and** nothing was *touched today*.
- **R4** — The gate's decision is made from `localStorage` only, synchronously, before first
  paint. The dashboard's data fetch must not be able to change which state is shown.
- **R5** — Firing the gate records that it fired, so a second visit the same day goes straight
  to the dashboard whether or not the user acted on the prompt.
- **R6** — The gate is evaluated **on mount only**. A tab left open across midnight keeps the
  state it has; there is no re-gate on window focus or visibility change.
- **R7** — The focus prompt keeps its current behavior: plan type, focus line, and a start
  action that navigates to `/create-plan` with `name`, `focus` and `planType`. Nothing about
  the funnel changes.
- **R8** — The prompt carries the greeting ("Welcome back, {name}", falling back to "there").
  It is the only surface in this feature that greets by name.
- **R9** — The prompt's skip affordance reveals the dashboard in place. Its copy must no longer
  say "plans", and it must not navigate to `/myplans`.
- **R10** — Skipping records that the gate fired (R5) but does **not** record a touch — skipping
  is not activity.

### The touched-today stamp

- **R11** — Creating, updating, deleting, reordering, archiving or re-dating a plan or a
  checklist item records *touched today* for the browser-local day.
- **R12** — The stamp is written in the **API layer** (`client/src/api/*.ts` mutating
  functions), at the single choke point every mutation already passes through — not sprinkled
  through components. A mutation path that forgets the stamp is a bug this placement prevents.

  **The paths this requirement quantifies over.** R11/R12 makes a claim about *every* mutation
  in the client, so the claim needs an inventory; this table is it. It was missing from the
  first draft of this spec, and its absence is why the broken row below was found at code
  review instead of at spec time. This feature adds no endpoints (see *Out of Scope*), so there
  is no `## API surface` section — but "adds an endpoint" was the wrong trigger for writing
  down a surface a requirement ranges over.

  | API-layer function | Reached from | Stamps |
  | --- | --- | --- |
  | `createPlan` | `app/create-plan/page.tsx` | yes |
  | `updatePlan` | *no caller* — dead function today | yes |
  | `deletePlan` | `app/myplans/page.tsx` | yes |
  | `toggleDailyReset` | `components/Todo.tsx` | yes |
  | `createChecklistItem` | `Todo.tsx`, `services/api.ts` adapter | yes |
  | `updateChecklistItem` | `Todo.tsx`, `services/api.ts` adapter | yes |
  | `updateChecklistDates` | `Todo.tsx`, `ItemActions`, `services/api.ts` adapter | yes |
  | `reorderChecklists` | `services/api.ts` adapter → `Todo.tsx` | yes |
  | `archiveChecklistItem` | `Todo.tsx`, `services/api.ts` adapter | yes |
  | `deleteChecklistItem` | `Todo.tsx` (both delete paths), `services/api.ts` adapter | yes — fixed by I-0058 |

  Every row now stamps. The delete row did not when this table was written: `Todo.tsx:707`
  and `:1373` deleted with a hand-written `fetch` that carried no `Authorization` header and
  read an undefined env var, so the call never reached this layer and did not work at all
  (**I-0058**, now fixed). R11 was left as written throughout — deleting an item *is* a touch,
  the product was what was wrong, and narrowing the requirement to match the bug would have
  written the bug into the record.
- **R13** — Reading is not touching. Viewing a plan, scrolling, opening the sidebar, searching
  and navigating record nothing.
- **R14** — A failed mutation does not record a touch.
- **R15** — Absent, malformed or unreadable `localStorage` is treated as "not touched today" and
  "gate has not fired": the gate errs toward showing the prompt, never toward an error.

### The dashboard

- **R16** — The dashboard opens on a **status line**, not a greeting and not a card: the
  weekday, the count of due work, and the number of plans that count was drawn from — e.g.
  "Thursday · 2 due across 3 plans". When Next up is showing it states that instead — e.g.
  "Thursday · nothing due — 6 waiting".
- **R17** — The status line's "across N plans" is the honest surfacing of ADR-0013: it says how
  far home looked, so the cap is visible in the product rather than discovered as a bug.
- **R18** — The dashboard does **not** use the `backdrop-blur-sm rounded-2xl p-8 shadow-lg`
  header card. That block remains on `/myplans` and `/plan/[planId]`.
- **R19** — Blocks render in order: Today/Next up, Continue where you left off, Your other
  plans.
- **R20** — Data is assembled by the capped fan-out of ADR-0013: `GET /api/plans`, sorted by
  `updatedAt` descending, then checklists for the **top three** plans — one request plus one per
  focused plan, so **at most four**, whatever the account holds.
- **R21** — The three item fetches are issued in parallel, and one failing does not blank the
  page: blocks that can be rendered from what arrived, render.
- **R22** — Loading is progressive, not a full-page spinner: the plan-derived blocks appear as
  soon as `GET /api/plans` lands; the item-derived parts settle in after. No block may change
  height as it settles in a way that moves a row under the cursor.

### Today / Next up

- **R23** — *Today* lists, from the three fetched plans: items due today, items overdue and not
  done, and items with `scope: "daily"`.
- **R24** — When that set is empty, the same block becomes *Next up* and lists the first few
  unchecked items of the most recently updated plan **that still has unchecked work**, in
  `sequence` order.

  > **Amended during implementation.** This originally read "the most recently updated plan",
  > flatly, which renders an empty block whenever that plan happens to be finished — the exact
  > dead block D5 exists to prevent. R30 names the source plan on every row, so skipping a
  > completed plan is visible to the reader rather than silent. The code was right and the
  > requirement was too literal.
- **R25** — The heading always describes what is actually listed. "Today" never appears above
  rows that are not due.
- **R26** — Items with `type: "note"` never appear: notes are not work.
- **R27** — Archived items never appear anywhere on home.
- **R28** — Completed items do not appear in the initial set (a row ticked *during the visit*
  stays — see R33).
- **R29** — An item qualifies at any depth. A qualifying item with a parent shows its parent's
  text as context on the row.
- **R30** — Every row names the plan it came from, and the plan name links into that plan.
- **R31** — "Due today" and "overdue" are computed with the local-time helpers in
  `client/src/lib/itemDates.ts`. Date-only strings must not be parsed with `new Date(...)`
  directly; a new `isDueToday`-style helper belongs beside `isRangePast`, not inline in the
  component.

  > **Wire format, settled by I-0059.** The contract publishes `startDate`/`dueDate` as
  > `format: date-time` (`model.go:222`, `*time.Time` → RFC3339), while these helpers first
  > accepted only `YYYY-MM-DD` — so every comparison returned false against the live API and
  > Today could show `scope: "daily"` items only. The rule now: **the date portion is
  > authoritative and the time and zone are ignored.** These are date-only values in the
  > domain — the client sends `YYYY-MM-DD`, the server stores midnight UTC — so resolving the
  > instant to a local date would shift every due date back a day west of Greenwich.
- **R32** — `scheduledTime` is the deprecated mirror of `startDate` and is read only as a
  fallback where `startDate` is absent, consistent with `ItemDateChip`.
- **R33** — A row can be ticked from home. The tick is optimistic: the row marks done
  immediately, stays in place dimmed for the rest of the visit, and the status line's count
  decrements.
- **R34** — A failed tick restores the row to not-done, restores the count, and tells the user
  through the existing toast.
- **R35** — Completing every row does not swap the heading to "Next up" mid-visit and does not
  pull in replacement rows. The block settles into a done state until the next load.

### Continue where you left off / Your other plans

- **R36** — *Continue* shows the three most recently updated plans — the same three the fan-out
  already fetched, so it costs no extra request.
- **R37** — Each Continue card shows the plan name, its focus line, its type, and done/total
  progress computed from the fetched items.
- **R38** — There is no staleness floor. Three dormant plans still fill the three slots;
  "recently updated" is ordering, not a claim of recency.
- **R39** — *Your other plans* lists the remaining plans, one quiet line each, excluding the
  three in Continue. No plan appears twice on the page.
- **R40** — The list caps at six rows and links out to `/myplans` for the rest. Rows show name
  and type only: progress is not available for these plans without breaking the cap.
- **R41** — With three or fewer plans the *Your other plans* block is absent entirely, not
  rendered empty.
- **R42** — Both blocks are read-only. Creating and deleting plans stay on their own surfaces.

### First run

- **R43** — With zero plans the dashboard renders a single invitation panel instead of the
  three blocks: what a plan is, and one action that reopens the focus prompt in place.
- **R44** — The invitation does not navigate away and does not duplicate the logged-out tour.

### Sidebar

- **R45** — The sidebar still starts collapsed on `/`.
- **R46** — The "your plans live in the side panel" discovery toast **no longer fires on `/`**.
  Home now lists plans on the page, so the tip points at a panel showing what the user can
  already see. It continues to fire on plan pages, unchanged.

### Visual conformance

- **R47** — `client/docs/design-guideline.md` governs every visual decision here and wins over
  this spec where they disagree.
- **R48** — Tokens only. No raw hex, no `rgb(...)` literals, no `*-gray-*` utilities. The
  existing drift on this route is removed, not extended.
- **R49** — **One coral moment.** On the dashboard the coral accent belongs to the Today/Next up
  block and nothing else — Continue cards, plan rows and the status line stay neutral. On the
  focus prompt it stays the start action, as today.
- **R50** — Both themes are handled at the token layer. No per-component dark-mode overrides.
- **R51** — Motion follows the guideline: colour/opacity transitions, no layout jumps as blocks
  settle.

## User Stories

1. As a returning user, I want home to tell me what's due today, so that I know where to start
   without opening three plans.
2. As a returning user, I want to tick something off from home, so that a two-minute task
   doesn't cost a page load.
3. As a returning user, I want to see which plans I was last working on, so that I can pick one
   back up without remembering its name.
4. As a returning user, I want the plans I'm not currently in to stay out of my way but still be
   reachable, so that home stays calm as my plan count grows.
5. As a user mid-project, I want home to stop asking me what I want to build today, so that I
   don't accumulate half-started plans.
6. As a user starting a genuinely new day with nothing touched, I want the focus prompt to
   greet me, so that the ritual I like is still there.
7. As a user who saw the prompt this morning, I want it gone for the rest of the day, so that
   every visit isn't an interruption.
8. As a user who skips the prompt, I want to land on my work, so that skipping is progress
   rather than a detour to another index page.
9. As a user who never sets due dates, I want the first block to still show me something to do,
   so that the page isn't dead for the way I work.
10. As a user who does set due dates, I want overdue items surfaced with today's, so that
    nothing quietly rots.
11. As a user with daily-reset habits, I want my daily items counted as today's work, so that
    recurring things aren't invisible.
12. As a user, I want each row on home to say which plan it belongs to, so that a task name out
    of context still means something.
13. As a user with nested checklists, I want a dated sub-item to appear with its parent's name,
    so that "call them back" isn't an orphan.
14. As a user, I want a ticked row to stay where it is, so that the next row doesn't jump under
    my cursor.
15. As a user with a flaky connection, I want a failed tick to visibly come back, so that I
    don't believe I finished something I didn't.
16. As a new user with no plans, I want home to invite me to start one, so that my first screen
    after signing up isn't three empty boxes.
17. As a user, I want home to tell me how far it looked, so that a missing item from an old
    plan is explainable rather than a bug I report.
18. As a user, I want home to look different from my plans list, so that I can tell at a glance
    which page I'm on.
19. As a user, I want home to feel warm rather than like an analytics console, so that the
    product still feels like the thing I chose.
20. As a user, I want home to open fast, so that the first thing I see isn't a spinner.
21. As a user on a second device, I want an extra prompt at worst, so that a stale signal costs
    me a click and not my work.
22. As a user, I want the sidebar tip to stop telling me where my plans are when they're on the
    screen, so that the product doesn't feel unfinished.
23. As a developer, I want the touched-today stamp written at one choke point, so that adding a
    mutation later doesn't silently break the gate.
24. As a developer, I want home's request count bounded, so that a user with forty plans doesn't
    become a load problem.
25. As a designer, I want this route on tokens, so that the next theme change doesn't have to
    hunt for hardcoded coral.

## Acceptance Criteria

- [ ] Logged out, `/` is the product tour, unchanged.
- [ ] Logged in with the gate firing, `/` shows the focus prompt with the name greeting.
- [ ] Logged in with the gate not firing, `/` shows the dashboard with no flash of the prompt.
- [ ] Skipping the prompt reveals the dashboard without a URL change or a navigation.
- [ ] A second visit the same day goes straight to the dashboard.
- [ ] Ticking an item anywhere in the app on day N stops the gate firing for the rest of day N.
- [ ] Only viewing plans on day N does not stop the gate firing.
- [ ] A mutation that fails does not count as a touch.
- [ ] Cleared `localStorage` shows the prompt rather than erroring.
- [ ] The dashboard's first line is the status line — weekday, due count, plans looked at.
- [ ] The status line reads correctly in both the Today and the Next up case.
- [ ] Home issues at most four requests at any plan count — `1 + min(planCount, 3)` — verified
      at 1 (two requests), 3, 4 and 12 (four).
- [ ] With due items present, the block is headed Today and lists due, overdue and daily items.
- [ ] With no due items, the same block is headed Next up and lists unchecked items from the
      most recent plan.
- [ ] Note-type items and archived items appear nowhere on home.
- [ ] A dated child item appears with its parent's text as context.
- [ ] Every row links to its plan.
- [ ] Ticking a row marks it done in place, dims it, and decrements the count.
- [ ] A failed tick restores the row, restores the count, and toasts.
- [ ] Completing every row leaves the heading alone and pulls in no replacement rows.
- [ ] Continue shows three plans with focus line, type and done/total.
- [ ] No plan appears in both Continue and Your other plans.
- [ ] With three or fewer plans, Your other plans is absent.
- [ ] With more than nine plans, Your other plans caps at six and links to `/myplans`.
- [ ] With zero plans, the dashboard is the invitation panel and its action reopens the prompt.
- [ ] The sidebar starts collapsed on `/` and the discovery toast does not fire there.
- [ ] The discovery toast still fires on a plan page.
- [ ] No raw hex, `rgb(...)` literal or `*-gray-*` utility remains in the home route's files.
- [ ] Coral appears once on the dashboard, in the Today/Next up block.
- [ ] Both themes render correctly with no per-component dark overrides.
- [ ] `/myplans` and `/plan/[planId]` keep their header card, unchanged by this work.

## Edge States

| Situation | Behavior |
| --- | --- |
| Zero plans | Invitation panel (R43); the gate still fires and its skip lands there. |
| Exactly one plan | Continue shows it; Your other plans absent; Today draws from that one plan. |
| All plans dormant for months | Continue still shows three. No staleness floor (R38). |
| `GET /api/plans` fails | Home shows the status line's neutral form and an error state with a retry. The gate's decision is unaffected — it never depended on the fetch. |
| One of the three item fetches fails | The other two render. The failed plan is omitted from Today and its Continue card shows no progress rather than a wrong zero. |
| **All three** item fetches fail | Today says it could not load today's work and offers a retry; the status line falls back to the weekday alone rather than claiming a count it does not have. Continue cards still render, without progress. |
| Tick fails | Row and count restored, toast shown (R34). |
| Two ticks in flight | Each resolves against its own row; a failure restores only its own. |
| Every Today row completed | Heading holds, block settles into a done state (R35). |
| Tab open across midnight | State holds until reload (R6). Accepted. |
| Second device / cleared storage | Prompt shows when it arguably shouldn't; one skip click (R15). Accepted. |
| Deep-nested qualifying item | Shown with parent context (R29). Grandparents are not chained into the breadcrumb. |
| Item due today **and** daily-scope | Appears once, not twice. |
| Plan with zero items | Continue card shows the plan with 0/0 progress rather than being hidden. |
| Unauthenticated | Product tour; none of this evaluates (R1). |

## Out of Scope

- **Any server change.** No new endpoints, no contract regeneration, no gRPC work. The
  aggregate `GET /api/home` was considered and rejected in ADR-0013; picking it up later
  supersedes that ADR.
- **Implementing the analytics data path.** `GET /api/analytics/user/{userId}` stays a 501 stub,
  and streaks, completion rates and activity charts stay out of the product until it isn't.
- **AI nudges on home.** `getDailyInsights` and `suggest-videos` stay on plan surfaces.
- **The logged-out landing and the product tour** (FS-0003).
- **The plan creation flow** at `/create-plan` — the prompt's funnel is unchanged.
- **`/myplans`** — its search, paging, deletion and header card are untouched. Home links to it.
- **Cross-device activity truth.** A server-side `lastActiveAt` is the named upgrade if the
  localStorage stamp proves insufficient; it is not built here.
- **Re-theming the rest of the app's drift.** Only the home route's files are retired off raw
  colours by this work.
- **Calendar, notes and sharing surfaces** — nothing about them appears on home.
