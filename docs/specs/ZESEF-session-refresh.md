# FS-ZESEF: Sessions that renew themselves

> Status: work-order · SPECIFICATION.md: api-gateway `## Users` "Refresh-token exchange for a new token pair", api-gateway `## Authorization` "Bearer accepts access tokens only", client `## API integration` "Sessions that renew themselves while in use" → this FS · Related FS: FS-0004 (serialized gateway surface) · Related ADRs: docs/adr/0002-contract-planes-code-first-openapi.md, docs/adr/0004-error-representation-rfc9457.md, docs/adr/0005-request-validation-and-contract-design.md, docs/adr/0009-consolidate-to-gateway-plan-insights.md

## Summary

Today a signed-in user is thrown back to `/auth` about once a day, when the access token expires.
With this feature the client swaps an expired access token for a new pair without the user
noticing, and each swap extends the session. Someone who uses Fireplace at least once every 30
days is never signed out. The access token gets shorter at the same time, down from 24 hours to
1 hour, so a leaked one is useful for less time.

Terms: **access token** and **refresh token** are the gateway's terms
(`services/api-gateway/SPECIFICATION.md`, Users & Identity). "Session" is used here in its plain
sense, the stretch during which the browser holds usable tokens. No server-side session exists,
before or after this feature.

## Background: what ships today

- `docker-compose.yml:143-144` hard-codes `ACCESS_TOKEN_TTL: 24h` and `REFRESH_TOKEN_TTL: 168h`.
  No other compose file or deploy file sets them. The gateway spec's claim that the deployed
  `.env` sets them is wrong.
- `services/api-gateway/config/routes.go:59-61` defaults to 24h / 168h, and its comment says these
  match the `.env`. The consts in `internal/gateway/auth/service.go` default to 1h / 168h. The two
  disagree.
- `client/src/context/AuthContext.tsx` stores `refreshToken` in localStorage on sign-in. Nothing
  reads it. Users signed in today therefore already hold a refresh token, so they pick up this
  feature with no new sign-in, provided that token has not expired.
- Two client HTTP paths exist, and both wipe tokens and redirect to `/auth` on any 401: the
  middleware in `client/src/api/client.ts` and `authFetch` in `client/src/services/api.ts`.
  `authFetch` has no call sites in the current tree, so the generated client is the path in real
  use. `authFetch` stays and is covered anyway, so it cannot become a second behaviour if it gets
  a caller.
- `Service.Refresh` exists in `internal/gateway/auth/service.go`, but `LocalClient` has no refresh
  method and `typed_users.go` registers no operation, so no HTTP route reaches it. The old
  `[x] ... → FS-none` thin line was false on the HTTP surface. This feature repoints it here,
  unchecked.
- `Service.Refresh` wraps a repository not-found as a generic error, where `SignIn` maps it to
  `ErrUnauthorized`.
- `authenticate` in `services/api-gateway/internal/auth/middleware.go` never checks `tokenType`,
  so a refresh token passes as a bearer. It is the only bearer verifier in the repo: `ParseToken`
  has no other caller outside `common/auth`. Comments and spec prose saying plan-service and
  insights-service verify independently are out of date on this point.
  `commonauth.ValidateRefreshToken` has no access-side equivalent.
- openapi-fetch 0.13.8 takes a custom `fetch` in `createClient`, calls it as
  `fetch(request, requestInitExt)`, and captures its default `fetch` when the client is created.

## Requirements

### Gateway: the exchange

1. A public operation `POST /api/users/refresh` exchanges a refresh token for a new access and
   refresh pair. It is serialized (huma) next to signin in `typed_users.go`, declares no
   `Security` and runs no auth middleware.
2. A success returns the same `AuthResponse` as signin, including the current user in `userInfo`
   and the nanosecond expiry fields, unchanged.
3. An invalid, expired or wrong-type token returns `401 · UNAUTHENTICATED`. An access token sent
   as the refresh token is a wrong-type token. An empty string is not a shape failure: it reaches
   the token check and gets the 401.
4. A valid refresh token whose user no longer exists returns `401 · UNAUTHENTICATED`, not 404,
   mirroring `SignIn`. The difference between these cases is never exposed.
5. Every refresh issues a new pair with a full refresh lifetime, so the window slides. No check
   caps how long a chain of refreshes can run.
6. The contract change is additive. `openapi.yaml` and the generated client types are regenerated
   per `docs/agents/contract.md`, and `make -C services/api-gateway gates` passes.

### Gateway: bearer type

7. The gateway rejects a refresh token used as a bearer, with the same 401 that any other invalid
   token gets, on both renderers (legacy `{statusCode, message}` and problem+json). The reason
   stays server-side.
