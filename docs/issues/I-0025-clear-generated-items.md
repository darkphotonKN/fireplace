---
id: I-0025
status: open
implements: FS-0006
blocked_by: [I-0024]
labels: [feature]
title: FS-0006 slice 5: clear generated items
---

Implements FS-0006 §Requirements, §API surface

## What to Build

One action that makes declining a generation cheaper than accepting it.

`DELETE /api/plans/{id}/checklists/generated` → 204. Deletes exactly the `status='generated'`
set — the same predicate regeneration uses, so there is one rule rather than two.

Given parent promotion (I-0024), it is ADR-0007-safe **including through the FK cascade**: any
parent holding edited children is already `touched` and therefore not in the set.

## Acceptance Criteria

- [ ] Removes every `generated` row and nothing else.
- [ ] On a plan whose `generated` parent has a `touched` child, **neither is deleted** — the
      cascade never reaches protected data. This is the test the whole design exists for.
- [ ] Authorization matches the rest of the plan-scoped surface (403 for a non-owner).
- [ ] Tests pass.

## Blocked By

I-0024 (needs the `status` column and parent promotion).

## Spec Reference

FS-0006 §Requirements R46 · §API surface (`clearGeneratedItems`)

## TDD Approach

- RED: a plan with generated parent + touched child; assert both survive.
- GREEN: delete scoped to `status='generated'`.
