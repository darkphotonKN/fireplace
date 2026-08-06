# FS-0002: Profile view and edit as a serialized (typed) surface

> Status: work-order · SPECIFICATION.md: services/api-gateway/SPECIFICATION.md "## Users → Typed (serialized) profile surface" → this FS · Related ADRs: docs/adr/0002-contract-planes-code-first-openapi.md (as amended), docs/adr/0003-serialized-transport-types.md, docs/adr/0004-error-representation-rfc9457.md, docs/adr/0001-grpc-identity-via-metadata-rs256.md (identity seam)

## Summary

`GET /api/users/profile` and `PATCH /api/users/profile` already exist and already work. This
feature gives them a **designed, published contract**: the shape is declared in this spec,
generated into `openapi.yaml` by Huma from typed handler signatures, and consumed by the
frontend through a generated TypeScript client instead of hand-written fetch.

Nothing about what a user can do changes. What changes is that the two endpoints stop being
implicitly defined by handler code and start being governed. This is the **first slice ⓪
under ADR-0002's serialize-on-touch policy** — chosen deliberately as the contract layer's
validation run, because profile is small, owned end-to-end, and crosses both contract planes
(HTTP edge + gRPC to auth-service) without much domain surface to get wrong.

## Requirements

**Serialization and contract**

1. `GET /api/users/profile` and `PATCH /api/users/profile` are served by Huma-registered
   typed handlers mounted on the existing protected route group, behind the existing edge
   authentication. No route path changes.
2. `openapi.yaml` is generated from those handler signatures, committed, and never
   hand-edited. The committed spec must equal the regenerated spec (CI gate, ADR-0002).
3. The generated OpenAPI document is **3.x**, superseding the swaggo-generated OpenAPI 2.0
   document for these two operations only. Legacy endpoints continue to be described by the
   existing `/swagger` surface until they are themselves serialized.
4. No new `@`-annotations are written for these endpoints (ADR-0002 §8).

**Response shape**

5. Success responses are **bare resources** — no `{statusCode, message, result}` envelope
   (ADR-0003). This is a deliberate, grandfathered break for these two endpoints only.
6. The response is backed by a dedicated `ProfileResponse` transport type declared for these
   operations. `models.User` must not appear in the generated schema, directly or nested
   (ADR-0003 §3). `ProfileResponse` publishes exactly: `id`, `email`, `name`, `displayName`,
   `bio`, `createdAt`, `updatedAt` — and therefore can never publish `password`.
7. `PATCH` returns the updated profile in the same `ProfileResponse` shape, preserving
   current behavior.

**Request shape and update semantics**

8. `UpdateProfileRequest` declares `name`, `displayName`, `bio`, all optional.
9. **Absent and `null` are equivalent and both mean "leave unchanged."** This documents
   existing repository behavior (a dynamic `SET` clause omits nil fields); it is not a new
   rule and must not be "fixed" in this feature.
10. An empty request body (`{}`) is a valid no-op returning `200` with the unchanged profile.
11. Setting a field to `""` sets it to the empty string. This is distinct from `null` and is
    existing behavior.
12. **No edge validation constraints are declared in the schema.** The schema pins shape
    only; validation remains in `auth-service`, per the standing principle in
    `services/api-gateway/docs/api-conventions.md`. Consequence: `name: ""` travels
    downstream and returns **400**, not Huma's 422.

**Errors**

13. Every error response from these operations is RFC 9457 `application/problem+json`
    carrying a `code` and, where available, `errors[]` (ADR-0004).
14. `code` values come from the platform-wide vocabulary in **`common/errcode`**, introduced
    by this feature. Initial members: `UNAUTHENTICATED`, `VALIDATION_FAILED`, `NOT_FOUND`,
    `ALREADY_EXISTS`, `FORBIDDEN`, `INTERNAL_ERROR`, plus `PROFILE_NAME_EMPTY`.
15. `apierr.StatusFor` grows a domain-code dimension, returning a code alongside the status
    and client-safe message. This is the gateway-wide error boundary; the change is
    structural, not local to these endpoints.
16. **`errors[]` will be empty for downstream validation failures.** gRPC statuses carry only
    a string message, so field-level detail cannot be reconstructed from `auth-service`
    without structured error details on plane 2. This is why `PROFILE_NAME_EMPTY` exists as
    a distinct code rather than being folded into `VALIDATION_FAILED` with a field pointer —
    with no `errors[]` to carry the field, the code must carry the precision.
17. **401 responses on these endpoints must also be problem+json.** The existing
    `auth.AuthMiddleware` aborts with the legacy `{statusCode, message}` shape before any
    Huma handler runs, which would leave a hole in the contract on its most common error.
    A **problem-emitting auth variant is mounted on the serialized group only**; the existing
    middleware and every legacy protected route remain byte-identical.

**Identity**

18. The Huma handler reads the authenticated user from `context.Context`, not from
    `*gin.Context`. `humagin`'s `Context()` returns the _request_ context, so a bridge
    middleware copies the authenticated identity into it after authentication.
