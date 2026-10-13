# CONTEXT.md — Gateway (ubiquitous language)

> The shared vocabulary for this bounded context. One term, one meaning — used
> identically in conversation, code, and specs. Populated by the `domain-model` skill.

## Gateway

**Relation** — the caller's relation to a plan (`owner`, `member` or `none`), owned and answered
by plan-service (`GetPlanAccess`) and consumed here, never redefined. Every plan response
carries it as `relation: "owner" | "member"`; it is never `none` in a response, because a `none`
plan is never returned (FS-12ZMF D14).

**Plan-access seam** — `plangw.Access.Require`, the gateway's one place that turns a relation
into allow or refuse for an operation, asked first on every plan-scoped route (plans,
checklists, members, notes, calendar, insights). Fails closed: no answer or no checker means no
access (ADR-0015 §3).

**Owner-only** / **owner-or-member** — the two requirements (`Need`) an operation can place on a
relation. Owner-only: update plan, toggle daily reset, delete plan, add or remove member.
Everything else plan-scoped is owner-or-member (`NeedMember`, despite the name it admits the
owner too).

**Not found vs forbidden** — relation `none` → 404 `NOT_FOUND` on every operation, so a guessed
id is indistinguishable from a missing plan; a member on an owner-only operation → 403
`FORBIDDEN`. A child (item, note) addressed under a plan it does not belong to is also 404.

**Member** — as plan-service defines it: a user with a share row on the plan. Not
"collaborator" or "sharee". Added and removed by email through the members routes, owner only.
