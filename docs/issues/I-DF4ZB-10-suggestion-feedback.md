---
id: I-DF4ZB-10
status: open
implements: FS-DF4ZB
blocked_by: [I-DF4ZB-1, I-DF4ZB-5, I-DF4ZB-8, I-DF4ZB-9]
labels: [feature]
title: "FS-DF4ZB slice 10: recordSuggestionFeedback, wired from Today's moves, and the gate's dismissal bypass"
---
Implements FS-DF4ZB §Requirements (R46–R48, R60–R62 recording, R18 dismissal bypass), §API surface

## What to Build

Record what the user did with a suggestion, end to end.

**insights-service.**

- **Feedback (R46).** `RecordSuggestionFeedback(id, outcome, user_id)` sets `accepted` or
  `dismissed` on a suggestion the caller owns (any kind: move, step suggestion, item-tree
  suggestion). Idempotent; a later outcome replaces an earlier one (S8). Unknown or foreign id →
  `NotFound`.
- **Dismissal embedding (R48).** A dismissed suggestion's text is embedded once, at dismissal,
  through I-DF4ZB-1's embedder, and kept with the suggestion, **not** in `rag_chunks`. Re-dismissing
  does not re-embed. Flipping to `accepted` keeps or clears it consistently; pick one and test it.
- **No plan writes (R47).** insights-service never writes plan data; accepting is the client's
  job.
- **Gate bypass (R18, second clause tail; S3).** Extend I-DF4ZB-5's gate: the most recent brief is
  carried forward or served on refresh only if **none of its moves has been dismissed**.
  Otherwise the gate is bypassed (and a refresh counts).

**Gateway.** Typed Huma handler `recordSuggestionFeedback`: `POST
/api/insights/suggestions/{id}/feedback`, body `{outcome}` enum `accepted | dismissed`, `204`.
Identity from the JWT. Regenerate `openapi.yaml` and the client; gates green.

**Client (R60–R62).** In Today's moves: after a successful tick, record `accepted`; after a
successful add, record `accepted`; on dismiss, record `dismissed`. A failed create records
nothing. A failed feedback call is silent (the user action already happened) and never reverts
the tick or the add.

## Acceptance Criteria

- [ ] Feedback on the caller's suggestion returns 204, idempotently; a later outcome replaces an
      earlier one.
- [ ] Feedback on another user's or an unknown suggestion returns `404 · NOT_FOUND`.
- [ ] A dismissal embeds the text once and stores it with the suggestion, not in `rag_chunks`.
- [ ] After a dismissal on the most recent brief, an unchanged-plan read on a new day and an
      unchanged-plan refresh both bypass the gate (the refresh counts).
- [ ] Ticking and adding from Today's moves record `accepted`; dismissing records `dismissed`;
      a failed create records nothing.
- [ ] insights-service makes no plan-service write.
- [ ] `openapi.yaml` and the generated client are regenerated; gates green.
- [ ] Tests pass.

## Blocked By

- I-DF4ZB-1 — the embedder (R48) and pgvector for the stored vector.
- I-DF4ZB-5 — the gate this extends.
- I-DF4ZB-8 — the Today's moves block (and its refresh) this wires into; serializes edits to
  `TodaysMoves.tsx`.
- I-DF4ZB-9 — serializes the shared contract hotspot (proto, `openapi.yaml`, generated client),
  following the FS's suggested order.

## Expected write set

- `common/api/proto/insights/insights.proto` + regenerated stubs **(hotspot)**
- `services/insights-service/internal/suggestion/` (new, proposed: outcome store, dismissal
  embedding) and `internal/guided/` (gate extension)
- `services/insights-service/migrations/00000N_*` (outcome, dismissal vector on suggestions)
- `services/insights-service/config/services.go` **(hotspot)**
- `services/api-gateway/internal/insights/typed_insights.go` (+ test),
  `services/api-gateway/internal/gateway/insights/client.go`
- `services/api-gateway/openapi.yaml` + `client/src/api/generated/` **(hotspot, regenerated)**
- `client/src/api/insights.ts`, `client/src/components/TodaysMoves.tsx` (+ test)

## Spec Reference

FS-DF4ZB §Requirements R46–R48, R60–R62 (the "records accepted/dismissed" halves), R18 (dismissal
bypass); §Decisions L3, L11; §Spec defaults S3, S8, S9; §API surface (recordSuggestionFeedback);
§Edge States ("Refresh after dismissing a move", "Feedback for an unknown or foreign suggestion
id", "Accept create fails").

## TDD Approach

- RED: a service test: dismiss twice → one embedder call; feedback by another user → NotFound.
- GREEN: outcome upsert with ownership check.
- Then: gate test with a dismissed move on the last brief → generation; handler and contract;
  client tests for the three recording points and the silent feedback failure.
