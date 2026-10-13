# FS-12ZMF: Share a plan with other accounts

> Status: work-order · SPECIFICATION.md: client `## Plans` "Share a plan with other accounts"; api-gateway `## Authorization` "Plan access for owners and members"; plan-service `### Plans` "Plan members and an owner-or-member access check" → this FS · Brownfield sibling: plan-service "Plan sharing, and listing owned ∪ shared-with-me" (FS-none: schema + list only, no way to create a share from the API) · Related FS: FS-0005 (per-user authorization: this FS closes its plans half and answers its open questions), FS-043WP (milestone URLs inherit plan access) · Related ADRs: docs/adr/0015-plan-access-is-a-relation.md (governing), docs/adr/0001-grpc-identity-via-metadata-rs256.md, docs/adr/0004-error-representation-rfc9457.md, docs/adr/0005-request-validation-and-contract-design.md, docs/adr/0007-user-edits-win.md, docs/adr/0013-home-composes-capped-fan-out.md

## Summary

A plan's owner can add other Fireplace accounts to it by email. Those **members** see the plan
in their sidebar, My Plans and home, marked as shared, and work in it as the owner does: items,
notes, the plan calendar. The plan itself (its name, daily reset, deletion and who is on it)
stays the owner's. Anyone with no relation to a plan gets "not found" everywhere, so a plan id
alone is no longer enough to read someone's plan.

The case this is for is two people keeping each other going on shared work (Barrowspire's
milestones are split "Nick ·" / "Kranti ·"). It is not a team handing out tickets. There are no
roles to pick, no invitations to accept and no assignment.

## Background: what ships today (measured 2026-10-13, from code)

- **plan-service** has `plan_shares(user_id, plan_id → plans ON DELETE CASCADE, role
  'edit'|'view' DEFAULT 'edit', created_at, updated_at, PK(user_id, plan_id))`, indexed both
  ways (migration 000015). `repository.CreateShare` exists (`ON CONFLICT DO NOTHING`) but no RPC
  reaches it. Nothing lists or removes members.
- `ListSharedPlans` (owned ∪ shared, `UNION ALL`, limit/offset, default 20) is served as
  `GET /api/plans/shared`. The client's `listSharedPlans()` in `client/src/api/plans.ts` has no
  caller. Sidebar, My Plans and home all call `listPlans()`, which is owned-only and unpaged.
- **No plan access is enforced** (FS-0005's probe). `GET /api/plans/{id}` returns any plan to
  any signed-in caller. `UpdatePlan` and `ToggleDailyReset` are scoped `WHERE id AND user_id`,
  so a non-owner's PATCH matches no row, then the service reloads the plan and answers 200: a
  silent no-op. `DeletePlan` 404s a non-owner.
- **Checklist item RPCs carry no plan at all.** `GetItem`, `UpdateItem`, `UpdateItemDates`,
  `ArchiveItem` and `DeleteItem` take only the item id. The gateway route
  `/api/plans/{id}/checklists/{checklist_id}` ignores `{id}`, so any caller with an item id can
  change it.
- **Notes** (gateway-local) and the **plan calendar** call `AssertPlanOwnership`, which is
  owner-only and answers a stranger with 403. Applied to plain reads, that check is what would
  break sharing, and it is why FS-0005 stalled. Notes already confirm a note belongs to the
  path's plan (`authorizeNote`).
- **Insights** (`/api/insights/*?plan_id=`) forward to insights-service, whose
  `GetPlanContext` comment says plan-service's `GetPlan` "enforces ownership". It does not, so
  these three routes are open to any plan id today.
- **Users live in the gateway's database** (`internal/gateway/auth/repository.go`:
  `GetByEmail`, exact match, the same lookup sign-in uses). plan-service never sees an email.
- The gateway publishes `user.deleted`. plan-service consumes it and deletes the user's own plans
  (`CascadeDeleteForUser`), but leaves their `plan_shares` rows. `plan_shares.user_id` has no FK
  because users live in another database.
