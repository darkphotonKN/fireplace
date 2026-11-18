---
id: I-PGG2N-4
status: done
implements: FS-PGG2N
blocked_by: [I-PGG2N-3]
labels: [feature]
title: "FS-PGG2N slice 4: open a note from its card, its row, or its menu"
---
Implements FS-PGG2N §Requirements (R12–R15)

## What to Build

Every live note leads to its page. For notes only, these replace today's inline rename on
click. Tasks and milestones keep theirs.

- **Note card (R12), `ChecklistGrid.tsx`.**
  - Clicking anywhere on the card opens `/plan/<planId>/n/<id>`, except on its menu and drag
    handle, which already stop propagation.
  - Use a real link: for example a stretched `<Link>` over the card text, named "Open note:
    ‹firstLine(cardText)›", so it works by keyboard, opens in a new tab and copies.
  - A drag never opens. Check the dnd activation constraint and keep it.
- **Child note rows (R13).** In a milestone card (`ChecklistGrid.tsx`) and on the milestone page
  (`MilestonePage.tsx`), clicking the note's text opens its page as a link. `InlineText` rename
  stays for task steps.
- **Checklist reading (R13), `Todo.tsx`.** A live note row's text is a link to its page.
- **Menu Rename (R14), `ItemActions.tsx`.** For a note, Rename navigates to the note page in
  writing mode. Pass a signal such as `?edit=1` that `NotePage` reads once, then strips. The card
  gains no other control, and no other menu entry changes.
- **Archived list (R15).** Archived notes render as today, not as links.

Files expected:
- `client/src/components/{ChecklistGrid,MilestonePage,Todo,ItemActions}.tsx` and their tests
- `client/src/components/NotePage.tsx` (read the writing-mode signal)

## Acceptance Criteria

- [ ] Clicking a note card anywhere but its menu or handle opens its note page. A drag does not.
- [ ] The card is a link named "Open note: ‹first line›" that can open in a new tab.
- [ ] Clicking a child note row's text in a milestone card, or on the milestone page, opens its
      note page.
- [ ] Clicking a live list note row's text opens its note page.
- [ ] Task steps, milestone titles and task rows still rename inline on click.
- [ ] A note's menu Rename opens its note page with the body in writing mode. Tasks' Rename is
      unchanged.
- [ ] The note card has no new controls, and its menu's entries are unchanged.
- [ ] Archived notes in the archived list are not links.
- [ ] `npm test`, `npm run lint` and `npm run build` pass.

## Blocked By

I-PGG2N-3 (the note page must exist to link to).

## Spec Reference

FS-PGG2N:
- §Requirements R12–R15 (charter defaults C12, C14; gap defaults G8, G9).
- §Acceptance Criteria: the card, row, menu Rename and archived items.
- §Edge States: archived note in the archived list.

## TDD Approach

- RED: a `ChecklistGrid` test. Clicking a note card's text navigates to `/plan/p/n/<id>`, and
  clicking its menu does not.
- GREEN: the stretched link on the card.
- Then: the child row, milestone page and list row links, Rename's writing-mode navigation, and
  a regression test that a task step still renames inline.
