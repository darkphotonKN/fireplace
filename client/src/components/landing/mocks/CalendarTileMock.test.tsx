import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { render } from '@testing-library/react';
import { renderToString } from 'react-dom/server';
import { expectMockContract } from '@/test/mockContract';
import { REVEAL_MS } from '../bento/BentoTile';
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

/** A part's stagger, in ms (0 when it has none). */
function delayOf(el: Element) {
  return parseInt((el as HTMLElement).style.getPropertyValue('--reveal-delay') || '0', 10);
}

const steps = new Map(
  [DATABASE_MILESTONE, MICROSERVICES_MILESTONE].flatMap((m) =>
    m.steps.map((s) => [s.label, { milestone: m.title, done: s.done }] as const)
  )
);

describe('CalendarTileMock', () => {
  it('should honour the mock contract', () => {
    const { container } = render(<CalendarTileMock />);

    expectMockContract(container);
  });

  it('should render the same month every time, server and client alike', () => {
    expect(renderToString(<CalendarTileMock />)).toBe(renderToString(<CalendarTileMock />));
  });

  it('should take its month from content.ts, never from the clock', () => {
    for (const file of ['CalendarTileMock.tsx', 'content.ts']) {
      const source = readFileSync(resolve(__dirname, file), 'utf8');
      expect(source).not.toMatch(/\bDate\b/);
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

  it('should finish its longest stagger inside the tile reveal window', () => {
    const css = readFileSync(resolve(__dirname, '../bento/bento.module.css'), 'utf8');
    const revealMs = parseInt(/--reveal-ms:\s*(\d+)ms/.exec(css)![1], 10);
    const { container } = render(<CalendarTileMock />);

    const longest = Math.max(...Array.from(container.querySelectorAll('[data-reveal-part]'), delayOf));
    expect(longest + revealMs).toBeLessThanOrEqual(REVEAL_MS);
  });

  it('should import no product component, context, service layer or fetch', () => {
    const source = readFileSync(resolve(__dirname, 'CalendarTileMock.tsx'), 'utf8');

    expect(source).not.toMatch(/from '@\/components\/(?!landing)/);
    expect(source).not.toMatch(/@\/(context|api|services|lib\/api|lib\/calendar)/);
    expect(source).not.toMatch(/\bfetch\(|useContext|date-fns/);
  });
});
