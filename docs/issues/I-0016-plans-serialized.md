---
id: I-0016
status: done
implements: FS-0004
blocked_by: [I-0015]
labels: [ready-for-agent]
---

> **All criteria met**, including the before/after probe (R14) — see
> `docs/notes/fs0004-envelope-probe.md`. The plans half of that probe initially
> ran against an EMPTY list, which would have passed vacuously; a plan was
> created and it was re-run so the record comparison is real.
>
> **Three defects found and fixed that were not in scope**, all of which would have
> shipped silently:
> 1. `CreatePlanReq.Description` would have been published REQUIRED (huma reads
>    omitempty; gin reads binding:"required" — independent mechanisms, and this
>    field had neither).
> 2. `SearchPlans` returned `[]*pb.SearchPlanResult` straight from the client, so a
>    protobuf would have entered `components.schemas` (ADR-0003 §3).
> 3. **A live frontend bug**: `myplans` typed search hits as full plans, but search
>    returns `{id, name, description, similarity}` with no `planType` — so the label
>    ternary rendered EVERY search result as "Learning". The generated types are what
>    surfaced it; the hand-written interface had been asserting a shape the API never
>    returned.
>
> The shared transport package this issue inherited from I-0015 was still not needed:
> plans introduced no type name that collides. I-0017 is where checklists and plans
> genuinely share, so it lands there.

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
- [x] Before/after probe recorded showing the envelope removed, payload otherwise unchanged — `docs/notes/fs0004-envelope-probe.md`
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
