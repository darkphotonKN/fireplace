---
id: I-DF4ZB-19
status: open
implements: FS-DF4ZB
blocked_by: [I-DF4ZB-16]
labels: [feature]
title: "FS-DF4ZB slice 19: suggestItemTree, a proposed parent item with about three steps (server and gateway; first to cut)"
---
Implements FS-DF4ZB §Requirements (R7–R9, R13, R24, R25, R49, R51, R56), §API surface

## What to Build

`POST /api/insights/item-tree-suggestion` with `{planId}` returns one proposed parent item: a
title, a one-line reason and 1–5 (aim about 3) steps. **Last slice and first to cut** (L19).

- Works on a plan with no parent items (an empty plan is the obvious case for it).
- One strict structured-output call through the shared prompt assembly (R29 seam for snippets),
  its own generator config, usage recorded as `item_tree`.
- **Validation (R13).** Non-empty title not duplicating (case/whitespace-insensitive) an existing
  top-level task; steps non-empty and distinct. An invalid title drops the whole proposal (there
  is nothing to anchor its steps to); invalid steps are dropped individually. Never repaired or
  regenerated. Decide and test what the response is when the proposal is dropped (the schema
  has `steps` 1–5 and a required `title`, and §API surface does not say how "no proposal" is
  represented; raise it with the owner rather than inventing an error code, since R53 allows no
  new codes beyond its three).
- **Dismissal filter (R49).** Reuse I-DF4ZB-11's filter on the title.
- **No cache (R25).** Stored with an id so `recordSuggestionFeedback` works on it.
- **RPC** `SuggestItemTree`; **gateway** typed Huma handler `suggestItemTree`, response
  **ItemTreeSuggestion**. Regenerate `openapi.yaml` and the client; gates green.

Built against the R29 seam/stub; not blocked by H2.

**Owner decision still open: O2.** Item-tree suggestions are **uncapped in v1** by default; usage
rows make spend visible. Do not add a cap unless the owner decides O2 first.

## Acceptance Criteria

- [ ] Item-tree suggestion returns a title, reason and 1–5 steps, including for an empty plan, in
      one model call.
- [ ] A title duplicating an existing top-level task, or near a dismissed title on this plan, is
      not returned; duplicate or empty steps are dropped.
- [ ] The suggestion has an id accepted by the feedback endpoint.
- [ ] Usage is recorded with kind `item_tree`.
- [ ] No path, contract type, proto message or server type name contains "milestone".
- [ ] `openapi.yaml` and the generated client are regenerated; gates green.
- [ ] Tests pass.

## Blocked By

I-DF4ZB-16 — follows the FS order (item-tree after step suggestions) and serializes the shared
contract hotspot. It reuses the filter and generator pattern 16 extends.

## Expected write set

- `common/api/proto/insights/insights.proto` + regenerated stubs **(hotspot)**
- `services/insights-service/internal/guided/` (item-tree flow, schema, generator config)
- `services/insights-service/internal/insights/handler.go` (or the guided handler)
- `services/insights-service/config/services.go` **(hotspot)**, `.env.example`
- `services/api-gateway/internal/insights/typed_insights.go` (+ test),
  `services/api-gateway/internal/gateway/insights/client.go`
- `services/api-gateway/openapi.yaml` + `client/src/api/generated/` **(hotspot, regenerated)**

## Spec Reference

FS-DF4ZB §Requirements R7–R9, R13, R24, R25, R49, R51, R56; §Decisions L19; §Spec defaults S6;
§Open items O2; §API surface (suggestItemTree, ItemTreeSuggestion).

## TDD Approach

- RED: a validation table test: duplicate title (case/whitespace variants), empty title, duplicate
  steps, empty steps.
- GREEN: validation.
- Then: empty-plan generation; filter on title; the handler and contract.