8. The check lives in a new access-side validator next to `ValidateRefreshToken` in `common/auth`,
   and `authenticate` uses it. A token with a missing `tokenType` claim is rejected too.
   `GenerateJWT` sets the claim on every token it issues.

### Token lifetimes

9. Refresh TTL is 30 days (720h), so a holiday does not sign anyone out.
10. Access TTL is 1 hour, down from 24, because refresh makes the expiry invisible.
11. `docker-compose.yml`, the `config/routes.go` defaults and the `service.go` consts all say
    1h / 720h, so all three sources agree and the misleading `routes.go` comment goes.
12. The access TTL drop must not reach production ahead of the client seam. Against a client that
    cannot refresh, a 1-hour token means being signed out every hour, worse than today. The client
    may deploy first: see R24. The client deploys separately from the gateway's compose stack, so
    this is an ordering rule for the deploy, not only for the merge.

### Client: one session-aware fetch

13. One function in a new module (for example `client/src/api/session.ts`) owns attaching the
    bearer, refreshing and signing out. Both HTTP paths use it: `createClient` receives it as its
    `fetch` option, and `authFetch` delegates to it. The `api.use` middleware and `authFetch`'s own
    token and 401 logic are removed.
14. The function accepts both call shapes it receives: openapi-fetch's `(request, init)` and
    `authFetch`'s `(url, options)`.
15. It attaches the stored access token as the bearer when one is stored.
16. On a 401 for a request that carried a bearer, it refreshes once and retries the request once
    with the new access token. The retry's response goes to the caller, whatever it is.
17. A request's body must be resendable: the request is cloned before the first send. Method,
    headers, body and abort signal all survive into the retry.
18. The refresh call goes through a generated-client instance that does not use this fetch, which
    prevents recursion and honours the rule against hand-written fetch to serialized endpoints.
    That instance is the one stated exception to `client.ts`'s "only import path" comment, and the
    comment says so.

### Client: concurrency

19. Concurrent 401s in a tab share one refresh exchange, and each retries once it settles.
20. A 401 for a request whose bearer is no longer the stored access token does not start a refresh.
    Another exchange has already replaced the token (in this tab or another), so the request
    retries with the stored one. This covers a 401 that arrives just after the shared exchange
    settled. It still counts as that request's one retry.
21. The exchange belongs to no single caller: aborting the request that triggered it does not abort
    the exchange for the others sharing it.
22. Across tabs there is no coordination. Tokens are stateless and old ones stay valid until they
    expire, so two tabs may each refresh, and the last write to localStorage wins.

### Client: the sign-out rule

23. Tokens are cleared and the browser redirects to `/auth` only when the refresh is **rejected**
    (a 401 from refresh, or no refresh token stored) or when the retried request is still 401.
24. A network error, a 5xx or any other non-401 from refresh keeps the tokens, does not redirect,
    and hands the original 401 response to the caller. This includes a 404 from a gateway that does
    not have the endpoint yet, which is what makes it safe for the client to deploy first.
25. On a sign-out `authFetch` still throws `Error("Unauthorized")` after redirecting. On the
    keep-tokens path it returns the 401 response.
26. A request with no bearer attached does not attempt a refresh and keeps today's behaviour on a
    401: tokens cleared, redirect to `/auth`. Nothing else about sign-in errors changes.

### Client: what a refresh writes

27. A successful refresh writes `accessToken` and `refreshToken` to localStorage. The stored `user`
    is left as it is, even though the response carries one.
28. A refresh result is written only if the stored `user` still has the id of the response's
    `userInfo`. Otherwise the user signed out, or signed in as someone else, while the exchange was
    in flight: the result is discarded, nothing is written, there is no redirect, and the original
    401 goes to the caller. Without this rule a late response would restore a session the user just
    ended, or put one account's tokens under another account's `user`.
29. Refresh is reactive only. There is no timer. One extra round trip per hour of use is accepted.

## User Stories

1. As a daily user, I want to stay signed in across days, so that I stop retyping my password.
2. As a user back from three weeks away, I want to find myself still signed in, so that a holiday
   costs me nothing.
3. As a user away for more than 30 days, I want to be asked to sign in again, so that an abandoned
   browser does not stay signed in forever.
4. As a user in the middle of saving an item when my access token expires, I want the save to go
   through, so that I never lose what I typed or notice the expiry.
5. As a user whose page fires several requests at once after the token expired, I want all of them
   to succeed, so that the page loads whole.
6. As a user with two tabs open, I want both to keep working when the token expires, so that I do
   not have to think about tabs.
7. As a user on a flaky connection, I want a failed refresh to leave me signed in, so that a blip
   does not throw me out and I can carry on once it clears.