- **Serialization:** every route this feature touches is already serialized (FS-0004, ADR-0006
  discharged ADR-0002's serialize-on-touch clause). So there is **no slice ⓪**: new and changed
  operations go straight into the typed surface and `openapi.yaml`.

## Decisions and why

Carried from the scoping session (2026-10-13). **Owner** decisions were made by the owner;
**charter defaults** were approved under the charter, each with its reason. Decisions marked
**write-a-spec** were left open by the notes and settled here with the smallest sensible v1.

**D1. Per plan, not per milestone** (owner). A milestone URL (FS-043WP) is a deep link into
something already shared, not its own permission.

**D2. Signed-in accounts only** (owner). Public links are parked (see Out of Scope).

**D3. Access is a relation: owner, or a `plan_shares` row** (charter default, recorded as
**ADR-0015**). plan-service answers *what is this user's relation to this plan* (`owner`,
`member` or `none`). The gateway asks on every plan-scoped route and decides per operation. Why:
the relation is plan data, so plan-service owns it. Which operation needs which relation is
transport policy, so the gateway owns it. Neither restates the other.

**D4. No relation → 404; member on an owner-only operation → 403** (charter default). 404 doesn't
leak that the plan exists. A member already knows the plan exists, so 403 tells them nothing new
and names the real reason. This answers FS-0005's "403 or 404" question. It also means strangers
on notes and calendar now get 404 where they got 403 before.

**D5. One role in v1: `edit`** (charter default). Members read and write items, notes and the
plan calendar. `view` stays a schema value only: no UI, and the API neither offers nor returns
it. Why: two people on shared work don't need a read-only tier, and every role is a branch in
every check.

**D6. Owner only: edit the plan, toggle daily reset, delete the plan, add and remove members**
(charter default). The plan's identity and lifecycle stay with one person.

**D7. Add by the email of an existing account** (charter default). An unknown email gets a plain
refusal, "No Fireplace account uses that email". **Accepted tradeoff:** the owner can learn
whether an email has an account. That's fine between friends and worth saying out loud in the
interview story. Self-share is refused. Re-adding an existing member is an idempotent success.

**D8. Removal is the owner's; no "leave plan"** (charter default). A removed member loses access
on their next request.

**D9. An item must belong to the path's plan** (charter default). Every item-by-id operation
checks the item's `plan_id` against the path plan. Otherwise a member of plan A could change
plan B's items by putting A's id in the path (IDOR). Mismatch → 404, the same as "no such item".

**D10. UI placement** (charter default). There's a **Sharing** section in the plan's existing
Settings view (`Todo.tsx`). The owner gets add-by-email, the member list and remove. A member
sees the owner and the other members, with no controls. Owner-only settings are hidden from
members.

**D11. Lists show owned ∪ shared, with a "shared" mark** (charter default). This covers the
sidebar, My Plans and home.

**D12. `user.deleted` also clears that user's shares** (charter default). Their own plans
already cascade.

**D13. Self-share and unknown email are 400 + domain code, not 422** (write-a-spec; this
**departs from the charter default's "422"**). ADR-0005 §1 is accepted and splits the two: 422
means the request is malformed, which the boundary decides from the type. 400 + code means the
domain refused a well-formed request. A well-formed email with no account behind it, or your
own email, is a domain refusal. A malformed email is still 422. The charter's intent (a plain
refusal the client can word exactly) is kept through the codes `NO_ACCOUNT_FOR_EMAIL` and
`MEMBER_IS_OWNER`.

**D14. Plan responses carry `relation`** (write-a-spec; the notes left it open). Every `PlanResp`
gains `relation: "owner" | "member"`, so the client picks controls without a second call. On a
single plan it comes from the access answer. In a list it's `owner` when `userId` is the caller,
otherwise `member`. That's not a second rule: the list only ever contains owned or shared plans.
Rejected alternative: the client compares `plan.userId` to "me". That needs the caller's id
threaded into every list and settings component, and it reads `userId` as a permission when
it's just data.

