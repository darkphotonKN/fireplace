# FS-0008: Paged list with a pinned add bar

> Status: work-order · SPECIFICATION.md: client/SPECIFICATION.md "## Checklists" entry → this FS · Related ADRs: docs/adr/0006 (referenced only: it flags adding pagination to an already-published unbounded list as breaking)

**Client only.** No HTTP, gRPC, or database change, so this FS carries no `## API surface` section. The client already fetches every item of a plan and needs the whole set for parent counts, the type filter, and collapse; paging is a view concern layered on top, exactly as collapse was in FS-0007.

## Summary

A plan's checklist grows until the card runs off the screen and the add bar sits somewhere below the fold. This feature bounds the list: top-level items are shown ten at a time with quiet page controls in the bottom-right corner, the add bar is pinned to the bottom of the card so it is always reachable, and the type filter now also decides what the add bar creates.

## Requirements

### Paging (client-side)

1. R1 Paging happens entirely in the client, over the items already fetched. No request gains a page parameter and no response shape changes.
2. R2 The page unit is the **top-level item** (a Parent Item or a childless row), never an individual row. A parent and its children are always on the same page, whatever their number.
3. R3 A page holds at most **10 top-level items**. A collapsed parent still counts as one, so folding changes the card's height but never the paging.
4. R4 Pages are cut from the list **after** the type filter has been applied, so what is paged is what is shown.
5. R5 Page controls sit in the **bottom-right**, below the list and beside the add bar's row of air: previous, `1 of 3`, next. They are quiet by default (muted text, no filled buttons) and warm to the primary colour on hover and focus, matching the header's collapse-all toggle.
6. R6 Controls are **not rendered at all when everything fits on one page**.
7. R7 Previous is disabled on the first page and next on the last; disabled controls are not focusable and read as disabled to assistive tech.
8. R8 Changing the type filter, or switching between the daily and long-term lists, returns to **page 1**.
9. R9 The current page is **not remembered** across reloads: a fresh mount starts on page 1. (Collapse state is remembered, per FS-0007 R6; the page is not.)
10. R10 If the current page stops existing — its last items were archived, deleted, outdented into another family, or filtered away — the view **clamps to the new last page** rather than showing an empty page.
11. R11 Counts, filters, collapse state and the guide rail continue to be computed over the **whole** set, not the current page.
12. R12 Page changes are announced to assistive tech (a polite live region naming the new page), and the controls carry explicit labels ("Previous page", "Next page").

### The pinned add bar

13. R13 The add bar is **pinned to the bottom of the list card**, staying in view while the list area scrolls, and keeps its current order: it belongs to the end of the list, not the top.
14. R14 Pinning must not cover the last row: the list area reserves the bar's height plus the existing breathing room beneath it.
15. R15 Tab-to-nest (FS-0007 R8) keeps working and now resolves against the **current page**: the target is the nearest eligible top-level task walking up from the bottom of the rows *on that page*.
16. R16 A nested add still lands in its target family. If that family is not on the current page, the view **jumps to the page holding it** so the new row is seen arriving with its animation.
17. R17 A top-level add appends to the end of the list; the view jumps to the **last page** so the new row is seen. The add bar keeps focus either way (FS-0007 R8.6 and the add-bar focus-return behaviour).
18. R18 When a jump changes the page, the add bar's nesting state and typed text are untouched.

### Filter sets what the add bar creates

19. R19 Choosing the **Notes** filter sets the add bar to create notes; choosing **Checklist** sets it to create tasks.
20. R20 Choosing **All** leaves the add bar's type as it was.
21. R21 The add bar's type icon stays clickable and **wins after a filter has set it**: the filter only sets the type at the moment it is chosen, and never blocks a manual override.
22. R22 The add bar's placeholder and labels follow the resulting type, as they already do.

### Interaction with collapse (FS-0007)

23. R23 "Collapse all" / "Expand all" acts on the parents **on the current page**; parents on other pages keep their remembered state, the same rule that already applies to parents the filter hides (FS-0007 R7.4).
24. R24 The header toggle's label reflects the parents on the current page.

## User Stories

