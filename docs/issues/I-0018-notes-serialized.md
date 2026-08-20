---
id: I-0018
status: open
implements: FS-0004
blocked_by: [I-0015]
labels: [ready-for-agent]
---
Implements FS-0004 §Requirements, §API surface, §Edge States

## What to Build

Serialize the **notes** group — 6 operations — and cut the frontend over in the same slice.

`GET /plans/{id}/notes` (filterable) · `GET .../{noteId}` · `POST` · `PATCH .../{noteId}` ·
`DELETE .../{noteId}` · `POST .../generate-ai`

Notes is a **gateway-local domain** — it is backed by the gateway's own `notes` table, not a
downstream service. So the transport mirror is transcribed from `internal/notes/model.go`
rather than from a proto, and R5 applies with force: `models.Note` must not leak into
`components.schemas`.

Traps specific to this group:

- **`tags` is read with `QueryArray`** — `?tags=a&tags=b` is a two-element filter today. The
  typed param must be a slice, or this silently collapses to the last value. This is a
  behavior regression that no gate would catch.
- Filters are camelCase (`isRead`, `isDismissed`, `relatedTaskId`) while `insights` uses
  snake_case. Both are transcribed as they are (R6) — do not normalize.
- `isRead` / `isDismissed` are parsed from strings today; an absent filter is distinct from
  `false`. Preserve that distinction in the typed param.
- `ai_metadata` is JSONB. Declare a shape for it or declare it explicitly free-form — do not
  let it become an untyped `any` by accident.

## Acceptance Criteria

- [x] All 6 operations appear in `openapi.yaml` with correct methods, paths, and params
- [x] `tags` is a repeatable query param and `?tags=a&tags=b` still filters on both
- [x] Absent `isRead`/`isDismissed` remains distinct from `false`, proven by a test
- [x] All 6 declare `Security: bearerAuth`
- [x] `models.Note` does not appear in `components.schemas`, directly or nested
- [x] Round-trip test over a **populated** fixture asserts JSON equality
- [x] Empty list responses marshal to `[]` and never `null`
- [ ] Before/after probe recorded showing the envelope removed, payload otherwise unchanged
- [x] `make openapi-diff`, `make lint-contract`, `make openapi-breaking` green
- [x] FE notes call sites cut over to the generated client; typecheck green *(notes-domain only — see close-out)*
- [x] Full Go test suite green

## Close-out

**Evidence per criterion**

- 6 operations, methods, paths, params — `openapi.yaml`; document went 23 → 29 operations.
- `tags` repeatable — published as `explode: true`; proven by
  `TestListNotes_RepeatedTagsReachServiceAsSlice`. Huma does **not** do this by default: a
  plain `[]string` query param is comma-separated and collapsed `?tags=a&tags=b` to `[a]`.
  The `query:"tags,explode"` tag is what makes it repeatable, and the test is what caught it.
  The legacy edge cases (`?tags=&tags=b` suppresses the whole filter, because gin's
  `c.Query` read only the first value) are pinned by
  `TestListNotes_TagFilterEdgeCasesMatchLegacy` rather than left to a comment.
- absent vs false — `TestListNotes_AbsentBooleanFilterIsDistinctFromFalse`, 7 cases.
  `isRead`/`isDismissed` are typed as `string`, not `*bool`: huma panics on pointer query
  params, and the legacy parse treated any non-`true` value as false, so an enum would have
  turned a long-accepted request into a 422.
- `Security: bearerAuth` — all 6 verified against the parsed document.
- no domain leak — `components.schemas` holds only `NoteResponse`, `AIMetadataPayload`,
  `CreateNoteRequest`, `UpdateNoteRequest`, `GenerateAINotesRequest`.
- round trip — `TestNoteResponse_RoundTripMatchesDomainJSON` over a fully populated fixture
  including `aiMetadata`.
- empty list — `TestListNotes_EmptyResultIsArrayNotNull`; `toNoteResponses` is explicitly
  non-nil.
- gates — `make gates` green (spec matches, spectral clean, client matches, no breaking).

**Behaviour differences, deliberate**

1. `DELETE` answers **204 no body**, was 200 + `result: "success"` — per §API surface.
2. `POST /generate-ai` with **no body at all** is now 400, where the legacy handler fell back
   to `requestType: "all"`. Huma derives "body required" from the typed Body field; the
   alternative (RawBody + hand-rolled unmarshal) would erase the schema this feature exists
   to publish. `{}` still means "all", and the only consumer always sends a body. Pinned by
   `TestGenerateAINotes_AbsentBodyIs400`.
3. Ownership behaviour is unchanged and re-proven — `TestNoteOperations_NoteFromAnotherPlanIs404AndDoesNotMutate`
   and `TestNotesOperations_NilOwnershipFailsClosed`.

**Dead code removed with the group**

- `internal/notes/handler.go` — the gin handler, replaced not duplicated.
- `client/src/context/NotesContext.tsx` — never mounted, never consumed, and structurally
  broken (treated async service calls as synchronous).
- `client/src/components/notes/TaskNoteRelations.tsx` — no callers.
- `NotesService.getNotesForTask` / `.clearNotes` / the deprecated mock `generateAINote` —
  all called a `saveNotes()` that does not exist on the class, or `.filter` on a Promise.

**OUTSTANDING — the one criterion not met**

The **before/after probe (R14)** is not recorded. R14 requires it "against a running
gateway", and the gateway needs Postgres, Consul, and plan-service up; the environment had
no fireplace infra running. The round-trip test proves the payload shape is identical, but
that is R15's evidence, not R14's. This criterion is genuinely unproven, not passed.

**FE typecheck, precisely**

`tsc --noEmit` went from **26 errors to 8**. All 18 notes-domain errors are resolved. The
remaining 8 are pre-existing and outside this group — 7 in `Todo.tsx` (5 missing
`@/components/ui/*` modules, 2 `ScopeEnum` mismatches) and 1 in `app/plans/[planId]/page.tsx`
(`TodoProps` missing `planId`). None were introduced here and none are notes-related.

## Blocked By

I-0015 (shared transport package and registration path)

## Spec Reference

FS-0004 §Requirements R1–R8, R14, R15, R19 · §API surface (notes) · §Edge States

## TDD Approach

- RED: test asserting `?tags=a&tags=b` reaches the service as a two-element filter
- GREEN: declare the typed param as a slice
- RED: test asserting an absent `isRead` filter is not treated as `false`
- GREEN: use a pointer or explicit-presence param type
