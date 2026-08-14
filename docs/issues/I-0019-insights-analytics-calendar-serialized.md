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

- [ ] All 5 operations appear in `openapi.yaml` with correct methods, paths, and params
- [ ] `plan_id` stays snake_case; `view`/`date` transcribed as-is
- [ ] `view` is documented as `week`|`month` and behaves identically to today
- [ ] All 5 declare `Security: bearerAuth`
- [ ] `getUserAnalytics` lists 501 among its errors, and the close-out states plainly that its
      success shape is unproven until the data path lands
- [ ] No `models.*` type in `components.schemas`
- [ ] Round-trip test over a **populated** fixture asserts JSON equality for each response type
      that has a live producer
- [ ] Before/after probe recorded for the operations that return real data
- [ ] `make openapi-diff`, `make lint-contract`, `make openapi-breaking` green
- [ ] FE call sites cut over to the generated client; typecheck green
- [ ] Full Go test suite green

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
