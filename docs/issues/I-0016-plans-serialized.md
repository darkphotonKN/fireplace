---
id: I-0016
status: open
implements: FS-0004
blocked_by: [I-0015]
labels: [ready-for-agent]
---
Implements FS-0004 §Requirements, §API surface, §Edge States

## What to Build

Serialize the **plans** group — 8 operations proxied to plan-service — and cut the frontend
over in the same slice.

`GET /plans` · `GET /plans/search` · `GET /plans/shared` · `GET /plans/{id}` ·
`POST /plans` · `PATCH /plans/{id}` · `PATCH /plans/{id}/toggle-daily-reset` ·
`DELETE /plans/{id}`

Notes specific to this group:

- `listSharedPlans` takes `limit` (default 20) and `offset`. The defaults are applied in the
  handler today (`limit <= 0 → 20`); declare them on the typed param so the document states
  them, and keep the effective behavior identical.
- `searchPlans` binds a `SearchParam` struct — transcribe its fields as they are.
- `createPlan` returns **201**, unlike the rest of the group.
- `deletePlan` returns 204 with no body.
- Ownership checks live in plan-service and surface as 403. The gateway does not restate them.

## Acceptance Criteria

- [ ] All 8 operations appear in `openapi.yaml` with correct methods, paths, params, and
      status codes (201 on create, 204 on delete)
- [ ] `limit`/`offset` defaults are declared in the document and behave identically
- [ ] All 8 declare `Security: bearerAuth`
- [ ] Transport mirrors transcribed field-for-field including `omitempty`; no proto message or
      `models.*` type in `components.schemas`
- [ ] Round-trip test over a **populated** fixture asserts JSON equality
- [ ] List responses marshal to `[]` and never `null`
- [ ] Before/after probe recorded showing the envelope removed, payload otherwise unchanged
- [ ] `make openapi-diff`, `make lint-contract`, `make openapi-breaking` green
- [ ] FE plan call sites cut over to the generated client; client typecheck green
- [ ] Full Go test suite green

## Blocked By

I-0015 (registration path and the frontend client module)

> **This slice now owns the shared transport package.** I-0015 originally carried it, but users
> needed no shared type — fireplace converts protobuf timestamps to `time.Time` at the
> boundary, so barrowspire's `Timestamp` collision does not arise there. Plans and checklists
> both mirror plan-service types, so the first genuine duplicate-type-name risk lands here.
> Create the shared package when the first shared type actually appears, not before.

## Spec Reference

FS-0004 §Requirements R1–R8, R14, R15, R19 · §API surface (plans) · §Edge States

## TDD Approach

- RED: round-trip test for `PlanResponse` against a populated `pb.Plan`
- GREEN: transcribe the mirror from the plan proto's json tags
- RED: test asserting an empty plan list marshals to `[]`
- GREEN: use the slice converter that returns an empty slice rather than nil
