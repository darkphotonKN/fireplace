---
id: I-02JBZ-5
status: done
implements: FS-02JBZ
blocked_by: [I-02JBZ-3, I-02JBZ-4]
labels: [feature]
title: "FS-02JBZ slice 5: rule-based day plan when the budget is spent or generation fails"
---
Implements FS-02JBZ §Requirements (R16, R36–R40), §API surface (planMyDay `source = rules`)

## What to Build

Plan my day degrades instead of failing (ADR-0019 clause 4, D8, D9).

- **When (R36, S9).** Plan my day's generation fails with `LLM_BUDGET_EXHAUSTED` (I-02JBZ-3), or
  the model is unreachable or times out after transport retries, or the output is invalid
  against the schema → draw by rules instead. The response is `200` with `source = rules`; the
  plan is stored. (`BRIEF_REFRESH_LIMIT_REACHED` is still a 429: the cap is checked before any
  draw.)
- **Ranking (R37).** Due revisions (oldest due first) → overdue items (most overdue first) →
  items with a due date (soonest first) → the next open step of the most recently updated parent
  item (first open, non-archived task child in stored order; a parent with no open task child
  contributes itself if open). Ties keep plan order. Due revisions come from I-02JBZ-7; until
  then the input is an empty list, so leave the parameter in place.
- **Shape (R38, S10).** Due revisions in a first group labelled "Revisions"; every other entry in
  a group per top-level ancestor, labelled with its description cut to 40 characters. Take in
  rank order until 6 entries, skipping any entry that would open a fourth group. All
  `existing_item`, `minutes` absent, reason names the rule ("Revision 2 due", "Overdue since
  Oct 20", "Due Oct 28", "Next step"). Same validation bounds as R9.
- **Free (R16, R40).** No count, free draw not consumed, no model call, no ledger row.
- **Gate (R17).** A stored `rules` plan never satisfies the gate: the next Plan my day tries the
  model; if that succeeds and is the key's first successful AI draw, it is free.
- **Ticks (R39).** When I-02JBZ-6 lands, ticked entries are pinned into a rules redraw exactly as
  into an AI one; build the draw so pinned entries can be passed in.

No new operation and no contract shape change (`source` already exists from I-02JBZ-4).

## Acceptance Criteria

- [ ] Over budget, Plan my day returns `200` with a stored `source = rules` plan, makes no model
      call, writes no ledger row and does not lower `refreshesRemaining`.
- [ ] Model unreachable / timeout / schema-invalid output → same.
- [ ] Ranking table: overdue before due-soonest before next-step; most overdue first; ties in plan
      order; revision input (when supplied) ranks first in a "Revisions" group.
- [ ] Groups are top-level ancestors with ≤40-char labels; ≤6 entries; never a fourth group; no
      `minutes`; rule-named reasons.
- [ ] After a rules plan, Plan my day calls the model; a success there is free if it is the first
      successful AI draw of the key.
- [ ] Tests pass.

## Blocked By

- I-02JBZ-3 — `LLM_BUDGET_EXHAUSTED` from the generator seam.
- I-02JBZ-4 — the day plan, its storage and Plan my day.

## Expected write set

- `services/insights-service/internal/dayplan/rules.go` (new) + `rules_test.go`
- `services/insights-service/internal/dayplan/service.go` (catch and fall back) (+ test)

## Spec Reference

FS-02JBZ §Decisions D8, D9; §Requirements R16, R17 (rules plan never gates), R36–R40; §Spec
defaults S9, S10; §User Stories 25, 26; §Edge States ("Budget spent before Plan my day", "Budget
spent mid-day after an AI plan", "Provider down or timing out", "Plan again after a basic plan,
budget back"); ADR-0019 clause 4.

## TDD Approach

- RED: a table test for the ranker over a fixture tree (overdue, due-soon, undated parents with
  and without open children, updatedAt order) → expected entries, groups and labels.
- GREEN: the ranker.
- Then: service tests with a generator returning `LLM_BUDGET_EXHAUSTED`, `Unavailable`, and
  schema-invalid output → rules plan stored, no count, no ledger row; gate bypass after a rules
  plan.
