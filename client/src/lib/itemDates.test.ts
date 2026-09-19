import { describe, it, expect, vi, afterEach } from 'vitest';
import {
  formatDateRange,
  isRangePast,
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
});
