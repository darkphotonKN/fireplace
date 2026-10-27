# ADR-0019 — Generation is explicit and bounded; reads never generate; rules own timing

Status: accepted
Date: 2026-10-25
Scope: root — governs when any LLM generator may run, the platform-wide spend bound across all
generators, and which decisions may be delegated to a model
Related: ADR-0009 (insights-service is the home of every model call, and is expensive per call
in a way no other domain is), ADR-0007 (user edits win: generated output adds or proposes,
never overwrites), ADR-0013 (home composes from per-plan reads, the roll-up this keeps
model-free)
Realized by: FS-02JBZ (day plan: on-demand generation, the global budget, the rule-based
fallback, the revision ladder). Builds on FS-DF4ZB (per-plan refresh cap, rule gate, and the
per-generation usage ledger this sums).

## Context

Fireplace is growing more than one generator. FS-DF4ZB adds the daily brief, step suggestions
and the item-tree suggestion. FS-02JBZ adds the day plan on top of the brief. Each runs on
`gpt-5-mini`: cheap per call, but a call is still money, latency and a dependency on a provider
we don't control. ADR-0009 already recorded that insights-service is the one domain that is
expensive per call.

FS-DF4ZB bounds cost **per feature**: a cache per (plan, user, date), a rule gate that skips the
model when the plan hasn't changed, a cap of 3 counted refreshes per plan per day, one strict
structured call per action, and a usage ledger with input, cached input and output tokens per
generation. That bounds one plan's daily brief. It does not bound the platform. Step and
item-tree suggestions are uncapped in v1 (FS-DF4ZB O2), every new generator brings its own
limits or none, and nothing adds them up.

Three further forces came out of the FS-02JBZ scoping session:

- **The owner wants to test generation for real in dev**, against the real provider, without
  burning budget. A fixture/replay mode was rejected. Real calls, but guarded.
- **Generation that fires on a read spends tokens on days nobody looks.** Today
  `GET /api/insights/checklist-suggestion-daily` calls the model on every read, and FS-DF4ZB R15
  generates a brief on a cache miss ("draw on first read"). Auto-generation on first visit, or
  in a nightly job, pays for every plan of every user whether or not anyone opens the page.
- **Some decisions are timing.** The day plan's revision ladder decides when a learning topic is
  due again. A model could guess that, but it would guess differently on identical input, cost
  tokens each time, and need history we don't have yet.

The ADR also has to hold when the provider is down or the budget is spent. The platform should
degrade, not break.

Recorded without adversarial review — locked collaboratively in the FS-02JBZ scoping session and
not run through challenge-me.

## Decision

**LLM generation is explicit and bounded. Reads never generate. Rules own timing.**

1. **Generation happens only on an explicit user action.** Pressing "Plan my day", asking a
   milestone card for steps, refreshing a brief: a user did something that asks for model
   output. No generator runs because a page loaded, a cache missed, a job ticked, or a
   first visit happened.
2. **Read paths never call the model.** GET endpoints and the home roll-up return cached output
   for the key, or an empty state that offers the explicit action. A cache miss on a read is an
   empty state, not a generation.
3. **A global daily token budget is a circuit breaker across all generators.** The budget is set
   by `LLM_DAILY_TOKEN_BUDGET`, and spend is summed from the per-generation usage ledger FS-DF4ZB
   introduces. It is one bound for the whole platform, on top of each feature's own limits, not
   instead of them. Dev runs with a lower value than production.
4. **Over budget, generators fail with domain code `LLM_BUDGET_EXHAUSTED`** (a `common/errcode`
   code, RFC 9457 per ADR-0004). A feature that can degrade falls back to a deterministic,
   rule-based result instead, **labelled as such** in the response and the UI, so nobody
   mistakes it for model output. A feature with no sensible deterministic result surfaces the
   code. The same fallback serves a provider outage.
5. **Timing is a rule, never model output.** When something is due (the +1/+3/+7 revision ladder
   in FS-02JBZ) is computed by a rule. The model decides **what** goes into today and **how** it
   groups. It never decides **when**. Due items reach the model as input it must work with, not
   as a question it answers.

## Consequences

**Good**

