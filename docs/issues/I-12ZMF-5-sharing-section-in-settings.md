---
id: I-12ZMF-5
status: done
implements: FS-12ZMF
blocked_by: [I-12ZMF-3]
labels: [feature]
title: "FS-12ZMF slice 5: a Sharing section in plan Settings, and owner-only settings hidden from members"
---
Implements FS-12ZMF §Requirements (R28–R31), §API surface (listPlanMembers, addPlanMember, removePlanMember: client side)

## What to Build

The owner shares a plan from its Settings view, and a member sees who they're working with.

**API layer.** In `client/src/api/plans.ts` add `listPlanMembers(planId)`,
`addPlanMember(planId, email)` and `removePlanMember(planId, userId)`, through the generated
client. Add and remove call `markTouchedToday()` like the other mutations. `PlanMember` is the
generated `PlanMemberResp`.

**Settings view.** `Todo.tsx` already loads the plan (`getPlan`) for `dailyReset`. Also keep its
`relation`. Add a **Sharing** section to the settings shift; a small component such as
`components/PlanSharing.tsx` keeps `Todo.tsx` from growing further.
- **Owner:**
  - an email input and an Add button;
  - the people list (owner first, marked "owner"), each row showing name and email;
  - a Remove control on each member row.
- **Member:** the same list with the owner marked, and no input or Remove.
- **Add refusals (R29),** switched on the problem's `code`:
  - `NO_ACCOUNT_FOR_EMAIL` → "No Fireplace account uses that email.";
  - `MEMBER_IS_OWNER` → "That's you. You already own this plan.";
  - status 422 → "Enter an email address.";
  - otherwise the existing generic error.
- **Successful add:** clear the field and show the person. Re-adding leaves the list unchanged.
- **Remove (R30):** update the list after the call resolves. On failure, keep the row and show
  the error.
- **Members (R31):** hide the "Refresh daily tasks" toggle. "View archived tasks" stays.

## Acceptance Criteria

- [ ] As owner: Settings shows Sharing with the input, the list (owner first) and Remove on
      member rows.
- [ ] Adding a known email shows the new person. Adding the same one again changes nothing.
- [ ] Unknown email, own email and malformed email each show their R29 wording.
- [ ] Remove drops the row after success. A failed remove keeps it and shows the error.
- [ ] As member: Sharing shows the list with the owner marked, and no input or Remove. No
      daily-reset toggle. Archived view still opens.
- [ ] No component parses `detail`. Refusals switch on `code` (and on status for 422).
- [ ] Existing Todo settings tests still pass. New tests cover the owner and member renders and
      each refusal.
- [ ] Tests pass.

## Blocked By

I-12ZMF-3, for the member endpoints and the generated `PlanMemberResp`. It also needs
I-12ZMF-2's `relation` on `getPlan`, which I-12ZMF-3 already depends on.

## Spec Reference

FS-12ZMF §Requirements R28–R31; §API surface (the three member operations and their codes);
§Edge States (owner adds own email, unknown email, re-add, remove while viewing); decisions D6,
D10, D13, D16. User stories 1–7, 14, 15.

Files expected:
- `client/src/api/plans.ts`
- `client/src/api/client.ts` (add the two new codes to the `ErrorCode` / `MESSAGES` map with
  the R29 wording)
- `client/src/components/Todo.tsx` (settings view, plan `relation` state)
- `client/src/components/PlanSharing.tsx` (new), `client/src/components/PlanSharing.test.tsx`
  (new)
- `client/src/components/Todo.*.test.tsx`, only if a settings assertion moves

## TDD Approach

- RED: PlanSharing renders for `relation: "member"` with no email input and no Remove buttons.
- RED: submitting an email whose add rejects with code `NO_ACCOUNT_FOR_EMAIL` shows "No
  Fireplace account uses that email."
