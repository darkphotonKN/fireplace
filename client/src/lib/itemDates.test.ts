import { describe, it, expect, vi, afterEach } from 'vitest';
import {
  formatDateRange,
  isDueToday,
  isRangePast,
  itemStartDate,
  parseDateOnly,
  toDateOnly,
} from './itemDates';

describe('itemDates', () => {
  afterEach(() => vi.useRealTimers());

  it('should read a date-only string as that day locally, not as UTC midnight', () => {
    const d = parseDateOnly('2026-03-09')!;
    expect(d.getFullYear()).toBe(2026);
    expect(d.getMonth()).toBe(2);
    // `new Date('2026-03-09')` is midnight UTC, which is the 8th in every
    // western zone — the whole reason this helper exists.
    expect(d.getDate()).toBe(9);
  });

  it('should write a date back as the day it is locally', () => {
    // Late evening, where toISOString() would roll over to the next day.
    expect(toDateOnly(new Date(2026, 2, 9, 23, 30))).toBe('2026-03-09');
  });

  it.each([null, undefined, '', 'nonsense'])(
    'should read %s as no date at all',
    (value) => {
      expect(parseDateOnly(value as string | null)).toBeNull();
    }
  );

  it('should show one day when only one end is set, from either end', () => {
    vi.setSystemTime(new Date(2026, 0, 1));
    expect(formatDateRange('2026-03-09', null)).toBe('Mar 9');
    expect(formatDateRange(null, '2026-03-09')).toBe('Mar 9');
    expect(formatDateRange('2026-03-09', '2026-03-09')).toBe('Mar 9');
  });

  it('should show both ends of a real range', () => {
    vi.setSystemTime(new Date(2026, 0, 1));
    expect(formatDateRange('2026-03-09', '2026-03-16')).toBe('Mar 9 – Mar 16');
  });

  it('should name the year only when it is not this one', () => {
    vi.setSystemTime(new Date(2026, 0, 1));
    expect(formatDateRange('2027-03-09', null)).toBe('Mar 9, 2027');
  });

  it('should have nothing to show when neither end is set', () => {
    expect(formatDateRange(null, null)).toBeNull();
    expect(formatDateRange(undefined, undefined)).toBeNull();
  });

  it('should call a range past only once its last day has gone', () => {
    vi.setSystemTime(new Date(2026, 2, 16, 9, 0));
    // Today is the last day: still live, all day.
    expect(isRangePast('2026-03-09', '2026-03-16')).toBe(false);
    expect(isRangePast('2026-03-09', '2026-03-15')).toBe(true);
    // A start alone is its own last day.
    expect(isRangePast('2026-03-16', null)).toBe(false);
    expect(isRangePast(null, null)).toBe(false);
  });

  it('should call a range due today only on its last day', () => {
    vi.setSystemTime(new Date(2026, 2, 16, 9, 0));
    expect(isDueToday('2026-03-09', '2026-03-16')).toBe(true);
    // Mid-range is not due yet, and a finished range is not due any more.
    expect(isDueToday('2026-03-09', '2026-03-17')).toBe(false);
    expect(isDueToday('2026-03-09', '2026-03-15')).toBe(false);
    // A start alone is its own last day; no dates at all is never due.
    expect(isDueToday('2026-03-16', null)).toBe(true);
    expect(isDueToday(null, '2026-03-16')).toBe(true);
    expect(isDueToday(null, null)).toBe(false);
  });

  it.each([
    ['just after midnight', new Date(2026, 2, 16, 0, 1)],
    ['midday', new Date(2026, 2, 16, 12, 0)],
    ['late evening', new Date(2026, 2, 16, 23, 59)],
  ])(
    'should hold the due-today fence at %s wherever the machine sits',
    (_when, now) => {
      vi.setSystemTime(now);
      // Both sides are local, so this reads the same in UTC, in Los Angeles
      // (where `new Date('2026-03-16')` is still the 15th) and in Kolkata.
      expect(isDueToday(null, '2026-03-16')).toBe(true);
      expect(isDueToday(null, '2026-03-15')).toBe(false);
      expect(isDueToday(null, '2026-03-17')).toBe(false);
      expect(isRangePast(null, '2026-03-16')).toBe(false);
    }
  );

  it('should never call the same range both due today and past', () => {
    vi.setSystemTime(new Date(2026, 2, 16, 9, 0));
    for (const day of ['2026-03-15', '2026-03-16', '2026-03-17']) {
      expect(isDueToday(null, day) && isRangePast(null, day)).toBe(false);
    }
  });

  it('should fall back to scheduledTime only where startDate is absent', () => {
    expect(
      itemStartDate({ startDate: '2026-03-09', scheduledTime: '2026-03-01T10:00:00Z' })
    ).toBe('2026-03-09');
    // The deprecated mirror carries a timestamp; only its date half is a date.
    expect(itemStartDate({ scheduledTime: '2026-03-01T10:00:00Z' })).toBe('2026-03-01');
    expect(itemStartDate({})).toBeNull();
  });
});

