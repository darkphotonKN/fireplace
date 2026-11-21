import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/react';
import { renderToString } from 'react-dom/server';
import {
  CALENDAR_HIGHLIGHT,
  CALENDAR_ITEMS,
  CALENDAR_MONTH,
  DATABASE_MILESTONE,
  DAY_LABEL,
  DAY_PLAN,
  MICROSERVICES_MILESTONE,
} from './content';
import CalendarTileMock from './CalendarTileMock';

const WEEKDAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

/** The weekday a day of the fixed month falls on, from the month's own first weekday. */
function weekdayOf(day: number) {
  return WEEKDAYS[(CALENDAR_MONTH.firstWeekday + day - 1) % 7];
}

/** Which grid row (0-based) a day of the month sits in. */
function weekOf(day: number) {
  return Math.floor((CALENDAR_MONTH.firstWeekday + day - 1) / 7);
}

function days(container: HTMLElement) {
  return Array.from(container.querySelectorAll<HTMLElement>('[data-calendar-day]'));
}

function items(container: HTMLElement) {
  return Array.from(container.querySelectorAll<HTMLElement>('[data-calendar-item]'));
}

/** Present at start: not inside a part that only arrives on reveal. */
function atStart(el: Element) {
  return !el.closest('[data-reveal-part="enter"], [data-reveal-part="glow"]');
}

const steps = new Map(
  [DATABASE_MILESTONE, MICROSERVICES_MILESTONE].flatMap((m) =>
    m.steps.map((s) => [s.label, { milestone: m.title, done: s.done }] as const)
  )
);

describe('CalendarTileMock', () => {
  it('should render the same month whatever the clock says, server and client alike', () => {
    vi.useFakeTimers();
    try {
      vi.setSystemTime(new Date('2020-02-29T23:59:00Z'));
      const leapDay = renderToString(<CalendarTileMock />);
      vi.setSystemTime(new Date('2031-07-04T12:00:00Z'));
      expect(renderToString(<CalendarTileMock />)).toBe(leapDay);
    } finally {
      vi.useRealTimers();
    }
  });

  it('should draw one cell per day of the fixed month, under its name and no year', () => {
    const { container, getByText } = render(<CalendarTileMock />);

    expect(days(container).map((d) => Number(d.dataset.calendarDay))).toEqual(
      Array.from({ length: CALENDAR_MONTH.days }, (_, i) => i + 1)
    );
    expect(atStart(getByText(CALENDAR_MONTH.name))).toBe(true);
    expect(container.textContent).not.toMatch(/\b(19|20)\d\d\b/);
  });

  it('should place each day under its weekday column', () => {
    const { container } = render(<CalendarTileMock />);

    days(container).forEach((cell) => {
      const day = Number(cell.dataset.calendarDay);
      expect(cell).toHaveAttribute('data-weekday', weekdayOf(day));
    });
  });

  it('should date steps from content.ts, done or open as the PLAN tile shows them', () => {
    const { container } = render(<CalendarTileMock />);
    const shown = items(container);

    expect(CALENDAR_ITEMS.length).toBeGreaterThanOrEqual(3);
    expect(shown.map((el) => el.dataset.calendarItem)).toEqual(CALENDAR_ITEMS.map((i) => i.step));
    CALENDAR_ITEMS.forEach((item, k) => {
      expect(steps.get(item.step)).toEqual({ milestone: item.milestone, done: item.done });
      expect(shown[k]).toHaveTextContent(item.step);
      expect(atStart(shown[k])).toBe(true);
    });
  });

  it('should keep every dated item inside one week row of the month', () => {
    for (const item of CALENDAR_ITEMS) {
      expect(item.start).toBeGreaterThanOrEqual(1);
      expect(item.end).toBeLessThanOrEqual(CALENDAR_MONTH.days);
      expect(item.end).toBeGreaterThanOrEqual(item.start);
      expect(weekOf(item.end)).toBe(weekOf(item.start));
    }
  });

  it("should light up the Day plan's Tuesday, with a Day plan step starting there", () => {
    expect(weekdayOf(CALENDAR_HIGHLIGHT)).toBe(DAY_LABEL);
    const planned = new Set(DAY_PLAN.flatMap((g) => g.entries.map((e) => e.step)));
    expect(
      CALENDAR_ITEMS.some((item) => item.start === CALENDAR_HIGHLIGHT && planned.has(item.step))
    ).toBe(true);
  });

  it('should hold one highlight, on that day only, hidden until reveal', () => {
    const { container } = render(<CalendarTileMock />);
    const glows = container.querySelectorAll('[data-reveal-part="glow"]');

    expect(glows).toHaveLength(1);
    expect(glows[0].closest('[data-calendar-day]')).toHaveAttribute(
      'data-calendar-day',
      String(CALENDAR_HIGHLIGHT)
    );
  });

});
