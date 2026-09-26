---
id: I-KSJFR-4
status: in-progress
implements: FS-KSJFR
blocked_by: [I-KSJFR-2]
labels: [feature]
title: "FS-KSJFR slice 4: Today / Next up, read-only, and the status line"
---
Implements FS-KSJFR §Requirements (R16–R17, R23–R32, R47–R51), §Edge States

## What to Build

The block that makes home worth opening, plus the line that describes it. Ticking is slice 5 —
rows render read-only here.

**The date helper first (R31).** `client/src/lib/itemDates.ts` has `isRangePast` and nothing for
"due today". Add the missing predicate *there*, beside it, not inline in a component. That file
opens with the reason: checklist dates are date-only strings parsed in **local** time on
purpose, because `new Date("2026-03-09")` is midnight UTC and lands on the 8th west of
Greenwich. A date comparison written inline in this block is the most likely bug in this slice.
`scheduledTime` is the deprecated mirror of `startDate` and is read only as a fallback where
`startDate` is absent, consistent with `ItemDateChip` (R32).

**Today (R23).** From the three fetched plans: items due today, items overdue and not done, and
items with `scope: "daily"`. An item that is both due today and daily-scope appears once.

**Next up (R24–R25).** When that set is empty the same block — not a second block — becomes
*Next up* and lists the first few unchecked items of the most recently updated plan in
`sequence` order. The heading always describes what is actually listed: "Today" never appears
above rows that are not due.

**Exclusions.** `type: "note"` items never appear, because notes are not work (R26). Archived
items never appear anywhere on home (R27). Completed items are not in the initial set (R28).

**Rows (R29–R30).** An item qualifies at any depth; one with a parent carries its parent's text
as context on the row so a sub-item isn't orphaned. Grandparents are not chained into the
breadcrumb. Every row names its plan, and the plan name links into that plan.

**The status line (R16–R17).** The dashboard's first line — not a greeting, not a card:
weekday, the count of due work, and how many plans that count was drawn from, e.g.
"Thursday · 2 due across 3 plans". In the Next-up case it says so instead, e.g.
"Thursday · nothing due — 6 waiting". The "across N plans" is deliberate: it is ADR-0013's cap
made visible in the product rather than discovered as a bug report.

**Visual conformance (R47–R51), and note R49:** this block is the page's **one coral moment**.
Continue cards, plan rows and the status line stay neutral. Tokens only, both themes at the
token layer.

## Acceptance Criteria

- [ ] `isDueToday` (or equivalent) lives in `itemDates.ts` beside `isRangePast` and is
      local-time correct across a UTC-offset boundary.
- [ ] No date-only string is parsed with a bare `new Date(...)` in this slice.
- [ ] With due items present, the heading is Today and the rows are due + overdue + daily.
- [ ] An item both due today and daily-scope appears exactly once.
- [ ] With no qualifying items, the heading is Next up and rows come from the most recently
      updated plan in `sequence` order.
- [ ] Note-type items never appear.
- [ ] Archived items never appear.
- [ ] Already-completed items are absent from the initial render.
- [ ] A dated child item appears, carrying its parent's text as context.
- [ ] A dated grandchild shows only its immediate parent, not a chain.
- [ ] Every row names its plan and the plan name links into that plan.
- [ ] `startDate` absent falls back to `scheduledTime`; where both exist, `startDate` wins.
- [ ] The status line reads correctly in the Today case and in the Next-up case.
- [ ] The status line states how many plans the count was drawn from.
- [ ] Coral appears exactly once on the dashboard, in this block.
- [ ] Tokens only; both themes correct.
- [ ] Tests pass.

## Blocked By

I-KSJFR-2 — consumes the fan-out's items and renders inside the shell.

## Spec Reference

FS-KSJFR §Requirements (R16–R17, R23–R32), §Edge States (due-and-daily, deep nesting, no due
items), §Decisions and why (D5, D8, D14). ADR-0013 for why the count is scoped to three plans.

## TDD Approach

- RED: a row-per-case test over the qualification rule — due today, overdue, daily, note,
  archived, done, due-and-daily — asserting exactly which items surface.
- GREEN: the derivation plus the `itemDates` predicate.
- Then: the Next-up fallback with a fixture where nothing has a date, the parent-context row,
  and both status-line strings.