describe('itemDates — the format the API actually sends (I-0059)', () => {
  afterEach(() => vi.useRealTimers());

  // The contract publishes startDate/dueDate as `format: date-time`: the
  // server stores a *time.Time and Go marshals it as RFC3339, so this is the
  // string production hands these helpers. Every fixture above is the shape
  // the client *sends*, which is why the old parser looked correct.
  const TODAY = '2026-03-16T00:00:00Z';
  const YESTERDAY = '2026-03-15T00:00:00Z';
  const TOMORROW = '2026-03-17T00:00:00Z';

  it('should read a timestamp as its date portion, as that day locally', () => {
    const d = parseDateOnly(TODAY)!;
    expect([d.getFullYear(), d.getMonth(), d.getDate()]).toEqual([2026, 2, 16]);
    // And the midnight is local, so it compares against a local today.
    expect([d.getHours(), d.getMinutes()]).toEqual([0, 0]);
  });

  it('should ignore the time and the zone rather than converting them', () => {
    // 02:00Z on the 17th is the evening of the 16th in Los Angeles. The rule
    // is that the date portion wins: this is the 17th everywhere, because the
    // 17th is the day the user picked.
    const d = parseDateOnly('2026-03-17T02:00:00Z')!;
    expect(d.getDate()).toBe(17);
    expect(parseDateOnly('2026-03-17T23:59:59+05:30')!.getDate()).toBe(17);
  });

  it('should still read a bare date-only string', () => {
    expect(parseDateOnly('2026-03-16')!.getDate()).toBe(16);
  });

  it.each(['', 'nonsense', 'T00:00:00Z', null, undefined])(
    'should still read %s as no date at all',
    (value) => {
      expect(parseDateOnly(value as string | null)).toBeNull();
    }
  );

  it('should call an item due today when its RFC3339 due date is today', () => {
    vi.setSystemTime(new Date(2026, 2, 16, 9, 0));
    expect(isDueToday(null, TODAY)).toBe(true);
    expect(isDueToday(YESTERDAY, TODAY)).toBe(true);
    expect(isDueToday(null, TOMORROW)).toBe(false);
    expect(isDueToday(null, YESTERDAY)).toBe(false);
  });

  it('should call an RFC3339 range past only once its last day has gone', () => {
    vi.setSystemTime(new Date(2026, 2, 16, 9, 0));
    expect(isRangePast(null, YESTERDAY)).toBe(true);
    expect(isRangePast(null, TODAY)).toBe(false);
    expect(isRangePast(TODAY, null)).toBe(false);
  });

  it('should label an RFC3339 range, so the date chip has something to render', () => {
    vi.setSystemTime(new Date(2026, 2, 16, 9, 0));
    expect(formatDateRange(null, TODAY)).toBe('Mar 16');
    expect(formatDateRange(YESTERDAY, TOMORROW)).toBe('Mar 15 – Mar 17');
    expect(formatDateRange('2027-03-09T00:00:00Z', null)).toBe('Mar 9, 2027');
  });

  it('should hold the fence at every hour of the local day', () => {
    // The helpers are compared against a LOCAL today, so the boundary has to
    // sit at local midnight wherever the machine is. Run under TZ=UTC and
    // under a non-UTC zone: both sides of every comparison are local.
    for (const now of [
      new Date(2026, 2, 16, 0, 1),
      new Date(2026, 2, 16, 12, 0),
      new Date(2026, 2, 16, 23, 59),
    ]) {
      vi.setSystemTime(now);
      expect(isDueToday(null, TODAY)).toBe(true);
      expect(isDueToday(null, TOMORROW)).toBe(false);
      expect(isRangePast(null, TODAY)).toBe(false);
      expect(isRangePast(null, YESTERDAY)).toBe(true);
    }
  });

  it('should read startDate and the scheduledTime fallback by the same rule', () => {
    const start = itemStartDate({ scheduledTime: '2026-03-16T23:00:00Z' });
    expect(start).toBe('2026-03-16');
    vi.setSystemTime(new Date(2026, 2, 16, 9, 0));
    // The fallback's date portion and a startDate's date portion now reach
    // the same day — the exception this file used to carry is gone.
    expect(isDueToday(start, null)).toBe(true);
    expect(isDueToday(itemStartDate({ startDate: '2026-03-16T23:00:00Z' }), null)).toBe(
      true
    );
  });
});
