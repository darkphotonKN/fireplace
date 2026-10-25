---
id: I-02JBZ-10
status: done
implements: FS-02JBZ
blocked_by: [I-02JBZ-7, I-02JBZ-9]
labels: [feature]
title: "FS-02JBZ slice 10: revision labels and the \"Don't revise\" switch in the Day plan section"
---
Implements FS-02JBZ §Requirements (R66 revision label, R80, R81)

## What to Build

The revision ladder made visible on learning plans.

- **Label (R66, R81).** An entry with `revision = N` reads "Revision N" next to its text.
- **Switch (R80).** On learning plans (entries carrying `revisionsEnabled`), an `existing_item`
  entry offers a small "Don't revise" switch bound to `revisionsEnabled`, calling
  `setItemRevisions(itemId, planId, enabled)`, optimistic, reverted on failure. Entries without
  the field show no switch (project plans, new steps).
- Update the "i" tooltip copy only if it doesn't already mention revisions.

## Acceptance Criteria

- [ ] "Revision N" renders for stamped entries and nowhere else.
- [ ] The switch appears only where `revisionsEnabled` is present, reflects it, and calls
      `setItemRevisions` optimistically, reverting on failure.
- [ ] Tests pass.

## Blocked By

- I-02JBZ-7 — `revision`, `revisionsEnabled` and `setItemRevisions`.
- I-02JBZ-9 — serializes edits to `DayPlanSection.tsx` and `client/src/api/insights.ts`.

## Expected write set

- `client/src/components/DayPlanSection.tsx` (+ test) **(hotspot)**
- `client/src/api/insights.ts` **(hotspot)**: `setItemRevisions`

## Spec Reference

FS-02JBZ §Decisions D6; §Requirements R66, R80, R81; §User Stories 19–24.

## TDD Approach

- RED: component test with a fixture plan: a revision-2 entry shows "Revision 2"; a learning
  entry shows the switch; a project entry does not.
- GREEN: rendering.
- Then: optimistic toggle and revert.
