# FS-02JBZ: Day plan

> Status: work-order · SPECIFICATION.md: insights-service `## Day Plan` (day plan built on the daily brief, day ticks, revision ladder, global spend cap with a rule-based fallback); plan-service `### Checklist Items` "Daily scope retired, existing daily items kept as long-term", plus the struck lines "Per-plan daily-reset toggle", "Bulk daily reset" and "Daily-items-only-via-AI rule"; api-gateway `## Insights` "Day plan endpoints"; client `## Suggestions` "Day plan on the plan page"; client `## Home` "Day plan roll-up on home" → this FS · Builds on: FS-DF4ZB (plan tree, prompt assembly, strict structured call, output validation, plan-state hash, refresh counter and cap, usage ledger, Today's moves; amended here, §Amendments to FS-DF4ZB) · Related ADRs: docs/adr/0019-generation-is-explicit-and-bounded.md, docs/adr/0013-home-composes-capped-fan-out.md (request bound raised, R90), docs/adr/0014-milestones-are-a-client-reading-of-items.md, docs/adr/0007-user-edits-win.md, docs/adr/0002-contract-planes-code-first-openapi.md, docs/adr/0004-error-representation-rfc9457.md, docs/adr/0005-request-validation-and-contract-design.md

## Summary

A plan's **Daily** card becomes the **Day plan**: when the user presses **Plan my day**, the AI
picks up to six things to do today from what is left in the plan, groups related ones under a
short label, puts nearer due dates first and gives each a rough time estimate. Each entry is
ticked **done for today**, which is not the same as finishing the item; a light inline ask offers
to mark the item itself done too. On learning plans, a topic ticked for the first time comes
back as "revision 1" the next day, then "revision 2" and "revision 3" on a fixed ladder. If the
AI is unavailable or the platform's daily token budget is spent, the user gets a **basic** plan
built by rules instead of an error. Home shows a short roll-up of the day plans of the three
plans home already follows.

Alongside, the old **daily scope** goes: existing daily items become ordinary long-term items,
nothing new can be created as daily, and the nightly reset and its per-plan toggle are removed.

Terms used here (`day plan`, `entry`, `group`, `day tick`, `revision`, `revision ladder`,
`rules plan`, `token budget`, plus FS-DF4ZB's `plan tree`, `parent item`, `plan-state hash`,
`rule gate`, `generation`) are defined in `services/insights-service/CONTEXT.md`. Client terms
(`Day plan`, `entry`, `basic plan`, `day plan roll-up`) are in `client/CONTEXT.md`.

## Background: what exists when this FS starts

This FS is built after FS-DF4ZB's I-DF4ZB-1 and I-DF4ZB-3 to I-DF4ZB-9 have landed in the same
run. It assumes, and reuses without re-specifying:

- **insights-service:** the plan tree (DF4ZB R1), prompt assembly order (R2–R6), one strict
  structured call per generation with a per-generator model and token ceiling (R7–R8), the usage
  ledger with kind, model and input / cached input / output tokens per generation (R9), output
  validation that drops, never repairs (R10–R12), the plan-state hash (R19), the per
  (plan, user, date) refresh counter with an atomic cap of 3 (R17), the date window and
  `BRIEF_DATE_OUT_OF_RANGE` (R21), and the `ErrorInfo` error seam (R52).
- **gateway:** `getDailyBrief` and `refreshDailyBrief` (typed Huma, DF4ZB §API surface);
  `checklist-suggestion-daily` already removed (I-DF4ZB-9).
- **client:** Today's moves in the plan page's Daily card. That card is
  `<Todo fixedTaskType="daily" dailyAIOnly />` in `client/src/app/plan/[planId]/layout.tsx`;
  Today's moves is its own component mounted from `Todo.tsx` on the daily side (I-DF4ZB-7), with a
  refresh action (I-DF4ZB-8).

And on the daily-scope side, as the code stands on this branch:

- **plan-service:** `checklist_items.scope` is `daily | longterm` (CHECK, migration 000008);
  create and update accept both (`internal/checklistitem/service.go`). `DailyReset` RPC runs one
  bulk CTE that un-completes done daily items in plans with `daily_reset = true`
  (`BulkResetDailyItems`). `ToggleDailyReset` RPC flips `plans.daily_reset`. Latest migration is
  000021.
- **gateway:** a robfig/cron job at `0 0 14 * * *` (`internal/jobs/daily_reset_job.go`, wired in
  `config/routes.go`) calls plan-service `DailyReset`. `PATCH /api/plans/{id}/toggle-daily-reset`
  (`toggleDailyReset`) is **serialized** (Huma, `internal/gateway/plan/typed_plans.go`), as are
  `createChecklist`, `updateChecklist` and `generateAINotes`. **No unserialized endpoint is
  touched by this FS, so no slice ⓪ is needed.** AI notes (`internal/notes/service.go`) reads the
  plan's **daily** items as its context.
- **client:** the settings view in `Todo.tsx` offers "Refresh daily tasks" (owner only), calling
  `toggleDailyReset` (`client/src/api/plans.ts`). The legacy `/plans/[planId]` route renders
  `<Todo />` with Daily | Long-term tab buttons. Home's Today rule counts daily-scope items as
  today's work (`client/src/lib/homeWork.ts`, `isTodays`).

## Decisions and why

Every decision the scoping session left open was pre-answered by the owner's charter for this
run. Each is folded in below, marked **(charter default)** with its reason. Where the charter
was silent, this FS chose, marked **(spec default)** and listed in §Spec defaults. Decisions
the scoping session locked itself are marked **(scoped)**. Nothing here was run through
challenge-me (the owner declined), so the expensive calls are flagged: the daily-scope
retirement (a data migration and a removed scheduled job) and the new server state and
endpoints (contract surface, ADR-0002).

**D1. Day-tick and revision state lives in insights-service.** (charter default) In
`fireplace_insights`, keyed per (user, plan, item). plan-service is untouched except for the
daily-scope retirement (D12). The day plan is insights' read model beside FS-DF4ZB's brief.
ADR-0014 holds: no milestone server entity; the API says item and parent item. *Reason:* a day
tick is a fact about one user's day, not about the item; keeping it out of plan-service keeps
`done` meaning one thing, keeps shared plans per user (FS-12ZMF) without a plan-service schema
change, and keeps the generated read model next to the generator that reads it.

**D2. A day is at most 6 entries in at most 3 groups.** (charter default) Each group has a short
label and each AI entry a rough minutes estimate. 6 and 3 are fixed constants until the
"how much do you plan to do in a day" knob exists (parked). *Reason:* small enough to finish,
big enough for a mixed day, and bounded output keeps the structured call cheap.

**D3. Only entries pointing at existing items are tickable.** (charter default) A proposed new
step shows **Add**, which is FS-DF4ZB's accept path (ADR-0007 "propose": nothing is written until
the user accepts), and becomes tickable once added. *Reason:* a day tick has to be about a real
item, or the revision ladder has nothing to come back to.

**D4. The day tick is the primary checkbox.** (charter default) It means "done for today".
After a tick, an inline link "Also mark '<item>' done?" appears and dismisses itself; no modal.
**Yes** sets the item's `done` through the existing item update. Ticking the real item in the
plan view keeps working as today, independently. *Reason:* most daily work moves an item along
without finishing it (one LeetCode topic, one session of a course); finishing is the rarer
event and gets the lighter ask.

