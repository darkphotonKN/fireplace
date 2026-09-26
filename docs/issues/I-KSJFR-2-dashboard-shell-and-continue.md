---
id: I-KSJFR-2
status: done
implements: FS-KSJFR
blocked_by: []
labels: [feature]
title: "FS-KSJFR slice 2: the dashboard shell, capped fan-out, and Continue cards"
---
Implements FS-KSJFR §Requirements (R9, R18–R22, R36–R38, R42, R47–R51), §Edge States

## What to Build

The dashboard exists and skipping the focus prompt reveals it. The prompt still shows on every
visit in this slice — making it conditional is slice 6.

**The reveal (R9).** "Skip to plans" stops navigating to `/myplans`. It swaps the dashboard in
beneath the prompt, in place, with no URL change and no router push. `/` is one route in two
states. Reword the affordance so it no longer says "plans".

**The fan-out (R20–R22), per ADR-0013.** `GET /api/plans`, sort by `updatedAt` descending, then
checklists for the **top three** plans only — `1 + min(planCount, 3)`, so at most four, whatever
the account holds. The
three item fetches go in parallel. One failing does not blank the page: whatever arrived,
renders (R21). Loading is progressive, not a full-page spinner — plan-derived content paints as
soon as `GET /api/plans` lands and the item-derived parts settle in after, without a block
changing height in a way that moves a row under the cursor (R22).

**Continue where you left off (R36–R38).** The three fetched plans as cards: name, focus line,
type, and done/total progress computed from the fetched items. No extra request — these are the
plans the fan-out already has. No staleness floor (R38): three dormant plans still fill the
three slots, because "recently updated" is ordering and not a claim about recency. A plan with
no items shows 0/0 rather than being hidden. A plan whose item fetch failed shows no progress
rather than a wrong zero.

Read-only (R42). Creating and deleting plans stay on their own surfaces.

**No header card (R18).** The `backdrop-blur-sm rounded-2xl p-8 shadow-lg` block stays on
`/myplans` and `/plan/[planId]` and does not come to home. There is no status line yet — it
ships in slice 4 where its numbers come from.

**Visual conformance (R47–R51).** `client/docs/design-guideline.md` governs and wins over the
FS where they disagree. Tokens only — the raw `rgb(247,111,83)` and `bg-white/5` /
`dark:bg-gray-900/10` on this route go, they do not get extended. Dark mode at the token layer.
Continue cards are **not** the page's coral moment (R49 reserves it for slice 4's block), so
they stay neutral.

## Acceptance Criteria

- [ ] Skipping the prompt reveals the dashboard with no URL change and no navigation.
- [ ] The skip affordance no longer mentions plans and no longer links to `/myplans`.
- [ ] Home issues `1 + min(planCount, 3)` requests — two at 1 plan, four at 3, 4 and 12.
- [ ] Plans are ordered by `updatedAt` descending and only the top three are fetched for items.
- [ ] The three item fetches are issued in parallel.
- [ ] One failed item fetch leaves the rest of the page rendered.
- [ ] `GET /api/plans` failing shows an error state with a retry, not a blank page.
- [ ] Continue shows three cards with name, focus, type and done/total.
- [ ] A plan with zero items renders 0/0, not hidden.
- [ ] A plan whose items failed to load shows no progress rather than 0/0.
- [ ] With fewer than three plans, Continue shows what exists without empty placeholders.
- [ ] No full-page spinner; plan-derived content paints before item data arrives.
- [ ] No raw hex, `rgb(...)` literal or `*-gray-*` utility remains in the files this slice
      touches.
- [ ] Both themes render correctly with no per-component dark overrides.
- [ ] `/myplans` and `/plan/[planId]` are unchanged.
- [ ] Tests pass.

## Blocked By

None.

## Spec Reference

FS-KSJFR §Requirements (R9, R18–R22, R36–R38, R42, R47–R51), §Edge States (zero items, failed
plans fetch, failed item fetch, dormant plans), §Decisions and why (D1, D7, D10).
ADR-0013 is the authority for the cap and owns its consequence.

## TDD Approach

- RED: a test that mounts home with a stubbed API at 12 plans and asserts exactly four requests
  went out, and that the three fetched are the three most recently updated.
- GREEN: the fan-out hook.
- Then: skip reveals the dashboard without navigating; one item fetch rejecting still renders
  the other two cards; progress renders from fetched items.
