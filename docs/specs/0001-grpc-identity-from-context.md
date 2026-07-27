# FS-0001: gRPC caller identity from context (distributed JWT validation)

> Status: work-order · SPECIFICATION.md: services/plan-service/SPECIFICATION.md "Security / Identity" entry → this FS · Related ADRs: docs/adr/0001-grpc-identity-via-metadata-rs256.md

## Summary

Downstream gRPC services stop trusting a plaintext `user_id` field in the request body and
instead learn *who the caller is* from a verified JWT carried in gRPC metadata. A shared
unary server interceptor validates the token locally (RS256, verify-only) and injects the
caller's identity into `context.Context`; handlers read identity from context. This is
Phase 1 — proving the full loop end-to-end on **plan-service** (edge → service, and the
insights→plan service-to-service hop) — of a platform-wide move off edge-only trust.

The architectural constraint (metadata + RS256 verify-only, client-side propagation
mandatory) is fixed in **ADR-0001**; this spec is the plan-service realization of it.

## Context (harvested from scoping)

- **Today:** JWT is validated only at the api-gateway HTTP edge. Identity is then passed to
  gRPC services as a plaintext `user_id` string in the request message body, trusted blindly
  over `insecure` (plaintext) transport. Zero gRPC interceptors exist (6 bare `grpc.NewServer()`).
  Anyone with mesh access can impersonate any user by setting a field.
- **Reference (barrowspire):** good interceptor/context *seam* (`NewValidator`, `Auth`
  interceptor, `MemberIDFromCtx` with an unexported struct context key) but a half-finished
  migration — registered on 1/8 services, **no client injects the metadata** (the path is dead
  code), and an HS256 shared secret handed to every service (any holder can *mint* tokens).
- **This spec deliberately diverges:** finish the client-side propagation barrowspire skipped,
  and use **RS256** so distributing verification material never grants minting.

## Locked decisions (from ADR-0001 / scope-it)

1. Identity travels as `authorization: Bearer <jwt>` in gRPC metadata, extracted into context.
2. Services verify **locally with RS256** (auth-service signs private, services verify public).
3. Legacy body `user_id` is **kept and cross-checked** against the token (mismatch → `Unauthenticated`).
4. **AuthN only** (no roles). No RPC left fully open: health is allowlisted; no-user calls
   (DailyReset cron) use a **service-principal** credential, not a user JWT.
5. **Phased:** Phase 1 = plan-service end-to-end incl. insights→plan forwarding. Phase 2 =
   insights + calendar (same pattern, separate rollout).

### Phase-1 defaults for the three parked open questions

These were parked in scoping and are resolved here as Phase-1 defaults (synthesized in
`--from-thread` mode — override before `/develop` if desired):

- **Public-key distribution → env/mounted PEM.** Each verifying service loads the auth-service
  RS256 **public** key from `AUTH_JWT_PUBLIC_KEY` (PEM string, or a file path via
  `AUTH_JWT_PUBLIC_KEY_FILE`), mirroring the existing `JWT_SECRET` env plumbing. A JWKS
  endpoint is the rotation-friendly evolution but is **out of scope** for Phase 1 (see below).
- **Key rotation → dual-key overlap.** Verifiers accept a set of public keys
  (`AUTH_JWT_PUBLIC_KEY` current + optional `AUTH_JWT_PUBLIC_KEY_NEXT`) so a token signed by
  the outgoing key still verifies during a rotation window. Rotation is operational (redeploy
  with the new key promoted); no automatic rotation in Phase 1.
- **Internal service credential → auth-service-minted RS256 service token.** The gateway's
  nightly-cron call to `DailyReset` presents a short-lived RS256 token whose claims mark it a
  **service principal** (e.g. `principal: "service"`, `sub: "service:api-gateway"`) rather than
  a user. The interceptor recognizes service-principal tokens on the same RS256 verification
  path (one verifier, one interceptor) and authorizes the internal method allowlist. No static
  shared secret is introduced.

## Requirements

1. A shared unary **server interceptor** lives in `common/` (extend `common/auth`/`commonauth`
   or a new `common/grpcauth`) exposing a barrowspire-shaped seam:
   `NewValidator(keys ...) func(token string) (Principal, error)`, an `Auth(validate) grpc.UnaryServerInterceptor`,
   and `IdentityFromContext(ctx) (Principal, bool)` backed by an **unexported struct context key**.
2. The `Principal` carries at least the user UUID and a principal kind (`user` | `service`); for
   service principals the service name is available.