8. As a user during a gateway outage or restart, I want to keep my session, so that the outage does
   not also cost me a sign-in.
9. As a user whose refresh token really is dead, I want to land on `/auth`, so that I know to sign
   in rather than staring at failures.
10. As a user whose account was deleted, I want my next refresh refused, so that I do not keep a
    working session on an account that is gone.
11. As a user who clicks sign out while a refresh is in flight, I want to stay signed out, so that
    signing out means what it says.
12. As a user already signed in when this ships, I want it to work with the tokens I hold, so that
    the fix does not itself sign me out.
13. As a visitor with no tokens, I want the app to behave as it does today, so that this change
    costs me nothing.
14. As the owner, I want a leaked access token to stop working within the hour, so that the damage
    from a leak is limited.
15. As the owner, I want a refresh token refused as a bearer, so that the long-lived token cannot
    be used to call the API directly.
16. As the owner, I want token lifetimes in one consistent set of values, so that I can read what
    the system does without comparing three files.
17. As the owner, I want the spec to tell the truth about refresh and about where TTLs are set, so
    that the next reader is not misled.
18. As a client developer, I want refresh on the generated client, so that a contract change breaks
    at `tsc` rather than at runtime.
19. As a client developer, I want one function to own token handling, so that the two HTTP paths
    cannot drift apart.
20. As a client developer, I want the refresh endpoint in `openapi.yaml`, so that it is governed by
    the same gates as every other operation.

## Acceptance Criteria

Gateway:

- [ ] `POST /api/users/refresh` with a valid refresh token and no `Authorization` header returns
      200 and an `AuthResponse`, and the new access token works as a bearer.
- [ ] The new refresh token has a full lifetime from the moment of the exchange, and can itself be
      exchanged.
- [ ] A garbage token, an expired refresh token, an access token and an empty string each return
      `401 · UNAUTHENTICATED`.
- [ ] A valid refresh token for a deleted user returns `401 · UNAUTHENTICATED`.
- [ ] A missing `refreshToken` or an unknown body member returns `422 · VALIDATION_FAILED`.
- [ ] A repository failure during refresh returns 500, not 401, so an outage is never mistaken for
      a rejected token.
- [ ] A refresh token used as a bearer returns 401 on a protected route, in both renderers; an
      access token still passes.
- [ ] `TestMount_PublicSurfacesUnaffected` covers refresh as a public operation, and its comment no
      longer says signin and signup are the only public ones.
- [ ] `openapi.yaml` and `client/src/api/generated/schema.d.ts` are regenerated, and
      `make -C services/api-gateway gates` passes with no allowlist entry.
- [ ] Compose, `config/routes.go` and `service.go` all say 1h / 720h, and the `routes.go` comment no
      longer mentions a `.env`.
- [ ] A fresh sign-in against the compose stack issues a 1-hour access token and a 30-day refresh
      token.
- [ ] `services/api-gateway/SPECIFICATION.md` tells the truth: the JWT lifecycle prose says compose
      sets the TTLs and gives the new values, the REST catalog has the refresh row, and the edge
      auth 401 list includes a wrong-type token.

Client:

- [ ] A request with an expired access token and a valid refresh token resolves with the retried
      response. The caller sees no error, and localStorage holds the new pair.
- [ ] A retried POST, PATCH or DELETE resends the identical body.
- [ ] N concurrent 401s produce exactly one refresh call and N retries, and all resolve.
- [ ] A 401 arriving after the shared exchange settled, or after another tab replaced the token,
      produces no second refresh and retries with the stored token.
- [ ] A 401 from refresh clears tokens and redirects to `/auth`.
- [ ] No stored refresh token clears tokens and redirects to `/auth`.
- [ ] A retried request that is still 401 clears tokens and redirects, with no second refresh and
      no second retry.
- [ ] A network error, a 500, a 503 and a 404 from refresh each keep all three localStorage keys,
      do not redirect, and return the original 401 to the caller.
- [ ] A 401 with no bearer attached makes no refresh call and clears and redirects as today.
- [ ] After a refresh the stored `user` is byte-identical to before.
- [ ] Signing out while the exchange is in flight leaves localStorage empty and does not redirect.
- [ ] Signing in as another user while the exchange is in flight leaves that user's tokens in place.
- [ ] Aborting the request that started the exchange does not fail other requests sharing it.
- [ ] `authFetch` and the generated client behave identically across the cases above, apart from
      `authFetch` throwing on sign-out (R25).
- [ ] `client.ts` has no `api.use` token or 401 logic left, and `authFetch` has no token or 401
      logic left.
- [ ] Verified once in the running app: with a tampered or expired `accessToken` in localStorage,
      the next action succeeds with no redirect and the stored tokens change.

