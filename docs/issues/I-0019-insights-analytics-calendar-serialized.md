---
id: I-0019
status: open
implements: FS-0004
blocked_by: [I-0015]
labels: [ready-for-agent]
---
Implements FS-0004 §Requirements, §API surface, §Edge States

## What to Build

Serialize the three remaining small groups — 5 operations total — and cut the frontend over in
the same slice. They are folded into one issue because each is thin and none shares transport
types with the others.

**insights (3)** — `GET /insights/checklist-suggestion` ·
`GET /insights/checklist-suggestion-daily` · `GET /insights/suggest-videos`.
All take `plan_id` as a **query** param (snake_case — transcribed as-is per R6, do not
normalize to match notes' camelCase).

**analytics (1)** — `GET /analytics/user/{userId}`.

**calendar (1)** — `GET /plans/{id}/calendar` with `view` (`week`|`month`) and `date`.

The honest part of this slice:

- **`getUserAnalytics` returns 501 today.** Its handler and repository are documented stubs.
  Publish the operation with 501 in its `Errors` list and declare the success shape from
  `internal/useranalytics/model.go`. That success shape is **unproven until the data path
  lands** — say so in the close-out rather than claiming the criterion passed. Serializing a
  stub is legitimate; claiming it verified is not.
- `insights` shapes were flagged during scoping as possibly still moving. The user chose to
  serialize them anyway. If a shape churns shortly after this lands, that is a contract change
  through the normal path, not a surprise.

## Acceptance Criteria

- [x] All 5 operations appear in `openapi.yaml` with correct methods, paths, and params
- [x] `plan_id` stays snake_case; `view`/`date` transcribed as-is
- [x] `view` is documented as `week`|`month` and behaves identically to today
- [x] All 5 declare `Security: bearerAuth`
- [x] `getUserAnalytics` lists 501 among its errors, and the close-out states plainly that its
      success shape is unproven until the data path lands
- [x] No `models.*` type in `components.schemas`
- [x] Round-trip test over a **populated** fixture asserts JSON equality for each response type
      that has a live producer
- [ ] Before/after probe recorded for the operations that return real data
- [x] `make openapi-diff`, `make lint-contract`, `make openapi-breaking` green
- [x] FE call sites cut over to the generated client; typecheck green *(pre-existing unrelated
      errors remain — see close-out)*
- [x] Full Go test suite green

## Close-out

**The document is now complete: 34 of 34 operations.** FS-0004's whole surface is serialized;
only I-0020 (retire swaggo, harden gates) remains.

**§API surface correction — the one place the table could not be transcribed literally**

The table named `SuggestionResponse` as the response type for **both** `getChecklistSuggestion`
and `getDailyChecklistSuggestion`. Those two operations return different structures — a single
string and an array of strings — and huma keys schemas by Go type name, so one name for both is
a duplicate declaration that panics the generator. Every other array row in the table is written
`NoteResponse[]` / `PlanResponse[]`, so the daily row is missing its `[]`.

Resolved by giving each operation its own output type and publishing the bodies **bare**:

| Operation | Body | Schema |
|---|---|---|
| `getChecklistSuggestion` | `string` | inline `type: string` |
| `getDailyChecklistSuggestion` | `[]string` | inline `type: array` |
| `getSuggestedVideos` | `[]VideoSuggestionResponse` | named component ✓ |

Verified empirically: **huma emits a named component only for struct types** — defined scalar
and slice types are inlined at the use site. So "named `SuggestionResponse` in
`components.schemas`" and "body is an object" are the same decision, not two.

Bare was chosen over a wrapper on two independent grounds. **Consistency:** this contract
already publishes nine array-returning operations bare (`listPlans`, `searchPlans`,
`listSharedPlans`, three checklist lists, `listUsers`, `listNotes`, `generateAINotes`); a
wrapper would make insights the only group with a different collection convention. **R6:** a
wrapper invents a field name (`{"suggestion": ...}`) that ships nowhere today, and R4 already
spends this feature's one sanctioned break on removing the envelope.

Consequence to record honestly: **`SuggestionResponse` does not appear in
`components.schemas`.** The FS table rows should read `string` and `string[]`. If a named type
is wanted, that is a deliberate shape change and belongs in an FS — cheapest after
insights-service owns these endpoints.

**New error vocabulary**

`errcode.NotImplemented` (`NOT_IMPLEMENTED`) and `commonconstants.ErrNotImplemented` were added
and wired through `apierr.StatusFor` / `CodeFor` / `CodeForStatus`. Adding a code is explicitly
non-breaking. Without it `getUserAnalytics` would answer 501 with no domain code, and ADR-0004
says clients switch on `code`, never `detail` — a 501 without one is a contract break wearing a
passing status. Pinned by `TestGetUserAnalytics_IsNotImplementedWithDomainCode`.

**`getUserAnalytics` is a published stub.** It answers 501 today. Its success shape is declared
from `internal/useranalytics/model.go` and is **unproven until the data path lands**. Serializing
it is legitimate; claiming the success shape verified would not be, and this slice does not.

**`view` is deliberately not an enum.** The legacy handler forwarded any value to
calendar-service and let it decide, so an enum would turn a long-accepted request into a 422.
`TestGetCalendar_UnrecognisedViewIsForwardedNotRejected` pins the passthrough;
`TestGetCalendar_AbsentDateDefaultsToCurrentWindow` pins the `YYYY-MM-DD` / `YYYY-MM` defaulting.

**Live bug fixed incidentally.** `suggest-videos` was called with bare `fetch` and
`credentials: "include"` — **no bearer token** — against a route behind `AuthMiddleware`. That
call could only ever 401; the video panel was broken in production. Cutting it onto the generated
client attaches the token.

**Dead parameter dropped.** The FE sent `&scope=daily|longterm` to `checklist-suggestion`; the
handler never read it. Same situation I-0017 hit with `archived`. The table lists only `plan_id`.

**Refactor.** `errNoIdentity` had been copied into every serialized group. Collapsed onto a
single `apierr.ErrNoIdentity()`; the contract regenerated byte-identical afterwards.

**Deleted with the groups:** `internal/insights/handler.go`, `internal/useranalytics/handler.go`,
`internal/gateway/calendar/handler.go`, and the three FE functions in `services/api.ts`
(389 → 309 lines). `Todo.scope-clobber.test.tsx` had its mocks realigned — `Todo.tsx` now imports
from `@/api/insights`, so mocking those two functions on `@/services/api` intercepted nothing and
left the real module unmocked.

**OUTSTANDING — same criterion as I-0018**

The **before/after probe (R14)** is not recorded. It requires a running gateway, and the
environment has no fireplace infra up. Round-trip evidence exists (R15); this is R14's and is
genuinely unproven.

**FE typecheck:** 8 errors before this slice, 8 after — the same 8, all pre-existing and outside
these groups (7 in `Todo.tsx`, 1 in `app/plans/[planId]/page.tsx`). This slice added none.

**Next:** these three insights routes are now serialized, which discharges ADR-0002 §6 and
unblocks repointing them at insights-service over gRPC — the strangler step this was slice ⓪ for.

## Blocked By

I-0015 (shared transport package and registration path)

## Spec Reference

FS-0004 §Requirements R1–R8, R14, R15, R19 · §API surface (insights, analytics, calendar) ·
§Edge States

## TDD Approach

- RED: round-trip test for `CalendarResponse` against a populated proto message
- GREEN: transcribe the mirror
- RED: test asserting an unrecognized `view` behaves as it does today
- GREEN: transcribe the current handling rather than inventing validation
