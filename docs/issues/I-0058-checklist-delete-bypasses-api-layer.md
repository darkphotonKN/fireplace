---
id: I-0058
status: done
implements: FS-none
blocked_by: []
labels: [bug]
title: "Deleting a checklist item is broken: raw fetch, no auth header, wrong env var"
---
Implements FS-none — a defect in behavior that predates the spec system. `FS-none` is a legal
anchor (docs/specs/README.md); note that `develop`'s anchor gate expects FS-NNNN or ADR-NNNN,
so this may need to be picked up by hand.

## What's wrong

`client/src/components/Todo.tsx` deletes checklist items through a hand-written `fetch` in two
places — `deleteTodo` (:707, wired at :1875, :2091, :2104) and `handleDeleteConfirm` (:1373,
wired at :1783):

```
fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/plans/${planId}/checklists/${todoId}`,
      { method: 'DELETE', credentials: 'include' })
```

**Two independent defects, either of which alone breaks the call:**

1. **No `Authorization` header.** `client/src/api/client.ts` attaches
   `Bearer ${localStorage.accessToken}` to every request through the generated client. This
   call doesn't. The gateway reads identity from exactly one place —
   `services/api-gateway/internal/auth/middleware.go:33`, `c.GetHeader("Authorization")` —
   there is no cookie path, and the operation is registered with `Middlewares: mw, Security:
   secured` (`services/api-gateway/internal/gateway/plan/typed_checklists.go:310`). So
   `credentials: 'include'` ships cookies nothing reads. **401.**
2. **The env var does not exist.** It reads `NEXT_PUBLIC_API_URL`; the rest of the client uses
   `NEXT_PUBLIC_API_BASE_URL` (`client/src/config/environment.ts:6`). `NEXT_PUBLIC_API_URL`
   appears in exactly two places in the whole repo — these two calls — and is defined in no
   `.env`. The URL becomes the literal string `undefined/api/plans/...`, resolved against the
   page origin. **404**, before auth is even reached.

Either way `!response.ok` throws and the user sees "Failed to delete task".

**Why no test caught it.** Ten `Todo.*.test.tsx` files mock `deleteChecklistItem` — the API
function `Todo.tsx:8` imports and never calls. The suite mocks the correct path while the
component uses the broken one, so the mocks are green and the product is not.

**The correct path already exists,** twice over: `deleteChecklistItem` in
`client/src/api/checklists.ts:187`, and the delegating adapter at
`client/src/services/api.ts:259`. Nothing needed building.

## What to Build

- Replace both raw `fetch` calls with `deleteChecklistItem` (or the `services/api.ts` adapter,
  matching whichever idiom the surrounding call sites use).
- Delete the now-unused raw-fetch error handling that duplicates what the API layer does.
- A test that fails against the current code: assert the delete path actually calls the API
  layer, not that a mock was configured.
- Grep for any other hand-written `fetch` in components. At the time of writing these two are
  the only ones in `src/components` and `src/app`; keep it that way.

## Acceptance Criteria

- [ ] Both delete call sites go through the API layer.
- [ ] A deleted item sends an `Authorization: Bearer …` header.
- [ ] No reference to `NEXT_PUBLIC_API_URL` remains in the codebase.
- [ ] A test fails if a delete call stops reaching the API layer.
- [ ] Deleting an item records touched-today (FS-KSJFR R11 — free once the path is correct).
- [ ] No hand-written `fetch` remains in `src/components` or `src/app`.
- [ ] Tests pass.

## Blocked By

None.

## Spec Reference

FS-none. Related: FS-KSJFR R11–R12 (the touched-today stamp assumes every plan and item
mutation crosses the API layer; this is the one path that doesn't, and
`docs/issues/I-KSJFR-1-touched-today-stamp.md`'s delete criterion cannot be true until this is
fixed). ADR-0002 / `client/src/api/client.ts` state that hand-written fetch against a
serialized endpoint is a high code-review finding — this is one.

## TDD Approach

- RED: a test asserting the mocked `deleteChecklistItem` is called when the user confirms a
  delete. It fails today, because the component calls `fetch`.
- GREEN: swap both call sites.
