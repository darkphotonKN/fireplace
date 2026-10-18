---
id: I-F6AEE-3
status: done
implements: FS-F6AEE
blocked_by: [I-F6AEE-2]
labels: [feature]
title: "FS-F6AEE slice 3: duplicate from grid cards, their steps and the milestone page"
---
Implements FS-F6AEE §Requirements (R14, R16, R17, R20)

## What to Build

The other two places the menu renders get the same Duplicate row: the grid (the milestone
reading) and the milestone page.

**Grid (R14, R16).** `ChecklistGrid` gains an `onDuplicate: (id: string) => unknown` prop. It
passes `onDuplicate={() => onDuplicate(item.id)}` to the card's `ItemActions` and to each step
row's `ItemActions` (both milestone cards and note cards). `Todo` passes the `duplicateTodo` from
slice 2 where it renders `<ChecklistGrid …>`. The grid reads `Todo`'s list, so the copy card
(with its steps) appears directly after the source card with no extra state. A copy is not the
traced milestone (§Edge States): don't touch the composer's tracing.

**Milestone page (R14, R16, R17, R20).** `MilestonePage`'s `MilestoneActions` gains `duplicate(id):
Promise<boolean>`. It is not optimistic: it calls `duplicateChecklistItem` and, on success,
splices the returned items into the page's loaded long-term list right after the source's family
via `setItems`. That way the copy is already in place when Back returns to the plan. On failure
it calls the page's existing `setError` with "Failed to duplicate item. Please try again." The
page stays on the source milestone. Pass `onDuplicate={() => actions.duplicate(milestone.id)}`
to the page's one `ItemActions`. The page has no per-step menu, and this slice adds none. Update
the comment above that menu ("The card's menu less what would make the page's subject something
else") so it still reads true with Duplicate in the list. This widens FS-043WP R17 by one row, as
FS-F6AEE R14 records.

Files expected to touch:
- `client/src/components/ChecklistGrid.tsx` + `client/src/components/ChecklistGrid.test.tsx`
- `client/src/components/Todo.tsx` (one prop on `<ChecklistGrid>` only)
- `client/src/components/MilestonePage.tsx` + `client/src/components/MilestonePage.test.tsx`

## Acceptance Criteria

- [ ] A grid card's menu and each step row's menu offer Duplicate.
- [ ] Duplicating a milestone card shows the copy card, with its copied steps not done, directly
      after the source card.
- [ ] Duplicating a step in a card shows the copy directly after it in the same card.
- [ ] Duplicating does not change which milestone the composer is tracing.
- [ ] The milestone page's menu offers Duplicate. On success the page stays on the source, and
      the copy is in the page's list right after the source's family.
- [ ] A failed duplicate on the milestone page inserts nothing and shows the page's error
      message. Grid failures show `Todo`'s error message (slice 2's handler).
- [ ] `npm run lint`, `npm run build` and the test suite pass.

## Blocked By

I-F6AEE-2 (the `ItemActions` row and `Todo`'s `duplicateTodo`).

## Spec Reference

FS-F6AEE §Requirements R14 (grid cards, grid step rows and the milestone page's menu), R16, R17
and R20. §Acceptance Criteria (the grid and milestone page items). §Edge States (composer
tracing). §Out of Scope (no per-step menu on the milestone page, no navigation to the copy). User
stories 8–11, 14, 19.

## TDD Approach

- RED: in `ChecklistGrid.test.tsx`, render with an `onDuplicate` spy, choose Duplicate on a card,
  and assert the spy got the card's id. Do the same for a step row.
- GREEN: the prop and the two `ItemActions` wirings.
- Then: in `MilestonePage.test.tsx`, mock `duplicateChecklistItem`, choose Duplicate in the
  page's menu, and assert the page still shows the source and the mock was called with the
  milestone id. Add a rejection test that the error text shows and nothing is inserted.