## Edge States

**Expired**

- Access token expired, refresh token valid: the normal case. Invisible to the user.
- Both expired (away more than 30 days): refresh returns 401, sign-out, `/auth`.
- A refresh token issued before this feature, with its 168h lifetime: still valid until it expires,
  and exchanging it yields a 30-day one. No migration is needed.
- An access token issued before this feature, with its 24h lifetime: valid until it expires, then
  refreshed like any other.

**Concurrent**

- Several 401s in a tab: one exchange (R19).
- A 401 landing just after the exchange settled, or after another tab refreshed: retries with the
  stored token and makes no new exchange (R20). If that retry is still 401, the sign-out rule
  applies.
- Two tabs refreshing at once: both exchanges succeed because nothing is revoked, and the last
  write wins. Whichever pair survives is valid.
- One tab signs out while another stays open: the other tab's next request carries no bearer, so
  it follows R26 and lands on `/auth`.
- Sign-out or a switch of account during an exchange: the result is discarded (R28).

**Failed or disconnected**

- Refresh fails with a network error, 5xx or 404: tokens are kept, there is no redirect, and the
  caller gets the original 401 (R24). The next request tries again, because no failure is remembered.
- The typed client turns that 401 into `ApiError` with code `UNAUTHENTICATED`, whose message says
  "Your session has expired. Please sign in again." That wording is wrong for this one transient
  case. It is accepted for now and the message map is not changed here.
- Refresh succeeds and the retry hits a network error: the caller sees the fetch rejection it would
  see for any request. The new tokens stay stored.
- Refresh succeeds and the retry returns any non-401 error: that response goes to the caller. Tokens
  are kept.
- The caller aborts while waiting: its own request rejects with the abort. The shared exchange
  continues and still stores its result (R21).

**Unauthorized**

- Deleted user: refresh returns 401, sign-out, `/auth` (R4).
- `JWT_SECRET` rotated: every token fails, refresh returns 401, and everyone is signed out. This is
  the only global kill switch.
- A request with a bearer that still gets 401 with fresh tokens: sign-out after one retry, so no
  loop is possible.
- Sign-in with a wrong password while stale tokens are stored: a bearer was attached, so a refresh
  runs and the retry is still 401, which ends in sign-out and `/auth`. That is the same landing as
  today, with one extra round trip.
- Sign-in with a wrong password and no tokens stored: R26, unchanged, including today's redirect.

**Empty**

- A bearer stored but no refresh token: sign-out at the first 401 (R23).
- No stored `user` when a refresh returns: discarded under R28. `AuthContext` already treats a
  token with no stored user as signed out.

**Deploy order**

- Client first: 401s get a 404 from refresh, tokens are kept, and the caller sees the 401 until the
  gateway ships. No one is signed out by this, though requests made with an expired token fail
  until then.
- Gateway first: harmless as long as the access TTL has not dropped (R12).

## API surface

| Op | Method + Path | Query/Params | Request body | Response | Errors |
|----|---------------|--------------|--------------|----------|--------|
| `refresh` | `POST /api/users/refresh` | none. Public: no `Security`, no middleware | `refreshToken`: string, required. No other members | `200` · `AuthResponse`, same as signin | `401 · UNAUTHENTICATED` (invalid, expired, wrong type, deleted user); `422 · VALIDATION_FAILED` (missing field, unknown member); `500 · INTERNAL_ERROR` (repository or signing failure); `503 · SERVICE_UNAVAILABLE` (declared to match signin) |

As with signin, a documented 401 here does not mean a token is required in the header. Its summary
says so.

## Out of Scope

- **Refresh-token revocation.** The existing open line stays open. Be clear about what this means:
  refresh tokens are stateless and nothing is stored server-side, so a **stolen refresh token can
  keep renewing itself indefinitely**, because rolling has no hard cap, and each renewal yields a
  fresh 30 days. Sign-out only clears the browser that clicked it and does not invalidate the
  token. Rotating `JWT_SECRET` is the only global kill switch, and it signs out everyone. This
  exposure already exists in a smaller form for the life of one refresh token. This feature
  lengthens it, knowingly.
- Password change and reset.
- Moving tokens to httpOnly cookies. Tokens stay in localStorage, with the XSS exposure that has.
- An absolute session cap.
- Proactive refresh, timers and refresh on page load.
- Rate limiting on the refresh endpoint.
- Changing what `/auth` shows on sign-out, or any other sign-in error handling, including the
  redirect a wrong password causes today.
- The `UNAUTHENTICATED` message wording.
- The nanosecond expiry fields.
- Removing `authFetch`, even though it has no call sites.