**D15. `GET /api/plans` becomes owned ∪ shared, and stays unpaged** (write-a-spec). The client's
three lists keep calling `listPlans()`. Switching them to `/api/plans/shared` would cap them at
20 (its default page) and add paging to three components for no gain at friends scale.
`/api/plans/shared` keeps its shape and paging and gains `relation`. plan-service's `ListPlans`
gains an `include_shared` flag that defaults off, so the RPC's meaning doesn't change under any
other caller.

**D16. Members see each other's emails** (write-a-spec; the notes' open question). The owner
added them by email, and the list is the only place to tell two people with the same name apart.
`name` is shown first.

**D17. Search stays owned-only** (write-a-spec). `/api/plans/search` is unchanged in v1. A member
finds shared plans in the list and sidebar, not by search.

**D18. Touched today is unaffected** (write-a-spec; the notes' open question). The touched-today
stamp (FS-KSJFR) is per-browser and written by the caller's own mutations. A member's edits mark
the member's day, never the owner's. Nothing to build.

**D19. Concurrent edits: last write wins, no realtime** (from the notes). ADR-0007 is about
generation versus the user, not user versus user. You see the other person's changes on refetch.

**D20. Daily items and daily reset are shared like everything else** (from the notes). Reset is
the owner's setting and applies to every member's view of the plan.

## Requirements

### Access relation (plan-service)

1. **R1.** plan-service answers a user's relation to a plan as exactly one of `owner`, `member`
   or `none`. `owner` means `plans.user_id` is the user. `member` means a `plan_shares` row
   exists for (user, plan). Anything else is `none`, **including a plan that does not exist**,
   so the answer never tells a stranger apart from a missing plan.
2. **R2.** The relation answer also carries the plan's owner id.
3. **R3.** A `plan_shares` row's `role` is irrelevant to the relation in v1. Any row makes a
   member (D5).

### Access enforcement (gateway)

4. **R4.** Every plan-scoped route asks plan-service for the caller's relation before doing
   anything else. A route is plan-scoped when it takes a plan id in its path or query: plans by
   id, checklists, notes, the plan calendar, the three insights routes, and the new members
   routes.
5. **R5.** Relation `none` → **404 `NOT_FOUND`**, whatever the operation.
6. **R6.** Relation `member` on an owner-only operation → **403 `FORBIDDEN`**. Owner-only
   operations are: update plan, toggle daily reset, delete plan, add member, remove member.
7. **R7.** Every other plan-scoped operation is allowed to `owner` and `member` alike: get plan,
   every checklist operation (including reorder and archive), every notes operation (including
   generate with AI), the plan calendar, the three insights routes, and list members.
8. **R8.** The check fails **closed**. If the gateway cannot get an answer (plan-service down),
   the request fails with the existing 503/500 mapping and never proceeds unchecked. A missing
   access checker is a 403, never an allow (the pattern notes already use).
9. **R9.** A non-owner's PATCH no longer answers 200 while changing nothing (FS-0005 decision 3).
   A member gets 403 and a stranger gets 404, both before plan-service is asked to update.
10. **R10.** The gateway stops calling `AssertPlanOwnership` from notes and calendar. Both use the
    relation check under R5–R7. The RPC itself stays, uncalled by the gateway.

### Item belongs to the path's plan

11. **R11.** Every checklist operation addressed by item id (get, update, update dates, archive,
    delete) passes the path plan id to plan-service. plan-service acts only on an item whose
    `plan_id` equals it. An item that exists in another plan is answered exactly like an item
    that doesn't exist: **404 `NOT_FOUND`**.
12. **R12.** plan-service refuses an item-by-id request that carries no plan id
    (`InvalidArgument`). The gateway is the only caller and ships with plan-service. A missing
    plan id is a caller bug, not a request to act unscoped.
13. **R13.** Create item, list items and reorder are already scoped by plan id. Their existing
    cross-plan guards are unchanged: a parent in another plan is 400, and an id outside the
    sibling set is 400 (FS-0009).
14. **R14.** Notes keep their existing note-belongs-to-plan check (`authorizeNote`), now behind
    the relation check instead of ownership.

### Members

15. **R15.** **List members:** the owner and every member may list a plan's people. The list is
    the owner first, then members in the order they were added. Each entry carries the user id,
    a display name (display name if set, otherwise account name), email, relation
    (`owner` | `member`), and `since` (the plan's creation time for the owner, the share's
    creation time for a member).
16. **R16.** A member whose account no longer exists (a share row not yet cleaned up, see R25) is
    left out of the list, not shown as blank.
17. **R17.** **Add member** (owner only): the request names an email. The gateway trims
    surrounding whitespace and resolves it the same way sign-in does (exact match on the stored
    email).
    - No account → **400 `NO_ACCOUNT_FOR_EMAIL`**. The detail tells the owner no Fireplace
      account uses that email.
    - The owner's own email → **400 `MEMBER_IS_OWNER`**, decided by plan-service (it owns the
      rule that the owner is not a member).
    - Already a member → **201** with the existing entry. Its `since` keeps the original add
      time, and no duplicate row is written.
    - Otherwise a share row is written with role `edit` and the new entry is returned (201).