**D5. Unticking on the same day removes the tick and the revision that tick scheduled.**
(charter default) *Reason:* an untick is an undo; it must leave no trace that would bring the
item back as a revision.

**D6. The revision ladder.** (charter default) Learning plans only. +1, +3 and +7 days in the
user's local date, at most 3 revisions, started by an item's **first** day tick. A per-item
"no revisions" switch. Due revisions are injected into the prompt as top-priority candidates
labelled "revision N". *Reason:* spaced repetition with a fixed, learnable rule; ADR-0019 clause 5
says timing is a rule and never model output. (Interpretation of the offsets: §Spec defaults S6.)

**D7. On demand, never on read.** (charter default) `GET` returns the cached day plan, or an
empty "not planned yet" state, and **never generates**. **Plan my day** generates. The first
generation of the day does not count against FS-DF4ZB's refresh cap of 3; later ones do.
*Reason:* ADR-0019 clauses 1–2: tokens are spent only on days someone asks. This amends FS-DF4ZB's
draw-on-first-read (§Amendments A1).

**D8. A global spend cap.** (charter default) Today's summed token usage from FS-DF4ZB's usage
ledger is compared with `LLM_DAILY_TOKEN_BUDGET` (dev default 200000, prod default 2000000).
Over budget, generators fail with errcode `LLM_BUDGET_EXHAUSTED`; the day plan catches it and
returns the rule-based plan. *Reason:* ADR-0019 clauses 3–4: one bound for the platform on top
of each feature's own limits, and a low dev value makes real calls safe to test.

**D9. Rule-based fallback ranking.** (charter default) Revisions due → overdue → due soonest →
next open step of the most recently updated milestone. Same entry shape, `source = rules`,
shown as a "basic" plan. *Reason:* deterministic, explainable, and built only from data the
service already reads.

**D10. Home roll-up, model-free.** (charter default) For ADR-0013's same three plans, home reads
cached day plans (`GET`, which never generates). A plan with none today contributes client-side
rule picks from the items home already holds. 2 entries per plan, up to 6; a single plan fills
the block. Read-only rows linking to the plan. ADR-0013's decision holds (same three plans,
shipped endpoints only); **its request bound rises from 4 to 7** (R90). *Reason:* no new
aggregate endpoint and no generation on a read, at the price of three cheap cached reads.

**D11. UI name "Day plan", with an "i" tooltip.** (charter default) The tooltip explains daily
progression, using the same pattern as the plan-focus tooltip in
`client/src/app/plan/[planId]/layout.tsx`. The Day plan replaces the plan page's Daily card;
FS-DF4ZB's Today's moves block becomes this section. *Reason:* the owner's "plan of the day",
named as a section of a plan, not as a plan type.

**D12. The daily scope is retired.** (charter default; scoped as decision 8) A migration sets
`scope = 'longterm'` on every daily row. Creating or updating an item to `daily` is refused
with 400 and a domain code. plan-service's `DailyReset` RPC, the gateway's nightly reset cron and
the daily-reset toggle endpoint are removed. The `daily_reset` column and the `scope` field stay
in schema and contract for now; dropping them is later cleanup. The breaking-change gate flags
the toggle removal; that is expected and allowlisted. Removing serialized endpoints follows
ADR-0002; no unserialized endpoint is touched (§Background). *Reason:* the day plan is the
daily surface now; a second, manual daily list would compete with it and the reset job would
keep resetting rows nobody can create.

**D13. Browser-local date.** (charter default) The client sends its local `YYYY-MM-DD`, with
FS-DF4ZB's bounds and code (`400 · BRIEF_DATE_OUT_OF_RANGE`). *Reason:* one date rule across the
insights surface.

**D14. The prompt.** (charter default) Groups entries by topic affinity with a short group
label, ranks nearer due dates higher, includes plan type and focus, and uses strict structured
output (FS-DF4ZB's single-call pattern). *Reason:* grouping and sizing are what a cheap model is
good at; the schema keeps output bounded and parseable.

**D15. Per plan.** (scoped, decision 4) The AI day plan is generated per plan, grouping within
that plan. A cross-plan AI day is parked.

**D16. Real LLM calls in dev, guarded.** (scoped, decision 1) No fixture or replay layer. The
guard is D8's low dev budget.

## Amendments to FS-DF4ZB

FS-DF4ZB is a work order in its own right and is **not edited**. This FS changes it from the
outside; where they disagree, this FS wins for the slices that implement it.

- **A1. No read generates.** DF4ZB R15 (a read generates on a miss) and the read-like branch of
  R16 are superseded by ADR-0019 clause 2. For the day plan this is R14 below. For the brief it
  is achieved by A2.
- **A2. The brief's HTTP surface is retired once the client reads the day plan.** (spec default
  S1) `getDailyBrief`, `refreshDailyBrief`, the `GetDailyBrief` and `RefreshDailyBrief` RPCs and
  the brief-only generation path are removed after the Day plan section replaces Today's moves
  (R62, R92). Their consumer is gone and A1 would otherwise need them rewritten. The brief's storage,
  hash, validation, counter and ledger stay, because the day plan reuses them.
- **A3. The refresh counter is shared.** A counted Plan my day and a counted brief refresh draw on
  the same per (plan, user, date) counter, so the two together can never exceed DF4ZB's cap of 3
  counted generations for a key (R20).
- **A4. The ledger becomes load-bearing.** Every insights-service model call records usage,
  including the single checklist suggestion and the video search terms, which DF4ZB left
  unrecorded (R31). ADR-0019 requires it.
- **A5. Today's moves becomes the Day plan section** (D11). Its show/hide toggle, Add, dismiss and
  quiet-retry behaviors carry over (R72–R84).

## Requirements

### Day plan: storage and read (insights-service)

- **R1.** A **day plan** belongs to one (plan, user, date), where `date` is the user's local
  calendar date sent by the client (D13). At most one day plan is stored per key, in
  `fireplace_insights`, beside the brief (D1, S1).
- **R2.** A stored day plan carries: its key, `source` (`ai` | `rules`), the time it was drawn,
  the plan-state hash it was drawn from (R17), and 0–3 **groups**, each with a label and 1–6
  **entries**, in order. Total entries across groups ≤ 6 (D2).
- **R3.** An **entry** has an id (usable as a suggestion id by FS-DF4ZB's feedback when it lands,
  see O1), a kind, text, a one-line reason, an optional minutes estimate and an optional
  revision number. Kinds (spec default S2):
  - `existing_item`: points at an item of the plan by id. Its parent item id is recorded when the
    item has a parent.
  - `new_step`: proposes text for a new child of a named parent item.
- **R4.** `minutes` is an integer 5–240 on AI entries and absent on rules entries (S10).
  `revision` is 1–3 and only ever set by the server (R45), never by the model.
- **R5.** A group label is non-empty, trimmed and at most 40 characters (S14).

### Day plan: generation (insights-service)

- **R6.** **Plan my day** draws a day plan for the key with **one** strict structured-output call
  (DF4ZB R7) through its own generator config: kind `day_plan` in the ledger, model and output
  ceiling configurable, defaulting to `gpt-5-mini` with minimal reasoning (DF4ZB R8).
- **R7.** The prompt follows DF4ZB R2's order (static rules → snippets → plan data → request). The
  static rules tell the model to group entries by topic affinity under short labels, to put
  nearer due dates and overdue items first, to give each entry a rough minutes estimate, and to
  respect the D2 bounds. The plan data adds, to DF4ZB's plan tree: the plan's type and focus,
  today's date, the due revisions as top-priority candidates labelled "revision N" (R44), and
  the entries already ticked today with their group labels (R12). Prompt prose may say
  "milestone"; no server type, proto message or path does (DF4ZB R6).
