---
id: I-043WP-2
status: done
implements: FS-043WP
blocked_by: []
labels: [feature]
title: "FS-043WP slice 2: open a milestone into the plan's middle card, at its own URL"
---
Implements FS-043WP §Requirements (R1–R14, R20, R29, R33–R36), §Edge States

## What to Build

This is the tracer bullet. A milestone card opens a page at `/plan/<planId>/m/<milestoneId>`
that takes over only the plan's middle card, shows the milestone, and goes back. The page is
read-only apart from navigation; slices 3–5 add the editing.

**Shared layout (R2–R4, G1).** Create `client/src/app/plan/[planId]/layout.tsx`. It holds
everything `page.tsx` renders today except the long-term Items card. Above `{children}` go the
title card (with the `getPlan` fetch and the focus tooltip) and the Daily card. Below it go the
Calendar and Recommended videos (with `getSuggestedVideos`). `page.tsx` shrinks to the Items
card. `Todo` reads `planId` through `useParams`, which keeps working under the layout. Moving
between the plan and a milestone must not refetch the plan, Daily, Calendar or videos (R3), and
the plan route must behave exactly as before (R4).

**Route (R1).** `client/src/app/plan/[planId]/m/[milestoneId]/page.tsx` renders a new
`client/src/components/MilestonePage.tsx` inside the same Card the Items card uses.

**Way in (R5–R8, G5).** In `ChecklistGrid.tsx`, every milestone card gets an open control in its
header. A milestone card is a top-level task: not a note card and not a step. The control is a
Next `<Link>` to the page, named "Open ‹title›", muted at rest and brighter on hover or focus.
Clicking the title still renames (R7). `ItemActions.tsx` gains an optional `onOpen`; when it is
passed, the menu gets an **Open** entry. The grid passes it for milestone cards only. Update the
row count `menuHeightGuess` uses. List rows don't change (R8).

**Load and read (R10, R14, R33–R36, G2–G4).** `MilestonePage` fetches the plan's long-term list
with the same `fetchChecklist(planId, 'longterm')` call the grid uses. It finds the item there
and reads it through `lib/milestones.ts`. If a small helper there keeps the page thin (say, one
milestone with its ordered steps), add it. The page has four states:
- Loading: a quiet loading line.
- Found, and it is a milestone (top-level, task, long-term, non-archived, `planId` matches the
  URL): the page.
- Not found: the id isn't in the list, is a note, a step or another plan's item, or the list
  answers 404, 403 or 422. Show "This milestone isn't here" with a link back to the plan.
- Any other failure: "Couldn't load this milestone" with Try again.

The page reads only through the plan's existing checklist routes, so it inherits the plan's
access. Add no access logic (R36).

**The page's frame (R9, R11–R13, R20, R29).** A "Back to plan" button in the Settings back
button's style navigates to `/plan/<planId>` (G13). Below it:
- the title, with its checkbox shown read-only;
- the date chip (`ItemDateChip`);
- a progress meter and done/total, with no meter when there are no task steps;
- every non-archived step in stored order: task steps with their done state, notes drawn as
  notes.

Leave the slots for the next step and why it matters to slices 3 and 5. While the page is
mounted, `document.title` is the milestone's title (R12). Nothing from the banned list appears
(R13).

Files expected: `client/src/app/plan/[planId]/layout.tsx` (new), `client/src/app/plan/[planId]/page.tsx`,
`client/src/app/plan/[planId]/m/[milestoneId]/page.tsx` (new), `client/src/components/MilestonePage.tsx`
(new) + test, `client/src/components/ChecklistGrid.tsx` + test, `client/src/components/ItemActions.tsx`
+ test, `client/src/lib/milestones.ts` (+ test if it grows).

## Acceptance Criteria

- [ ] `/plan/<planId>/m/<milestoneId>` renders the page in the middle card, with the title card
      and Daily above and the Calendar and videos below.
- [ ] Going plan → milestone → plan does not refetch the plan, Daily, Calendar or videos.
- [ ] The plan route's settings shift, archived view, toggle, paging and composer are unchanged.
- [ ] Every milestone card shows an "Open ‹title›" link; note cards, steps and list rows do not.
- [ ] A milestone card's menu has Open; menus on steps and note cards do not.
- [ ] Clicking a card's title still renames it.
- [ ] Back to plan goes to `/plan/<planId>`, and browser back works.
- [ ] The tab title is the milestone's title.
- [ ] Progress reads the same done/total as the card; a milestone with no steps shows no meter.
- [ ] Steps are listed in order; notes show as notes and are not counted.
- [ ] Unknown, archived, daily, note, step, other-plan and unreachable-plan ids show "This
      milestone isn't here" with a link to the plan.
- [ ] A 5xx or network failure shows a retryable error, and Try again reloads.
- [ ] While loading, a quiet loading line shows.
- [ ] Tests pass.

## Blocked By

None. Can run alongside I-043WP-1.

## Spec Reference

FS-043WP §Requirements R1–R14 (route, layout, way in, frame), R20 (progress), R29 (steps listed),
R33–R36 (not here, retryable error, no live push, inherited access). §Edge States (the not-here
rows, loading, failure, no steps, only notes, Back after paging, shared plan). Charter defaults
C1–C3, C8, C9, C12, C13; gap defaults G1–G5, G13–G15.

## TDD Approach

- RED: a `MilestonePage` test against a mocked long-term list. A milestone id renders its title,
  progress and steps. The ids of a note, a step, an unknown item and another plan's item each
  render "This milestone isn't here". A rejected 503 renders Try again.
- GREEN: the page and its states.
- Then: `ChecklistGrid` shows the open link on milestone cards only, and the title still
  renames. `ItemActions` shows Open only when `onOpen` is passed. A layout test checks that the
  plan fetch runs once across a child swap.
