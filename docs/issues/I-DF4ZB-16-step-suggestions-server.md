---
id: I-DF4ZB-16
status: open
implements: FS-DF4ZB
blocked_by: [I-DF4ZB-11]
labels: [feature]
title: "FS-DF4ZB slice 16: suggestSteps for a parent item (server and gateway)"
---
Implements FS-DF4ZB §Requirements (R7–R9, R13, R23, R25, R49, R51, R53, R56), §API surface

## What to Build

`POST /api/insights/step-suggestions` with `{planId, parentItemId}` returns 3–5 step suggestions
for that parent.

- **Parent check (R23).** The parent must be an open, top-level, long-term task of the plan. An
  item in the plan that fails that (top-level note, closed task, child item, daily item) is
  refused with `PARENT_ITEM_INVALID` as `ErrorInfo` on `FailedPrecondition` (→ 400 via
  I-DF4ZB-3). An item not in the plan is `NotFound`. Add `PARENT_ITEM_INVALID` to
  `common/errcode` (R53).
- **Generation (R7–R9).** One strict structured-output call through the same prompt assembly as
  the brief (system rules → snippets via the R29 seam → plan tree → request naming the target
  parent id), its own generator config (model, token ceiling), usage recorded as
  `step_suggestions`.
- **Validation (R13).** Drop suggestions that duplicate an existing child of the parent or each
  other, or are empty. Never repaired or regenerated; fewer than 3, down to zero, is valid.
- **Dismissal filter (R49).** Reuse I-DF4ZB-11's filter on the suggestion text.
- **No cache (R25, S6).** Each request is one generation. Each suggestion is stored with an id so
  `recordSuggestionFeedback` (I-DF4ZB-10) works on it unchanged.
- **RPC** `SuggestSteps`; **gateway** typed Huma handler `suggestSteps`, response
  **StepSuggestions** (`parentItemId`, `suggestions` never null). Regenerate `openapi.yaml` and the
  client; gates green.

Built against the R29 seam/stub; not blocked by H2.

**Owner decision still open: O2.** Step suggestions are **uncapped in v1** by default; usage rows
make spend visible. A per-plan daily cap like R17 would need its own code. Do not add a cap in
this slice unless the owner decides O2 first.

## Acceptance Criteria

- [ ] Step suggestions return up to 5 items (aim 3–5) for a valid parent, in one model call.
- [ ] A top-level note, closed task, child item or daily item as parent returns
      `400 · PARENT_ITEM_INVALID`; an item outside the plan returns `404 · NOT_FOUND`.
- [ ] Suggestions duplicating an existing child or each other are dropped.
- [ ] A suggestion near a dismissed one on this plan is filtered out.
- [ ] Each suggestion has an id accepted by the feedback endpoint.
- [ ] Usage is recorded with kind `step_suggestions`.
- [ ] No path, contract type, proto message or server type name contains "milestone".
- [ ] `openapi.yaml` and the generated client are regenerated; gates green.
- [ ] Tests pass.

## Blocked By

I-DF4ZB-11 — the dismissal filter (and through it the feedback store, I-DF4ZB-10, and the brief
pipeline, I-DF4ZB-4) this reuses.

## Expected write set

- `common/api/proto/insights/insights.proto` + regenerated stubs **(hotspot)**
- `common/errcode/errcode.go`
- `services/insights-service/internal/guided/` (step flow, schema, generator config)
- `services/insights-service/internal/insights/handler.go` (or the guided handler)
- `services/insights-service/config/services.go` **(hotspot)**, `.env.example`
- `services/api-gateway/internal/insights/typed_insights.go` (+ test),
  `services/api-gateway/internal/gateway/insights/client.go`
- `services/api-gateway/openapi.yaml` + `client/src/api/generated/` **(hotspot, regenerated)**

## Spec Reference

FS-DF4ZB §Requirements R7–R9, R13, R23, R25, R49, R51, R53 (`PARENT_ITEM_INVALID`), R56; §Decisions
L2, L3; §Spec defaults S6; §Open items O2; §API surface (suggestSteps, StepSuggestions,
StepSuggestion); §Edge States ("Step suggestions on a parent that already has many open steps").

## TDD Approach

- RED: a table test of the parent check over a fixture tree (valid parent, top-level note, closed
  task, child, daily item, foreign id) asserting status and code.
- GREEN: the check.
- Then: generation with a fake generator returning duplicates; filter applied; ids stored; the
  handler and contract.