3. auth-service **signs access + refresh tokens with RS256** (private key from
   `AUTH_JWT_PRIVATE_KEY`/`_FILE`); `commonauth` gains RS256 sign + verify and the existing
   `ParseToken` / `ValidateRefreshToken` switch to RS256. Signing method is pinned to RSA
   (reject non-RSA `alg`, mirroring today's HMAC-pinning).
4. Verifying services load the RS256 **public** key(s) from env/file (§Phase-1 defaults) and
   accept a current + optional next key for rotation overlap.
5. The interceptor: reads `authorization` metadata, requires the `Bearer ` prefix, verifies the
   token, and injects the resulting `Principal` into context. Missing/malformed/invalid →
   `codes.Unauthenticated`.
6. A **public-method allowlist** (at minimum `grpc.health.v1.Health/Check`) bypasses auth.
7. **Cross-check:** for any RPC whose request still carries a `user_id` field, the handler (or a
   shared helper) rejects the call with `Unauthenticated` when the body `user_id` ≠ the context
   user identity. Context identity is the source of truth.
8. plan-service registers the interceptor via `grpc.NewServer(grpc.ChainUnaryInterceptor(...))`
   at `services/plan-service/config/services.go` (the single `grpc.NewServer()` site).
9. plan-service handlers read identity via `IdentityFromContext(ctx)` instead of
   `uuid.Parse(req.UserId)` (all PlanService + ChecklistService user-scoped RPCs).
10. **Client-side propagation (mandatory):** the api-gateway plan client
    (`services/api-gateway/internal/gateway/plan/client.go`) injects `authorization: Bearer <jwt>`
    into outgoing gRPC context via `metadata.AppendToOutgoingContext`, forwarding the end-user's
    token from the edge. The gateway already holds the raw bearer at the edge.
11. **Service-to-service propagation (mandatory):** the insights-service plan client
    (`services/insights-service/internal/insights/plan_client.go`) **forwards the original
    caller's token unchanged** (does not re-mint) into outgoing metadata when calling plan-service.
12. `DailyReset` requires a **service-principal** token (§Phase-1 defaults); a user-principal or
    missing service token → `Unauthenticated`/`PermissionDenied`. The gateway nightly cron obtains
    and presents the service token.
13. Rollout is **non-breaking**: with the cross-check in place, existing body-field callers keep
    working as long as they also present a matching token; the proto `user_id` fields are **not**
    removed in this FS.

## User Stories

1. As an **end user**, I want my requests to reach plan-service carrying my verified identity, so
   that the plan I create is owned by me without any component re-typing my id into a body field.
2. As a **plan-service maintainer**, I want the service to authenticate its caller itself, so that
   it is not blindly trusting a spoofable `req.user_id` over plaintext transport.
3. As a **plan-service handler**, I want to read the user id from `context.Context`, so that my
   business logic can't be tricked by a mismatched body field.
4. As the **api-gateway**, I want to forward the end-user's bearer token as gRPC metadata, so that
   downstream services can verify identity end-to-end.
5. As **insights-service (a gRPC client)**, I want to forward the original caller's token when I
   call plan-service, so that the plan-service interceptor sees the real user — closing the exact
   gap barrowspire left open.
6. As **auth-service**, I want to sign tokens with an RS256 private key, so that other services can
   verify without holding a secret that would let them mint tokens.
7. As a **verifying service (plan/insights/calendar)**, I want to load only the public key, so that
   a leak of my environment lets an attacker verify but never mint.
8. As a **platform/ops engineer**, I want to rotate the signing key with an overlap window
   (current + next public key), so that in-flight tokens signed by the old key still verify during
   the rollover.
9. As a **platform/ops engineer**, I want the public key delivered the same way as the existing
   `JWT_SECRET` (env/mounted file), so that Phase 1 needs no new infrastructure.
10. As the **nightly-reset cron (system principal)** in api-gateway, I want to call
    `DailyReset` with a service-principal token, so that a no-user maintenance sweep is
    authenticated without pretending to be a user.
11. As the **interceptor**, I want to distinguish a service principal from a user principal via a
    token claim on one RS256 verification path, so that there is a single verifier and a clean
    system-vs-user split.
12. As a **liveness/readiness probe**, I want the gRPC health check to bypass auth, so that
    orchestration health-checking isn't rejected as unauthenticated.
13. As a **security reviewer**, I want an interceptor that rejects missing, malformed, or
    invalid-signature tokens with `Unauthenticated`, so that no unauthenticated call reaches a
    handler (outside the explicit allowlist).
14. As a **security reviewer**, I want the body `user_id` cross-checked against the token, so that
    a caller cannot act as a different user even if a stale/incorrect body id is sent.
15. As an **attacker on the mesh**, I want to be unable to impersonate a user by setting
    `req.user_id`, because without a valid matching token the call is rejected. (Negative story.)
16. As an **attacker who compromised a downstream service's env**, I want to be unable to mint
    tokens for arbitrary users, because that service holds only the RS256 public key. (Negative.)
17. As a **plan-service handler for an already-user-scoped RPC** (CreatePlan, ListPlans,
    UpdatePlan, ToggleDailyReset, DeletePlan, AssertPlanOwnership, SearchPlans, ListSharedPlans,
    ListItemsByUser, …), I want identity from context, so I stop calling `uuid.Parse(req.UserId)`.
18. As the **auth-service token-refresh path**, I want refresh tokens also RS256-signed and
    verified, so that the whole token family uses one asymmetric scheme.
19. As a **developer adding a new gRPC client call**, I want a shared helper that attaches the
    inbound token to the outbound context, so that forwarding identity is the easy default and
    the barrowspire "forgot to propagate" failure can't recur.
20. As a **plan-service operator**, I want the rollout to be non-breaking (body field kept +
    cross-checked), so that I can deploy the interceptor without a coordinated big-bang change.
21. As an **insights/calendar maintainer (Phase 2)**, I want plan-service to establish the exact
    interceptor + propagation pattern, so that replicating it to my service is mechanical.
22. As a **developer**, I want a clear `Principal` type in context (user id + kind), so that future
    authorization (roles) can build on it without reworking the transport.

## Acceptance Criteria

- [ ] `common/` exposes `NewValidator`, `Auth(validate) grpc.UnaryServerInterceptor`, and
      `IdentityFromContext(ctx) (Principal, bool)` with an unexported struct context key.
- [ ] auth-service issues RS256-signed access **and** refresh tokens; `commonauth` verifies RS256
      and rejects non-RSA `alg`.
- [ ] Verifying services load the public key from `AUTH_JWT_PUBLIC_KEY`/`_FILE`; a token signed by
      either the current or the configured next key verifies (rotation overlap).
- [ ] plan-service `grpc.NewServer` chains the auth interceptor; a call with no `authorization`
      metadata to a non-allowlisted method returns `Unauthenticated`.
- [ ] A call with a valid token reaches the handler, and `IdentityFromContext(ctx)` returns the
      token's user id.
- [ ] A call whose body `user_id` ≠ the token's user id returns `Unauthenticated`.
- [ ] `grpc.health.v1.Health/Check` succeeds without a token.
- [ ] The api-gateway plan client attaches `authorization: Bearer <jwt>` to outgoing gRPC context;
      a real HTTP → gateway → plan-service round trip authenticates via context, not the body.
- [ ] The insights-service plan client forwards the inbound caller's token to plan-service; an
      insights→plan call authenticates as the original user.
- [ ] `DailyReset` succeeds only with a valid service-principal token and is rejected for a
      user-principal or tokenless call; the gateway cron presents the service token.
- [ ] plan-service handlers no longer derive identity from `uuid.Parse(req.UserId)` for
      user-scoped RPCs (context is used; body is only cross-checked).
- [ ] Existing gateway flows continue to work end-to-end (non-breaking; proto fields unchanged).

## Edge States

- **Missing `authorization` metadata** (non-allowlisted method) → `Unauthenticated`.
- **Malformed header** (no `Bearer ` prefix / empty value) → `Unauthenticated`.
- **Invalid signature / wrong key / tampered token** → `Unauthenticated`.
- **Expired token** → `Unauthenticated` (verifier enforces `exp`).
- **Valid token, body `user_id` mismatch** → `Unauthenticated` (cross-check).
- **Valid token, RPC carries no `user_id`** (e.g. `GetPlan`) → authenticated; identity is in
  context; ownership remains separate domain logic (unchanged by this FS).
- **Service-to-service hop with no inbound token** (insights called without a token) → the
  outbound call to plan-service is `Unauthenticated`; insights surfaces it, doesn't fabricate one.
- **`DailyReset` with a user token instead of a service token** → rejected (needs service principal).
- **Health check with/without token** → always allowed (allowlist).
- **Key rotation window:** token signed by outgoing key while `NEXT` is promoted → still verifies
  until the outgoing key is removed; after removal, old tokens → `Unauthenticated`.
- **Public key misconfigured/absent at a verifying service** → service fails fast at startup
  (cannot verify anything) rather than silently accepting.
- **Plaintext transport:** metadata (incl. bearer) travels in clear on the mesh — accepted risk
  for Phase 1; mTLS is a noted follow-up, not covered here.
- **Refresh flow:** refresh token also RS256; an HS256-era token (pre-migration) fails to verify
  — acceptable given a coordinated deploy / token re-issue.

## Out of Scope

- **Authorization / roles.** No role claim, no role-based checks. Ownership stays in domain logic
  (`AssertPlanOwnership`, share checks). Barrowspire's dead `Role` enum is not adopted.
- **Removing `user_id` from proto request messages.** Kept and cross-checked; field removal is a
  later cleanup FS.
- **JWKS endpoint** and automated key rotation. Phase 1 uses env/file PEM + manual dual-key
  overlap; JWKS is the future evolution.
- **mTLS / transport encryption** between services.
- **Phase 2 rollout** to insights-service and calendar-service (same pattern, separate work).
- **Non-JWT auth** (API keys, sessions) and any change to the HTTP edge's public routes
  (signup/signin remain public and unchanged).
