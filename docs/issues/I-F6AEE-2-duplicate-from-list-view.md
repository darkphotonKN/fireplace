---
id: I-F6AEE-2
status: done
implements: FS-F6AEE
blocked_by: [I-F6AEE-1]
labels: [feature]
title: "FS-F6AEE slice 2: a Duplicate row in the item menu, working in the list view"
---
Implements FS-F6AEE §Requirements (R14–R18)

## What to Build

The planner can duplicate from a list row. `ItemActions` gains the row, and `Todo` makes it work:
it waits for the server, inserts the returned items after the source, follows the copy to its
page, and shows its error message on failure.

**`ItemActions` (R14, R15).** Add an optional `onDuplicate?: () => void` prop, documented like
`onOpen`: "offered only where it is handed over". When it is given, render a "Duplicate" row with
a lucide `Copy` icon, run through the same `run(...)` wrapper that closes the panel. Place it
after Rename. Update the panel's row-count comment ("Four rows are always there"), which no
longer holds. Nothing else in the menu changes.

**`Todo` list rows (R15–R18).** Add a `duplicateTodo(id)` handler and pass it as `onDuplicate`
on the live-row `ItemActions` (the one beside `onIndent`/`onOutdent`). The archived view already
renders a delete button instead of the menu. Keep it that way, and also don't hand the handler
over when `todo.archived` is true.

- Call `duplicateChecklistItem(planId, id)` and wait. Insert nothing before it resolves (R16).
- On success, splice the returned items into `todos` immediately after the source's family: for a
  top-level source, after the source and its children; for a child, directly after the child.
  The copy goes first, then its children in the order returned. No refetch.
- Paging (R18): set the existing pending-jump state (`pendingJumpId`, resolved by
  `pageOfItem`) to the copy's id, so the view jumps to the page the copy lands on, exactly as an
  added item does under FS-0008 R16/R17. Add no new paging rule.
- Failure (R17): insert nothing. Call `setError('Failed to duplicate item. Please try again.')`,
  which uses the same error surface as the other item actions.

Files expected to touch:
- `client/src/components/ItemActions.tsx` + `client/src/components/ItemActions.test.tsx`
- `client/src/components/Todo.tsx` (handler and the list-row prop only; it is a 2.6k-line
  hotspot, so keep the diff local)
- `client/src/components/Todo.duplicate.test.tsx` (new)

## Acceptance Criteria

- [ ] `ItemActions` shows "Duplicate" when `onDuplicate` is given and hides it otherwise.
      Choosing it calls the handler once and closes the panel.
- [ ] In the list view, Duplicate on a top-level item with children inserts the copy and its
      children right after the source's family, with no refetch call.
- [ ] Duplicate on a child inserts the copy directly after that child, inside the same family.
- [ ] Nothing is inserted while the request is pending.
- [ ] When the copy lands on a different page than the current one, the view moves to that page.
- [ ] A failed request inserts nothing and shows the item-action error message.
- [ ] No archived row offers Duplicate.
- [ ] `npm run lint`, `npm run build` and the test suite pass.

## Blocked By

I-F6AEE-1 (the endpoint and `duplicateChecklistItem`).

## Spec Reference

FS-F6AEE §Requirements R14 (the menu row; list rows), R15, R16, R17, R18. §Acceptance Criteria
(the menu row, list insert, nothing-before-answer, paging and failure items). §Edge States
(source deleted mid-request, daily item while AI-only). User stories 1, 2, 12, 14–18.

## TDD Approach

- RED: in `Todo.duplicate.test.tsx`, mock `duplicateChecklistItem` to resolve with
  `[copy, copyChild]`. Open a row's menu, choose Duplicate, and assert the rendered order is
  source, its child, copy, copyChild.
- GREEN: the `ItemActions` row and the `duplicateTodo` splice.
- Then: a pending-request test (no insert), a rejection test (error shown, list unchanged), and a
  paging test modelled on `Todo.add-page-jump.test.tsx`.
