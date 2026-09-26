---
id: I-0062
status: open
implements: FS-none
blocked_by: []
labels: [bug]
title: "The same date-only domain value is published two ways: date-only on calendar, RFC3339 on checklists"
---
Implements FS-none — predates the spec system. (`develop`'s anchor gate expects FS-NNNN or
ADR-NNNN, so this may need picking up by hand.)

## What's wrong

A checklist item's `startDate` / `dueDate` is a **date-only** value in the domain: the client
sends `"YYYY-MM-DD"`, and a user picks a day, not an instant. The gateway publishes that one
value in two different shapes depending on which endpoint you ask:

| Endpoint | Type | Wire |
| --- | --- | --- |
| `/api/plans/{id}/checklists` (`plan/model.go:222`) | `*time.Time`, `format: date-time` | `"2026-03-16T00:00:00Z"` |
| `/api/plans/{id}/calendar` (`calendar/local.go:72`) | `string`, no format | `"2026-03-16"` |

Both describe the same column. The `date-time` spelling is the misleading one — it advertises
an instant with a timezone for a value that has neither, which is what made I-0059's local-vs-UTC
question hard enough to need a written rule.

Consequences already paid: **I-0059** (the client's `itemDates` parser accepted only the
date-only spelling and silently returned `null` for every real checklist item, killing the
dashboard's due/overdue arms) and **I-0060** (the same strict parser duplicated in
`calendar.ts`, latent only because the calendar route happens to send the other spelling). Two
bugs, one root: nothing holds the two spellings in sync, and a change to either serializer
flips the other consumer silently.

## What to Build

Pick one spelling for this value across the surface and make the contract say it:

- **Date-only everywhere** is the honest shape — it matches the domain, and the calendar route
  already does it. Costs a change to `ChecklistResp` (a published contract change: breaking, so
  it wants the ratchet's attention and probably a version note).
- **`date-time` everywhere** keeps `ChecklistResp` as-is and is a smaller diff, but it keeps
  publishing a timezone for a value that does not have one, and every consumer forever has to
  know to ignore it.

Whichever is chosen, the client's rule — *the date portion is authoritative, time and zone are
ignored* (`client/src/lib/itemDates.ts`) — should stop being a workaround and become either
unnecessary or documented as the contract's own rule.

## Acceptance Criteria

- [ ] One wire shape for `startDate` / `dueDate` across every endpoint that publishes them.
- [ ] `openapi.yaml` states it, and the regenerated client agrees.
- [ ] The breaking-change gate has seen the change deliberately, not by surprise.
- [ ] `client/src/lib/itemDates.ts` reflects the settled contract rather than defending against
      two shapes.
- [ ] Tests carry the real wire format on both routes.

## Blocked By

None. Lands after I-0059 and I-0060, which made the client tolerant of both shapes — so this
can be done without a flag day.

## Spec Reference

FS-none. Root cause behind I-0059 and I-0060. Contract authority: ADR-0002 (code-first OpenAPI).