- **Cost is predictable.** The worst day is bounded by one number, whatever the generator count.
  On top of that it is defence in depth: *avoid the call* (cache, rule gate, no read-path
  generation), *cheap call* (small model, compact input, strict structured output), *bounded
  call* (per-plan caps, then the global budget), *fail soft* (labelled rule-based fallback),
  *measure* (the usage ledger). Each layer catches what the one before misses.
- **Dev can test for real.** A low `LLM_DAILY_TOKEN_BUDGET` in dev makes runaway loops and
  forgotten tabs cost a known, small amount. No fixture layer to keep in sync with the real
  provider.
- **Spend tracks intent.** Tokens are spent on days and plans someone actually asked about.
  Dormant plans and unvisited days cost nothing.
- **Behaviour is predictable where it matters.** Due dates and revisions follow a rule a user
  can learn and a test can assert. Model variance is confined to grouping and wording.
- **The platform keeps working without the provider.** Budget spent or provider down, the day
  plan still shows a basic, labelled plan. Reads never depended on the provider to begin with.
- **New generators inherit the bound.** Anything that writes to the usage ledger is under the
  global budget from its first call, including FS-DF4ZB's uncapped step and item-tree
  suggestions.

**Bad / accepted**

- **First visit shows an empty state, not content.** A user who never presses the button never
  sees a generated day. That is the point, but it is a less magical first experience than
  draw-on-first-read. Automatic generation is parked behind a future opt-in setting, which would
  need a new ADR or an amendment to clause 1.
- **FS-DF4ZB R15/R16 contradict clause 2.** R15 generates a brief on a read miss, and R16 treats a
  refresh with no stored brief as a read. FS-02JBZ replaces draw-on-first-read with on-demand
  generation (its decision 10), so the brief's read path must change before or with FS-02JBZ.
  Until then, and until `GET /api/insights/checklist-suggestion-daily` is removed (FS-DF4ZB's
  removal slice), the codebase carries read-path generation that this ADR forbids.
- **One user can spend the whole budget.** The global budget is a platform circuit breaker, not
  a fairness mechanism. Heavy use by one account trips fallback for everyone until the window
  resets. The per-plan caps limit how fast that can happen. Per-user budgets are a later
  decision if real usage shows the need.
- **The check is soft at the edge.** The budget is read from the ledger before a call, and a
  call's cost is only known after it. Concurrent calls near the limit can overshoot by roughly
  the in-flight calls' output ceilings. Accepted: it is a circuit breaker, not an invoice.
- **Every degradable feature carries two implementations.** The model path and the deterministic
  fallback must both be built, tested and kept producing the same response shape. The fallback
  label is part of the contract.
- **The ledger becomes load-bearing.** It was for measurement (FS-DF4ZB R9). Now a generation
  that isn't recorded is spend the breaker can't see, so every generator, including retries and
  calls lost to a race, must write to it.
- **Rules can't learn timing on their own.** The revision ladder is fixed until someone changes
  it. Personal memory may tune its parameters per user later (FS-02JBZ parking lot) without
  changing the rule's shape. Handing timing itself to the model supersedes clause 5.

**Left to the implementing FS (mechanism, not decision):** the budget window and its time zone,
the HTTP status for `LLM_BUDGET_EXHAUSTED`, whether retries count (FS-02JBZ says they do), the
exact fallback ranking, and whether embedding calls (playbook ingest, personal-memory ingest,
query embeddings) count toward the budget. Embedding ingest runs from events, not user actions;
clause 1 governs **generation** (model output shown to a user), and embeddings are not that.

**Rejected**

- **Auto-generation on first visit or nightly.** Spends tokens on days nobody looks, and makes
  cost scale with plans held rather than plans used.
- **AI-decided revision timing.** Unpredictable on identical input, costs tokens on every
  decision, and needs per-user history the platform doesn't have yet.
- **Per-feature ad-hoc limits only.** Each limit is local. Nothing bounds the sum, and a new
  generator ships with whatever limit its author remembered, or none.

**Revisit when:** a product need for unprompted generation appears (an opt-in "plan my day
automatically" setting amends clause 1), or one account's usage trips the global breaker in
practice (per-user budgets, likely an amendment to clause 3).
