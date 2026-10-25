---
id: I-DF4ZB-4
status: done
implements: FS-DF4ZB
blocked_by: [I-DF4ZB-3]
labels: [feature]
title: "FS-DF4ZB slice 4: getDailyBrief, one structured call over the plan tree, validated, stored, usage recorded"
---
Implements FS-DF4ZB §Requirements (R1–R15, R18 first clause, R21, R22, R29, R51, R53, R54, R56), §API surface

## What to Build

The first working path end to end: `GET /api/insights/daily-brief?planId=&date=` returns a stored
brief, generating it on a miss with **one** structured-output call. Retrieval is coded against
a seam with a stub that returns nothing, so the weak-retrieval path is the only path until H2
lands. The rest of the rule gate is I-DF4ZB-5, refresh and the cap I-DF4ZB-6.

**insights-service.**

- **Plan tree (R1).** From plan-service `ListItems`: non-archived, long-term **task** items with
  `id`, `parent_id`, `description`, `done`, `due_date`. Notes, daily and archived items are
  excluded. The single-suggestion and video generators keep their current input.
- **Retrieval seam (R29).** An interface owned by the brief flow (consumer-owned), plus a stub
  that returns zero snippets. H2 (I-DF4ZB-14) supplies the real implementation; nothing here may
  depend on it.
- **Prompt assembly (R2–R6).** Strictly: static system rules (byte-identical across calls of one
  generator) → retrieved snippets (omitted entirely when none) → compact plan tree with ids →
  the request. Snippets are labelled by origin (R4). With no snippets the rules forbid referring
  to the user's past work (R5). Prompt prose may say "milestone"; no server type, proto message
  or path does (R6, L13).
- **One structured call (R7–R8).** A strict JSON-schema structured-output call: moves 1–3, each
  `{parent_item_id, kind, step_item_id?, text, reason}`. No free-text parsing, no repair call.
  The model and output-token ceiling are configurable **per generator**, defaulting to
  `gpt-5-mini`, minimal reasoning. Transport retries stay as today and are not extra
  generations.
- **Usage (R9).** Every generation, including a failed one and one lost to the R22 race,
  records kind (`daily_brief` here; the enum also holds `step_suggestions` and `item_tree`), plan,
  user, model, input, cached input and output tokens, and time.
- **Validation (R10–R12).** Each move checked against the tree: open top-level parent; an
  `existing_step` names an open child of that parent; a `new_step` has non-empty text not
  duplicating (case/whitespace-insensitive) an existing child; non-empty reason; no duplicate
  `step_item_id` or text within one output. Invalid moves are dropped one by one, never repaired
  or regenerated; zero is valid.
- **Storage (R14, R15, R22).** One brief per (plan, user, date). Each stored move gets a
  suggestion id (the `Move.id` feedback will reference, I-DF4ZB-10). Concurrent first reads for
  one key store one brief and every caller gets it.
- **Rule gate, first clause only (R18).** A plan with no open, top-level, long-term task stores
  and returns an empty brief with no generation.