19. This bridge is the **single** identity seam for serialized routes. When ADR-0001's
    metadata-and-context identity lands, it converges here; a second, divergent identity path
    must not be created.

**Frontend**

20. The frontend consumes these two operations exclusively through the generated TypeScript
    client at `client/src/api/generated`. After this feature, hand-written fetch to these two
    paths is a HIGH review finding. All other endpoints remain grandfathered (ADR-0002 §7).

**Documentation surface**

21. Huma's docs and spec endpoints are mounted on a **public** group, mirroring the
    reachability `/swagger/index.html` has today. Huma's configured paths are relative to the
    group they mount on, so the doc surface and the operations mount separately.

## User Stories

1. As a **frontend developer**, I want profile requests and responses to be typed, so that a
   backend change that breaks my code fails at `tsc` instead of at runtime.
2. As a **frontend developer**, I want to import a generated client rather than hand-write
   fetch calls, so that I never restate the endpoint's shape by hand.
3. As a **frontend developer**, I want a stable `code` on every error, so that I can branch on
   failure kind without string-matching a human-readable message.
4. As a **frontend developer**, I want to know that `code` will not silently change meaning,
   so that my error handling stays correct across releases.
5. As a **frontend developer**, I want `401` to look like every other error, so that I have
   one error-handling path rather than a special case.
6. As a **frontend developer**, I want to read the current contract in a browser without a
   token, so that I can build against it without wiring auth first.
7. As an **end user**, I want to view my profile and see exactly the fields I own, so that
   nothing about my account is hidden or surprising.
8. As an **end user**, I want to update one field without resending the others, so that
   partial edits are safe.
9. As an **end user**, I want a rejected update to tell me which rule I broke, so that I can
   correct it.
10. As an **end user**, I want my password never returned by any endpoint, so that a client
    bug or a proxy log cannot expose it.
11. As a **gateway maintainer**, I want the published schema derived from typed signatures,
    so that it cannot drift from the code the way annotations did.
12. As a **gateway maintainer**, I want one error-mapping seam, so that a new endpoint gets
    correct problem+json without repeating a status switch.
13. As a **gateway maintainer**, I want serializing an endpoint to leave legacy endpoints
    untouched, so that adoption never requires a coordinated big-bang.
14. As a **gateway maintainer**, I want transport types declared per operation, so that a new
    database column cannot enter the public API by accident.
15. As a **platform engineer**, I want the error-code vocabulary shared in `common/`, so that
    every service's failures read the same way to the client.
16. As a **reviewer**, I want the PR's `openapi.yaml` diff to be the contract review surface,
    so that a contract change is visible rather than inferred from handler code.
17. As a **reviewer**, I want a breaking contract change to fail CI, so that it is chosen
    rather than discovered by a client.
18. As a **reviewer**, I want the `§API surface` table walked against the regenerated spec,
    so that "specified but not built" and "built but never designed" both stop the PR.
19. As a **future feature author**, I want this feature to be the worked example of slice ⓪,
    so that the next serialize-on-touch is mechanical rather than exploratory.
20. As a **future feature author**, I want the gotchas recorded, so that I do not rediscover
    the identity bridge or the group-relative doc paths.
21. As an **operator**, I want downstream error text never forwarded to clients, so that
    internal detail does not leak through the contract.

## Acceptance Criteria

- [ ] `GET /api/users/profile` returns `200` with a bare `ProfileResponse` — no envelope.
- [ ] `PATCH /api/users/profile` returns `200` with the updated bare `ProfileResponse`.
- [ ] The generated `openapi.yaml` contains both operations, and `password` appears nowhere
      in the document.
- [ ] `models.User` does not appear in the generated schema, directly or nested.
- [ ] `PATCH {}` returns `200` with the profile unchanged.
- [ ] `PATCH {"bio": null}` leaves `bio` unchanged (documented, tested).
- [ ] `PATCH {"bio": ""}` sets `bio` to the empty string.
- [ ] `PATCH {"name": ""}` returns `400` with `code: PROFILE_NAME_EMPTY` — **not** 422.
- [ ] Every error response carries `Content-Type: application/problem+json`.
- [ ] Every error response body carries a `code` from `common/errcode`.
- [ ] A request with no token returns `401` as **problem+json** with `code: UNAUTHENTICATED`.
- [ ] A legacy protected route (e.g. `GET /api/plans`) returns a `401` **byte-identical to
      today** — the fork did not leak.
- [ ] No downstream error text (e.g. `"name cannot be empty"` verbatim from auth-service)
      appears in any response body.
- [ ] The Huma handler resolves the caller's identity from `context.Context`.
- [ ] `GET /api/docs` and the spec endpoint are reachable **without** a token.
- [ ] The serialized operations still require a token.
- [ ] `/swagger/index.html` still serves and still describes the legacy endpoints.
- [ ] The frontend's profile read and update both go through the generated client; no
      hand-written fetch to these two paths remains.
- [ ] `oasdiff` reports the envelope removal and the 401 shape change as the **only** breaking
      changes, and both are present in the `.oasdiff.yaml` allowlist with a reason.
