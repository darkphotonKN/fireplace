---
id: I-DF4ZB-3
status: done
implements: FS-DF4ZB
blocked_by: []
labels: [enhancement]
title: "FS-DF4ZB slice 3: domain codes cross gRPC as ErrorInfo; the seam maps ResourceExhausted and FailedPrecondition"
---
Implements FS-DF4ZB §Requirements (R52)

## What to Build

The error seam the new insights operations need, landed before any of them so each later slice
only adds its own code.

- **Producer side.** A helper in `common/grpcerror` that builds a gRPC status carrying a
  `google.rpc.ErrorInfo` detail whose `reason` is an `errcode.Code`. insights-service uses it in
  later slices for `BRIEF_DATE_OUT_OF_RANGE`, `BRIEF_REFRESH_LIMIT_REACHED` and
  `PARENT_ITEM_INVALID`.
- **Consumer side.** `services/api-gateway/internal/apierr/problem.go`'s single seam reads
  `ErrorInfo.reason` **before** the status-class fallback: a known code is used as-is.
- **New class mappings.** `ResourceExhausted → 429` and `FailedPrecondition → 400` (status and
  code fallback). Every existing mapping, including `Unavailable → 503`, is unchanged.
- **The "name" substring rule** stays for auth-service's existing case (it sends no detail) but
  can no longer capture a status that carries an `ErrorInfo`: an `InvalidArgument` from insights
  whose message contains "name" is not reported as `PROFILE_NAME_EMPTY`.

No new `errcode` constants here: each code is added by the slice whose failure needs it
(`docs/agents/contract.md`: never speculatively). No HTTP operation changes, so no
`openapi.yaml` regen is expected.

## Acceptance Criteria

- [ ] A status with `ErrorInfo.reason = X` maps to code `X`, whatever its status class.
- [ ] `ResourceExhausted` without detail maps to 429; `FailedPrecondition` without detail to 400.
- [ ] An `InvalidArgument` carrying an `ErrorInfo` and a message containing "name" is **not**
      `PROFILE_NAME_EMPTY`.
- [ ] auth-service's profile-name failure (no detail) still maps to `PROFILE_NAME_EMPTY`.
- [ ] Existing mappings (`NotFound`, `AlreadyExists`, `InvalidArgument`, `Unauthenticated`,
      `PermissionDenied`, `Unavailable → 503`) are unchanged.
- [ ] Tests pass.

## Blocked By

None.

## Expected write set

- `common/grpcerror/grpcerror.go` (+ test)
- `services/api-gateway/internal/apierr/problem.go`, `services/api-gateway/internal/apierr/apierr_test.go`

## Spec Reference

FS-DF4ZB §Requirements R52; §Background ("Error seam"); §Design notes ("Message-text matching
for new domain codes"); §Acceptance Criteria ("Domain codes reach the gateway as
`ErrorInfo.reason`…"); ADR-0004.

## TDD Approach

- RED: a table test in `apierr_test.go` with rows for ErrorInfo-carrying statuses (including an
  `InvalidArgument` whose message says "name"), `ResourceExhausted`, `FailedPrecondition`, and the
  existing rows unchanged.
- GREEN: detail lookup first, then the two new cases.