18. **R18.** A malformed email (not an email, empty, over 254 characters) or an unknown body
    member → **422 `VALIDATION_FAILED`** at the boundary (ADR-0005).
19. **R19.** **Remove member** (owner only): deletes the share row for (plan, user) → 204. No such
    row (not a member, already removed, or the owner's own id) → **404 `NOT_FOUND`**.
20. **R20.** Removal takes effect on the removed member's next request (R5). There is no push and
    no session to revoke.
21. **R21.** No member cap in v1.

### Plans in lists, and the `relation` field

22. **R22.** `GET /api/plans` returns every plan the caller owns or is a member of, newest first
    (`createdAt` desc, as today), unpaged. Each plan appears once.
23. **R23.** Every plan response (list, get, create, update, toggle, the shared list) carries
    `relation`: `owner` or `member` (D14). It's never `none`, because a `none` plan is never
    returned.
24. **R24.** `GET /api/plans/shared` keeps its paging and shape and gains `relation`. Search is
    unchanged and owned-only (D17).

### Account deletion

25. **R25.** On `user.deleted`, plan-service deletes every `plan_shares` row whose `user_id` is
    that user, as well as the user's own plans (whose shares already cascade). This is
    idempotent: redelivery is a no-op. A failure follows the consumer's existing policy
    (transient → requeue, permanent → ack and log).

### Client

26. **R26.** The sidebar, My Plans and home show owned and shared plans together (they already
    call `listPlans()`). A plan with `relation: "member"` carries a small **shared** mark.
    Owned plans carry no mark in v1.
27. **R27.** My Plans hides the delete control on plans where `relation` is `member`.
28. **R28.** Settings (the settings shift in `Todo.tsx`) gains a **Sharing** section.
    - **Owner:** an email field with an Add button, the people list (owner first), and a Remove
      control on each member row.
    - **Member:** the same list, no controls, and the owner identified as the owner.
29. **R29.** The add form shows each refusal in words, by `code`, never by parsing `detail`:
    - `NO_ACCOUNT_FOR_EMAIL` → "No Fireplace account uses that email."
    - `MEMBER_IS_OWNER` → "That's you. You already own this plan."
    - 422 → "Enter an email address."
    - Anything else → the existing generic error.

    A successful add clears the field and shows the person in the list. Re-adding an existing
    member leaves the list as it was.
30. **R30.** Remove takes effect in the list once the call succeeds (not optimistically). A
    failure leaves the row and shows the existing error.
31. **R31.** Owner-only settings are **hidden** for members. In v1 that's the daily-reset toggle.
    Archived items stay reachable from Settings for members (item operations are member
    operations).
32. **R32.** The plan page, on a **404** from loading the plan, shows a "This plan isn't here"
    state with a link home, in place of "Failed to load plan data". This covers a stranger, a
    removed member and a deleted plan. Other failures keep the existing message. Mid-session, a
    404 on an item write follows the existing failed-write path (revert + error) and doesn't
    redirect.
33. **R33.** Milestone URLs (FS-043WP) need nothing from this FS beyond R4–R7. They're plan-scoped
    reads and writes, and members reach them like the owner does.

## User Stories

1. As a plan owner, I want to add a friend to my plan by their email, so that we can work on it
   together.
2. As a plan owner, I want to be told plainly when no account uses the email I typed, so that I
   can ask them to sign up or check the spelling.
3. As a plan owner, I want adding someone who's already on the plan to just work, so that a
   double-click or a retry doesn't confuse me.
4. As a plan owner, I want to be stopped from adding my own email, so that I don't think I've
   shared the plan when I haven't.
5. As a plan owner, I want to see everyone on my plan in one list, so that I know who can change
   it.
6. As a plan owner, I want to remove a member, so that someone who's stepped away no longer sees
   or changes the plan.
7. As a plan owner, I want my plan's name, daily reset and deletion to stay mine, so that a
   member can't rename it, reset it or delete it out from under me.
8. As a member, I want a plan shared with me to appear in my sidebar, My Plans and home, so that
   I can find it like my own plans.
9. As a member, I want a shared plan marked as shared, so that I can tell it apart from my own.
10. As a member, I want to add, tick, edit, nest, date, reorder, archive and delete items in a
    shared plan, so that I'm a real partner in the work.
11. As a member, I want to read and write the plan's notes, including generating notes with AI,
    so that our thinking lives in one place.
12. As a member, I want the plan calendar and suggestions to work for me, so that a shared plan
    isn't a lesser plan.
13. As a member, I want to open a milestone link my partner sent me, so that a shared deep link
    lands where they meant.
14. As a member, I want to see who owns the plan and who else is on it, so that I know who I'm
    working with.
15. As a member, I don't want to see settings I can't use, so that the page doesn't offer me
    buttons that fail.
16. As a member, I don't want a delete button on a plan I don't own, so that I can't try to
    delete someone else's plan.
17. As a removed member, I want a clear "this plan isn't here" with a way home, so that I'm not
    stuck on a broken page.
18. As a signed-in stranger with a plan id, I want to get "not found", so that a guessed or leaked
    id reveals nothing about someone else's plan.
19. As a member of plan A, I must not be able to touch plan B's items by putting plan A's id in
    the URL, so that membership in one plan grants nothing elsewhere.
20. As a plan owner, I want my plan's items safe from anyone who isn't on it, so that sharing one
    plan doesn't weaken the others.
21. As a user deleting my account, I want my memberships cleared too, so that nothing of mine
    lingers in other people's plans.
22. As a plan owner whose member deleted their account, I want them to drop out of my people
    list, so that it shows only real people.
23. As a member, I want my edits to count toward my own day, not the owner's, so that touched
    today stays about me.
24. As two people editing the same item, we accept that the last save wins and the other person
    sees it on refetch, so that v1 stays simple.
25. As the owner of a plan with daily reset on, I want reset to apply for everyone on the plan,
    so that the plan has one daily rhythm.
26. As a developer, I want plan access to be one relation that plan-service answers and the
    gateway enforces per operation, so that I never re-derive it per route (ADR-0015).
27. As a developer, I want the refusals to be stable codes, so that the client switches on `code`
    and never parses prose.
28. As the owner telling this story in an interview, I want the schema to show access becoming a
    relation you add, so that the design is explainable in one sentence.

## Acceptance Criteria

### Access

- [ ] A stranger (signed in, no relation) gets 404 `NOT_FOUND` on every plan-scoped route: get /
      patch / toggle / delete plan, every checklist route, every notes route, the calendar, the
      three insights routes, and the three members routes.
- [ ] A request for a plan id that doesn't exist answers the same as a stranger's.
- [ ] A member gets 200/201/204 on every member-allowed operation listed in R7, with the same
      bodies the owner gets.
- [ ] A member gets 403 `FORBIDDEN` on update plan, toggle daily reset, delete plan, add member
      and remove member, and nothing changes.
- [ ] A non-owner's PATCH never answers 200 (the FS-0005 silent no-op is gone).
- [ ] The owner sees no behaviour change on any route, apart from `relation` appearing on plan
      bodies.
- [ ] With plan-service unreachable, plan-scoped routes fail (503/500) and never act unchecked.
- [ ] Notes and calendar no longer call `AssertPlanOwnership`.

### Item belongs to plan

- [ ] get / update / update dates / archive / delete of an item from plan B, addressed under
      plan A's path, answers 404 and changes nothing, even when the caller owns or is a member
      of plan A.
- [ ] The same operations under the item's own plan work for owner and member.
- [ ] plan-service rejects an item-by-id RPC with no plan id (`InvalidArgument`).

### Members

- [ ] `GET /api/plans/{id}/members` returns the owner first, then members by add time, each with
      id, name, email, relation, since. It works for owner and member.
- [ ] `POST /api/plans/{id}/members {email}` by the owner, for an existing other account, answers
      201 with the entry, and that account now sees the plan in `GET /api/plans` with
      `relation: "member"`.
- [ ] Posting the same email again answers 201 with the same `since` and writes no second row.
- [ ] An email with no account → 400 `NO_ACCOUNT_FOR_EMAIL`.
- [ ] The owner's own email → 400 `MEMBER_IS_OWNER`.
- [ ] `{"email": "not-an-email"}`, `{}` or an extra body member → 422 `VALIDATION_FAILED`.
- [ ] `DELETE /api/plans/{id}/members/{userId}` by the owner → 204, and that user's next request
      on the plan → 404.
- [ ] Deleting a non-member, an already-removed member or the owner's own id → 404.
- [ ] A share row whose user no longer exists is left out of the members list.

### Lists and relation

- [ ] `GET /api/plans` returns owned ∪ shared, each once, newest first, each with `relation`.
- [ ] `GET /api/plans/{id}` carries `relation: "owner"` for the owner and `"member"` for a member.
- [ ] `GET /api/plans/shared` keeps paging and gains `relation`.
- [ ] `GET /api/plans/search` returns owned plans only, unchanged.

### Account deletion

- [ ] After `user.deleted` for user U, no `plan_shares` row has `user_id = U`. U's own plans are
      gone, and so are their shares.
- [ ] Delivering the same `user.deleted` twice succeeds both times.

### Client

- [ ] Sidebar, My Plans and home list shared plans alongside owned ones, with a "shared" mark on
      `relation: "member"` plans only.
- [ ] My Plans shows no delete control on a member plan.
- [ ] The owner's Settings shows Sharing with the email field, the people list and Remove on
      member rows. Adding and removing work, and each refusal shows its R29 wording.
- [ ] A member's Settings shows the people list with the owner marked, no add field, no Remove,
      and no daily-reset toggle. The archived view is still reachable.
- [ ] Opening a plan URL as a stranger or removed member shows "This plan isn't here" with a link
      home.
- [ ] Contract regenerated: `openapi.yaml` and `client/src/api/generated` updated, and the gates
      pass.
- [ ] Tests pass.

## Edge States

| Situation | Behaviour |
|---|---|
| Plan id doesn't exist | 404 on every route, the same as no relation (R1, R5) |
| Stranger guesses a plan id | 404. Nothing about the plan leaks (R5) |
| Member on owner-only operation | 403 `FORBIDDEN`, no change (R6) |
| Item id from another plan in the path | 404, no change (R11) |
| Item-by-id RPC without plan id | `InvalidArgument` from plan-service. The gateway never sends one (R12) |
| Owner adds own email | 400 `MEMBER_IS_OWNER` (R17) |
| Owner adds unknown email | 400 `NO_ACCOUNT_FOR_EMAIL`. Accepted account-existence disclosure (D7) |
| Email with surrounding spaces | Trimmed, then resolved (R17) |
| Email differing only by case from the stored one | No match → `NO_ACCOUNT_FOR_EMAIL`, the same as sign-in (R17) |
| Malformed or empty email | 422 `VALIDATION_FAILED` (R18) |
| Owner re-adds an existing member | 201, the existing entry, original `since` (R17) |
| Two owners' tabs add the same email at once | PK + `ON CONFLICT DO NOTHING`. One row, both answer 201 |
| Owner removes a member who isn't there | 404 (R19) |
| Owner removes a member who's viewing the plan | The member's next request is 404. A reload shows "This plan isn't here" (R20, R32) |
| Member's write in flight when removed | The write 404s and the client reverts with the existing error (R32) |
| Owner deletes the plan | Shares cascade (FK). Members' next request is 404 |
| Owner deletes their account | Their plans cascade, so do those plans' shares (R25) |
| Member deletes their account | Their share rows are deleted on `user.deleted` (R25). Until then the members list skips them (R16) |
| `user.deleted` delivered twice | Second run deletes nothing and succeeds (R25) |
| Two people edit the same item | Last write wins. Each sees the other's change on refetch (D19) |
| Member ticks a daily item | Shared. The owner's nightly reset clears it for everyone (D20) |
| Member generates notes with AI | Allowed, the same as any notes operation (R7) |
| Member opens a milestone URL (FS-043WP) | Same page, same edit rights. A stranger gets "not here" (R33) |
| Member searches My Plans | Search is owned-only, so shared plans don't match (D17) |
| A legacy `role = 'view'` row | Treated as a member with full edit access. `role` is ignored in v1 (R3) |
| Plan-service down during the access check | 503/500. Never proceeds unchecked (R8) |
| Plan with no members | The members list holds only the owner |

## API surface

### New resource: plan member

`PlanMemberResp` (response only):

| Field | Type | Notes |
|---|---|---|
| `id` | uuid | The member's user id |
| `name` | string | Display name if set, otherwise account name |
| `email` | string | |
| `relation` | `"owner"` \| `"member"` | |
| `since` | date-time | Owner: plan `createdAt`. Member: share `created_at` |

`AddPlanMemberReq` (request; writable fields only, strict):

| Field | Type | Constraints |
|---|---|---|
| `email` | string | Required, `format: email`, 1–254 chars. Surrounding whitespace is trimmed by the gateway before lookup |

### HTTP operations

| Op | Method + Path | Query/Params | Request body | Response | Errors |
|----|---------------|--------------|--------------|----------|--------|
| listPlanMembers | `GET /api/plans/{id}/members` | `id` uuid | none | 200 `PlanMemberResp[]` (bare array, owner first) | `401 · UNAUTHENTICATED`, `404 · NOT_FOUND` (no relation / no plan), `422 · VALIDATION_FAILED` (bad uuid), `503 · SERVICE_UNAVAILABLE` |
| addPlanMember | `POST /api/plans/{id}/members` | `id` uuid | `AddPlanMemberReq` | 201 `PlanMemberResp` (new or existing) | `400 · NO_ACCOUNT_FOR_EMAIL`, `400 · MEMBER_IS_OWNER`, `401 · UNAUTHENTICATED`, `403 · FORBIDDEN` (member), `404 · NOT_FOUND`, `422 · VALIDATION_FAILED`, `503 · SERVICE_UNAVAILABLE` |
| removePlanMember | `DELETE /api/plans/{id}/members/{userId}` | `id` uuid, `userId` uuid | none | 204, no body | `401 · UNAUTHENTICATED`, `403 · FORBIDDEN` (member), `404 · NOT_FOUND` (no relation, or not a member), `422 · VALIDATION_FAILED`, `503 · SERVICE_UNAVAILABLE` |

### Changed operations

| Op | Change |
|---|---|
| `PlanResp` (all plan responses) | Gains required `relation: "owner" \| "member"` (R23). Additive |
| listPlans `GET /api/plans` | Returns owned ∪ shared (R22). Description updated. Shape unchanged apart from `relation` |
| listSharedPlans `GET /api/plans/shared` | Gains `relation`. Otherwise unchanged |
| getPlan `GET /api/plans/{id}` | `404 · NOT_FOUND` for no relation. The FS-0005 note in its description is removed |
| updatePlan, toggleDailyReset, deletePlan | Owner only: `403 · FORBIDDEN` for a member, `404 · NOT_FOUND` for none. 403 was already declared, so not a contract change |
| all `/api/plans/{id}/checklists…` | Owner or member. `404 · NOT_FOUND` for none and for an item outside the path plan. 403 stays declared but is no longer produced |
| all `/api/plans/{id}/notes…` | Owner or member. A stranger now gets `404` (was an undeclared 403) |
| getCalendar `GET /api/plans/{id}/calendar` | Owner or member. A stranger now gets `404` (was 403) |
| getChecklistSuggestion, getDailyChecklistSuggestion, getSuggestedVideos (`?plan_id=`) | Owner or member. **Gain `404 · NOT_FOUND`** in their Errors list (additive) |
| searchPlans, analytics, users | Unchanged (not plan-scoped, or owned-only by D17) |

New domain codes in `common/errcode` (additive): `NO_ACCOUNT_FOR_EMAIL`, `MEMBER_IS_OWNER`. Both
are 400. `MEMBER_IS_OWNER` arrives from plan-service as `InvalidArgument`. The add-member
operation attaches the code itself rather than relying on `CodeFor`'s message sniffing, which
would misread any message containing "name" as `PROFILE_NAME_EMPTY`.

### plan-service RPCs (plane 2, additive)

Every request keeps carrying `user_id` (the caller), per the current convention pending ADR-0001.

| RPC | Request | Response | Errors |
|---|---|---|---|
| `GetPlanAccess` (new) | `plan_id`, `user_id` | `relation` (enum `PLAN_RELATION_NONE` \| `_OWNER` \| `_MEMBER`), `owner_id` | `InvalidArgument` on a malformed uuid. A missing plan is `NONE`, never `NotFound` (R1) |
| `ListPlanMembers` (new) | `plan_id`, `user_id` | `owner_id`, `plan_created_at`, `members[]` of (`user_id`, `since`) ordered by `since` | `NotFound` if no plan |
| `AddPlanMember` (new) | `plan_id`, `user_id` (caller), `member_user_id` | `PlanMember` (`user_id`, `since`); existing row returned on conflict | `InvalidArgument` (`member_user_id` is the owner: `MEMBER_IS_OWNER`), `NotFound` (no plan owned by caller: defensive, the gateway checks first) |
| `RemovePlanMember` (new) | `plan_id`, `user_id` (caller), `member_user_id` | Empty | `NotFound` (no such share, or no plan owned by caller) |
| `ListPlans` (changed) | + `include_shared` bool (default false) | unchanged | unchanged. When true, owned ∪ shared, deduplicated, `created_at` desc, unpaged |
| `GetItem`, `UpdateItem`, `UpdateItemDates`, `ArchiveItem`, `DeleteItem` (changed) | + `plan_id` (required by R12) | unchanged | `NotFound` when the item isn't in `plan_id`. `InvalidArgument` when `plan_id` is empty |

User names and emails are resolved by the gateway from its own users table. plan-service never
stores or returns an email.

## Out of Scope

- **Public, unauthenticated links.** Parked: a `plan_links(token, plan_id, role, created_by,
  revoked_at)` table resolved by an unauthenticated route. Good interview material on capability
  URLs versus relations.
- A view-only role in the UI or API (`view` stays schema-only).
- Invitations with accept/decline, email notifications, and "leave plan" for members.
- Transferring ownership.
- Per-milestone sharing.
- Realtime presence or sync, and conflict detection between two editors.
- A "you've shared this" mark on owned plans (optional in the notes, not built in v1).
- Shared plans in search results (D17).
- A member cap.
- Case-insensitive email matching. Matching follows sign-in, and normalizing stored emails is
  its own change.
- The not-here state on the standalone calendar page (`app/calendar`), which keeps its existing
  error. The milestone page's not-here belongs to FS-043WP.
- Removing the now gateway-uncalled `AssertPlanOwnership` RPC.
- Moving identity off the body `user_id` field (ADR-0001 / FS-0001).
- `/api/analytics/user/{userId}` authorization (user-scoped, not plan-scoped; FS-0005 territory).
- The `intention` field on checklist items (FS-043WP).
- Glossary entries for **member**, **owner** and **relation** in plan-service / api-gateway
  `CONTEXT.md`. These are new terms here, flagged for `domain-model`.
