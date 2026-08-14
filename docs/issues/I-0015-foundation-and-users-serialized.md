---
id: I-0015
status: open
implements: FS-0004
blocked_by: []
labels: [ready-for-agent]
---
Implements FS-0004 §Requirements, §API surface, §Edge States

## What to Build

The **users** group (4 operations), serialized — and, because it goes first, the frontend
client module that every later slice's cutover needs.

> **This issue was originally titled and scoped as a "foundation" slice. That framing was
> wrong and is corrected here**, because acting on it would have produced speculative work.
>
> It was carried over from barrowspire, where a shared transport package genuinely had to
> come first: those groups mirror protobuf timestamps as `{seconds, nanos}`, so the second
> group to declare `Timestamp` panicked huma's generator, which keys schemas by type name.
> **Fireplace converts to `time.Time` at the boundary instead**, so that collision does not
> exist here. Of the four original "foundation" items, one was speculative (the shared
> package), one already existed (`RegisterAPI`/`APIDeps`, built by FS-0002), and one is a
> three-line change (`StatusFor`). Only the frontend client module is genuinely once-only.
>
> The shared transport package is deferred to the first slice that actually needs it —
> I-0016 and I-0017 both mirror plan-service types and will collide there for real.

**Why users goes first** — it is the only group containing public operations, so it is the
only slice that proves the public/protected split end to end: that `signup` and `signin`
declare **no** security while the other two declare `bearerAuth`. That distinction is what
made the document misleading in the first place, so it is worth proving first rather than
last.

1. A single registration path called by **both** `config/routes.go` and `cmd/openapi`, running
   with no DB and no network (FS-0004 R3). `RegisterAPI`/`APIDeps` in `config/api.go` already
   is this — extend it, never add a parallel one.
2. `apierr.StatusFor` gains a `codes.Unavailable` case → 503 (R13). Absent today, so a
   downstream outage reports 500. Small, but it lands here because every later group inherits
   the mapping.
3. Scaffold the frontend client module: `openapi-fetch` over the generated `schema.d.ts`, the
   auth middleware that attaches the bearer token, and an `unwrap()` helper. No lint fence yet
   — the tree still has legitimate raw calls until I-0020.

**Users group** — `POST /users/signup`, `POST /users/signin` (both public),
`GET /users/{id}`, `GET /users` (both JWT).

4. Transport mirrors transcribed field-for-field from the current wire format, including
   `omitempty` (R6). Do not improve names. Map straight from the proto — never through
   `models.User` (ADR-0003 §3).
5. Envelope removed (R4), FE call sites for these four cut over in this same slice (R19).
6. `signup`/`signin` registered with no auth middleware and **no** `Security` block; the other
   two with both.

**Two facts found while planning, both of which constrain this slice:**

- **`models.User` carries `json:"password,omitempty"`** and is today the response type for
  `signin`, `getUser`, and `listUsers`. It is **not** leaking — `userFromProto` never assigns
  it and `pb.User` has no password field — but it is one careless assignment away, and the
  table behind that struct was dropped in migration 000020. Serializing removes the hazard
  structurally. Add a test asserting `password` appears in no users response.
- **`signup` discards its own result.** `client.SignUp` returns a full `LoginResponse` with
  access and refresh tokens; the handler throws it away and replies `{statusCode, message}`
  with no body. Transcribe as-is: **201, empty body**. Returning the tokens (auto-login on
  signup) is a real improvement and a real behavior change — out of scope per R6, worth its
  own feature spec.

## Acceptance Criteria

- [ ] The 4 users operations appear in `openapi.yaml` with correct methods, paths, and params
- [ ] `signup` and `signin` carry no `security:`; `getUser` and `listUsers` carry `bearerAuth`
- [ ] `cmd/openapi` generates the document with no DB and no network
- [ ] No `models.*` type and no protobuf message appears in `components.schemas`
- [ ] **R15, adapted to this repo:** a **populated** fixture proves the legacy body and the new
      body are JSON-equal — marshal `userFromProto(pb)` (what `result` holds today) and the new
      `UserResponse`, assert equality. The literal wording of R15 compares the proto's JSON to
      the mirror's; here the assertion that matters is legacy-vs-new, because that is what the
      frontend actually observes. A zero-valued fixture compares `{}` to `{}` and proves nothing
- [ ] `password` appears in no users response, proven by a test
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