- **R8.** **Candidates.** An `existing_item` entry may point at an open, non-archived, long-term
  **task** item of the plan, top-level or child, or at an item with a due revision even if it is
  done (S7). A `new_step` entry's parent must be an open, top-level, long-term task, and its text
  must not duplicate (case- and whitespace-insensitive) an existing child of that parent
  (DF4ZB R10).
- **R9.** **Validation** drops, never repairs, never regenerates (DF4ZB R12): an entry that fails
  R8; a second entry for the same item; a second `new_step` with the same text; an empty reason;
  a `minutes` outside 5–240 (the entry is kept with `minutes` absent rather than dropped,
  S14); an entry in a fourth or later group; an entry beyond the sixth. A group left with no
  entries is dropped. A plan may end up with zero entries (S14).
- **R10.** The server stamps `revision = N` on every `existing_item` entry whose item has
  revision N due (R44). The model's output carries no revision field.
- **R11.** A successful draw replaces the stored day plan for the key in one step; readers see
  the old plan or the new one, never a mix.
- **R12.** **Ticked entries survive a redraw.** (spec default S4) Every entry ticked today keeps
  its place, text, group and tick in the new plan. The model is told they are fixed and done and
  fills the remaining slots (6 minus the ticked count), either joining a ticked entry's group
  by its exact label or opening new groups, within the 3-group bound. R9 applies to what it adds.

### Day plan: on demand, cap and gate (insights-service + gateway)

- **R13.** **Plan my day** (`planMyDay`) draws or redraws the day plan for the key.
- **R14.** **Read** (`getDayPlan`) returns the stored day plan for the key, or, when there is
  none, a `not_planned` state with no groups. A read **never** calls the model, never runs a
  rule-based draw and never copies a plan forward from another date (D7, ADR-0019).
- **R15.** **The free draw.** The first successful AI draw for a key does not count against the
  cap. Every later successful AI draw for the key counts (D7), drawing on the counter shared with
  the brief (A3). At most 3 counted draws per key; one that would be the fourth is refused with
  `429 · BRIEF_REFRESH_LIMIT_REACHED` and the stored plan is left untouched (DF4ZB R17 semantics,
  atomic check-and-increment).
- **R16.** A draw that falls back to rules (R36) does not count and does not use up the free draw.
- **R17.** **Rule gate.** The day plan's plan-state hash covers DF4ZB R19's plan tree plus the
  plan's type and focus, the date, the set of due revisions for the date and the items ticked
  today. Plan my day on a key whose stored `ai` plan has an equal hash returns the stored plan
  with no generation and no count. (spec default S3: there is no cross-date carry-forward, since
  the date is in the hash.) A stored `rules` plan never satisfies the gate: Plan my day after a
  basic plan always tries the model.
- **R18.** A plan with no candidates (no open long-term task and no due revision) is stored as a
  planned `ai` plan with no groups and no generation; it does not count and does not use the free
  draw.
- **R19.** The response carries `refreshesRemaining` (0–3) so the client knows, before asking,
  whether a redraw is possible. Before the free draw is used, a redraw costs nothing and the
  number is 3.
- **R20.** The counter is per (plan, user, date) and shared with `refreshDailyBrief` while that
  operation exists (A3).
- **R21.** **Concurrency.** Two Plan my day requests for one key never spend more than one
  draw's allowance between them: one plan is stored and both callers receive it. A generation
  lost to the race is still recorded in the ledger (DF4ZB R22) and does not count.
- **R22.** `date` must lie within one day of the server's UTC date, inclusive; otherwise
  `400 · BRIEF_DATE_OUT_OF_RANGE` (D13, DF4ZB R21). Applies to read and Plan my day.
- **R23.** A plan the caller cannot access is `404 · NOT_FOUND`, as plan-service's `GetPlan`
  answers today. On a shared plan (FS-12ZMF) each user has their own day plan, ticks and
  revisions.

### Day ticks (insights-service + gateway)

- **R24.** A **day tick** marks an entry's item done for today. It is stored per (user, plan,
  item, date), where the date is the day plan's date, with a reference to the entry (D1).
- **R25.** **Tick** (`tickDayPlanEntry`) addresses an entry by id (spec default S5). For an
  `existing_item` entry, the body carries no item id (or the entry's own). For a `new_step`
  entry, the body must carry `itemId`: the id of the item the user created with Add. The server
  checks through plan-service that it is a non-archived task, a child of the entry's parent, in
  the same plan, then **binds** the entry to it: the entry becomes `existing_item` for that item,
  keeping its text and reason.
- **R26.** Tick refuses with `400 · DAY_PLAN_ENTRY_NOT_TICKABLE` when: a `new_step` entry has no
  `itemId`, or one that fails R25's check; an `existing_item` entry's item is now archived or
  deleted; or the body's `itemId` contradicts an already-bound entry. An entry that does not
  exist or is not the caller's is `404 · NOT_FOUND`.
- **R27.** Tick is idempotent: ticking a ticked entry returns it unchanged and schedules
  nothing new.
- **R28.** A tick never changes the item's `done`, and changing `done` never ticks or unticks
  (D4). The two are independent.
- **R29.** **Untick** (`untickDayPlanEntry`) removes the entry's tick for its date and undoes
  what that tick did to the revision ladder (R46). Unticking an unticked entry succeeds and does
  nothing.
- **R30.** Tick and untick are allowed only while the entry's day plan date is within R22's
  window; outside it they are refused with `400 · BRIEF_DATE_OUT_OF_RANGE` (spec default S17).

### Global spend cap (insights-service)

- **R31.** Every model call insights-service makes records usage in FS-DF4ZB's ledger:
  the guided generators, the day plan, the single checklist suggestion (kind
  `checklist_suggestion`) and the video search terms (kind `video_terms`) (A4). Each transport
  retry records what the provider reported. Embedding calls are not generation and are not
  recorded against the budget (spec default S8).
- **R32.** Before any model call, the service sums today's tokens in the ledger and compares the
  sum with `LLM_DAILY_TOKEN_BUDGET`. "Today" is the server's **UTC** calendar day; the sum is
  input plus output tokens (cached input is part of input) (S8).
- **R33.** At or over budget, the call is not made and the generator fails with
  `LLM_BUDGET_EXHAUSTED` as `ErrorInfo` detail on `ResourceExhausted`, which the gateway seam maps
  to `429 · LLM_BUDGET_EXHAUSTED` (S8). Retrying later is correct; retrying now is not, which is
  why this is not 503.
- **R34.** `LLM_DAILY_TOKEN_BUDGET` is read at boot. Unset, it is 200000. Local dev sets 200000
  (`docker-compose.yml`, `services/insights-service/.env`); production defaults to 2000000 in
  `docker-compose.prod.yml` (D8).
- **R35.** The check is a circuit breaker, not an invoice: concurrent calls near the limit may
  overshoot by roughly their output ceilings (ADR-0019, accepted).

### Rule-based fallback (insights-service)

- **R36.** When Plan my day's generation fails, the day plan is drawn by rules instead and
  stored with `source = rules`. "Fails" means: `LLM_BUDGET_EXHAUSTED` (D8), or the model
  unreachable or timing out after transport retries, or output invalid against the schema
  (scoped decision 9.5, spec default S9). The response is `200` with the rules plan, never 429 or
  503 for these causes.
- **R37.** **Ranking** (D9): due revisions (oldest due first), then overdue items (most overdue
  first), then items with a due date (soonest first), then the next open step of the parent item
  most recently updated. "Next open step" is the first open, non-archived task child in stored
  order; a parent with no open task child contributes itself if it is open. Ties keep plan order.
- **R38.** **Shape** (spec default S10): the same entries and groups. Due revisions sit in a
  first group labelled "Revisions". Every other entry is grouped under its top-level ancestor,
  labelled with that item's description cut to 40 characters. Entries are taken in rank order
  until 6 entries, skipping any that would open a fourth group. All entries are
  `existing_item`; `minutes` is absent; each `reason` names its rule ("Revision 2 due",
  "Overdue since Oct 20", "Due Oct 28", "Next step").
- **R39.** Ticked entries survive into a rules redraw exactly as in R12.
- **R40.** The rules draw makes no model call and records nothing in the ledger.

### Revision ladder (insights-service + gateway)

- **R41.** Revisions exist on **learning** plans only (D6). On a project plan no revision is ever
  scheduled or offered.
- **R42.** **Start.** An item's first day tick ever, for this user on this plan, schedules
  revision 1, due on the tick's date + 1 day, unless revisions are switched off for the item.
- **R43.** **Climb.** Ticking an entry stamped `revision = N` completes revision N. For N = 1 it
  schedules revision 2 due on that tick's date + 3 days; for N = 2, revision 3 due + 7 days; for
  N = 3 the ladder ends (spec default S6). Dates are the user's local dates as carried by the day
  plan. A day tick of an item that has no revision due (before the due date, or after the ladder
  ended) schedules nothing.
- **R44.** **Due.** Revision N is due for a date when its due date is on or before that date and
  it is not completed; it stays due (overdue) until ticked. A due revision is a top-priority
  candidate (R7, R37) as long as its item is not archived or deleted, **including when the item
  is done** (S7).
- **R45.** The server, never the model, decides which revision is due and stamps it on entries
  (R10); the model only decides whether and where a due revision goes in today (ADR-0019
  clause 5).
- **R46.** **Untick undoes** (D5): unticking the tick that started the ladder removes revision 1
  (the item has no first tick any more); unticking the tick that completed revision N reopens
  revision N and removes revision N+1 if that tick scheduled it.
- **R47.** **Switch.** Revisions can be switched off and on per (user, plan, item)
  (`setItemRevisions`). Off cancels the item's open revision and schedules no new one. On
  schedules nothing by itself; the item's next day tick schedules the next rung (revision
  completed-count + 1), up to 3 (S7). The switch is on by default.
