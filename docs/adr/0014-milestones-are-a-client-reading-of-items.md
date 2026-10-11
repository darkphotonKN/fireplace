# ADR-0014 — Milestones are a client reading of top-level checklist items

Status: accepted
Date: 2026-10-11
Scope: client — governs where the Milestone concept lives and what it may touch
Related: FS-FSFEC (milestones as blocks of progress, scoping decision 1), ADR-0013 (the
home fan-out the dashboard progress strip must stay inside)

## Context

The grid view (`ChecklistGrid`) already draws each top-level checklist item as a card, with
that item's children as its steps and a meter of done over total task children. The product
now names that unit a **Milestone** and wants to build progress features on it: a per-plan
progress strip, a composer that creates a milestone and then fills it with steps, and a
completion nudge when every step is done.

That raises the question of where a milestone lives. The data already fits: a plan's
top-level items and their one level of children carry everything a milestone needs (title,
`done`, children, notes vs tasks). What was missing was the name and the progress story, not
storage. Three options were weighed:

- **A. Keep it view-only.** Close the grid's gaps and make it the default, with no named
  concept behind it. Cheapest, but it leaves "progress" unnamed and so unadvertisable, and the
  product wants features built around the card as the unit of progress.
- **B. A product concept over the existing item model.** A milestone *is* a top-level task
  item; its steps are its children. The concept is named and designed for in the client, and
  the server keeps seeing checklist items.
- **C. A separate server entity.** A `milestone` table or type in plan-service with its own
  API. That means a migration, contract changes (ADR-0002), and plan-service work, all for no
  behaviour the item model can't already express.

## Decision

**A milestone is a client reading of a top-level checklist item, not a server entity.** A
plan's top-level task items are its milestones. Each milestone's children are its steps.
Top-level notes are not milestones. The grid view is the *milestone reading* of a plan, and
the list view is the *checklist reading* of the same plan. Both views render one model.

## Consequences

**Accepted:**

- The whole milestones feature is client-only. No migration, no new endpoint, no plan-service
  gRPC method, no contract regeneration. Like FS-0007/0008, it ships as a client change.
- **"Milestone" must not leak into API names or server code.** Requests and contract types
  keep saying checklist item, and the client translates at its edge. This is the same
  boundary client `CONTEXT.md` already draws for its other presentation terms.
- A milestone's completion is the item's own `done`. The completion nudge is client behaviour
  that, once the user confirms, sets `done` through the existing item update. Nothing new is
  stored.
- Progress (per milestone and the plan's strip) is derived on every render from items
  already fetched. On the dashboard, it is derived from the items ADR-0013's capped fan-out
  already holds, so the strip adds no request.

**Gained:**

- The progress story ships without coordinating a server release.
- One model behind both views, so the list and the grid can't disagree about what a plan
  contains.

**Revisit when:** a milestone needs server-side state that a checklist item cannot carry,
such as a stored completion distinct from `done`, a milestone-level date or order, or
milestone data that has to be queried across plans on the server. A decision to store that
state **supersedes** this ADR rather than amending it, because a milestone becoming
something the server knows about changes where the concept lives.
