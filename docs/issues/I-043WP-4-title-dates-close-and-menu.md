---
id: I-043WP-4
status: done
implements: FS-043WP
blocked_by: [I-043WP-2]
labels: [feature]
title: "FS-043WP slice 4: rename, date, close, archive and delete a milestone from its page"
---
Implements FS-043WP §Requirements (R15–R19, R32), §Edge States

## What to Build

This slice makes the milestone itself editable on its page.

**Title (R15).** Click the title to rename it inline. Enter or blur saves, Esc cancels, and an
empty rename keeps the old title. `document.title` follows the new title.

**Close and reopen (R16, C10).** The milestone's checkbox can be set by hand at any progress.
Closing keeps you on the page and shows the closed state: title struck through, faded, checkbox
ticked. Unticking reopens it. The change is optimistic and reverts with the existing error (R32).

**Item menu (R17, G6).** The page's `ItemActions` offers exactly Rename, Dates, Archive and
Delete. Make the note/task flip optional in `ItemActions`, as move and indent already are, and
pass no move, indent or flip handlers.

**Leaving (R18, C11, G7).** Delete asks for the same confirmation the plan's lists use; reuse or
extract the modal from `Todo.tsx` rather than writing a second one. Once the delete succeeds, the
page navigates to `/plan/<planId>`. A successful archive navigates to the plan too. If either
fails, the page stays put and shows the existing error.

**Dates (R19).** The chip shows the dates. With no dates, a quiet "Add dates" control stands in
its place. Both open the existing date picker, the one the menu's Dates entry opens, and setting
or clearing goes through `updateChecklistDates`. If the picker can't be opened from outside
`ItemActions` as it stands, expose the smallest seam that allows it, such as a prop or a small
extracted picker. How the grid and the list open it must not change.

Files expected: `client/src/components/MilestonePage.tsx` + test,
`client/src/components/ItemActions.tsx` + test, possibly a small extracted delete-confirm and/or
date-picker component under `client/src/components/`, and `client/src/components/Todo.tsx` only
if the delete confirmation is extracted from it.

## Acceptance Criteria

- [ ] Renaming the title on the page saves; an empty rename keeps the old title; the tab title
      follows the new one.
- [ ] Closing keeps the page open and shows the milestone closed; unticking reopens it; a failure
      reverts.
- [ ] The page's menu has exactly Rename, Dates, Archive and Delete.
- [ ] Delete asks for confirmation, then navigates to the plan. Archive navigates to the plan.
- [ ] A failed delete or archive stays on the page with the existing error.
- [ ] Dates show in the chip, and "Add dates" shows when there are none. Both open the date
      picker. Setting and clearing dates persist.
- [ ] The grid's and the list's menus and date picking are unchanged.
- [ ] Tests pass.

## Blocked By

I-043WP-2: needs `MilestonePage`. This slice edits the same component as I-043WP-3, in
different regions, so run the two one after the other or merge with care.

## Spec Reference

FS-043WP §Requirements R15–R19 (title, closing, menu, leaving, dates; charter defaults C4, C10,
C11; gap defaults G6, G7), R32 (optimistic with revert). §Edge States (closed with steps open,
delete or archive from the page, delete or archive fails, dates edited on the page).

## TDD Approach

- RED: a `MilestonePage` test that the menu lists exactly Rename, Dates, Archive and Delete, and
  that confirming Delete calls the delete API and navigates to `/plan/<planId>`.
- GREEN: the menu wiring and the navigation.
- Then: close and reopen, with a rejected update reverting; "Add dates" opening the picker and a
  set date reaching `updateChecklistDates`; renaming the title.