- **R48.** The switch addresses an item of the plan: an item not in the plan, or a plan the
  caller cannot access, is `404 · NOT_FOUND`. Setting it on a project plan is accepted and has no
  effect.
- **R49.** Entries on learning plans carry the item's switch state (`revisionsEnabled`); entries
  on project plans and `new_step` entries do not.

### Daily-scope retirement (plan-service + gateway)

- **R50.** A plan-service migration (000022) sets `scope = 'longterm'` on every `daily` row,
  archived or not. Nothing is deleted; `done`, dates, parent links, intention and archived state
  are kept. Migrated rows are placed after the existing long-term siblings at the same level
  (same plan and parent), keeping their relative order (spec default S16).
- **R51.** Creating an item with `scope = 'daily'`, or updating an item to it, is refused with
  `400 · DAILY_SCOPE_RETIRED`, carried as `ErrorInfo` (DF4ZB R52's seam) on `InvalidArgument`.
  The `scope` field and its `daily | longterm` enum stay in the proto and the HTTP contract, so
  the request is well-formed and the refusal is a domain decision (ADR-0005). Listing or
  reordering with `scope = daily` stays accepted and finds nothing (S15).
- **R52.** plan-service's `DailyReset` and `ToggleDailyReset` RPCs and their request/response
  messages are removed, with `BulkResetDailyItems`. Generated code is regenerated, never
  hand-edited.
- **R53.** The gateway's nightly reset job (`internal/jobs/daily_reset_job.go`, its manager
  wiring in `config/routes.go`) and the plan adapter's reset path are removed.
- **R54.** `PATCH /api/plans/{id}/toggle-daily-reset` (`toggleDailyReset`) is removed.
  `openapi.yaml` and the generated client are regenerated; the breaking gate's finding is
  recorded in `.oasdiff-ignore` citing FS-02JBZ.
- **R55.** `plans.daily_reset` stays in the schema; `createPlan` and `updatePlan` still accept
  `dailyReset` and plans still return it. It has no effect. Dropping the column and the field is
  later cleanup.
- **R56.** AI notes (`generateAINotes`) reads the plan's long-term task items as context instead
  of its daily items, so it does not silently lose all context (spec default S13).

### Gateway and contract

- **R57.** The operations in §API surface are typed Huma handlers. Request types list writable
  fields only; identity comes from the JWT and is forwarded as `user_id` over gRPC. JSON bodies
  and new query parameters are camelCase (DF4ZB S11). `openapi.yaml` and the generated client are
  regenerated, never hand-edited.
- **R58.** New codes in `common/errcode`: `LLM_BUDGET_EXHAUSTED`, `DAY_PLAN_ENTRY_NOT_TICKABLE`,
  `DAILY_SCOPE_RETIRED`. No others. Each is added by the slice whose failure needs it.
- **R59.** Every operation also returns `401`, `403`, `503 · SERVICE_UNAVAILABLE` (insights or
  plan-service unreachable) and `500`, as the existing insights operations do.
- **R60.** `getChecklistSuggestion` and `getSuggestedVideos` gain `429 · LLM_BUDGET_EXHAUSTED`
  (R33). Nothing else about them changes.
- **R61.** No API path, contract type, proto message or server type name contains "milestone"
  (ADR-0014).

### Client: Day plan section (plan page)

- **R62.** The plan page's Daily card is replaced by a **Day plan** section in the same place
  (`client/src/app/plan/[planId]/layout.tsx`). It mounts its own component, not a `<Todo>`, and
  absorbs FS-DF4ZB's Today's moves (A5). With the last daily `<Todo>` mount gone, `Todo.tsx`'s
  daily path (`fixedTaskType = 'daily'`, `dailyAIOnly`, daily fetches) is deleted.
- **R63.** The header reads "Day plan" with an "i" button whose hover tooltip explains daily
  progression: plan your day, tick what you did today, and finished items and revisions come back
  on their own. Same pattern as the plan-focus tooltip (D11).
- **R64.** It reads `getDayPlan` for the browser-local date on mount and again when the local date
  changes.
- **R65.** **Not planned:** a short line and a **Plan my day** button. Pressing it calls
  `planMyDay` and shows a loading state in the section only.
