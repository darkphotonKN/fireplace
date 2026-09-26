# ADR-0013 — Home composes from per-plan endpoints with a capped fan-out

Status: accepted
Date: 2026-09-25
Scope: client — governs how the signed-in home assembles cross-plan data
Related: FS-KSJFR (the dashboard home this was decided for, §D7), ADR-0002 (the
code-first contract this declines to extend), ADR-0009 (the service boundaries an
aggregate endpoint would have to cross)

## Context

The signed-in home is becoming a dashboard: a "Today" block of due, overdue and daily-scope
items drawn from across the user's plans, plus per-plan progress. Every one of those facts
lives behind a **per-plan** endpoint — `/api/plans/{id}/checklists`, `.../upcoming` — and no
cross-plan aggregate exists anywhere in the gateway surface. `GET /api/plans` returns plans
with no counts and no items.

So the page cannot be assembled in one request without building something new, and the shape
of "something new" is the decision. Three ways to pay for it were weighed:

- **Fan out over every plan.** Complete: nothing due is ever hidden. Unbounded: an account
  with twelve plans issues thirteen parallel requests on every visit to home, and the dormant
  plans cost exactly what the live ones do. The cost grows with a number the user never
  chose to keep small.
- **A new `GET /api/home` aggregate.** One request, one contract surface to test, the fastest
  page of the three. It is also gateway work plus plan-service gRPC plus contract
  regeneration — a server-side feature carried inside what is otherwise a client restyle, and
  a new endpoint whose only consumer is one page.
- **A capped fan-out.** `GET /api/plans`, sort by `updatedAt`, fetch checklists for the top
  three. Four requests, bounded forever, no new server surface.

The forces: no backend work was in scope for this slice; the analytics endpoint that would
have supplied aggregate counts is a documented 501 stub, so nothing on the server is ready to
be leaned on; and the product framing of home is *focus*, not inventory.

## Decision

**Home composes from the endpoints that already ship, fanning out over the three most
recently updated plans only. No bespoke aggregate endpoint is added for the home page.**

The cap is part of the contract with the user, not a hidden optimization: home shows today
**in the plans you are actually working on**. The UI says so in its framing rather than
implying a global sweep it does not perform.

## Consequences

**Accepted:**

- A due item in a user's eighth-most-recent plan **will not appear on home**. This is the
  price, and it is the honest reading of a focus tool rather than a bug to apologize for —
  but it is a real behavioral limit and must not be discovered by a confused user.
- "Recently updated" is relative, not recent. Three long-dormant plans still fill the three
  slots; staleness is not currently a floor.
- Request count is bounded at four regardless of how many plans an account accumulates.
- No new gateway route, no plan-service gRPC method, no contract regeneration, no new
  generated-client surface to keep honest.

**Gained:**

- The dashboard ships as a client change, reviewable as one.
- Nothing new has to be kept in sync between the server contract and one page's needs.

**Revisit when:** users report missing due items from older plans, or a second consumer wants
the same aggregate. At that point the `GET /api/home` endpoint is the upgrade path, and it
supersedes this ADR rather than amending it — the cap and the endpoint are different
contracts with the user about what home shows.
