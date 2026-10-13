# ADR-0015 — Plan access is a relation: plan-service answers it, the gateway enforces it per operation

Status: accepted
Date: 2026-10-13
Scope: root. Governs every plan-scoped operation: plans, checklist items, notes, the plan
calendar, plan-scoped insights, plan members, and every future route that takes a plan id.
Realized by: FS-12ZMF (share a plan with other accounts)
Related: FS-0005 (per-user authorization; this answers its open questions), ADR-0005 (domain
rules live in the owning service), ADR-0004 (status + code), ADR-0001 (caller identity)

## Context

Recorded without adversarial review. This is a **charter default**, approved by the owner under
the charter in the FS-12ZMF scoping session, not grilled in a `challenge-me`. It concerns
authorization, which is expensive to reverse, so the marker is deliberate.

Fireplace started private: one owner per plan, and everything assumed the caller was that
owner. In practice nothing enforced it. FS-0005's two-account probe found plans readable by any
signed-in user, and a non-owner's PATCH answering 200 while changing nothing. Checklist item
RPCs take only an item id, so the plan in the URL is decoration. Notes and the calendar call an
owner-only `AssertPlanOwnership`, and FS-0005 stalled on exactly that: applying the owner check
to reads would break the sharing the schema already models (`plan_shares`, migration 000015).

Sharing (FS-12ZMF) forces the question FS-0005 couldn't settle: who may touch a plan, and who
decides. Forces:

- **The relation is plan data.** Ownership is `plans.user_id`, and membership is a
  `plan_shares` row. Both live in plan-service's database.
- **Which operation needs which relation is transport policy.** "A member may tick items but
  not delete the plan" is a statement about HTTP operations, and the gateway is where those are
  defined, documented (`openapi.yaml`) and mapped to status codes.
- **ADR-0005:** one place per rule. If plan-service enforced per operation *and* the gateway
  did, the two would drift.
- **Existence leaks.** "Forbidden" on a stranger's guess confirms the plan exists.
- **IDOR.** Once a member legitimately holds plan A, any item-by-id route that ignores the path
  plan lets them reach plan B's items by putting A's id in the URL.

## Decision

1. **Access is a relation, not a flag.** A user's relation to a plan is exactly one of
   `owner` (`plans.user_id` is the user), `member` (a `plan_shares` row exists for the pair),
   or `none`. A plan that doesn't exist is `none`. Nothing else grants access. A share row's
   `role` doesn't change the relation while only `edit` is offered.
2. **plan-service answers the relation; it doesn't enforce operations.** It exposes one
   question, "what is this user's relation to this plan", as an RPC (`GetPlanAccess`). Its
   owner-scoped mutations (`UpdatePlan`, `DeletePlan`, member add/remove) keep their
   `WHERE user_id` scoping as a defensive backstop, not as the policy.
3. **The gateway enforces per operation, on every plan-scoped route.** Before doing anything
   else, each plan-scoped handler asks for the relation and checks it against the operation's
   requirement: *owner* or *owner-or-member*. There is one seam for this in the gateway, and it
   fails closed. No answer means no access, and a missing checker means no access.
4. **Status codes:**
   - relation `none` → **404 `NOT_FOUND`**, for every operation, so a stranger can't tell a
     guessed id from a missing plan;
   - relation `member` on an owner-only operation → **403 `FORBIDDEN`**. A member already
     knows the plan exists, so 403 leaks nothing and names the real reason.
5. **A child resource must belong to the path's plan.** Every operation that addresses a child
   by its own id (a checklist item, a note) verifies that the child's `plan_id` equals the path
   plan. A mismatch is answered exactly like a missing child: **404**. For checklist items the
   path plan id is passed to plan-service, which scopes the row by it. An item-by-id request
   without a plan id is refused, not run unscoped.

## Consequences

**Positive:**

- One sentence explains the model: *you can reach a plan if you own it or it's shared with you*.
  The schema tells the same story: access becomes a relation you add.
- FS-0005's open questions are answered: 404 for strangers, shared plans in scope, and notes'
  ownership derives from the plan.
- The FS-0005 silent PATCH no-op ends: a non-owner is refused before the update runs.
- The rule lives in two halves that can't drift. The data question is answered once, in its
  owner, and the operation policy is written once, at the edge where operations are defined.
- A second kind of relation later (public link tokens, a view role) extends the relation answer
  and the per-operation table. It doesn't add a parallel check.

**Costs:**

- **One extra RPC per plan-scoped request** (two indexed primary-key lookups). That's
  acceptable at current scale. If it starts to matter, the answer can be cached briefly at the
  gateway or folded into the downstream call. Neither changes this decision.
- **plan-service trusts its caller to have asked.** A future caller of plan-service other than
  the gateway must make the same check. The owner-scoped mutations' `WHERE user_id` backstop
  narrows, but doesn't close, that gap. ADR-0001's identity-in-metadata work is what would let
  plan-service verify the caller itself.
- **Strangers on notes and the calendar now get 404 instead of 403.** That's a behaviour change
  on routes that previously answered 403.
- **404 hides authorization failures from the legitimate user too.** A removed member sees "not
  here", not "you were removed". This is accepted: the client words 404 as "this plan isn't
  here" with a way home.

**Alternatives rejected:**

- **Owner-only `AssertPlanOwnership` everywhere.** Correct for a private app, but it breaks
  sharing on the first read. This is why FS-0005 stalled.
- **plan-service enforces every operation itself** (each RPC takes the caller and decides).
  Pushes HTTP-shaped policy (which verb is owner-only, 403 vs 404) into a service that doesn't
  define the operations. Every RPC grows the same branch, and the gateway still has to shape
  statuses for its local domains (notes). That makes two policy homes.
- **403 for strangers.** Confirms existence for every guessed id.
- **Trust the item id alone** ("item ids are globally unique, the plan in the path is
  decoration"). True of uniqueness, false of authorization: it's the IDOR this ADR exists to
  close.