- [ ] Regenerating the spec produces no diff against the committed `openapi.yaml`.
- [ ] Spectral passes: descriptions, examples, and error schemas present on both operations.
- [ ] The full api-gateway test suite passes on the bumped gin/Go versions.

## Edge States

| Situation                                | Behavior                                                                                                      |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| No `Authorization` header                | `401` problem+json, `UNAUTHENTICATED`                                                                         |
| Malformed or expired token               | `401` problem+json, `UNAUTHENTICATED`; specific reason logged server-side only                                |
| Valid token, user row deleted            | `404` problem+json, `NOT_FOUND` (auth-service returns `codes.NotFound`)                                       |
| Empty PATCH body `{}`                    | `200`, profile unchanged                                                                                      |
| PATCH with all fields `null`             | `200`, profile unchanged — identical to `{}` by design                                                        |
| PATCH with `name: ""`                    | `400`, `PROFILE_NAME_EMPTY`, `errors[]` empty (see Requirement 16)                                            |
| PATCH with unknown fields                | Ignored; not an error. Huma does not reject unknown members by default and this feature does not change that  |
| Malformed JSON body                      | `400`, `VALIDATION_FAILED`                                                                                    |
| `auth-service` unreachable               | `500`, `INTERNAL_ERROR` — `codes.Unavailable` maps to 500 today and this feature does not change that mapping |
| Concurrent PATCHes                       | Last write wins per field; no optimistic concurrency. Existing behavior, explicitly unchanged                 |
| Downstream returns an unmapped gRPC code | `500`, `INTERNAL_ERROR`; the wire message is logged, never returned                                           |
| Client requests a non-JSON `Accept`      | Huma content negotiation applies; JSON remains the only representation                                        |

## API surface

**Transport types** (per ADR-0003; declared for these operations, not shared)

`ProfileResponse` — `id` uuid · `email` string · `name` string · `displayName` string|null ·
`bio` string|null · `createdAt` timestamp · `updatedAt` timestamp

`UpdateProfileRequest` — `name` string? · `displayName` string? · `bio` string?
(all optional; absent ≡ null ≡ unchanged)

| Op              | Method + Path              | Query/Params | Request body           | Response                | Errors (status · code)                                                                                                        |
| --------------- | -------------------------- | ------------ | ---------------------- | ----------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| `getProfile`    | `GET /api/users/profile`   | none         | none                   | `200` `ProfileResponse` | `401 · UNAUTHENTICATED` · `404 · NOT_FOUND` · `500 · INTERNAL_ERROR`                                                          |
| `updateProfile` | `PATCH /api/users/profile` | none         | `UpdateProfileRequest` | `200` `ProfileResponse` | `400 · PROFILE_NAME_EMPTY` · `400 · VALIDATION_FAILED` · `401 · UNAUTHENTICATED` · `404 · NOT_FOUND` · `500 · INTERNAL_ERROR` |

All error responses are `application/problem+json` (RFC 9457) with members `type`, `title`,
`status`, `detail`, plus extensions `code` and `errors[]`. Prose semantics for each operation
and each failure live in Requirements; this table pins **shape** only.

**Doc surface** (registered by this feature, not part of the operation contract): the Huma
docs UI and the OpenAPI document mount on a public group, reachable without a token.

> **Format note:** the `Errors` column carries `status · code` pairs per ADR-0004.
> `docs/specs/README.md` has not yet been updated to codify this column — that format-authority
> edit is outstanding and this table is its first instance.

## Out of Scope

- **NULL-clearing** of `displayName` / `bio`. Requires a nullable wrapper at the gateway plus
  a proto change and an `auth-service` repository change — both contract planes would move
  during the validation run. Recorded as a future capability.
- **Structured field-level errors from downstream.** Populating `errors[]` for `auth-service`
  validation failures needs structured gRPC error details (plane 2). Until then `errors[]` is
  empty for downstream failures.
- **Edge validation constraints** (`minLength` etc.). Declining these is Requirement 12; the
  400→422 status move it would cause is explicitly rejected.
- **Other `## Users` endpoints** — `POST /users/signup`, `POST /users/signin`,
  `GET /users/:id`, `GET /users`. Same bounded context and same handler file, but out of this
  touch; they stay legacy and enveloped until serialized.
- **Migrating any non-profile endpoint's 401** to problem+json. The auth fork is deliberately
  scoped to the serialized group.
- **Retiring swaggo.** `make gen`, the generated `docs/` package, and the OpenAPI 2.0 Swagger
  UI all remain until the last legacy endpoint serializes (ADR-0002 §8).
- **Standing up the gate tooling itself** — authoring oasdiff/Spectral/buf rules and the CI
  jobs is ADR-0002's implementation. This feature _consumes_ those gates and is the first
  thing to exercise them.
- **Un-gitignoring `services/api-gateway/go.sum`.** It weakens ADR-0002's regenerate-and-diff
  gate and should be fixed, but it is a repo-hygiene change, not this feature's work.
- **The buf / plane-2 governance work.** Untouched here.