- **R66.** **Planned:** groups in order, each with its label; each entry shows its milestone's
  name (the entry's top-level ancestor, resolved from loaded items), its text, its reason, its
  minutes estimate when present, and "Revision N" when stamped.
- **R67.** Entries are resolved against the plan's loaded long-term items at render time
  (DF4ZB R59 parity): an entry whose item is deleted or archived is hidden; a `new_step` entry
  whose parent is gone, archived or closed is hidden. An `existing_item` entry whose item is
  done shows it as completed and stays tickable.
- **R68.** **Plan again** calls `planMyDay`, shows `refreshesRemaining`, and retires at 0. A redraw
  that comes back with the same `generatedAt` says nothing has changed. `429 ·
  BRIEF_REFRESH_LIMIT_REACHED` retires the action and keeps the plan on screen. The client
  switches on `code`, never on `detail`.
- **R69.** A `source = rules` plan is labelled **basic plan** with a one-line explanation that
  the AI wasn't available, and still offers Plan my day.
- **R70.** A plan the caller cannot reach (`404`) renders no section; the page's own "not here"
  state applies (FS-12ZMF R32).
- **R71.** Loading shows a skeleton of the section; a failed read shows a quiet retry, never an
  error banner over the plan.
- **R72.** The section keeps Today's moves' show/hide toggle and its remembered `localStorage`
  preference (DF4ZB R57).
- **R73.** A `new_step` entry shows **Add**: it creates a long-term task as a child of the entry's
  parent through the existing create endpoint (DF4ZB R61); the entry then shows as added and
  becomes tickable (D3). A failed create keeps the entry and shows the existing error.
- **R74.** Every entry can be dismissed; it hides for the rest of the session (S18). Recording
  dismissals is FS-DF4ZB's feedback, not this FS (O1).

### Client: day ticks and revisions in the section

- **R75.** A tickable entry has a checkbox: the **day tick** (D4). Ticking calls
  `tickDayPlanEntry` (optimistic, reverted on failure); unticking calls `untickDayPlanEntry`
  (optimistic, reverted on failure).
- **R76.** A just-added `new_step` entry is ticked with the created item's id (R25). After a
  reload, an added entry is recognised by a child of its parent whose description matches the
  entry text (case- and whitespace-insensitive) and is ticked with that child's id.
- **R77.** After a tick, when the entry's item is not already done, an inline link "Also mark
  '<item>' done?" appears beside the entry and dismisses itself after 8 seconds or on any other
  action (S12). **Yes** sets the item's `done` through the existing item update (optimistic,
  reverted on failure). No modal. Unticking removes the link.
- **R78.** Ticking or completing the real item in the plan view keeps working exactly as today
  and does not change the day tick (R28).
- **R79.** `400 · DAY_PLAN_ENTRY_NOT_TICKABLE` reverts the tick quietly and refreshes the entry's
  state from a fresh read.
- **R80.** On learning plans, an `existing_item` entry offers a small "Don't revise" switch bound
  to `revisionsEnabled`, calling `setItemRevisions` (optimistic, reverted on failure).
- **R81.** An entry stamped `revision = N` reads "Revision N" next to its text.
- **R82.** Ticked entries stay in the section, shown ticked, for the rest of the day, and survive
  Plan again (R12).
- **R83.** A `rules` plan's entries tick exactly like AI entries.
- **R84.** The section never creates a daily-scope item.

### Client: home roll-up

- **R85.** Home gains a **Day plan** block (spec default S11: above the Today block). For each of
  ADR-0013's three focused plans it calls `getDayPlan` for the browser-local date. It never calls
  `planMyDay`.
- **R86.** From a plan with a stored day plan, the block takes its first two entries in plan
  order that resolve against home's items (R67's rules); a ticked entry is shown ticked.
- **R87.** From a plan with no day plan today (or whose read failed), the block takes two
  client-side rule picks from the items home already holds, ranked overdue → due soonest → next
  open step of the most recently updated milestone (D9 without revisions, which the client does
  not hold). Those rows carry the "basic" mark.
- **R88.** Two entries per plan, up to 6. With one focused plan, that plan fills the block (up to
  6). Rows are read-only: no tick, no Add; each links to its plan.
- **R89.** A plan with nothing to show contributes nothing; a block with no rows is not rendered.
- **R90.** **ADR-0013's request bound rises from 4 to 7:** `GET /api/plans`, three checklist
  fetches and three day-plan reads. ADR-0013's decision itself holds: the same three plans, only
  shipped endpoints, no home aggregate.

### Daily-scope retirement (client)

- **R91.** The legacy `/plans/[planId]` route's Daily | Long-term tab buttons are removed; it
  shows the long-term list. The settings view's "Refresh daily tasks" toggle and
  `toggleDailyReset` in `client/src/api/plans.ts` are removed. Home's Today rule no longer counts
  daily-scope items (`isTodays`). New items are always created long-term.

### Retiring the brief's HTTP surface (A2)

- **R92.** Once the client reads the day plan (R62) and nothing calls them, `GET
  /api/insights/daily-brief`, `POST /api/insights/daily-brief/refresh`, the `GetDailyBrief` and
  `RefreshDailyBrief` RPCs and the brief-only generation path are removed. The brief's storage,
  plan-state hash, validation, counter and ledger are kept for the day plan. If the breaking gate
  flags the removal, the finding is allowlisted citing FS-02JBZ.

## User Stories

1. As a planner, I want the app to pick what I should do today from what's left, so that I don't
   have to decide every morning.
2. As a planner, I want items due sooner to come first, so that deadlines don't sneak up on me.
3. As a planner, I want related things grouped under one short label, so that my day reads as a
   few themes rather than a scattered list.
4. As a planner, I want a rough time estimate on each entry, so that I can tell whether the day
   fits.
5. As a planner, I want no more than six things in my day, so that it stays doable.
6. As a planner, I want the plan made only when I ask, so that nothing is spent on days I don't
   look.
7. As a planner, I want my day plan to stay put when I reload, so that it doesn't shuffle under me.
8. As a planner, I want to plan again a few times a day, so that a bad draw isn't final.
9. As a planner, I want to see how many redraws I have left, so that the cap doesn't surprise me.
10. As a planner, I want my first plan of the day not to use up a redraw, so that asking once is
    always free.
11. As a planner, I want a redraw to tell me when nothing has changed, so that I know why I got the
    same plan.
12. As a planner, I want to tick an entry as done for today without finishing the item, so that a
    session of work counts without closing the milestone.
13. As a planner, I want to be offered "also mark it done?" right after ticking, so that finishing
    an item costs one more click, not a trip to the plan.
14. As a planner, I want that offer to disappear on its own, so that it never blocks me.
15. As a planner, I want ticking an item in the plan view to keep working as before, so that
    nothing I know breaks.
16. As a planner, I want to untick by mistake and leave no trace, so that a misclick doesn't
    schedule revisions.
17. As a planner, I want entries I already ticked to stay when I plan again, so that a redraw
    never hides what I did.
18. As a planner, I want proposed new steps to be addable and then tickable, so that a good idea
    becomes real work in one flow.
19. As a learner, I want a topic I worked on to come back as a revision the next day, then after
    three days, then after a week, so that I actually remember it.
20. As a learner, I want revisions to stop after three, so that old topics don't haunt my days.
21. As a learner, I want to switch revisions off for an item, so that topics I don't need to
    revise stay out.
22. As a learner, I want revisions labelled "Revision N", so that I know why they're back.
23. As a learner, I want revisions to come back even for topics I've marked done, so that
    finishing a topic doesn't end its review.
24. As a project planner, I want no revisions on project plans, so that tasks don't repeat.
25. As a planner, I want a basic plan when the AI is down or out of budget, so that I always get a
    plan.
26. As a planner, I want a basic plan clearly labelled, so that I don't mistake it for the AI's.
27. As a planner on several plans, I want home to show a few entries from each plan's day, so that
    I see my day at a glance.
28. As a planner with one plan, I want home's block filled from that plan, so that it isn't sparse.
29. As a planner, I want home rows to link to their plan, so that I can act on them there.
30. As a planner, I want a short explanation of what the Day plan is, so that I understand ticks
    and revisions.
