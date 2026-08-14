---
id: I-0018
status: open
implements: FS-0004
blocked_by: [I-0015]
labels: [ready-for-agent]
---
Implements FS-0004 §Requirements, §API surface, §Edge States

## What to Build

Serialize the **notes** group — 6 operations — and cut the frontend over in the same slice.

`GET /plans/{id}/notes` (filterable) · `GET .../{noteId}` · `POST` · `PATCH .../{noteId}` ·
`DELETE .../{noteId}` · `POST .../generate-ai`

Notes is a **gateway-local domain** — it is backed by the gateway's own `notes` table, not a
downstream service. So the transport mirror is transcribed from `internal/notes/model.go`
rather than from a proto, and R5 applies with force: `models.Note` must not leak into
`components.schemas`.

Traps specific to this group:

- **`tags` is read with `QueryArray`** — `?tags=a&tags=b` is a two-element filter today. The
  typed param must be a slice, or this silently collapses to the last value. This is a
  behavior regression that no gate would catch.
- Filters are camelCase (`isRead`, `isDismissed`, `relatedTaskId`) while `insights` uses
  snake_case. Both are transcribed as they are (R6) — do not normalize.
- `isRead` / `isDismissed` are parsed from strings today; an absent filter is distinct from
  `false`. Preserve that distinction in the typed param.
- `ai_metadata` is JSONB. Declare a shape for it or declare it explicitly free-form — do not
  let it become an untyped `any` by accident.

## Acceptance Criteria

- [ ] All 6 operations appear in `openapi.yaml` with correct methods, paths, and params
- [ ] `tags` is a repeatable query param and `?tags=a&tags=b` still filters on both
- [ ] Absent `isRead`/`isDismissed` remains distinct from `false`, proven by a test
- [ ] All 6 declare `Security: bearerAuth`
- [ ] `models.Note` does not appear in `components.schemas`, directly or nested
- [ ] Round-trip test over a **populated** fixture asserts JSON equality
- [ ] Empty list responses marshal to `[]` and never `null`
- [ ] Before/after probe recorded showing the envelope removed, payload otherwise unchanged
- [ ] `make openapi-diff`, `make lint-contract`, `make openapi-breaking` green
- [ ] FE notes call sites cut over to the generated client; typecheck green
- [ ] Full Go test suite green

## Blocked By

I-0015 (shared transport package and registration path)

## Spec Reference

FS-0004 §Requirements R1–R8, R14, R15, R19 · §API surface (notes) · §Edge States

## TDD Approach

- RED: test asserting `?tags=a&tags=b` reaches the service as a two-element filter
- GREEN: declare the typed param as a slice
- RED: test asserting an absent `isRead` filter is not treated as `false`
- GREEN: use a pointer or explicit-presence param type
