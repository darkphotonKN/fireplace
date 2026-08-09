---
id: I-0004
status: done
implements: FS-0002
blocked_by: [I-0002, I-0003]
labels: [enhancement]
title: FS-0002 slice 3: GET /api/users/profile serialized (ProfileResponse, bare resource)
migrated_from: github#49
---
Implements FS-0002 §Requirements, §Acceptance Criteria, §API surface

## What to Build

The first genuinely serialized operation — `GET /api/users/profile` as a typed Huma handler.

- **`ProfileResponse` transport type**, declared for these operations (not shared), plus a
  mapping from the domain user. Publishes exactly: `id`, `email`, `name`, `displayName`,
  `bio`, `createdAt`, `updatedAt`.
- **Typed Huma handler** replacing the gin handler for this route only. Identity read from
  `context.Context` via the slice-1 bridge.
- **Bare resource response** — no `{statusCode, message, result}` envelope. This is a
  deliberate, grandfathered break for this endpoint.
- The operation appears in the generated `openapi.yaml`, which is committed.

## Why a dedicated type (do not shortcut this)

`models.User` carries `Password string` with `json:"password,omitempty"`. It is never
populated today, so it never ships — **but a schema generated from that struct would publish
a `password` field.** Serialization makes an invisible field visible. Per ADR-0003, storage
models are never serialized, *including nested inside a transport type* — a transport type
that embeds `models.User` compiles fine and silently violates the rule.

## Acceptance Criteria

- [ ] `GET /api/users/profile` returns `200` with a bare `ProfileResponse` — no envelope.
- [ ] The generated `openapi.yaml` contains the operation and **`password` appears nowhere
      in the document**.
- [ ] `models.User` does not appear in the generated schema, directly or nested.
- [ ] The handler resolves the caller from `context.Context`, not `*gin.Context`.
- [ ] A valid token whose user row was deleted returns `404` problem+json, `code: NOT_FOUND`.
- [ ] `auth-service` unreachable returns `500` problem+json, `code: INTERNAL_ERROR`; the wire
      message is logged, never returned.
- [ ] No token returns `401` problem+json, `code: UNAUTHENTICATED`.

## Gate-dependent (blocked on I-0001)

- [ ] Regenerating the spec produces no diff against the committed `openapi.yaml`.
- [ ] Spectral passes: description, example, and error schemas present on the operation.
- [ ] `oasdiff` reports the envelope removal as breaking, and it is present in
      `.oasdiff.yaml` with a stated reason.

## Blocked By

I-0002, I-0003. Gate-dependent acceptance criteria additionally require I-0001.

## Spec Reference

FS-0002 §Requirements 5, 6, 18 · §API surface (`getProfile` row) · §Edge States.
Constraints: ADR-0003 (transport types), ADR-0004 (error representation).

## TDD Approach

- RED: request `GET /api/users/profile` with a valid token; assert a bare body with the seven
  fields and **no** `statusCode`/`message`/`result` keys.
- GREEN: `ProfileResponse` + mapping + typed handler registration.
- RED: assert the generated spec contains no `password` anywhere.
