import { format } from 'date-fns';

/**
 * Checklist dates are date-only **values** in the domain, inclusive at both
 * ends, and either end may be absent — one alone is a single day, which is
 * how the plan calendar already reads them (src/lib/calendar.ts).
 *
 * THE RULE, once, for this whole file (I-0059): **the first ten characters of
 * a value — its `YYYY-MM-DD` date portion — are the calendar day. Any time and
 * zone that follow are ignored, never converted.**
 *
 * Why, rather than reading the instant: the client sends `"YYYY-MM-DD"`, the
 * server stores it as a `*time.Time` (so, midnight UTC) and the contract
 * publishes it as `format: date-time`, so the wire carries
 * `"2026-03-16T00:00:00Z"`. Converting that instant to a local date would move
 * every due date back a day for every user west of Greenwich. The date the
 * user picked is the date portion, so the date portion is what is read — and
 * `itemStartDate` below, and `ItemDateChip`, have always read it that way.
 *
 * The day itself is then built and written in LOCAL time. `new Date("2026-03-09")`
 * is midnight UTC, which is the 8th anywhere west of Greenwich, and
 * `toISOString()` on a local midnight makes the same mistake in reverse — so a
 * date picked in the evening would be saved as the day before. Both sides of
 * every comparison here are local midnights, which is what makes them agree.
 */

export function parseDateOnly(value?: string | null): Date | null {
  if (!value) return null;
  // The date portion, whether the value is a bare "2026-03-09" or the
  // "2026-03-09T00:00:00Z" the API actually sends. Anything after it is
  // clock time and zone, and this domain has neither.
  const [y, m, d] = value.slice(0, 10).split('-').map(Number);
  if (!y || !m || !d) return null;
  const date = new Date(y, m - 1, d);
  return Number.isNaN(date.getTime()) ? null : date;
}

export function toDateOnly(date?: Date | null): string | null {
  return date ? format(date, 'yyyy-MM-dd') : null;
}

/**
 * What the item shows once it has dates: one day, or a range. The year is
 * left off unless it isn't the current one, so the common case stays short.
 */
export function formatDateRange(
  startDate?: string | null,
  dueDate?: string | null
): string | null {
  const start = parseDateOnly(startDate);
  const due = parseDateOnly(dueDate);
  if (!start && !due) return null;

  const thisYear = new Date().getFullYear();
  const show = (d: Date) =>
    format(d, d.getFullYear() === thisYear ? 'MMM d' : 'MMM d, yyyy');

  const from = start ?? (due as Date);
  const to = due ?? (start as Date);
  return from.getTime() === to.getTime() ? show(from) : `${show(from)} – ${show(to)}`;
}

/**
 * Midnight today, local — the fence the two range predicates below compare
 * against. Local for the same reason `parseDateOnly` is: a UTC fence would put
 * the boundary in the middle of somebody's evening.
 */
function startOfToday(): Date {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  return today;
}

/**
 * The last day a range covers: its due date, or its start when that is all it
 * has. One day is its own last day, which is what lets `isRangePast` and
 * `isDueToday` be the two sides of one fence — no item is ever both.
 */
function lastDay(startDate?: string | null, dueDate?: string | null): Date | null {
  return parseDateOnly(dueDate) ?? parseDateOnly(startDate);
}

/** True once the whole range is behind us, so the chip can say so. */
export function isRangePast(
  startDate?: string | null,
  dueDate?: string | null
): boolean {
  const end = lastDay(startDate, dueDate);
  return end ? end.getTime() < startOfToday().getTime() : false;
}

/**
 * True when the range runs out today — what home's Today block means by "due"
 * (FS-KSJFR R23, R31).
 *
 * A range still in progress is not due today: an item spanning the 9th to the
 * 16th is due on the 16th and silent until then. A start with no due date is
 * its own last day, the same reading `isRangePast` takes.
 */
export function isDueToday(
  startDate?: string | null,
  dueDate?: string | null
): boolean {
  const end = lastDay(startDate, dueDate);
  return end ? end.getTime() === startOfToday().getTime() : false;
}

/**
 * The start an item actually has (FS-KSJFR R32).
 *
 * `scheduledTime` is the deprecated mirror of `start_date` and carries a full
 * timestamp, so it is cut back to its date half and only ever stands in where
 * `startDate` is absent — the same precedence `ItemDateChip` applies.
 *
 * That `.slice(0, 10)` is the rule at the top of this file, not an exception
 * to it: the date portion is the calendar day and the time and zone after it
 * are ignored. `parseDateOnly` now reads every value exactly the same way, so
 * `startDate` and the `scheduledTime` fallback no longer disagree about which
 * day an item starts — and neither does `ItemDateChip`.
 */
export function itemStartDate(item: {
  startDate?: string | null;
  scheduledTime?: string | null;
}): string | null {
  return item.startDate ?? item.scheduledTime?.slice(0, 10) ?? null;
}
