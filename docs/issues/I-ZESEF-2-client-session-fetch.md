---
id: I-ZESEF-2
status: done
implements: FS-ZESEF
blocked_by: [I-ZESEF-1]
labels: [bug]
title: "FS-ZESEF slice 2: one session-aware fetch on both client paths, and the 1-hour access token"
---
Implements FS-ZESEF §Requirements (R10–R29), §Edge States

## What to Build

An expired access token stops signing the user out. This is the slice the owner feels. It also
lowers the access TTL to 1 hour, because that is only safe once this seam exists (R12).

**The seam (R13–R18).** A new module, for example `client/src/api/session.ts`, exports one
session-aware fetch. `createClient` in `client.ts` takes it as its `fetch` option, and `authFetch`
in `services/api.ts` delegates to it. Remove the `api.use` middleware and `authFetch`'s own token
and 401 logic. `session.ts` must not import from `client.ts`, since `client.ts` imports it. It uses
`config` and the generated `paths` type directly.

It handles both call shapes by normalising to a `Request`, and clones that request before the first
send so the retry can resend the body (R17). On a 401 for a request that carried a bearer: refresh
once, retry once, return the retry's response whatever it is. Never read the original 401's body, so
it can still be handed to the caller.

**The refresh call (R18).** A second `createClient<paths>` instance local to `session.ts`, with no
custom session fetch. Update the "ONLY import path" comment in `client.ts` to name this as the one
exception, and its "mirror the legacy 401 handling" comment. Do not add refresh to `api/users.ts`:
it must not go through `api`.

**Single flight (R19–R21).** A module-level in-flight promise, cleared when it settles. Before
starting an exchange, compare the bearer the failed request carried with the stored access token; if
they differ, retry with the stored one and start nothing (R20). The exchange carries no caller's
abort signal (R21).

**Sign-out rule (R23–R26).** Clear and redirect only on a 401 from refresh, no stored refresh token,
or a retry that is still 401. Everything else from refresh keeps the tokens and returns the original
401: network error, 5xx, 404, any other status. A 401 on a request that carried no bearer keeps
today's clear and redirect, with no refresh call. `authFetch` keeps throwing `Error("Unauthorized")`
on sign-out only; on the keep-tokens path it returns the 401.

**What a refresh writes (R27, R28).** Only `accessToken` and `refreshToken`, and only if the stored
`user` still has the id of the response's `userInfo`. Otherwise discard, do not redirect, and return
the original 401. Never write `user`.

**Access TTL (R10–R12).** `ACCESS_TOKEN_TTL: 1h` in `docker-compose.yml` and a 1-hour default in
`config/routes.go`, and update the access value in the gateway's JWT lifecycle prose. The client
deploys separately from the compose stack: **deploy the client first, or at the same time as the
gateway**. Say so in the PR description.

**Testing note.** openapi-fetch captures `globalThis.fetch` when `createClient` runs, so a spy
installed later never sees the refresh call. Give the refresh instance a call-time wrapper, for
example `fetch: (r) => globalThis.fetch(r)`, so tests can stub one global. The existing API-module
tests mock `./client` and should not need changes.

Files expected to touch:
- `client/src/api/session.ts` (new)
- `client/src/api/session.test.ts` (new)
- `client/src/api/client.ts`
- `client/src/services/api.ts`
- `docker-compose.yml`
- `services/api-gateway/config/routes.go`
- `services/api-gateway/SPECIFICATION.md`

Not expected: `AuthContext.tsx`. Sign-in already stores everything this slice reads.

## Acceptance Criteria

- [ ] A request with an expired access token and a valid refresh token resolves with the retried
      response. The caller sees no error, and localStorage holds the new pair.
- [ ] A retried POST, PATCH or DELETE resends the identical body.
- [ ] N concurrent 401s produce exactly one refresh call and N retries, and all resolve.
- [ ] A 401 arriving after the shared exchange settled, or after another tab replaced the token,
      produces no second refresh and retries with the stored token.
- [ ] A 401 from refresh clears tokens and redirects to `/auth`.
- [ ] No stored refresh token clears tokens and redirects to `/auth`.
- [ ] A retried request that is still 401 clears tokens and redirects, with no second refresh and no
      second retry.
- [ ] A network error, a 500, a 503 and a 404 from refresh each keep all three localStorage keys, do
      not redirect, and return the original 401 to the caller with its body readable.
- [ ] A 401 with no bearer attached makes no refresh call and clears and redirects as today.
- [ ] After a refresh the stored `user` is byte-identical to before.
- [ ] Signing out while the exchange is in flight leaves localStorage empty and does not redirect.
- [ ] Signing in as another user while the exchange is in flight leaves that user's tokens in place.
- [ ] Aborting the request that started the exchange does not fail other requests sharing it.
- [ ] `authFetch` and the generated client behave identically across these cases, apart from
      `authFetch` throwing on sign-out.
- [ ] `client.ts` has no `api.use` token or 401 logic left, and `authFetch` has no token or 401 logic
      left. The `client.ts` comments are accurate.
- [ ] Compose, `config/routes.go` and `service.go` all say 1h / 720h, and the gateway spec prose says
      the same. A fresh sign-in against the compose stack issues a 1-hour access token and a 30-day
      refresh token.
- [ ] Verified once in the running app: with a tampered or expired `accessToken` in localStorage, the
      next action succeeds with no redirect and the stored tokens change.
- [ ] Tests pass: `npx vitest run` and `npx tsc --noEmit` in `client`, `go test ./config/...` in
      `services/api-gateway`.

## Blocked By

I-ZESEF-1

## Spec Reference

FS-ZESEF §Requirements R10–R29, §Edge States (Expired, Concurrent, Failed or disconnected, Empty,
Deploy order), §Acceptance Criteria (Client, and the compose and TTL lines of Gateway). User stories
1, 2, 4–9, 11–14, 18, 19.

## TDD Approach

- RED: with a stubbed fetch answering 401 then 200 and a stored refresh token, the session fetch
  resolves with the 200 and the new tokens are stored. Fails: the module does not exist.
- RED: a stubbed refresh failing with a network error keeps all three localStorage keys and returns
  the original 401.
- GREEN: the seam, then single flight, then the R28 identity check, then the wiring of both paths.
