import {
  addDays,
  endOfMonth,
  parse,
  startOfMonth,
  startOfWeek,
  endOfWeek,
  eachDayOfInterval,
  format,
  isValid,
} from "date-fns";

import type { CalendarItem, CalendarView } from "@/services/api";

export interface CalendarWindow {
  start: Date;
  end: Date;
}

/**
 * Resolve the visible calendar window from a view + anchor date string.
 * Mirrors the backend resolveWindow:
 *   - month: date is "YYYY-MM" → first..last day of that month
 *   - week:  date is "YYYY-MM-DD" → Sun..Sat week containing that date
 */
export function resolveWindow(view: CalendarView, date: string): CalendarWindow {
  if (view === "month") {
    const t = parse(date, "yyyy-MM", new Date());
    if (!isValid(t)) throw new Error(`invalid month date: ${date}`);
    return { start: startOfMonth(t), end: endOfMonth(t) };
  }
  const t = parse(date, "yyyy-MM-dd", new Date());
  if (!isValid(t)) throw new Error(`invalid week date: ${date}`);
  return {
    start: startOfWeek(t, { weekStartsOn: 0 }),
    end: endOfWeek(t, { weekStartsOn: 0 }),
  };
}

/** Format a date as "YYYY-MM-DD". */
export const formatDateKey = (d: Date) => format(d, "yyyy-MM-dd");

/** Format a date as the API anchor for a given view. */
export const formatViewAnchor = (d: Date, view: CalendarView) =>
  view === "month" ? format(d, "yyyy-MM") : format(d, "yyyy-MM-dd");

/** All days in [start, end] inclusive. */
export const getDaysInWindow = ({ start, end }: CalendarWindow) =>
  eachDayOfInterval({ start, end });

/** Parse a "YYYY-MM-DD" string into a UTC midnight Date. Returns null for empty. */
export function parseISODate(s: string | undefined | null): Date | null {
  if (!s) return null;
  const t = parse(s, "yyyy-MM-dd", new Date());
  return isValid(t) ? t : null;
}

export type RenderShape = "bar" | "chip" | "none";

export interface RenderedItem {
  item: CalendarItem;
  shape: RenderShape;
  /** Anchor date for chips, or the visible-window-clipped start of a bar. */
  visibleStart: Date;
  /** Anchor date for chips (== visibleStart) or window-clipped end of a bar. */
  visibleEnd: Date;
  /** True if the item's actual start is before the visible window. */
  clipsLeft: boolean;
  /** True if the item's actual end is after the visible window. */
  clipsRight: boolean;
}

/**
 * Decide how each calendar item should render relative to the visible window.
 * Items with both dates null get shape "none" (caller should not render them).
 */
export function layoutItems(
  items: CalendarItem[],
  window: CalendarWindow,
): RenderedItem[] {
  return items.map((item) => layoutItem(item, window));
}

export function layoutItem(item: CalendarItem, window: CalendarWindow): RenderedItem {
  const start = parseISODate(item.startDate);
  const due = parseISODate(item.dueDate);

  if (!start && !due) {
    return {
      item,
      shape: "none",
      visibleStart: window.start,
      visibleEnd: window.start,
      clipsLeft: false,
      clipsRight: false,
    };
  }

  const anchor = (start ?? due) as Date;
  const end = (due ?? start) as Date;

  // chip: single day
  if (anchor.getTime() === end.getTime()) {
    return {
      item,
      shape: "chip",
      visibleStart: anchor,
      visibleEnd: anchor,
      clipsLeft: false,
      clipsRight: false,
    };
  }

  // bar: clip to window
  const clipsLeft = anchor < window.start;
  const clipsRight = end > window.end;
  return {
    item,
    shape: "bar",
    visibleStart: clipsLeft ? window.start : anchor,
    visibleEnd: clipsRight ? window.end : end,
    clipsLeft,
    clipsRight,
  };
}

/** Step the anchor date forward or backward by one view-unit (week or month). */
export function stepAnchor(anchor: Date, view: CalendarView, direction: -1 | 1): Date {
  if (view === "month") {
    const next = new Date(anchor);
    next.setMonth(anchor.getMonth() + direction);
    return next;
  }
  return addDays(anchor, 7 * direction);
}
