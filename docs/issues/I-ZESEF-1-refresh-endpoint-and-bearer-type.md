---
id: I-ZESEF-1
status: done
implements: FS-ZESEF
blocked_by: []
labels: [bug]
title: "FS-ZESEF slice 1: refresh endpoint, access-only bearer, 30-day refresh TTL"
---
Implements FS-ZESEF §Requirements (R1–R9, R11 for the refresh TTL), §API surface, §Edge States (Unauthorized)

## What to Build

The gateway can exchange a refresh token over HTTP, and stops accepting one as a bearer. Nothing in
the client calls it yet, so no user-visible behaviour changes in this slice.

**Exchange (R1–R6).** `POST /api/users/refresh`, public, registered next to signin in
`typed_users.go`, through the same `ProblemFor` path. New `RefreshRequest` / `RefreshInput` /
`RefreshOutput` beside the signin types in `users_transport.go`, with the output body being
`AuthResponse`. `RefreshProto` goes on `UsersClient` and `LocalClient`, calling the `Service.Refresh`
that already exists and already issues a full new pair. Follow the shape of signin's
registration, including a description saying that a documented 401 does not mean a token is
required in the header.

**Deleted user (R4).** `Service.Refresh` maps a repository `ErrNotFound` to `ErrUnauthorized`, as
`SignIn` does. Other repository errors stay wrapped, so an outage stays a 500.

**Bearer type (R7, R8).** A new access-side validator next to `ValidateRefreshToken` in
`common/auth/jwt.go`. `authenticate` in `internal/auth/middleware.go` uses it. Both the legacy and
the problem+json renderers go through `authenticate`, so one change covers both. Do not write a
second verifier in the gateway (ADR-0009 §5).

**Refresh TTL (R9, R11).** 720h in `docker-compose.yml`, in the `config/routes.go` default and in
the `service.go` const. Replace the `routes.go` comment that says the defaults match the `.env`.

**Leave the access TTL at 24h in compose and in `routes.go`.** R12 forbids a 1-hour access token
reaching production ahead of the client seam, so slice 2 owns that change. The `service.go` access
const is already 1h.

**Contract (R6).** Additive. Run `make -C services/api-gateway openapi`, then `client`, then `gates`.

**Spec prose.** In `services/api-gateway/SPECIFICATION.md`: correct the JWT lifecycle bullet (compose
sets the TTLs, not a deployed `.env`; refresh is 720h; fallback consts match), add the refresh row
to the REST catalog, and add a wrong-type token to the edge auth 401 list. Do not flip the thin-line
checkboxes here.

Files expected to touch:
- `common/auth/jwt.go`
- `common/auth/jwt_test.go` (new)
- `services/api-gateway/internal/auth/middleware.go`
- `services/api-gateway/internal/auth/middleware_test.go`
- `services/api-gateway/internal/gateway/auth/service.go`
- `services/api-gateway/internal/gateway/auth/service_test.go` (new)
- `services/api-gateway/internal/gateway/auth/local.go`
- `services/api-gateway/internal/gateway/auth/typed_users.go`
- `services/api-gateway/internal/gateway/auth/users_transport.go`
- `services/api-gateway/internal/gateway/auth/users_transport_test.go`
- `services/api-gateway/config/routes.go`
- `services/api-gateway/config/api_test.go`
- `services/api-gateway/openapi.yaml` (generated)
- `client/src/api/generated/schema.d.ts` (generated)
- `services/api-gateway/SPECIFICATION.md`
- `docker-compose.yml`

## Acceptance Criteria

- [ ] `POST /api/users/refresh` with a valid refresh token and no `Authorization` header returns 200
      and an `AuthResponse`, and the new access token works as a bearer.
- [ ] The new refresh token has a full lifetime from the moment of the exchange, and can itself be
      exchanged.
- [ ] A garbage token, an expired refresh token, an access token and an empty string each return
      `401 · UNAUTHENTICATED`.
- [ ] A valid refresh token for a deleted user returns `401 · UNAUTHENTICATED`.
- [ ] A missing `refreshToken` or an unknown body member returns `422 · VALIDATION_FAILED`.
- [ ] A repository failure during refresh returns 500, not 401.
- [ ] A refresh token used as a bearer returns 401 on a protected route, in both renderers; an
      access token still passes. A token with no `tokenType` claim is rejected.
- [ ] `TestMount_PublicSurfacesUnaffected` covers refresh as a public operation, and its comment no
      longer says signin and signup are the only public ones.
- [ ] `openapi.yaml` and `schema.d.ts` are regenerated, and `make -C services/api-gateway gates`
      passes with no allowlist entry.
- [ ] Compose, `config/routes.go` and `service.go` all say 720h for refresh. Compose and `routes.go`
      still say 24h for access. The `routes.go` comment no longer mentions a `.env`.
- [ ] `services/api-gateway/SPECIFICATION.md` is corrected as described above.
- [ ] Tests pass: `go test ./...` in `common` and in `services/api-gateway`.

## Blocked By

None

## Spec Reference

FS-ZESEF §Requirements R1–R9 and R11 (refresh half), §API surface, §Edge States (Unauthorized,
Expired), §Acceptance Criteria (Gateway). User stories 3, 10, 15, 16, 17, 20.

## TDD Approach

- RED: `Service.Refresh` with a fake repository returning `ErrNotFound` expects `ErrUnauthorized`.
  Fails today with a wrapped generic error.
- RED: a middleware test sends a `TokenTypeRefresh` token as the bearer and expects 401. Passes
  through today.
- GREEN: the mapping, then the validator wired into `authenticate`, then the operation.