1. As a planner with a long list, I want it broken into pages, so that the card stays a readable height instead of running off the screen.
2. As a planner, I want a parent and its children to stay together on a page, so that a family is never split down the middle.
3. As a planner, I want page controls tucked in the bottom-right, so that they are there when I need them and quiet when I don't.
4. As a planner with a short list, I want no page controls at all, so that nothing is added to a view that doesn't need it.
5. As a planner, I want the add bar always within reach at the bottom of the card, so that I can add an item without scrolling to the end.
6. As a planner, I want the add bar to stay at the end of the list rather than the top, so that adding still reads as appending and Tab-to-nest keeps its meaning.
7. As a planner, I want Tab to nest under the item above on the page I'm looking at, so that the gesture still matches what I can see.
8. As a planner adding a child to a family on another page, I want the view to jump there, so that I see my item arrive.
9. As a planner adding at the top level, I want to be taken to the last page, so that the new item isn't added somewhere out of sight.
10. As a planner who was half-way through typing when the page jumped, I want my text and nesting kept, so that nothing is lost.
11. As a planner, I want switching to the Notes filter to set the add bar to notes, so that I don't have to flip the icon every time.
12. As a planner, I want switching to the Checklist filter to set it back to tasks, so that the bar matches the list I'm looking at.
13. As a planner on the All filter, I want the add bar's type left alone, so that a neutral filter doesn't override my choice.
14. As a planner, I want to override the type by clicking the icon after the filter set it, so that the shortcut never traps me.
15. As a planner changing filters, I want to be returned to page 1, so that I'm not stranded on a page that no longer exists.
16. As a planner who just archived the last items on a page, I want to land on the new last page, so that I never see an empty list with items elsewhere.
17. As a planner, I want counts like `1/2` to keep counting the whole family, so that paging never changes what a number means.
18. As a planner, I want "Collapse all" to act on what I'm looking at, so that it behaves like the filter rule I already know.
19. As a keyboard user, I want the page controls focusable with clear labels, so that I can move between pages without a mouse.
20. As a screen reader user, I want the page change announced, so that I know the list content changed under me.
21. As a keyboard user, I want disabled previous/next skipped in the tab order, so that I don't land on dead controls.
22. As a planner on a phone, I want the pinned add bar not to cover the last row, so that I can still read and tap it.
23. As a planner returning to a plan, I want to start on page 1, so that the view is predictable.
24. As a planner with a plan of hundreds of items, I want the app to stay responsive, so that paging is a real fix rather than a cosmetic one.

## Acceptance Criteria

### Paging
- [ ] A list of 25 top-level items shows 10 on page 1 and page controls reading `1 of 3`.
- [ ] A parent with 8 children counts as one of the ten, and its children render on the same page as the parent.
- [ ] A list of 10 or fewer top-level items shows no page controls.
- [ ] Previous is disabled on page 1, next on the last page, and neither is reachable by keyboard while disabled.
- [ ] Switching the type filter returns to page 1; switching between daily and long-term does too.
- [ ] Unmounting and remounting shows page 1, while collapse state is still restored.
- [ ] Archiving the last item on the last page moves the view to the new last page, never an empty one.
- [ ] Parent counts, the filter tabs and collapse state are computed over the full set while a page is shown.
- [ ] Page controls carry "Previous page" / "Next page" labels, and a page change is announced politely.
- [ ] No network request is made when changing pages.

### Pinned add bar
- [ ] The add bar stays visible at the bottom of the card while the list area scrolls.
- [ ] The last row is fully visible and clickable, never hidden behind the pinned bar.
- [ ] On page 2 of 3, Tab nests under the last eligible top-level task of page 2.
- [ ] Adding a child to a family on another page moves the view to that page and plays the arrival animation.
- [ ] A top-level add moves the view to the last page and plays the arrival animation.
- [ ] Typed text, nesting state and input focus survive a page jump caused by an add.

### Filter sets add type
- [ ] Choosing Notes sets the add bar to note; the placeholder follows.
- [ ] Choosing Checklist sets it to task.
- [ ] Choosing All leaves the current type unchanged.
- [ ] Clicking the type icon after a filter set it overrides the type and sticks.

### Regression
- [ ] FS-0007 behaviour holds: guide rail, chevrons and counts, remembered collapse, collapse/expand all, Tab-to-nest, and opening a collapsed parent on add.
- [ ] Client test suite passes.

## Edge States

- **Empty list:** no page controls, no rows; the add bar is still pinned and usable.
- **Exactly 10 top-level items:** one page, no controls.
- **Filter empties the list** (for example Notes with no notes): page 1, the existing empty-state text, no controls.
- **A page's items all archived at once:** clamp to the new last page (R10).
- **Last item on the list deleted while on the last page:** clamp; if the list is now empty, fall back to the empty state.
- **Add while on page 1 of 3:** jump to the page the item lands on (R16, R17).
- **Nested add bar whose target is filtered away or archived:** the bar returns to top level (FS-0007 R8.11), and paging is unaffected.
- **Collapse all on page 2:** only page 2's parents change; page 1 and 3 keep their remembered state.
- **A single family larger than a page** (a parent with 30 children): the family is never split; that page simply runs longer than the others.
- **Reduced motion:** page changes do not animate.
- **Narrow screens:** the controls stay bottom-right and do not wrap under the add bar.
- **Storage unavailable:** paging is unaffected, since the page is never stored.

## Out of Scope

- **Server-side pagination.** `listChecklists` returns a bare array today; adding a paged envelope would break a published shape (FS-0004 §Out of Scope, ADR-0006). Revisit only when a plan's initial fetch is measurably slow — hundreds of items — and then as its own FS superseding FS-0004's API surface, with a plan-service `ListItems` change to match.
- Virtualised scrolling, infinite scroll, or a "show more" affordance.
- Sorting or reordering items, and drag-and-drop between pages.
- Per-user or persisted page size.
- Remembering the page across reloads or sharing it in the URL.
- Paging inside the archived view or the daily insights section.
- Changing what the type filter itself filters.
