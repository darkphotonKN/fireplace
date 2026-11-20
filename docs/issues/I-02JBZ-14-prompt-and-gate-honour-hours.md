---
id: I-02JBZ-14
status: done
implements: FS-02JBZ
blocked_by: [I-02JBZ-13]
labels: [feature]
title: "FS-02JBZ slice 14: Plan my day sizes the day to the user's hours, and a changed setting redraws"
---
Implements FS-02JBZ §Change: Hours a day (§Requirements R96–R100)

## What to Build

When the caller has a `dailyMinutes` setting for the plan, Plan my day asks the model to fit the
day inside it; when they don't, the draw is exactly as today. All in insights-service
`internal/dayplan`; no contract change.

- **Draw input** (`service.go`, `startDraw`/the draft): read the caller's setting at the start of
  a draw and carry `dailyMinutes` into the prompt input and the hash.
- **Prompt** (`prompt.go`, `buildPrompt`): unset → the user message is byte-identical to today
  (R96). Set → the plan data / request states the user's daily time on this plan and asks that
  the entries' minutes sum within it, fewer entries being fine (R97). `systemRules` is **not**
  edited (it stays byte-identical across calls). With ticked entries, the ALREADY TICKED TODAY
  lines carry each entry's `minutes` when present, and the request asks the added entries to fit
  the time left; with none left, it says adding nothing is fine (R98, spec default, owner item
  O4).
- **Hash** (`hash.go`, `stateHash`/`hashInput`): `dailyMinutes` joins the hash input with
  `omitempty`, so unset hashes exactly as before (R99).
- **No trimming** (R100): validation (`validate.go`) and the rules draw (`rules.go`) are
  unchanged; bounds in `model.go` are unchanged.

## Acceptance Criteria

- [ ] With no setting, `buildPrompt`'s system and user messages are byte-identical to the
      pre-change output (golden test), and `stateHash` is identical too.
- [ ] With `dailyMinutes` 150, the user message states the daily time ("2½ hours" or
      "150 minutes", stated once) and asks that entry minutes sum within it; `systemRules` is
      byte-identical with and without a setting.
- [ ] With ticked entries carrying minutes, the ticked block lists them and the request states
      the minutes left; with the time used up, the request says adding nothing is fine.
- [ ] Changing `dailyMinutes` (unset → 120, 120 → 180, 120 → unset) changes the hash, so Plan
      again draws and counts (free if the free draw is unused); no change keeps the gate.
- [ ] Model output whose minutes exceed the budget is stored as drawn (nothing trimmed); the
      6 / 3 / 5–240 bounds and the rules draw are unaffected by the setting.
- [ ] Tests pass (`make test` in insights-service).

## Blocked By

I-02JBZ-13 (the settings store and the `Settings` the draw reads)

## Spec Reference

FS-02JBZ §Change: Hours a day: R96–R100; Edge States "Setting changed after today's plan was
drawn", "Setting changed while a Plan my day is in flight", "Ticked entries already use the whole
budget", "Model ignores the budget", "A chip, then the plan falls back to rules".

## TDD Approach

- RED: `prompt_test.go`: a golden comparison that an unset setting leaves the user message
  unchanged, then a case with 120 minutes asserting the budget line and an unchanged `System`.
- GREEN: the optional budget in `promptInput` and its line in `buildPrompt`.
- RED: a `hash` test that unset equals the pre-change hash and 120 differs.

## Files expected to touch

- `services/insights-service/internal/dayplan/prompt.go`, `prompt_test.go`
- `services/insights-service/internal/dayplan/hash.go` (and its test, in `service_test.go` or a new
  `hash_test.go`)
- `services/insights-service/internal/dayplan/service.go`, `service_test.go`, `fakes_test.go`
