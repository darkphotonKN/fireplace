# ADR-0001 — gRPC caller identity via metadata + RS256 verify-only in services

Status: accepted
Date: 2026-07-25
Realized by: FS-0001 (plan-service Phase 1)

## Context

Recorded without adversarial review — locked in a `scope-it` session and marked
**(not challenged)** by user choice. The decision concerns authentication, which is
expensive to reverse, so this marker is deliberate.

Today JWT is validated in exactly **one** place: the api-gateway HTTP (Gin) edge
middleware. After the edge, the authenticated user's identity is passed to downstream
gRPC services as a **plaintext `user_id` string field inside the request message body**,
and every service blindly `uuid.Parse(req.UserId)` and trusts it. There are **zero gRPC
interceptors** in the platform (all six `grpc.NewServer()` calls are bare), and the
transport is `insecure` (plaintext). The consequence: anything with network access to a
downstream service can impersonate any user by setting a body field. This is an
authentication gap, not merely a code-style issue.

We studied the sibling repo **barrowspire** as a reference. Its interceptor/context seam
design is good and worth adopting, but its *state* is a cautionary tale, not a template:

- The auth interceptor is registered on **1 of 8 services** (wallet only).
- **No client anywhere injects the `authorization` metadata** — so the entire
  context-propagation path is correctly-written **dead code**. The part that makes it work
  end-to-end (client-side propagation) was never built.
- It uses an **HS256 shared secret** distributed to every service. With symmetric HMAC the
  *verify* key **is** the *sign* key, so every service — and every leaked service env — can
  **mint** tokens, not just verify them. This is the weakest link and it gets strictly worse
  as validation is distributed.

Forces: we want validation *distributed into the critical services* (defense-in-depth,
not edge-only trust); we want identity read from `context.Context` rather than a spoofable
body field; and we want to avoid barrowspire's two failures — an unfinished vertical and a
blast-radius-widening key model.

## Decision

1. **Identity propagates via gRPC metadata, extracted from context.** Caller identity
   travels as `authorization: Bearer <jwt>` in gRPC metadata and is injected into
   `context.Context` by a shared unary **server interceptor** in `common/`. Seam shape
   (adopted from barrowspire): `NewValidator`, an `Auth` interceptor, and
   `MemberIDFromCtx(ctx)` backed by an **unexported struct context key** (collision-safe).
   **Client-side propagation is mandatory** — the api-gateway clients *and* every
   service-to-service client (e.g. insights→plan) MUST inject the metadata. This is the
   exact step barrowspire skipped and is non-negotiable for the path to carry traffic.

2. **Services verify tokens locally with RS256 (asymmetric).** auth-service signs with a
   **private** key; downstream services (plan, insights, calendar) verify with the **public**
   key. We explicitly reject the HS256 shared-secret model: distributing asymmetric
   *verification* material lets a compromised service verify but **never mint**, whereas a
   distributed symmetric secret grants minting to every holder. This is the deliberate
   improvement over barrowspire.

3. **Legacy body `user_id` is kept and cross-checked during migration.** The
   context-injected identity is the source of truth; where a request still carries a
   `user_id` body field, a mismatch against the token → `Unauthenticated`. Removing the
   proto fields is deferred to a later cleanup, keeping this change non-breaking.

4. **Scope is authentication only, and nothing is left fully open.** No roles/authZ this
   round (ownership stays in domain logic). A small public-method allowlist (gRPC health)
   bypasses auth; no-user calls such as plan-service `DailyReset` (fired by the gateway's
   nightly cron) carry a **separate internal service-principal credential** rather than
   being left unauthenticated — a clean system-principal vs user-principal split.

## Consequences

**Accepted / positive:**
- Every critical service becomes self-defending: it authenticates its caller instead of
  trusting a body field over plaintext transport.
- A leaked downstream-service environment cannot mint tokens (RS256), only verify.
- Non-breaking rollout: the cross-check lets old (body) and new (context) coexist.

**Costs / follow-ups:**
- Every critical service's `grpc.NewServer` gains a `ChainUnaryInterceptor`; every gRPC
  client (gateway + service-to-service) must inject `authorization` metadata.
- auth-service switches signing from HS256 to RS256 (issue, verify, and refresh paths in
  `commonauth`), and the **public key must be distributed** to each verifying service. The
  distribution mechanism is an **open question** (env PEM vs mounted file vs JWKS endpoint)
  deferred to FS-0001, as is key rotation.
- Handlers migrate `req.GetUserId()` → `MemberIDFromCtx(ctx)`.
- Metadata (including the bearer token) still travels over `insecure` transport in clear;
  mTLS is out of scope here and noted as a future follow-up.
- **Rollout is phased:** Phase 1 proves the full loop end-to-end on plan-service (interceptor
  + gateway injection + insights→plan forwarding); Phase 2 replicates to insights-service and
  calendar-service. Realized by **FS-0001**.
