---
id: I-0003
status: done
implements: FS-0002
blocked_by: [I-0002]
labels: [enhancement]
title: FS-0002 slice 2: error layer — common/errcode, apierr domain codes, problem+json model, 401 fork
migrated_from: github#48
---
Implements FS-0002 §Requirements, §Edge States, §API surface

## What to Build

The error layer. This is the **gateway-wide error boundary** — every serialized endpoint
afterwards depends on it, so it is built once, deliberately.

- **`common/errcode`** — the platform-wide SCREAMING_SNAKE vocabulary. Initial members:
  `UNAUTHENTICATED`, `VALIDATION_FAILED`, `NOT_FOUND`, `ALREADY_EXISTS`, `FORBIDDEN`,
  `INTERNAL_ERROR`, plus `PROFILE_NAME_EMPTY`.
- **`apierr.StatusFor` grows a domain-code dimension** — it returns `(int, string)` today.
  It must return a code alongside the status and client-safe message, without changing the
  status/message mapping for any existing caller.
- **Custom Huma error model** implementing `GetStatus()`, `Error()`, **and `ContentType()`**,
  carrying `code` and `errors[]` as RFC 9457 extension members, registered by overriding the
  `huma.NewError` var.
- **Problem-emitting auth variant mounted on the serialized group ONLY.** The existing
  `auth.AuthMiddleware` aborts with the legacy `{statusCode, message}` shape before any Huma
  handler runs; without this fork the contract would lie about its most common error.

## Critical gotcha

**`ContentType()` is a silent failure mode.** The stock `huma.ErrorModel`'s `ContentType`
method is what emits `application/problem+json`. A custom error model that omits it degrades
to `application/json` while still passing any test that asserts only on status and body.
Assert the header explicitly.

## Acceptance Criteria

- [ ] `common/errcode` exists with the seven initial members.
- [ ] `apierr.StatusFor` returns a domain code; existing status/message mappings are
      unchanged (regression test over the gRPC code table: NotFound→404, AlreadyExists→409,
      InvalidArgument→400, Unauthenticated→401, PermissionDenied→403, Internal/Unavailable→500).
- [ ] The custom error model serializes `type`, `title`, `status`, `detail`, `code`, and
      `errors[]`.
- [ ] Every error from a serialized route carries `Content-Type: application/problem+json`
      — asserted on the header, not inferred from the body.
- [ ] A request with no token to a serialized route returns `401` as **problem+json** with
      `code: UNAUTHENTICATED`.
- [ ] **A legacy protected route's 401 is byte-identical to today** — the fork did not leak.
- [ ] No downstream error text (e.g. auth-service's `"name cannot be empty"`) appears in any
      response body.

## Blocked By

I-0002 — the problem-emitting auth variant mounts on the serialized group created there.

## Spec Reference

FS-0002 §Requirements 13–17 · §Edge States (401 rows) · §API surface (Errors column).
Constraint: ADR-0004 (error representation), which amends ADR-0002's error clause.

## TDD Approach

- RED: table test asserting each gRPC code maps to the right `(status, code)` pair and that
  the response `Content-Type` is `application/problem+json`.
- GREEN: `errcode` package + `StatusFor` extension + the custom model and `NewError` override.
- RED: a test asserting a legacy route's 401 body is unchanged.

## Known limitation to encode, not fix

`errors[]` will be **empty for downstream validation failures** — gRPC statuses carry only a
string message, so field-level detail cannot be reconstructed from auth-service without
structured error details on plane 2 (out of scope). This is exactly why `PROFILE_NAME_EMPTY`
is a distinct code rather than `VALIDATION_FAILED` with a field pointer: with no `errors[]`
to carry the field, the code must carry the precision.