31. As a planner sharing a plan, I want my day plan and ticks to be mine, so that a co-member's day
    doesn't overwrite mine.
32. As a planner, I want my old daily items kept as ordinary items, so that nothing I wrote is lost
    when the daily list goes.
33. As the owner, I want a single daily token budget across every generator, so that the worst day
    costs a known amount.
34. As the owner, I want a low budget in dev, so that testing with real calls can't run away.
35. As the owner, I want every model call recorded, including the old generators, so that the
    budget sees all spend.
36. As the owner, I want revision timing decided by a rule, so that it is predictable and costs
    nothing.
37. As a client developer, I want distinct codes for "out of budget", "not tickable" and "daily
    retired", so that I can switch on them.
38. As an operator, I want the nightly reset job gone, so that there is no scheduled job left
    resetting rows nobody can create.

## Acceptance Criteria

### Server: day plan

- [ ] `getDayPlan` for a key with no stored plan returns `not_planned` with empty `groups` and makes
      no model call and no rules draw.
- [ ] `planMyDay` makes at most one structured call and stores at most 6 entries in at most 3
      non-empty groups, each label 1–40 characters.
- [ ] The prompt carries plan type, focus, today's date, due revisions labelled "revision N" and
      today's ticked entries; the static rules are byte-identical across two calls.
- [ ] Invalid entries (bad item, duplicate item or text, empty reason, fourth group, seventh
      entry) are dropped; an out-of-range `minutes` is dropped from its entry; nothing is
      repaired or regenerated.
- [ ] The first successful AI draw of a key leaves `refreshesRemaining` at 3; the next three
      lower it to 0; the fourth counted draw returns `429 · BRIEF_REFRESH_LIMIT_REACHED` and
      leaves the plan unchanged.
- [ ] A counted brief refresh and a counted Plan my day on the same key share the cap.
- [ ] Plan my day with nothing changed (same hash) returns the stored plan with no model call and
      no count; after a rules plan it always tries the model.
- [ ] Two concurrent Plan my day requests on one key store one plan and both receive it; the
      allowance spent is at most one draw.
- [ ] A plan with no candidates stores an empty planned plan with no model call and no count.
- [ ] A date outside server UTC ±1 day returns `400 · BRIEF_DATE_OUT_OF_RANGE` on read, Plan my
      day, tick and untick.
- [ ] Day plans, ticks and revisions on a shared plan are per user.
- [ ] Usage is recorded with kind `day_plan`.

### Server: ticks and revisions

- [ ] Ticking an `existing_item` entry stores a tick for (user, plan, item, date) and does not
      change the item's `done`; ticking it again changes nothing.
- [ ] Ticking a `new_step` entry without `itemId`, or with one that isn't a non-archived task
      child of its parent, returns `400 · DAY_PLAN_ENTRY_NOT_TICKABLE`; with a valid `itemId`
      it binds the entry and ticks it.
- [ ] Ticking an entry whose item was archived or deleted returns `400 ·
      DAY_PLAN_ENTRY_NOT_TICKABLE`; another user's entry returns `404 · NOT_FOUND`.
- [ ] Ticked entries keep their place, group and tick through Plan my day (AI and rules).
- [ ] On a learning plan, an item's first tick schedules revision 1 for date + 1; ticking
      revision 1 schedules revision 2 for that date + 3; ticking revision 2 schedules revision 3
      for that date + 7; ticking revision 3 schedules nothing.
- [ ] On a project plan, no revision is ever scheduled.
- [ ] A due revision stays due until ticked and is offered even if its item is done; not if it is
      archived or deleted.
- [ ] Unticking the first tick removes revision 1; unticking a revision tick reopens that revision
      and removes the one it scheduled.
- [ ] Switching revisions off cancels the open revision and schedules no more; switching on lets
      the next tick schedule the next rung.
- [ ] Entries for items with a due revision carry `revision = N` set by the server; the model
      schema has no revision field.

### Server: spend cap and fallback

- [ ] With today's ledger sum at or over `LLM_DAILY_TOKEN_BUDGET`, no model call is made and the
      generator fails with `LLM_BUDGET_EXHAUSTED` (`429` on `getChecklistSuggestion` and
      `getSuggestedVideos`).
- [ ] The single checklist suggestion and video search terms record usage; transport retries are
      recorded; embedding calls are not counted.
- [ ] The budget window is the server's UTC day; unset, the budget is 200000; dev config sets
      200000 and prod defaults to 2000000.
- [ ] Plan my day over budget, with the model down, or with schema-invalid output returns `200`
      with a `source = rules` plan, stored, not counted, with no ledger row for the rules draw.
- [ ] The rules plan ranks due revisions, then overdue, then due soonest, then the next open step
      of the most recently updated parent item; revisions sit in a "Revisions" group; other
      groups are top-level ancestors; entries have no `minutes` and a rule-named reason.

### Server: daily-scope retirement

- [ ] After migration 000022 no `daily` row exists; migrated rows keep `done`, dates, parents,
      intention and archived state, and sort after existing long-term siblings in their old order.
- [ ] Creating or updating an item to `daily` returns `400 · DAILY_SCOPE_RETIRED`; listing with
      `scope=daily` returns an empty list.
- [ ] `DailyReset` and `ToggleDailyReset` are gone from the proto, plan-service and the gateway;
      the nightly job no longer exists.
- [ ] `PATCH /api/plans/{id}/toggle-daily-reset` is gone; the breaking-gate finding is allowlisted
      citing FS-02JBZ; `make -C services/api-gateway gates` is otherwise green.
- [ ] `createPlan`/`updatePlan` still accept `dailyReset` and plans still return it.
- [ ] AI notes reads long-term task items.

### Server: brief retirement and contract

- [ ] `getDailyBrief`, `refreshDailyBrief` and their RPCs are removed after the client switches;
      the day plan still uses the brief's hash, validation, counter and ledger.
- [ ] `common/errcode` gains exactly `LLM_BUDGET_EXHAUSTED`, `DAY_PLAN_ENTRY_NOT_TICKABLE` and
      `DAILY_SCOPE_RETIRED`.
- [ ] No API path, contract type, proto message or server type name contains "milestone".
- [ ] `openapi.yaml` and `client/src/api/generated` are regenerated, never hand-edited.

### Client

- [ ] The plan page shows a "Day plan" section where the Daily card was, with an "i" tooltip
      explaining daily progression; no daily `<Todo>` remains.
- [ ] Not planned → "Plan my day"; planned → groups, labels, milestone names, text, reasons,
      minutes, "Revision N".
- [ ] Plan again shows the count, retires at 0, says when nothing changed, and handles 429 by
      `code`.
- [ ] A basic plan is labelled as such.
- [ ] Ticking and unticking are optimistic and revert on failure; ticks never change `done`.
- [ ] After a tick, "Also mark '<item>' done?" appears inline, disappears after 8 seconds, and Yes
      sets `done`.
- [ ] A new step can be added, then ticked; no daily-scope item is ever created.
- [ ] On learning plans the "Don't revise" switch reflects and sets `revisionsEnabled`.
- [ ] Home shows the Day plan block: two entries per focused plan, six from a single plan, rule
      picks marked basic where no plan exists, read-only rows linking to the plan; at most 7
      requests on load and never `planMyDay`.
- [ ] The legacy route has no Daily tab; settings has no daily-reset toggle; home's Today ignores
      scope.
- [ ] Tests pass.

## Edge States

