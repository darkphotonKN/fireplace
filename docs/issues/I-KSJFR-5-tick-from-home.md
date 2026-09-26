---
id: I-KSJFR-5
status: in-progress
implements: FS-KSJFR
blocked_by: [I-KSJFR-4]
labels: [feature]
title: "FS-KSJFR slice 5: tick an item off from home"
---
Implements FS-KSJFR §Requirements (R33–R35, R47–R51), §Edge States

## What to Build

The interaction that makes home a place you use rather than a table of contents (D9).

**Optimistic tick (R33).** A row can be ticked from home. It marks done immediately via
`updateChecklistItem`, and the status line's count decrements.

**The row does not disappear (R33, R35).** It stays exactly where it is, marked done and dimmed,
for the rest of the visit. This was refined during spec-writing and the reasoning is load-
bearing: a vanishing row collapses the list under the cursor and turns the next tick into a
mis-click. For the same reason, completing every row does **not** swap the heading to "Next up"
mid-visit and does **not** pull in replacement rows from another plan — the block settles into
a done state until the next load.

**Failure restores honestly (R34).** A rejected tick returns the row to not-done, restores the
count, and tells the user through the existing toast (`@/components/ui/use-toast`, the
bottom-left convention already used in `Todo.tsx`). A user must never be left believing they
finished something they didn't.

**Concurrency.** Two ticks in flight resolve against their own rows; one failing restores only
its own row and only its own share of the count.

**This writes, so it stamps.** The mutation goes through the API layer, which means I-KSJFR-1's
touched-today stamp fires from here for free — verify that it does, since it is what stops the
gate re-firing after a user acts from home.

Visual conformance per R47–R51; motion is a colour/opacity settle, not a layout jump (R51).

## Acceptance Criteria

- [ ] Ticking a row marks it done immediately, before the request resolves.
- [ ] The ticked row stays in place, dimmed, for the rest of the visit.
- [ ] The status line's count decrements on tick.
- [ ] A rejected tick restores the row to not-done.
- [ ] A rejected tick restores the count.
- [ ] A rejected tick surfaces a toast.
- [ ] Two ticks in flight, one failing: only the failed row and its count are restored.
- [ ] Completing every row leaves the heading unchanged and pulls in no new rows.
- [ ] Ticking from home records touched-today.
- [ ] Ticking does not re-order or re-fetch the block.
- [ ] No layout jump on tick; the transition is colour/opacity.
- [ ] Tests pass.

## Blocked By

I-KSJFR-4 — there are no rows to tick until Today / Next up renders them.

## Spec Reference

FS-KSJFR §Requirements (R33–R35), §Edge States (tick fails, two ticks in flight, every row
completed), §Decisions and why (D9, including the refinement away from "the row fades out").

## TDD Approach

- RED: a test that ticks a row against a rejecting stub and asserts the row is not-done again,
  the count is back, and a toast was raised.
- GREEN: optimistic update with rollback.
- Then: two-in-flight with one rejection; the all-complete case asserting the heading does not
  change and no new rows appear.
