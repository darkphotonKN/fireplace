---
id: I-PGG2N-6
status: done
implements: FS-PGG2N
blocked_by: []
labels: [feature]
title: "FS-PGG2N slice 6: insights writes a note summary on request (SummarizeNote + summarizeNote)"
---
Implements FS-PGG2N §Requirements (R38–R44), §API surface

## What to Build

A new insights-service RPC `SummarizeNote` and the gateway operation that routes to it,
`summarizeNote` (`POST /api/insights/note-summary`). It reads one note's body from plan-service,
makes one metered model call and returns the text. It stores nothing, writes nothing to
plan-service and publishes no event (R43). The call happens only because the user pressed a
button (R44, ADR-0019).

**Error codes.** In `common/errcode`:
- `ITEM_NOT_A_NOTE` (400). Insights sends it as an ErrorInfo detail on `FailedPrecondition`.
- `NOTE_SUMMARY_FAILED` (503). Insights sends it as an ErrorInfo detail on `Unavailable`.

Document both the way `LLM_BUDGET_EXHAUSTED` is.

**Proto.** In `common/api/proto/insights/insights.proto`:
- `rpc SummarizeNote(SummarizeNoteRequest) returns (SummarizeNoteResponse)`, with a doc comment
  listing its refusals.
- `SummarizeNoteRequest { plan_id = 1; user_id = 2; item_id = 3 }`.
- `SummarizeNoteResponse { summary = 1 }`.

Regenerate with `make proto-gen`.

**insights-service.**
- Access (R39). Ask plan-service `GetPlanAccess`, and refuse relation `none` as NotFound
  (ADR-0015: callers enforce). Then call `GetItem(plan_id, item_id)`.
  - A missing item, another plan's item or an archived one is NotFound.
  - `type != "note"` is `ITEM_NOT_A_NOTE`.
  - The plan client injects identity metadata, as every client does (ADR-0001).
- Input (R40). Only the body: the first `NOTE_SUMMARY_INPUT_CAP = 8000` runes of
  `description`. Not the summary, not other items. Ask for one plain paragraph of at most 280
  characters, in the note's language.
- Metering (R41).
  - Add `ai.KindNoteSummary = "note_summary"`.
  - Check the global daily token budget first. Spent → `ResourceExhausted` with
    `LLM_BUDGET_EXHAUSTED`, and no call is made.
  - Record usage in the `generations` ledger through the existing caller and meter.
  - **Migration:** widen `generations_kind_check` to allow `note_summary`, in the next insights
    migration number. Down restores the previous list. See the watch note below.
- Output (R42).
  - Trim the output and collapse its whitespace.
  - Over 280 characters → cut at the last word boundary within 279 and append "…".
  - Empty output, a provider error or a timeout → `Unavailable` with `NOTE_SUMMARY_FAILED`.
- Register the handler, and add plan client methods (`GetPlanAccess` if it is missing, `GetItem`).

**Gateway.**
- In `services/api-gateway/internal/gateway/insights/client.go`, add `SummarizeNote`.
- In `internal/insights/`, add a typed operation `summarizeNote`:
  - body `SummarizeNoteRequest { planId uuid, itemId uuid }`, both required and strict;
  - response `{ summary: string }`;
  - it runs `authorizer(access)` for the plan first, as every insights operation does;
  - the description states that nothing is stored and what each code means.
- Errors leave through `apierr`. Check that `FailedPrecondition` and `Unavailable` with an
  ErrorInfo carry their code, not a generic one.
- Regenerate `openapi.yaml` and `client/src/api/generated/schema.d.ts` (`make openapi client`),
  and run `make gates`.
- Add `summarizeNote(planId, itemId)` to `client/src/api/insights.ts`, returning the text.
  There is no UI in this slice.

**Watch.** The plan's charter bounds listed only a plan-service migration. The `generations`
ledger's kind check makes an insights migration unavoidable, because ADR-0019 requires every
generator to be metered. Confirm this with the owner before merging, rather than recording usage
under another kind.

Files expected:
- `common/errcode/errcode.go`
- `common/api/proto/insights/insights.proto` and the generated `insights.pb.go` and
  `insights_grpc.pb.go`
- `services/insights-service/internal/ai/meter.go`
- `services/insights-service/internal/insights/{handler,service,plan_client}.go`, or a new
  `internal/notesummary/` package, and tests
- `services/insights-service/migrations/0000NN_allow_note_summary_kind.{up,down}.sql`
- `services/api-gateway/internal/gateway/insights/client.go`
- `services/api-gateway/internal/insights/typed_insights.go`, or a new
  `typed_note_summary.go`, and tests
- the gateway route wiring
- `services/api-gateway/openapi.yaml`, `client/src/api/generated/schema.d.ts`,
  `client/src/api/insights.ts`

## Acceptance Criteria

- [ ] An owner or member gets `200 { summary }` of 1–280 characters for a live note. Nothing is
      written to plan-service.
- [ ] Relation `none` is refused before any item read or model call.
- [ ] A missing, archived or other-plan item is `404 · NOT_FOUND`. A task is
      `400 · ITEM_NOT_A_NOTE`.
- [ ] Malformed or missing ids, or an unknown member, are `422 · VALIDATION_FAILED`.
- [ ] The budget spent gives `429 · LLM_BUDGET_EXHAUSTED`, and no model call is made.
- [ ] A model error, timeout or empty output gives `503 · NOTE_SUMMARY_FAILED`. Insights
      unreachable gives `503 · SERVICE_UNAVAILABLE`.
- [ ] The model receives at most 8000 characters of body and nothing else from the plan.
- [ ] Output over 280 is cut at a word boundary with "…". Whitespace is collapsed.
- [ ] Each successful or failed attempt records a `note_summary` row in the ledger. The kind
      migration runs up and down.
- [ ] `openapi.yaml` and `schema.d.ts` are regenerated, and gates pass (additive).
- [ ] No GET or read path calls `SummarizeNote`.
- [ ] Tests pass.

## Blocked By

None. It can run in parallel with slices 1–5. Coordinate regeneration of `openapi.yaml` and
`schema.d.ts` with I-PGG2N-1 if both are open at once.

## Spec Reference

FS-PGG2N:
- §Requirements R38–R44 (charter defaults C6, C7; gap defaults G13–G15).
- §API surface: the `summarizeNote` row, the new codes, the insights gRPC and the insights
  storage note.
- §Acceptance Criteria: the summary endpoint items.
- §Edge States: budget spent, output over 280, very long body.

## TDD Approach

- RED: an insights service test with a fake plan client and a fake model.
  - Relation none → NotFound, and the model is not called.
  - A task → `ITEM_NOT_A_NOTE`.
  - A 9000-rune body → the model sees 8000.
  - A 400-character output → ≤ 280 ending in "…".
  - Empty output → `NOTE_SUMMARY_FAILED`.
- GREEN: the service and the handler mapping.
- Then: a budget-exhausted test through the meter. A gateway transport test for codes and 422s.
  Regenerate and run the gates.
