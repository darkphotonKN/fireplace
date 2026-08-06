---
id: I-0002
status: open
implements: FS-0002
blocked_by: []
labels: [enhancement]
title: FS-0002 slice 1: Huma mount, identity bridge, public doc surface
migrated_from: github#47
---
Implements FS-0002 §Requirements, §API surface

## What to Build

The Huma mount and its supporting seams — no operations registered yet. This slice proves
the serialized surface can coexist with everything already running.

- Bump dependencies: `huma v2`, which forces **gin v1.10.0 → v1.12.0** and the **Go
  directive to 1.25.0** (MVS floor, confirmed from the module graph — not optional while
  huma is a dependency).
- Mount a Huma API on the **existing protected group** via `humagin.NewWithGroup`, so
  operations inherit edge auth without changing how the group is built.
- **Identity bridge middleware.** `humagin`'s `Context()` returns `c.Request.Context()`, not
  the `*gin.Context`, so `c.Set("userId", …)` is invisible to typed handlers. Copy the
  authenticated identity into the request context after authentication.
- **Doc surface on a PUBLIC group** — Huma's docs UI and the OpenAPI document mount outside
  the protected group, mirroring the reachability `/swagger/index.html` has today.

## Gotchas (found in the spike — do not rediscover)

- **Huma's config paths are relative to the group they mount on.** `DocsPath: "/api/docs"` on
  a group based at `/api` yields `/api/api/docs`. This is why the doc surface and the
  operations mount on different groups.
- The identity bridge is the **single** identity seam for serialized routes. When ADR-0001's
  metadata-and-context identity lands, it converges here — do **not** create a second path.

## Acceptance Criteria

- [ ] The full api-gateway suite passes on gin 1.12 / Go 1.25 with no code changes elsewhere.
- [ ] A Huma API is mounted on the protected group; no operations registered yet.
- [ ] The identity bridge places the authenticated user in the request context, verified by
      a test that reads it from `context.Context`.
- [ ] `GET /api/docs` and the OpenAPI document are reachable **without** a token.
- [ ] `/swagger/index.html` still serves and still describes the legacy endpoints.
- [ ] Every legacy protected route behaves byte-identically (spot-check `GET /api/plans`
      with and without a token).
- [ ] Public routes (`POST /api/users/signin`) still public.

## Blocked By

None.

## Spec Reference

FS-0002 §Requirements 1, 18, 19, 21 · §API surface (doc surface note). Background:
`docs/notes/contract-pioneer-log.md` (2026-08-04 spike) proved this mount against the real
`auth.AuthMiddleware`.

## TDD Approach

- RED: a test asserting a Huma-registered probe route inherits `AuthMiddleware` (401 without
  a token) while `/swagger` and a legacy sibling route are unaffected.
- GREEN: `humagin.NewWithGroup(engine, protected, cfg)` + the bridge middleware.

## Not proven yet

The spike used a shape-equivalent harness, **not** the real `SetupRouter` (which needs a DB
and Consul). Wiring into the real `SetupRouter` is the first thing this slice should do.
