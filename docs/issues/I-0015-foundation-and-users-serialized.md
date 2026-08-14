---
id: I-0015
status: open
implements: FS-0004
blocked_by: []
labels: [ready-for-agent]
---
Implements FS-0004 §Requirements, §API surface, §Edge States

## What to Build

Slice ⓪ of the batch retrofit: the shared machinery every later group depends on, plus the
**users** group (4 operations) as its first real consumer. Users goes first because it is the
only group containing public operations, so it proves the public/protected split end to end.

**Foundation**

1. A shared transport package (`internal/wire` or equivalent) holding types that more than one
   group needs, plus JSON round-trip converters. Huma keys schemas by **type name** — a second
   group declaring `Timestamp` panics the generator, so this must exist before the second group
   lands, not after.
2. A single registration path called by **both** `config/routes.go` and `cmd/openapi`. It must
   run with no DB and no network (FS-0004 R3). The existing `MountSerialized`/`APIDeps` shape in
   `config/api.go` is the starting point — extend it rather than adding a parallel one.
3. `apierr.StatusFor` gains a `codes.Unavailable` case → 503 (R13). It is absent today, so a
   downstream outage currently reports 500. This lands in foundation because every later group
   inherits the mapping.
4. Scaffold the frontend client module: `openapi-fetch` over the generated `schema.d.ts`, the
   auth middleware that attaches the bearer token, and an `unwrap()` helper. No lint fence yet
   — the tree still has legitimate raw calls until I-0020.

**Users group** — `POST /users/signup`, `POST /users/signin` (both public),
`GET /users/{id}`, `GET /users` (both JWT).

5. Transport mirrors transcribed field-for-field from the current wire format, including
   `omitempty` (R6). Do not improve names.
6. Envelope removed (R4), FE call sites for these four cut over in this same slice (R19).
7. `signup`/`signin` registered with no auth middleware and **no** `Security` block; the other
   two with both.

## Acceptance Criteria

- [ ] The 4 users operations appear in `openapi.yaml` with correct methods, paths, and params
- [ ] `signup` and `signin` carry no `security:`; `getUser` and `listUsers` carry `bearerAuth`
- [ ] `cmd/openapi` generates the document with no DB and no network
- [ ] No `models.*` type and no protobuf message appears in `components.schemas`
- [ ] Round-trip test over a **populated** fixture asserts JSON equality between the downstream
      message and its transport mirror (R15) — a zero-valued fixture compares `{}` to `{}` and
      proves nothing
- [ ] A downstream `codes.Unavailable` maps to 503, proven by a test
- [ ] Before/after probe recorded against a running gateway showing the envelope removed and
      the payload otherwise unchanged (R14) — the ratchet structurally cannot see this break
- [ ] `make openapi-diff` and `make lint-contract` green
- [ ] FE calls to these four endpoints go through the generated client; client typecheck green
- [ ] Full Go test suite green

## Blocked By

None

## Spec Reference

FS-0004 §Requirements R1–R8, R13, R14, R15, R16, R19 · §API surface (users) · §Edge States

## TDD Approach

- RED: round-trip test for the users transport mirrors against a populated fixture — fails
  because the mirrors do not exist
- GREEN: transcribe the mirrors from the auth proto's json tags
- RED: test asserting `codes.Unavailable` → 503
- GREEN: add the case to `StatusFor`