| Situation | Behavior |
| --- | --- |
| First visit of the day | `not_planned`; the section offers Plan my day; nothing generated (R14). |
| Plan has no open long-term task and no due revision | Plan my day stores an empty planned plan, no call, no count (R18); the section says there is nothing to plan and invites a first milestone. |
| Model returns only invalid entries | Planned `ai` plan with zero entries (R9); same message as above. |
| Budget spent before Plan my day | Rules plan, 200, labelled basic (R36). |
| Budget spent mid-day after an AI plan | The stored AI plan stays; a later Plan my day returns a rules plan replacing it, ticked entries kept (R39). |
| Provider down or timing out | Rules plan (R36). |
| Insights-service down | `503`; the section shows the quiet retry (R71). |
| Plan again after a basic plan, budget back | Tries the model; if it is the first successful AI draw it is free (R15–R17). |
| Fourth counted draw | `429 · BRIEF_REFRESH_LIMIT_REACHED`; action retired, plan kept (R15, R68). |
| Two tabs press Plan my day at once | One plan stored, both receive it (R21). |
| Plan edited after the plan was drawn | Plan kept for the day; Plan again is not gated (hash differs) and counts unless it is the free draw. |
| Entry's item ticked done in the plan view | Entry shows the item completed, still tickable; day tick unaffected (R28, R67). |
| Entry's item archived or deleted | Entry hidden (R67); a tick on it is refused (R26). |
| New step added, then page reloaded | Recognised by its matching child and tickable with that id (R76). |
| New step added, text edited before ticking | Not recognised after reload; the entry still offers Add (a second Add would create a duplicate child; accepted, rare). |
| Tick then untick the same day, first tick | Tick gone, revision 1 removed (R46). |
| Untick a revision tick | Revision reopened, next rung removed (R46). |
| Revision slips several days | Stays due (overdue) and ranks first in a rules plan; the next rung is counted from the day it is finally ticked (R43, R44). |
| Item done, revision due | Offered; done items still revise (R44, S7). |
| Revisions switched off with one open | Open revision cancelled (R47). |
| Plan type changed learning → project | No revision offered or scheduled from then on (R41); stored rows kept. |
| Plan type changed project → learning | The next first-tick-ever of an item starts its ladder; earlier ticks on the project plan already count as first ticks, so those items start nothing (S6). |
| Local midnight passes with the section open | Next read sends the new date and shows `not_planned` (R64). |
| Client clock off by more than a day | `400 · BRIEF_DATE_OUT_OF_RANGE`; quiet retry. |
| Tick on yesterday's plan within the UTC window | Allowed (R30); the client only ever shows today's plan, so in practice unreachable. |
| Shared plan | Each member has their own day plan, ticks and revisions; "Also mark done?" sets the shared item's `done` for everyone, as any item edit does (R23). |
| Home: one focused plan | Up to 6 rows from it (R88). |
| Home: a plan's day plan read fails | That plan contributes rule picks, marked basic (R87). |
| Home: plan with no items | Contributes nothing (R89). |
| Existing daily items at migration | Become long-term items after the existing ones (R50). |
| Old client creates a daily item | `400 · DAILY_SCOPE_RETIRED` (R51). |
| Daily item that was done at migration | Stays done as a long-term item; nothing resets it any more (R50, R53). |
| Ledger sum crosses the budget mid-flight | Concurrent calls may overshoot slightly (R35). |

## API surface

All operations are JWT-protected, routed to insights-service over gRPC (plan-service for the
removed toggle), with identity from the token, never the body. Shape failures (malformed uuid or
date, unknown member, bad enum) are `422 · VALIDATION_FAILED` at the boundary; domain refusals
come from the owning service (ADR-0005). Every operation also returns `401 · UNAUTHENTICATED`,
`403 · FORBIDDEN`, `503 · SERVICE_UNAVAILABLE` and `500 · INTERNAL_ERROR` (R59); the Errors column
lists only what is specific.

