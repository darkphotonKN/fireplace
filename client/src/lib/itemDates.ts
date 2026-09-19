import { format } from 'date-fns';

/**
 * Checklist dates are date-only strings ("YYYY-MM-DD"), inclusive at both
 * ends, and either end may be absent — one alone is a single day, which is
 * how the plan calendar already reads them (src/lib/calendar.ts).
 *
 * They are parsed and written in LOCAL time on purpose. `new Date("2026-03-09")`
 * is midnight UTC, which is the 8th anywhere west of Greenwich, and
 * `toISOString()` on a local midnight makes the same mistake in reverse — so a
 * date picked in the evening would be saved as the day before.
 */

export function parseDateOnly(value?: string | null): Date | null {
  if (!value) return null;
  const [y, m, d] = value.split('-').map(Number);
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

/** True once the whole range is behind us, so the chip can say so. */
export function isRangePast(
  startDate?: string | null,
  dueDate?: string | null
): boolean {
  const end = parseDateOnly(dueDate) ?? parseDateOnly(startDate);
  if (!end) return false;
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  return end.getTime() < today.getTime();
}
