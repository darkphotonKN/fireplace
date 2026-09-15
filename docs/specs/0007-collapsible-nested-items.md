# FS-0007: Visible nesting and collapsible groups

> Status: work-order · SPECIFICATION.md: client/SPECIFICATION.md "## Checklists" entry → this FS · Related ADRs: none

**Client only.** No backend, contract, or migration change. The Parent Item / Child Item
relationship and Two-Tier Nesting already exist server-side (services/plan-service/SPECIFICATION.md,
"Nested Items + Notes"); the client create call already accepts `parentId`
(`client/src/services/api.ts` `createChecklistItem`). This feature makes the relationship
read as one, lets a group fold away, and lets new items be nested straight from the add bar.

## Summary

On a plan's checklist, a Child Item currently looks like a row that happens to be shifted
right. This feature ties children to their Parent Item with a quiet guide rail, lets each
parent collapse and expand its children (with a count of what's hidden), remembers that on
the device, offers a collapse/expand-all toggle, and lets the add bar nest a new item under
the parent above with Tab, so a user can write a parent and then its children back to back
without touching the mouse.

## Requirements

### Background rules this feature relies on (unchanged)

- R0.1 Nesting is two tiers: a Parent Item is a top-level item; a Child Item has a `parentId`
  pointing at a top-level item. A child cannot have children.
- R0.2 Notes cannot be parents. Only `task` items can be Parent Items. Children may be tasks or notes.
- R0.3 The client must never send a nest the server would refuse (tier-3, note parent). It
  decides eligibility locally rather than discovering it from a 400.

### Visual attachment

1. R1 A Parent Item with at least one visible child renders a **guide rail**: a single vertical
   hairline that starts beneath the parent's checkbox and runs down alongside all of its
   children, ending at the last child (or at the nested add bar, see R5.4).
2. R2 The rail is horizontally centred on the parent's checkbox column. Child rows sit to
   its right, indented so the child's checkbox/text starts clear of the rail.
3. R3 The rail uses the same token and weight as the add bar's resting hairline
   (`foreground` at ~15% opacity, 1px) in both light and dark themes. It does not animate
   except when the group expands/collapses or the add bar joins it.
4. R4 A parent whose children are all hidden by the active type filter, or which has no
   children, renders no rail, no chevron, and no count. Children whose parent is not rendered
   keep today's behaviour (shown at top level without indent).

### Collapse and expand

5. R5.1 Every Parent Item with at least one child has a **chevron** in the left gutter, outside
   the checkbox column, pointing down when expanded and right when collapsed.
6. R5.2 Chevron visibility: on devices that support hover, it is hidden until the parent row is
   hovered or focused **when expanded**, and always visible **when collapsed**. On devices without
   hover (`(hover: none)`), it is always visible at low opacity, full opacity when collapsed.
7. R5.3 Activating the chevron (click, tap, Enter or Space when focused) toggles the parent's
   children between shown and hidden. It does not toggle the parent's done state or start editing.
8. R5.4 A collapsed parent shows a quiet **count** after its text:
   - if it has at least one `task` child: `done/total` counting **only task children**
     (e.g. `1/2` for 2 tasks and 1 note);
   - if every child is a note: `N notes` (`1 note` when N is 1).
   No count is shown while expanded.
9. R5.5 Hidden children are removed from the layout and from the keyboard tab order, and are
   not reachable by screen readers until expanded. The chevron exposes `aria-expanded` and an
   accessible label naming the action ("Collapse <parent text>" / "Expand <parent text>").
10. R5.6 Expanding/collapsing uses a short height+opacity transition consistent with the list's
    existing motion, and respects `prefers-reduced-motion` (the global backstop already does).

### Remembering collapse state

11. R6.1 Collapsed state is remembered **on this device only**, in browser storage. Nothing is
    sent to the backend.
12. R6.2 State is keyed per plan **and** per list scope (daily vs long-term are separate), and
    stores the ids of collapsed parents. Default for any parent not in storage is expanded.
13. R6.3 Stored ids that no longer correspond to a Parent Item with children (child outdented,
    parent archived/deleted, converted) are ignored on render and pruned the next time state is written.
14. R6.4 Storage access is wrapped so an unavailable or throwing storage (private mode,
    blocked site data) degrades to "everything expanded, nothing remembered" with no error shown.

### Collapse / expand all

15. R7.1 The list header shows one quiet toggle when the list contains at least one Parent Item
    with children; otherwise it is not rendered.
16. R7.2 Label: **"Expand all"** if any parent is currently collapsed, otherwise **"Collapse all"**.
17. R7.3 It **overwrites** per-parent state: Collapse all collapses every parent; Expand all
    expands every parent; the result is what gets remembered (R6). It is not a temporary lens.
18. R7.4 It acts on parents rendered under the active type filter. Parents hidden by the filter
    keep their stored state.

### Nesting from the add bar

19. R8.1 The add bar has two positions: **top level** (default) and **nested under a target parent**.
20. R8.2 Pressing **Tab** in the add bar input, when at top level, finds the **target parent**:
    walking up from the bottom of the rendered list, the nearest top-level `task` item. Rows
    that are children resolve to their parent; top-level notes are skipped.
21. R8.3 If a target parent exists, the add bar moves under it: it indents to child position and
    the target's guide rail extends down to it. Focus and any typed text stay in the input.
22. R8.4 If no target parent exists (empty list, or only notes), Tab is a **silent no-op**: nothing
    moves, no hint, focus stays in the input, and Tab's default focus navigation is still prevented.
23. R8.5 While nested, pressing **Enter** (or Add) creates the item with `parentId` set to the
    target parent, in the current type (task or note), and appends it as that parent's last child.
24. R8.6 After a nested add, the add bar **stays nested** under the same target, the input is
    cleared and keeps focus (building on the existing focus-return behaviour), so several
    children can be entered back to back.
25. R8.7 **Shift+Tab** in the add bar returns it to top level. At top level, Shift+Tab is a
    silent no-op and does not move focus.
26. R8.8 Pressing Tab again while already nested does nothing (no tier-3).
27. R8.9 If the target parent is collapsed when the add bar nests under it, or when a nested add
    lands in it, the parent **auto-expands** (and that expansion is remembered), so the new item is
    visible and plays the existing arrival animation.
28. R8.10 The nested position is not remembered: on page load the add bar is always at top level.
29. R8.11 If the target parent stops being eligible while the add bar is nested under it (archived,
    deleted, converted to a note, outdented, hidden by the type filter), the add bar returns to
    top level without losing typed text.
30. R8.12 The "Tip: Tab to nest" hint remains, rate-limited as today; its description is updated
    to match R8 (Tab nests the add bar under the item above; Shift+Tab brings it back).

### Scope of surfaces

31. R9 Applies wherever the checklist list renders with the add bar (long-term and daily). When the
    add bar is hidden (daily list with AI-only mode, archived view), R8 does not apply but R1–R7 do.
    The archived view is out of scope (it shows no nesting today).

## User Stories

1. As a planner, I want children to be visibly tied to the item above them, so that I can see at a glance which sub-steps belong to which goal.
2. As a planner, I want the nesting line to feel quiet and match the rest of the list, so that the page stays calm and cozy rather than looking like a file tree.
3. As a planner with a long list, I want to fold a parent's children away, so that I can focus on the top-level picture.
4. As a planner, I want a collapsed parent to show how many of its checklist children are done, so that I can see progress without opening it.
5. As a note-taker, I want a parent whose children are all notes to say how many notes it holds, so that I know something is tucked away there.
6. As a planner, I want notes under a parent not to skew its done/total count, so that progress reflects actual to-dos.
7. As a returning user, I want groups I collapsed to still be collapsed when I come back to the plan, so that I don't have to re-tidy every visit.
8. As a user with daily and long-term lists, I want each list to remember its own collapsed groups, so that folding one doesn't fold the other.
9. As a user on a second device, I accept that collapse state isn't synced, so that this stays a lightweight, client-only preference.
10. As a planner, I want a single "Collapse all" / "Expand all" control, so that I can switch between overview and detail in one click.
11. As a planner, I want "Expand all" to really open everything, so that the control does what its label says.
12. As a planner typing a list, I want to press Tab in the add bar to write the next item as a child of the item above, so that I can outline without the mouse.
13. As a planner, I want to see the add bar move under the parent before I press Enter, so that I know where the item will land.
14. As a planner, I want the add bar to stay nested after adding a child, so that I can type several sub-items in a row.
15. As a planner, I want Shift+Tab to bring the add bar back to top level, so that I can start the next parent.
16. As a planner whose last row is a note, I want Tab to nest under the nearest checklist item instead, so that Tab still does something useful.
17. As a planner with nothing to nest under, I want Tab to simply do nothing and keep my cursor in place, so that I don't lose my typing or get thrown to another control.
18. As a planner adding into a collapsed parent, I want it to open, so that I see my new item arrive.
19. As a keyboard user, I want the chevron to be focusable and operable with Enter/Space, so that I can collapse groups without a mouse.
20. As a keyboard user, I want hidden children to be skipped when tabbing, so that I don't land on invisible rows.
21. As a screen reader user, I want the chevron to announce whether the group is expanded, so that I understand the structure.
22. As a phone user, I want to see the chevron without hovering, so that I can collapse groups on a touch screen.
23. As a desktop user, I want the chevron to stay out of sight until I hover an expanded parent, so that the list stays minimal.
24. As a user filtering to Notes or Checklist, I want rails, chevrons and counts to reflect only what's shown, so that the filtered view isn't cluttered with controls for hidden items.
25. As a user whose browser blocks storage, I want the list to still work (everything expanded), so that a privacy setting doesn't break my plan.
26. As a user who archives or outdents a parent's last child, I want the chevron and count to disappear, so that I'm not offered a collapse for an empty group.
27. As a user who prefers reduced motion, I want expand/collapse to happen without animation, so that the interface is comfortable.
28. As a planner, I want the Tab-to-nest tip to describe what Tab actually does, so that I learn the gesture correctly.

## Acceptance Criteria

### Attachment
- [ ] A parent with children shows one continuous hairline from beneath its checkbox to its last child; a parent without children shows none.
- [ ] The rail is centred on the checkbox column and uses the same colour/weight as the add bar's resting hairline in light and dark themes.
- [ ] With a type filter that hides a parent, its children render at top level with no rail (existing behaviour preserved).

### Collapse
- [ ] Clicking/tapping a parent's chevron hides its children; clicking again shows them. The parent's done state and edit mode are unaffected.
- [ ] Chevron is keyboard-focusable, toggles on Enter and Space, and exposes `aria-expanded` and an action label.
- [ ] Expanded parent: chevron hidden until hover/focus on hover-capable devices; visible at low opacity under `(hover: none)`. Collapsed parent: chevron always visible.
- [ ] Collapsed parent with 2 task children (1 done) and 1 note child shows `1/2`.
- [ ] Collapsed parent with 3 note children shows `3 notes`; with 1 note child shows `1 note`.
- [ ] No count is shown on an expanded parent.
- [ ] Hidden children are not in the tab order and not exposed to assistive tech.
- [ ] With reduced motion enabled, toggling has no visible transition.

### Persistence
- [ ] Collapsing a parent, reloading the page, shows it still collapsed.
- [ ] Collapsing a parent in the long-term list does not collapse anything in the daily list of the same plan, and vice versa.
- [ ] Collapse state in plan A does not affect plan B.
- [ ] No network request is made when collapsing/expanding.
- [ ] Removing a parent's last child (outdent/archive/delete) removes its chevron/count; a later reload does not error and the stale id is pruned on next write.
- [ ] With storage throwing on access, the list renders fully expanded and collapse still works for the session without errors.

### Collapse / expand all
- [ ] Toggle is absent when no parent has children; present otherwise.
- [ ] Label is "Expand all" when any rendered parent is collapsed, else "Collapse all".
- [ ] "Collapse all" collapses every rendered parent; "Expand all" expands every rendered parent; the result survives reload.
- [ ] Parents hidden by the active type filter keep their stored state after using the toggle.

### Add bar nesting
- [ ] Given a list ending in a top-level task, Tab in the add bar indents the bar under that task and the rail extends to it; focus and typed text are kept.
- [ ] Given a list ending in a child row, Tab nests the bar under that child's parent.
- [ ] Given a list whose last top-level row is a note with a task above it, Tab nests under that task.
- [ ] Given an empty list or a list of only notes, Tab changes nothing, focus stays in the input, and focus does not move to another control.
- [ ] Enter while nested creates the item with `parentId` of the target (verified in the create request), in the selected type, rendered as the target's last child with the arrival animation.
- [ ] After a nested add, the bar is still nested under the same parent, empty and focused.
- [ ] Shift+Tab while nested returns the bar to top level; Shift+Tab at top level does nothing and keeps focus.
- [ ] Tab while nested does nothing.
- [ ] Nesting under, or adding into, a collapsed parent expands it, and the expansion is remembered.
- [ ] Reloading the page always shows the add bar at top level.
- [ ] Archiving, deleting, converting to note, or filtering out the target parent while nested returns the bar to top level with typed text intact.
- [ ] No create request is ever sent with a `parentId` that is a note or a child item.
- [ ] The Tab hint's text describes nesting the add bar under the item above and Shift+Tab to return.

### Regression
- [ ] Existing row Tab/Shift+Tab indent/outdent, the hover-menu indent button, type toggle, edit, schedule, archive, and the add bar's focus-return after add all still work.
- [ ] Client test suite passes.

## Edge States

- **Empty list:** no rail, no chevrons, no collapse-all toggle; Tab in add bar is a no-op.
- **Only notes:** no parents possible; same as empty for R5–R8.
- **Parent loses last child** (outdent, archive, delete, or child converted and then moved): chevron, count and rail disappear immediately; stored id ignored, pruned on next write.
- **Parent archived with children remaining:** children render at top level (archive does not cascade, existing behaviour); no rail for the archived parent; if it was the add bar's target, the bar returns to top level.
- **Parent deleted:** children are cascade-deleted server-side; client removes them; stored id pruned.
- **Target parent converted to a note:** server refuses conversion if it has children; if it has none, it stops being eligible and the add bar returns to top level.
- **Type filter changes while collapsed:** stored state is kept; what renders follows R4 and R7.4.
- **Create fails while nested:** existing error handling applies; the bar stays nested with the typed text intact and focus in the input.
- **Optimistic insight/suggestion items** (temporary ids) are never Parent Items for Tab targeting until they have a real id.
- **Storage unavailable or corrupt JSON:** treated as empty (everything expanded); next write replaces it.
- **Very many children:** collapse and count still correct; no virtualisation required.
- **Concurrent tabs of the same plan:** each tab reads storage on load; last write wins; no live sync between tabs.
- **Touch device:** R5.2 visibility; chevron hit area at least 32×32px without shifting the text column.
- **Daily list in AI-only mode / archived view:** add bar hidden, R8 not applicable; archived view shows no nesting (unchanged).

## Out of Scope

- Any backend, API, or storage-schema change; syncing collapse state across devices.
- Changing Two-Tier Nesting or allowing notes as parents.
- The hover-menu indent button's vague error when the row above is a note (separate fix).
- Children orphaned to top level under type filters or when their parent is archived (beyond R4/Edge States).
- Completing or un-completing children when a parent is checked.
- Keyboard collapse with Left/Right arrow keys on a focused parent row.
- Drag-and-drop reordering or re-parenting.
- Nesting in the archived view.
- Tree-elbow connectors or grouped-card styling (considered and rejected in scoping in favour of the guide rail).