| Op | Method + Path | Query/Params | Request body | Response | Errors |
|----|---------------|--------------|--------------|----------|--------|
| getDayPlan | `GET /api/insights/day-plan` | `planId` uuid, required · `date` `YYYY-MM-DD`, required | — | `200` **DayPlan** | `400 · BRIEF_DATE_OUT_OF_RANGE` · `404 · NOT_FOUND` (plan) |
| planMyDay | `POST /api/insights/day-plan` | — | `planId` uuid, required · `date` `YYYY-MM-DD`, required | `200` **DayPlan** (`source` `ai` or, on fallback, `rules`) | `400 · BRIEF_DATE_OUT_OF_RANGE` · `404 · NOT_FOUND` (plan) · `429 · BRIEF_REFRESH_LIMIT_REACHED` |
| tickDayPlanEntry | `PUT /api/insights/day-plan/entries/{id}/tick` | `id` uuid (path) | `itemId` uuid, optional (required for an unbound `new_step` entry) | `200` **DayPlanEntry** | `400 · DAY_PLAN_ENTRY_NOT_TICKABLE` · `400 · BRIEF_DATE_OUT_OF_RANGE` · `404 · NOT_FOUND` (entry unknown or not the caller's) |
| untickDayPlanEntry | `DELETE /api/insights/day-plan/entries/{id}/tick` | `id` uuid (path) | — | `204` | `400 · BRIEF_DATE_OUT_OF_RANGE` · `404 · NOT_FOUND` |
| setItemRevisions | `PUT /api/insights/day-plan/items/{itemId}/revisions` | `itemId` uuid (path) | `planId` uuid, required · `enabled` boolean, required | `204` | `404 · NOT_FOUND` (plan, or item not in plan) |
| getChecklistSuggestion | `GET /api/insights/checklist-suggestion` | unchanged | — | unchanged | adds `429 · LLM_BUDGET_EXHAUSTED` |
| getSuggestedVideos | `GET /api/insights/suggest-videos` | unchanged | — | unchanged | adds `429 · LLM_BUDGET_EXHAUSTED` |
| createChecklist · updateChecklist | `POST /api/plans/{id}/checklists` · `PATCH /api/plans/{id}/checklists/{checklist_id}` | unchanged | `scope` enum unchanged (`daily` \| `longterm`) | unchanged | adds `400 · DAILY_SCOPE_RETIRED` (scope `daily`) |
| ~~toggleDailyReset~~ | ~~`PATCH /api/plans/{id}/toggle-daily-reset`~~ | — | — | **Removed** (R54) | breaking gate flags it; allowlisted citing FS-02JBZ |
| ~~getDailyBrief~~ | ~~`GET /api/insights/daily-brief`~~ | — | — | **Removed** (R92, A2) | allowlisted citing FS-02JBZ if flagged |
| ~~refreshDailyBrief~~ | ~~`POST /api/insights/daily-brief/refresh`~~ | — | — | **Removed** (R92, A2) | allowlisted citing FS-02JBZ if flagged |

**DayPlan**

| Field | Type | Notes |
|---|---|---|
| `planId` | uuid | |
| `date` | `YYYY-MM-DD` | the key's date, as sent |
| `status` | enum `planned` \| `not_planned` | |
| `source` | enum `ai` \| `rules`, optional | present iff `planned` |
| `generatedAt` | timestamp, optional | present iff `planned`; unchanged when a redraw was gated |
| `refreshesRemaining` | int 0–3 | 3 until the free draw is used (R19) |
| `groups` | **DayPlanGroup**[] 0–3 | never null; empty when `not_planned` |

**DayPlanGroup**: `label` string 1–40 · `entries` **DayPlanEntry**[] 1–6 (≤ 6 across all groups).

**DayPlanEntry**

| Field | Type | Notes |
|---|---|---|
| `id` | uuid | entry id |
| `kind` | enum `existing_item` \| `new_step` | |
| `itemId` | uuid, optional | present iff `existing_item` |
| `parentItemId` | uuid, optional | required for `new_step`; for `existing_item`, the item's parent if it has one |
| `text` | string | the item's description when drawn, or the proposed step text |
| `reason` | string | one line |
| `minutes` | int 5–240, optional | absent on rules entries |
| `revision` | int 1–3, optional | set by the server only |
| `ticked` | boolean | day tick for this entry's date |
| `revisionsEnabled` | boolean, optional | present only on `existing_item` entries of learning plans |

**New error codes** (`common/errcode`, R58): `LLM_BUDGET_EXHAUSTED` (429),
`DAY_PLAN_ENTRY_NOT_TICKABLE` (400), `DAILY_SCOPE_RETIRED` (400).

**Plane 2 (gRPC).** `insights.InsightsService` gains `GetDayPlan`, `PlanMyDay`,
`TickDayPlanEntry`, `UntickDayPlanEntry`, `SetItemRevisions`, and later loses `GetDailyBrief` and
`RefreshDailyBrief` (R92). `plan.PlanService` loses `ToggleDailyReset`; `plan.ChecklistService`
loses `DailyReset`. `plan.Plan.daily_reset` and `ChecklistItem.scope` stay.

## Out of Scope

- **The capacity knob** ("how much do you plan to do in a day"): parked; 6/3 are constants (D2).
- **Automatic generation** (first visit, nightly): parked behind a future opt-in (ADR-0019).
- **A cross-plan AI day** on home or anywhere: parked (D15).
- **RAG-tuned revision ladder**: parked; the ladder is fixed.
- **Dropping `plans.daily_reset`, the `dailyReset` field and the `daily` enum value**: later
  cleanup (D12).
- **Per-user budgets**: ADR-0019 revisit trigger.
- **Counting embedding calls against the budget** (S8).
- **The gateway's own `internal/ai` (Notes)**: it calls OpenAI from the gateway process, outside
  insights' ledger; not under this budget.
- **Recording dismissals and the dismissal filter on day plan entries**: FS-DF4ZB I-DF4ZB-10 and
  I-DF4ZB-11, retargeted per O1.
- **Purging ticks and revisions on item, plan or user deletion**: same gap as DF4ZB O3.
- **A UI for the ledger or the budget**: usage is stored; reading it is a query.
- **Ticking from home**: home rows are read-only (D10).

## Spec defaults

Where the charter and the scoping session were silent, this FS chose. Each is cheap to change
before the slice that implements it.

| # | Question | Default | Reason |
| --- | --- | --- | --- |
| S1 | Is the day plan the brief's row, or beside it? What happens to the brief's endpoints? | A separate record beside the brief that reuses its hash, validation, counter, ledger and structured call; the brief's HTTP ops are retired once the client reads the day plan (A2) | D1 says "beside"; separate storage keeps DF4ZB's slices untouched during the switch; ADR-0019 forbids the brief's read-path generation and nothing reads the brief after the switch |
| S2 | Can entries point at top-level items, not just steps? | Yes: `existing_item` covers any open long-term task, top-level or child | Overdue items, due-soonest items and revisions are often top-level; DF4ZB's `existing_step` only covered children |
| S3 | Does the gate carry yesterday's plan forward? | No; the date is in the hash, so the gate works within a day only | The prompt includes today's date for due ranking, so yesterday's answer is not today's |
| S4 | What happens to ticked entries on a redraw? | They are pinned; the model fills the rest | A redraw must never hide work already done, and it keeps tick validation local to the stored plan |
| S5 | How does a new step become tickable? | Tick by entry id; an unbound `new_step` tick carries the created item's id, verified and bound on the server | No extra "link" operation; the client already has the id after Add |
| S6 | Are +1/+3/+7 offsets from the first tick, or chained? | Chained: each tick schedules only the next rung, 1, 3 and 7 days after the tick that scheduled it | D5 speaks of "the revision that tick scheduled" (one); chaining adapts when a revision slips |
| S7 | Do done items revise? What does the switch do? | Done items still revise (not archived/deleted). Off cancels the open rung; on resumes at the next tick, up to 3 | Revising a finished topic is the point; the switch is reversible without retroactive scheduling |
| S8 | Budget window, token sum, status, embeddings, old generators | UTC day; input + output; 429 via `ResourceExhausted`; embeddings not counted; old generators recorded and guarded | ADR-0019 left these to the FS; 429 says "not now" (503 would invite a retry); embeddings are event-driven, not generation; a breaker that can't see old generators isn't one |
| S9 | Fallback only on budget, or also on provider failure? | Both, and on schema-invalid output | Scoping decision 9.5 and its edge case "budget exhausted or provider down: rule-based fallback" |
| S10 | Rules plan grouping, minutes and reasons | "Revisions" group first, then by top-level ancestor; no minutes; reasons name the rule | Deterministic and readable without a model; a made-up estimate would mislead |
| S11 | Where is home's block, and which entries? | Above the Today block; first two resolvable entries in plan order, ticked ones shown ticked | The day plan is the more deliberate list; plan order is the AI's or the rule's ranking |
| S12 | How long does "Also mark done?" stay? | 8 seconds, or until the next action; only when the item isn't done | Long enough to read and click, short enough not to clutter |
| S13 | What does AI notes read once daily items are gone? | Long-term task items | Otherwise it silently loses all context |
| S14 | Group label limits, bad minutes, empty plans | Label 1–40 chars; an out-of-range `minutes` is dropped, not the entry; an empty plan is stored as planned | Keeps a good entry over a bad estimate; "planned, nothing to do" differs from "not planned" |
| S15 | Is `scope=daily` refused on list and reorder too? | No; accepted, finds nothing | Refusing reads would break old clients for no gain |
| S16 | Where do migrated daily rows sort? | After existing long-term siblings, keeping relative order | Nothing jumps ahead of the user's long-term order |
| S17 | Can a past day's entry be ticked? | Only within the UTC ±1 window | Same rule as every other date in this surface |
| S18 | Does dismissing an entry persist? | Session-only hide | Parity with Today's moves until DF4ZB's feedback records it |

## Open items for the owner

Not decided on the owner's behalf. None blocks the slices.

- **O1. I-DF4ZB-10 and I-DF4ZB-11 predate this FS.** They wire feedback "from Today's moves" and
  filter "brief output", and extend the brief's gate. After this FS, Today's moves is the Day plan
  section and the brief's HTTP surface is gone (A2). Entry ids are usable as suggestion ids for
  exactly this reason. Retarget both issues at day plan entries before building them (this FS
  does not edit DF4ZB files).
- **O2. "plan of the day" in `client/CONTEXT.md`** says there is no plan entity scoped to a day.
  The day plan is not that (it is a section of one plan, stored per user and date), but the row
  now reads as contradicting it, and its note that recurrence lives on `scope: daily` goes stale
  when R50 ships. The new glossary row distinguishes them; amending the old row is a
  `domain-model` call.
- **O3. Prod budget.** 2,000,000 tokens a day is the charter's production default. At
  `gpt-5-mini` prices that is a small, bounded daily spend, but the owner should confirm it
  against expected users before the first prod deploy.
