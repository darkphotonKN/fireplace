---
id: I-PGG2N-9
status: done
implements: FS-PGG2N
blocked_by: [I-PGG2N-1]
labels: [feature]
title: "FS-PGG2N slice 9: insights prompts carry a note's summary, not its whole body"
---
Implements FS-PGG2N §Requirements (R45)

## What to Build

Long notes must not grow the tokens a suggestion spends. Wherever an insights prompt includes a
note's text, it uses the note's summary, or else the first 280 characters of its body.

- `services/insights-service/internal/insights/plan_client.go`: `GetPlanContext` maps each item's
  `type` and `summary` (from slice 1's proto field) into `ChecklistItem`, beside `description`.
- `services/insights-service/internal/insights/service.go`: `formatChecklist` uses a helper for a
  note's prompt text, such as `promptText(item)`. That is `summary` if set, else the first 280
  runes of `description`. Tasks still use their full `description`. The helper owns the line's
  wording for notes; "A … task:" is wrong for a note.
- This reaches `getChecklistSuggestion` and `getSuggestedVideos`.
- The day plan and guided suggestions already drop notes in `guided.NewPlanTree`, from their
  tree, prompt and hash. Leave that as it is, and add a test that pins it, so a long note never
  changes their prompt or their regeneration.

Files expected:
- `services/insights-service/internal/insights/{plan_client,service}.go` and tests
- a test beside `services/insights-service/internal/guided/tree.go`

## Acceptance Criteria

- [ ] A note with a summary appears in the suggestion and video prompts as its summary.
- [ ] A note without one appears as its first 280 characters.
- [ ] Tasks are unchanged.
- [ ] A 5000-character note adds at most 280 characters to those prompts.
- [ ] A test pins that `NewPlanTree` excludes notes, and that the day plan hash ignores a note's
      body and summary.
- [ ] Tests pass.

## Blocked By

I-PGG2N-1 (the `summary` field on the plan proto).

## Spec Reference

FS-PGG2N:
- §Requirements R45 (charter default C20).
- §Acceptance Criteria: the prompt item.
- §Edge States: day plan and guided suggestions with long notes.

## TDD Approach

- RED: a `formatChecklist` test. A note with summary "gist" renders "gist". A 400-rune note with
  no summary renders 280 runes. A task renders in full.
- GREEN: the mapping and the helper.
- Then: the `NewPlanTree` pin test.
