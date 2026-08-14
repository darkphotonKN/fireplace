---
id: I-0017
status: open
implements: FS-0004
blocked_by: [I-0016]
labels: [ready-for-agent]
---
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

- [ ] All 9 operations appear in `openapi.yaml` with correct methods, paths, params, and
      status codes (201 on create, 204 on delete)
- [ ] `scope` and `type` are optional and behave identically to the current handler
- [ ] All 9 declare `Security: bearerAuth`
- [ ] Transport mirrors transcribed field-for-field including `omitempty`; no proto message or
      `models.*` type in `components.schemas`
- [ ] Round-trip test over a **populated** fixture asserts JSON equality
- [ ] Empty list responses marshal to `[]` and never `null`
- [ ] Before/after probe recorded showing the envelope removed, payload otherwise unchanged
- [ ] `make openapi-diff`, `make lint-contract`, `make openapi-breaking` green
- [ ] `Todo.tsx` and `app/plan/[planId]/page.tsx` call the generated client; typecheck green
- [ ] Full Go test suite green

## Blocked By

I-0016 (shared plan-service transport mirrors)

## Spec Reference

FS-0004 §Requirements R1–R8, R14, R15, R19 · §API surface (checklists) · §Edge States

## TDD Approach

- RED: round-trip test for `ChecklistItemResponse` against a populated proto message
- GREEN: transcribe the mirror
- RED: test asserting `scope`/`type` absent produces the unfiltered request downstream
- GREEN: make the typed params optional and forward only when set
