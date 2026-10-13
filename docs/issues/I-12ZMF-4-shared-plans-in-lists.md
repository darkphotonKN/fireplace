---
id: I-12ZMF-4
status: done
implements: FS-12ZMF
blocked_by: [I-12ZMF-2]
labels: [feature]
title: "FS-12ZMF slice 4: shared plans appear in the sidebar, My Plans and home, marked, and a lost plan says so"
---
Implements FS-12ZMF §Requirements (R22, R24, R26, R27, R32), §API surface (listPlans change, `ListPlans.include_shared`)

## What to Build

Members find shared plans where they find their own, and a plan they can't reach says so plainly.

**plan-service.** `ListPlansRequest` gains `include_shared` (bool, default false). When it's true,
return owned ∪ shared:
- deduplicated (`UNION`, not `UNION ALL`);
- ordered by `created_at` desc;
- unpaged.

When it's false, behaviour is exactly as today (`CascadeDeleteForUser` uses the repo's
`ListByUser`; don't change that).

**Gateway.** `listPlans` (`GET /api/plans`) sends `include_shared: true` and updates its
description to "every plan the caller owns or is a member of". `relation` is already derived per
row (slice 2). `/api/plans/shared` and search are unchanged (R24, D17). Regenerate the contract.

**Client.**
- `Sidebar.tsx`, `app/myplans/page.tsx` and home (`useHomeData.ts` → `Dashboard` / `OtherPlans`)
  keep calling `listPlans()`.
- Each renders a small **shared** mark on `relation === "member"` plans. Owned plans get no mark.
- My Plans hides the delete control for member plans (R27).
- Plan page (`app/plan/[planId]/page.tsx`): when `getPlan` rejects with status 404, render "This
  plan isn't here" and a link to `/`, in place of "Failed to load plan data". Other errors keep
  today's message (R32).
- Don't add redirects for 404s on item writes; the existing revert-and-error path stands.

## Acceptance Criteria

- [ ] `GET /api/plans` for member M returns M's own plans and the plan shared with M, each once,
      newest first, with the right `relation`.
- [ ] `ListPlans` with `include_shared` unset returns owned plans only (existing tests still pass).
- [ ] The sidebar, My Plans and home show the shared plan with a "shared" mark. Owned plans show
      no mark.
- [ ] My Plans shows no delete control on the shared plan and still shows it on owned ones.
- [ ] Opening `/plan/{id}` as a stranger or removed member shows "This plan isn't here" with a
      link home. A 500 still shows the existing error.
- [ ] Home's capped fan-out (ADR-0013) is unchanged: shared plans count toward the same cap.
- [ ] Gates pass, and the contract is regenerated.
- [ ] Tests pass.

## Blocked By

I-12ZMF-2, for `relation` on `PlanResp` and the 404 for no relation. Doesn't need I-12ZMF-3:
tests seed `plan_shares` directly. It can run alongside I-12ZMF-3.

## Spec Reference

FS-12ZMF §Requirements R22, R24, R26, R27, R32; §API surface (listPlans, `ListPlans`
`include_shared`); §Edge States (member removed while viewing, owner deletes plan, member
searches); decisions D11, D14, D15, D17. User stories 8, 9, 16, 17.

Files expected:
- `common/api/proto/plan/plan.proto`, plus the regenerated `*.pb.go`
- `services/plan-service/internal/plan/{repository,service,handler}.go`, plus tests
- `services/api-gateway/internal/gateway/plan/{client.go,typed_plans.go}`, plus tests
- `services/api-gateway/openapi.yaml`, `client/src/api/generated/*`
- `client/src/components/Sidebar.tsx`
- `client/src/app/myplans/page.tsx`
- `client/src/components/home/{Dashboard.tsx,OtherPlans.tsx}` (and tests)
- `client/src/app/plan/[planId]/page.tsx`

## TDD Approach

- RED: a plan repo test with one owned plan and one shared plan, where `ListPlans` with
  `include_shared` expects 2 and without it expects 1.
- RED (client): Sidebar renders a "shared" mark for a `relation: "member"` plan and none for
  `"owner"`.
