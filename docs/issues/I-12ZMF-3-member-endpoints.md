---
id: I-12ZMF-3
status: done
implements: FS-12ZMF
blocked_by: [I-12ZMF-2]
labels: [feature]
title: "FS-12ZMF slice 3: owners add, list and remove plan members by email"
---
Implements FS-12ZMF §Requirements (R15–R21), §API surface (new resource: plan member; listPlanMembers, addPlanMember, removePlanMember; plan-service member RPCs; new error codes)

## What to Build

The three member endpoints, end to end, from plan-service through the gateway.

**plan-service.** Add `ListPlanMembers`, `AddPlanMember` and `RemovePlanMember` (shapes in
§API surface).
- `AddPlanMember`:
  - refuses `member_user_id == owner` with `ErrInvalidInput` (the `MEMBER_IS_OWNER` rule lives
    here, ADR-0005);
  - writes `role = 'edit'` via the existing `CreateShare` (`ON CONFLICT DO NOTHING`), then reads
    the row back so a re-add returns the original `since`;
  - is scoped to plans the caller owns (`NotFound` otherwise, defensive).
- `RemovePlanMember` deletes the (plan, member) share. No row → `NotFound`.
- `ListPlanMembers` returns `owner_id`, the plan's `created_at`, and the shares ordered by
  `created_at`.

**Gateway.** A new typed file, `internal/gateway/plan/typed_members.go`:
- `GET /api/plans/{id}/members` → Member access (slice 2's seam).
  - Hydrate each id with name (display name falling back to account name) and email from the
    gateway's users table, through a narrow lookup seam over `authgw` (a batch `WHERE id = ANY`
    is preferred over N `GetByID` calls).
  - Skip ids with no user (R16).
  - Owner first.
- `POST /api/plans/{id}/members` `{ email }`, strict, `format: email`, max 254 → Owner access.
  - Trim the email, then `GetByEmail`. No user → 400 `NO_ACCOUNT_FOR_EMAIL`.
  - Call `AddPlanMember`, and map its `InvalidArgument` to 400 `MEMBER_IS_OWNER` **at this
    operation**, not through `CodeFor`'s message sniffing.
  - Answer 201 with `PlanMemberResp`.
- `DELETE /api/plans/{id}/members/{userId}` → Owner access, then 204. Not a member → 404.

Add `NO_ACCOUNT_FOR_EMAIL` and `MEMBER_IS_OWNER` to `common/errcode` (additive). Register in
`config/api.go`. Regenerate `openapi.yaml` and the TS client.

## Acceptance Criteria

- [ ] Owner O adds existing account M by email → 201 `{id, name, email, relation: "member", since}`.
- [ ] Adding M again → 201 with the same `since`, and still one row.
- [ ] Unknown email → 400 `NO_ACCOUNT_FOR_EMAIL`. O's own email → 400 `MEMBER_IS_OWNER`.
- [ ] `"  m@x.com  "` resolves to M.
- [ ] `{"email":"nope"}`, `{}` and `{"email":"m@x.com","role":"view"}` → 422 `VALIDATION_FAILED`.
- [ ] M posts or deletes members → 403. Stranger S → 404 on all three routes.
- [ ] `GET` members as O and as M → the owner first, then members by add time.
- [ ] A share row for a deleted user is left out.
- [ ] O deletes M → 204. M's next `GET /api/plans/{id}` → 404. Deleting M again → 404. Deleting
      O's own id → 404.
- [ ] Problem responses carry `application/problem+json` and the codes above (assert the header
      in at least one 400 test).
- [ ] Gates pass, and `openapi.yaml` plus the generated client are regenerated.
- [ ] Tests pass.

## Blocked By

I-12ZMF-2, for the access seam (`Require`) and its 404/403 mapping.

## Spec Reference

FS-12ZMF §Requirements R15–R21; §API surface (plan member resource, the three operations,
plan-service member RPCs, new codes); §Edge States (self-share, unknown email, whitespace,
case, re-add, concurrent add, remove absent member); decisions D7, D8, D13, D16. User stories
1–6, 14, 22, 27.

Files expected:
- `common/api/proto/plan/plan.proto`, plus the regenerated `*.pb.go`
- `services/plan-service/internal/plan/{repository,service,handler,model}.go`, plus tests
- `services/api-gateway/internal/gateway/plan/{typed_members.go (new),client.go,model.go}`, plus
  tests
- `services/api-gateway/internal/gateway/auth/repository.go` (batch lookup by ids, if added)
- `common/errcode/*.go`
- `services/api-gateway/config/{api.go,routes.go}`
- `services/api-gateway/openapi.yaml`, `client/src/api/generated/*`

## TDD Approach

- RED: a typed-members test where POST with an email the fake user lookup doesn't know expects
  400 with code `NO_ACCOUNT_FOR_EMAIL`, and AddPlanMember is never called.
- GREEN: resolve the email before the RPC and map the miss.
