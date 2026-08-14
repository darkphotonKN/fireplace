---
id: I-0017
status: done
implements: FS-0004
blocked_by: [I-0016]
labels: [ready-for-agent]
---

> **All criteria met**, including the R14 probe — see `docs/notes/fs0004-envelope-probe.md`.
>
> **The `OptUUID`/`OptDate` landmine was real in two separate ways**, both verified by probe
> before any fix was written: with their `example` tag huma PANICS at registration; without it
> huma publishes `{Present, Valid, Value}` as a required object, from which a generated client
> would send an object where the wire has always carried a string. Fixed with
> `huma.SchemaProvider`, and the three states were then proved end to end against a live
> gateway — a document cannot prove them, because JSON Schema has no way to express "omitted".
>
> **A second live frontend bug, firing constantly.** `UpdateChecklistItemResponse` declared
> `result: "success" | "failure"` while the API always returned the item object, so
> `response.result !== 'success'` was always true and SEVEN Todo.tsx call sites ran their
> failure branch on every successful update — reverting the optimistic change and showing
> "Failed to update task status". Deleted rather than corrected: each site already had a catch
> doing the identical revert.
>
> **One behaviour change, flagged not buried:** `?scope=bogus` is now rejected at the boundary
> with 422 instead of travelling to plan-service. Correct per ADR-0005, but a change. It leaves
> an asymmetry — the same value in a request body still travels — which is recorded as a
> decision rather than silently resolved.
>
> **A premise I had wrong, twice.** I planned pointer query params to keep absent distinct from
> empty: huma rejects pointer query params outright, AND the distinction never existed — the
> legacy handler forwarded a filter only when non-empty. I also predicted this slice would
> finally need the shared transport package; it does not, because plans and checklists are the
> same Go package, so no type name can collide.

Implements FS-0004 §Requirements, §API surface, §Edge States

## What to Build

Serialize the **checklists** group — 9 operations proxied to plan-service — and cut the
frontend over in the same slice.

`GET /plans/{id}/checklists` (+ `scope`, `type` filters) · `.../archived` · `.../upcoming` ·
`GET .../{checklist_id}` · `POST` · `PATCH .../{checklist_id}` ·
`PATCH .../{checklist_id}/dates` · `PATCH .../{checklist_id}/archive` ·
`DELETE .../{checklist_id}`

This is the largest group and carries the heaviest frontend change: `components/Todo.tsx` and
`app/plan/[planId]/page.tsx` are both checklist-driven and both currently call raw
`fetch`/`axios`. Expect the FE half to be comparable in size to the Go half.

Notes specific to this group:

- Two path params on most operations (`id` and `checklist_id`) — both uuid.
- `scope` and `type` are optional query filters, applied only when non-empty today.
- `createChecklist` returns 201; `deleteChecklist` returns 204 with no body.
- Blocked by I-0016 rather than I-0015 because both groups mirror plan-service types and will
  contend for the same shared schema names.

## Acceptance Criteria

- [x] All 9 operations appear in `openapi.yaml` with correct methods, paths, params, and
      status codes (201 on create, 204 on delete)
- [x] `scope` and `type` are optional and behave identically to the current handler
- [x] All 9 declare `Security: bearerAuth`
- [x] Transport mirrors transcribed field-for-field including `omitempty`; no proto message or
      `models.*` type in `components.schemas`
- [x] Round-trip test over a **populated** fixture asserts JSON equality
- [x] Empty list responses marshal to `[]` and never `null`
- [x] Before/after probe recorded showing the envelope removed, payload otherwise unchanged
- [x] `make openapi-diff`, `make lint-contract`, `make openapi-breaking` green
- [x] `Todo.tsx` and `app/plan/[planId]/page.tsx` call the generated client; typecheck green
- [x] Full Go test suite green

## Blocked By

I-0016 (shared plan-service transport mirrors)

## Spec Reference

FS-0004 §Requirements R1–R8, R14, R15, R19 · §API surface (checklists) · §Edge States

## TDD Approach

- RED: round-trip test for `ChecklistItemResponse` against a populated proto message
- GREEN: transcribe the mirror
- RED: test asserting `scope`/`type` absent produces the unfiltered request downstream
- GREEN: make the typed params optional and forward only when set
