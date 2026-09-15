---
id: I-0047
status: open
implements: FS-0007
blocked_by: [I-0043]
labels: [feature]
title: FS-0007 slice 5: Tab nests the add bar under the parent above
---
Implements FS-0007 §Requirements (R8.1–R8.8, R8.10–R8.12), §Acceptance Criteria "Add bar nesting"

## What to Build

Give the add bar two positions: top level (default) and nested under a target parent.

- **Tab** in the add bar input finds the target: walking up from the bottom of the rendered list,
  the nearest top-level `task` item (child rows resolve to their parent; top-level notes are
  skipped). If found, the bar indents to child position and the target's guide rail (I-0043)
  extends to it; focus and typed text stay. If none (empty list, only notes), Tab is a silent
  no-op and focus does not move.
- **Enter / Add** while nested creates the item with `parentId` = target, in the selected type,
  appended as the target's last child with the arrival animation. The bar **stays nested**,
  cleared and focused.
- **Shift+Tab** returns to top level; at top level it is a silent no-op. Tab while nested does nothing.
- The nested position is not persisted: page load always starts at top level.
- If the target stops being eligible while nested (archived, deleted, converted to note,
  outdented, filtered out), the bar returns to top level keeping typed text.
- Update the "Tip: Tab to nest" toast text to describe this; rate limit unchanged.
- Never send a create request whose `parentId` is a note or a child item.

`createChecklistItem` in `client/src/services/api.ts` already accepts `opts.parentId`. No API change.

## Acceptance Criteria

- [ ] List ending in a top-level task: Tab indents the bar under it and the rail extends; focus and typed text kept.
- [ ] List ending in a child row: Tab nests under that child's parent.
- [ ] Last top-level row is a note with a task above: Tab nests under that task.
- [ ] Empty list / only notes: Tab changes nothing, focus stays in the input, focus does not move to another control.
- [ ] Enter while nested sends `parentId` of the target in the create request and renders the item as the target's last child with the arrival animation.
- [ ] After a nested add the bar is still nested under the same parent, empty and focused.
- [ ] Shift+Tab while nested returns to top level; Shift+Tab at top level does nothing and keeps focus.
- [ ] Tab while nested does nothing.
- [ ] Reload always shows the add bar at top level.
- [ ] Archiving, deleting, converting to note, or filtering out the target while nested returns the bar to top level with typed text intact.
- [ ] No create request is ever sent with a `parentId` that is a note or a child item.
- [ ] Tab hint text describes nesting the add bar under the item above and Shift+Tab to return.
- [ ] Create failure while nested keeps the bar nested with text intact and focus in the input.
- [ ] Regression checks from I-0043 still pass; client test suite passes.

## Blocked By

I-0043

## Spec Reference

FS-0007 §Requirements R8.1–R8.8, R8.10–R8.12; §User Stories 12–17, 28; §Acceptance Criteria "Add bar nesting" (all but the collapsed-parent item); §Edge States (empty list, only notes, target archived/converted/deleted, create fails while nested, optimistic temp ids never targets).

## TDD Approach

- RED: target resolver — `[A task, B child of A]` → A; `[A task, N note]` → A; `[N note]` → none; `[]` → none; a temp-id optimistic row is never a target.
- GREEN: pure resolver, then a render test that Tab in the input nests the bar and Enter calls `createChecklistItem` with `{ parentId: A }`.
