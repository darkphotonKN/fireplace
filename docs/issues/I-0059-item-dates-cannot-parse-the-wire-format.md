---
id: I-0059
status: in-progress
implements: FS-none
blocked_by: []
labels: [bug]
title: "itemDates parses only YYYY-MM-DD, but the API sends RFC3339 — every date comparison returns false"
---
Implements FS-none — a defect in behavior that predates the spec system. `FS-none` is a legal
anchor (docs/specs/README.md); note that `develop`'s anchor gate expects FS-NNNN or ADR-NNNN,
so this may need to be picked up by hand.

## What's wrong

`client/src/lib/itemDates.ts:14` `parseDateOnly` accepts only `"YYYY-MM-DD"`:

```
const [y, m, d] = value.split('-').map(Number);
if (!y || !m || !d) return null;
```

The contract publishes these fields as **`format: date-time`**, not date-only.
`services/api-gateway/internal/gateway/plan/model.go:222`:

```
StartDate *time.Time `json:"startDate,omitempty" format:"date-time"`
DueDate   *time.Time `json:"dueDate,omitempty"   format:"date-time"`
```

Go marshals `time.Time` as RFC3339, so the wire carries `"2026-03-16T00:00:00Z"`, and
`client/src/api/generated/schema.d.ts` declares it that way. Splitting that on `-` yields
`["2026", "03", "16T00:00:00Z"]`; `Number("16T00:00:00Z")` is `NaN`; the `!d` guard returns
`null`.

**Consequence:** `isRangePast`, `isDueToday` and `formatDateRange` all fall to their
null-branches for every item that came from the API. Date chips never render
(`ItemDateChip.tsx` returns `null` when `formatDateRange` gives `null`), and FS-KSJFR's Today
block can only ever list `scope: "daily"` items — its due and overdue arms are dead.

**Why no test caught it.** Every fixture in `itemDates.test.ts`, `homeWork.test.ts`,
`TodayBlock.test.tsx` and `ItemActions.test.tsx` is hand-written as `'2026-03-09'`, the format
the parser accepts and the server never sends. The suite tests the parser against itself.

## What to Build

- `parseDateOnly` accepts both shapes: a bare `YYYY-MM-DD` and an RFC3339 timestamp, taking the
  **local** calendar day in both cases. The file's existing local-time discipline is the whole
  point and must survive — `new Date("2026-03-09")` is midnight UTC, i.e. the 8th west of
  Greenwich, and `toISOString()` on a local midnight makes the same mistake in reverse.
- Decide explicitly what a timestamp's calendar day means. `"2026-03-17T02:00:00Z"` is the 17th
  in UTC and the evening of the 16th in Los Angeles; `itemDates.ts:105` `itemStartDate`
  currently takes the UTC day via `.slice(0, 10)`, matching `ItemDateChip`. Whatever is chosen,
  one rule, stated in the file.
- Fixtures that use the real wire format. A test whose fixture is date-only proves nothing
  about production.
- Check `client/src/lib/calendar.ts`, which the header comment says reads these same values.

## Acceptance Criteria

- [ ] `parseDateOnly` returns a correct local date for an RFC3339 input.
- [ ] `parseDateOnly` still returns a correct local date for a `YYYY-MM-DD` input.
- [ ] `isDueToday` is true for an item whose `dueDate` is today in RFC3339.
- [ ] `isRangePast` is true for an RFC3339 date in the past, false for today.
- [ ] `formatDateRange` renders a label for an RFC3339 range, so date chips appear.
- [ ] Tests carry at least one RFC3339 fixture per predicate.
- [ ] Tests pass under a non-UTC `TZ` as well as UTC.
- [ ] The file states one rule for which calendar day a timestamp belongs to.

## Blocked By

None.

## Spec Reference

FS-none. Related: FS-KSJFR R23 and R31 — the Today block's "due today" and "overdue" arms are
specified on top of these predicates and cannot work until this is fixed. Found by the
I-KSJFR-4 code review.

## TDD Approach

- RED: `isDueToday({ dueDate: <today as RFC3339> })` — expected true, returns false today.
- GREEN: widen the parser, keeping the local-time rule.
- Then: the same fixture through `formatDateRange` and `isRangePast`, run under two timezones.
