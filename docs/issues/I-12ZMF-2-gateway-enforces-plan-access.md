---
id: I-12ZMF-2
status: done
implements: FS-12ZMF
blocked_by: [I-12ZMF-1]
labels: [feature]
title: "FS-12ZMF slice 2: the gateway enforces plan access on every plan-scoped route, and plans carry relation"
---
Implements FS-12ZMF §Requirements (R4–R10, R14, R23), §API surface (changed operations, `PlanResp.relation`)

## What to Build

One access seam in the gateway, applied to every plan-scoped route (ADR-0015 §3–4).

**The seam.** A small type in `internal/gateway/plan/` (e.g. `access.go`) with one operation,
`Require(ctx, planID, userID, need) (Relation, error)`, where `need` is `Owner` or `Member`
(owner-or-member).
- `none` → an `ErrNotFound`-wrapped error (404 `NOT_FOUND`).
- `member` when `Owner` is needed → `ErrForbidden` (403 `FORBIDDEN`).
- An RPC failure propagates (503/500).
- A nil checker fails closed (R8).

Every route asks before it does anything else.

**Apply it (R4–R7):**
- `typed_plans.go`:
  - getPlan → Member;
  - updatePlan, toggleDailyReset, deletePlan → Owner. This ends the 200 silent no-op (R9).
  - Remove the FS-0005 caveat from getPlan's description and the file header.
- `typed_checklists.go`: every operation → Member.
- `internal/notes/typed_notes.go`: `authorize` uses the seam (Member) instead of
  `AssertPlanOwnership`. `authorizeNote`'s note-belongs-to-plan stage stays (R14). Strangers now
  get 404, not the undeclared 403.
- `internal/gateway/calendar/local.go`: `PlanGateway` swaps `AssertPlanOwnership` for the
  relation check (Member) (R10).
- `internal/insights/typed_insights.go`: all three operations → Member on `plan_id`, and add 404
  to their `Errors`.
- `config/api.go` / `routes.go`: wire the seam where `NotesOwnership` and the calendar's plan
  gateway are wired today.

**`relation` on `PlanResp` (R23).** A required `relation` enum `owner|member` on every plan
response:
- getPlan → from the seam's answer;
- create, update, toggle → `owner`;
- list and listShared → `owner` iff `userId == caller`, else `member`.

Regenerate `openapi.yaml` and the TS client. `GET /api/plans` stays owned-only in this slice;
slice 4 widens it.

## Acceptance Criteria

- [ ] Two-account test (owner O, stranger S): S gets 404 on GET / PATCH / toggle / DELETE plan,
      on every checklist route, on every notes route, on the calendar and on the three insights
      routes.
- [ ] With a share row seeded for member M: M gets 200 on getPlan with `relation: "member"`, and
      success on every checklist, notes, calendar and insights operation.
- [ ] M gets 403 on PATCH plan, toggle and DELETE plan, and the plan is unchanged.
- [ ] O sees no change on any route except `relation: "owner"` in plan bodies.
- [ ] A fake access checker that errors → the route fails and the downstream call is never made.
      A nil checker → 403.
- [ ] Notes and calendar no longer reference `AssertPlanOwnership`.
- [ ] The insights operations declare 404. `make -C services/api-gateway gates` passes (additive
      changes only). The generated client is regenerated.
- [ ] Tests pass.

## Blocked By

I-12ZMF-1, for `GetPlanAccess` and the plan-scoped item RPCs.

## Spec Reference

FS-12ZMF §Requirements R4–R10, R14, R23; §API surface "Changed operations"; §Edge States
(stranger, nonexistent plan, member on owner-only, plan-service down); ADR-0015 §3, §4. User
stories 7, 10–12, 18, 26.

Files expected:
- `services/api-gateway/internal/gateway/plan/{access.go (new),client.go,model.go,typed_plans.go,typed_checklists.go,adapter.go}`,
  plus tests
- `services/api-gateway/internal/notes/typed_notes.go`, `typed_notes_test.go`
- `services/api-gateway/internal/gateway/calendar/{local.go,typed_calendar_test.go}`
- `services/api-gateway/internal/insights/{typed_insights.go,typed_insights_test.go}`
- `services/api-gateway/config/{api.go,routes.go,api_test.go}`
- `services/api-gateway/openapi.yaml`, `client/src/api/generated/*`

## TDD Approach

- RED: a typed-plans test with a fake access checker answering MEMBER, where PATCH
  `/api/plans/{id}` expects 403 and the client's UpdatePlan is never called.
- GREEN: `Require(…, Owner)` at the top of the update handler.
