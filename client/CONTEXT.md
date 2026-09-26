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
| **Continue where you left off** | The 2–3 most recently updated plans, surfaced as cards on the dashboard. | Not "active plans" — there is no active/inactive state on a plan. Recency is `updatedAt` ordering and nothing more; three dormant plans still fill the three slots. |

## Boundaries

The client owns **presentation vocabulary only** — the terms above name screens, blocks and
interaction rules that exist nowhere on the server. `day gate`, `touched today` and
`Next up` are client-side inventions and must not leak into API names or server code.

**Plan**, **checklist item**, **scope** (`daily` | `longterm`), **dailyReset**, **archived**
and **done** are owned by **plan-service** and reach the client through the generated
contract. The client renders and re-labels them; it does not redefine them. Where a screen
needs a word the server does not have — "recently updated", "due today" — that word is a
client term, derived from server fields (`updatedAt`, `dueDate`) and defined here.