- **Date range (R21).** `date` outside server UTC date ±1 day is refused with
  `BRIEF_DATE_OUT_OF_RANGE` as `ErrorInfo` detail (I-DF4ZB-3's helper). Add the code to
  `common/errcode` (R53).
- **Failure (R54).** Model unreachable, timed out after retries, or schema-invalid output →
  `Unavailable`; nothing stored.
- **RPC.** `insights.InsightsService.GetDailyBrief` with `user_id` from verified identity. Name
  fields "parent item", never "milestone".

**Gateway and contract (R51, R56).** A typed Huma handler for `getDailyBrief` with `planId` and
`date` (camelCase), identity from the JWT forwarded as `user_id`. Response **DailyBrief** exactly
as §API surface: `planId`, `date`, `generatedAt`, `refreshesRemaining` (3 until I-DF4ZB-6 makes
refresh exist), `moves` never null. Shape failures are 422 at the boundary. Regenerate
`openapi.yaml` and the generated client; never hand-edit them. Run `make -C services/api-gateway
gates`.

The old `checklist-suggestion-daily` endpoint and `GenerateDailySuggestions` stay until
I-DF4ZB-9.

## Acceptance Criteria

- [ ] A brief costs at most one structured-output call; the new path makes no sequential calls.
- [ ] The plan tree sent to the model carries ids, parent ids, done and due for long-term,
      non-archived tasks; notes, daily and archived items are absent.
- [ ] Prompt order is system rules → snippets → plan tree → request; system rules are
      byte-identical across two calls; with the stub retriever the snippet block is absent and
      the rules forbid claims about past work.
- [ ] A second read for the same (plan, user, date) makes no model call and returns the same moves.
- [ ] Two concurrent first reads store one brief and both callers receive it.
- [ ] A plan with no open top-level long-term task returns an empty brief with no model call.
- [ ] Moves with an invalid parent, a non-child or done step, a duplicate, or an empty reason are
      dropped; nothing is repaired or regenerated.
- [ ] A date more than one day from the server's UTC date returns `400 · BRIEF_DATE_OUT_OF_RANGE`.
- [ ] A failed generation returns `503 · SERVICE_UNAVAILABLE` and stores no brief.
- [ ] Every generation stores model, input, cached input and output tokens.
- [ ] The model is set per generator; the default is `gpt-5-mini` with minimal reasoning.
- [ ] No new path, contract type, proto message or server type name contains "milestone".
- [ ] `openapi.yaml` and `client/src/api/generated` are regenerated; `make -C services/api-gateway gates` is green.
- [ ] Tests pass.

## Blocked By

I-DF4ZB-3 — `BRIEF_DATE_OUT_OF_RANGE` must cross gRPC as `ErrorInfo`.

Not blocked by H2 (I-DF4ZB-14): retrieval is the R29 seam with a stub.

## Expected write set

- `common/api/proto/insights/insights.proto` + regenerated `insights.pb.go`,
  `insights_grpc.pb.go` **(hotspot)**
- `common/errcode/errcode.go`
- `services/insights-service/internal/guided/` (new, proposed name: plan tree, prompt, schema,
  generator, validation, brief service and repository, retrieval seam + stub)
- `services/insights-service/internal/ai/generator.go` (structured output, per-generator model,
  usage capture)
- `services/insights-service/internal/insights/plan_client.go` (full item fields)
- `services/insights-service/internal/insights/handler.go` (or a new handler for the RPC)
- `services/insights-service/migrations/00000N_*` (briefs, suggestions, generations/usage)
- `services/insights-service/config/services.go` **(hotspot)**
- `services/insights-service/.env.example` (per-generator model and token ceiling)
- `services/api-gateway/internal/insights/typed_insights.go` (+ test),
  `services/api-gateway/internal/gateway/insights/client.go`
- `services/api-gateway/openapi.yaml` + `client/src/api/generated/` **(hotspot, regenerated)**

## Spec Reference

FS-DF4ZB §Requirements R1–R15, R18 (first clause), R21, R22, R29, R51, R53
(`BRIEF_DATE_OUT_OF_RANGE`), R54, R56; §Decisions L1, L4, L7, L12, L13; §API surface
(getDailyBrief, DailyBrief, Move); §Edge States ("Plan has no long-term tasks", "Every milestone
closed", "Milestone with no steps", "Model returns only invalid moves", "Two first reads at once",
"Client clock wrong", "Model or OpenAI down", "`rag_chunks` empty"); §Cost rationale.

## TDD Approach

- RED: a brief-service test with a fake plan client, a fake generator returning one valid and two
  invalid moves, and the stub retriever: assert one generator call, one stored move, usage
  recorded, and a second read with no generator call.
- GREEN: plan tree → prompt → generate → validate → store.
- Then: prompt-order golden test (system rules byte-identical, no snippet block); date-range
  refusal; concurrent first reads; the Huma handler and regenerated contract.
