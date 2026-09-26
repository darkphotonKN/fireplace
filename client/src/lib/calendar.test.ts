import { describe, it, expect } from 'vitest';

import { layoutItem, parseISODate, resolveWindow } from './calendar';
import { parseDateOnly } from './itemDates';
import type { CalendarItem } from '@/services/api';

/**
 * What the plan calendar makes of an item's dates (I-0060).
 *
 * The fixtures here are RFC3339, because that is the shape a `*time.Time`
 * reaches the client in. Every calendar fixture written before this file was
 * hand-typed `YYYY-MM-DD` — the shape the client *sends* — which is why a
 * strict `parse(s, "yyyy-MM-dd")` looked correct for as long as it did.
 *
 * The rule under test is I-0059's, unchanged: the date portion is the calendar
 * day, and the time and zone that follow are ignored rather than converted.
 * These assertions read local fields (`getDate()`), so they mean the same thing
 * under `TZ=UTC` and under a zone west or east of Greenwich.
 */

const item = (extra: Partial<CalendarItem> = {}): CalendarItem =>
  ({
    id: 'i1',
    description: 'Learn Go',
    scope: 'longterm',
    done: false,
    startDate: '',
    dueDate: '',
    ...extra,
  }) as CalendarItem;

const march = resolveWindow('month', '2026-03');

describe('parseISODate', () => {
  it('should read a timestamp as its date portion, as that day locally', () => {
    const d = parseISODate('2026-03-16T00:00:00Z')!;
    expect([d.getFullYear(), d.getMonth(), d.getDate()]).toEqual([2026, 2, 16]);
    // Local midnight, matching the days resolveWindow hands out.
    expect([d.getHours(), d.getMinutes()]).toEqual([0, 0]);
  });

  it('should ignore the time and the zone rather than converting them', () => {
    // 02:00Z on the 17th is the evening of the 16th in Los Angeles, and
    // 23:59+05:30 is still the 17th's morning in UTC. Neither moves the day:
    // the 17th is the day the user picked.
    expect(parseISODate('2026-03-17T02:00:00Z')!.getDate()).toBe(17);
    expect(parseISODate('2026-03-17T23:59:59+05:30')!.getDate()).toBe(17);
  });

  it('should still read the bare date-only string the client sends', () => {
    const d = parseISODate('2026-03-16')!;
    expect([d.getFullYear(), d.getMonth(), d.getDate()]).toEqual([2026, 2, 16]);
  });

  it.each(['', 'nonsense', 'T00:00:00Z', null, undefined])(
    'should read %s as no date at all',
    (value) => {
      expect(parseISODate(value as string | null)).toBeNull();
    }
  );

  it.each([
    '2026-03-16',
    '2026-03-16T00:00:00Z',
    '2026-03-16T23:30:00+09:00',
    '',
    'nonsense',
  ])('should agree with itemDates about %s, not carry its own rule', (value) => {
    // The point of the fix: one rule, one implementation. If these ever
    // diverge again, the calendar and the date chip disagree about which day
    // an item is on — which is the defect, not a detail of it.
    expect(parseISODate(value)?.getTime() ?? null).toBe(
      parseDateOnly(value)?.getTime() ?? null
    );
  });
});

describe('layoutItem — the format the API actually sends (I-0060)', () => {
  it('should draw a chip for a one-day item carrying RFC3339 dates', () => {
    const laid = layoutItem(
      item({ startDate: '2026-03-16T00:00:00Z', dueDate: '2026-03-16T00:00:00Z' }),
      march
    );
    // Before the fix both dates parsed to null and this was "none": the
    // calendar rendered nothing for any item the API returned.
    expect(laid.shape).toBe('chip');
    expect(laid.visibleStart.getDate()).toBe(16);
    expect(laid.visibleEnd.getDate()).toBe(16);
  });

  it('should draw a bar across an RFC3339 range', () => {
    const laid = layoutItem(
      item({ startDate: '2026-03-09T00:00:00Z', dueDate: '2026-03-16T00:00:00Z' }),
      march
    );
    expect(laid.shape).toBe('bar');
    expect(laid.visibleStart.getDate()).toBe(9);
    expect(laid.visibleEnd.getDate()).toBe(16);
    expect([laid.clipsLeft, laid.clipsRight]).toEqual([false, false]);
  });

  it('should draw a chip from one end alone, whichever end it is', () => {
    expect(layoutItem(item({ startDate: '2026-03-16T00:00:00Z' }), march).shape).toBe(
      'chip'
    );
    expect(layoutItem(item({ dueDate: '2026-03-16T00:00:00Z' }), march).shape).toBe(
      'chip'
    );
  });

  it('should clip an RFC3339 range to the visible window at either end', () => {
    const laid = layoutItem(
      item({ startDate: '2026-02-20T00:00:00Z', dueDate: '2026-04-05T00:00:00Z' }),
      march
    );
    expect(laid.shape).toBe('bar');
    expect([laid.clipsLeft, laid.clipsRight]).toEqual([true, true]);
    expect(laid.visibleStart.getTime()).toBe(march.start.getTime());
    expect(laid.visibleEnd.getTime()).toBe(march.end.getTime());
  });

  it('should treat a timestamp and a bare date for the same day as one day', () => {
    // A mixed pair is what a wire value and a just-saved local value look like
    // side by side. They have to land on the same midnight, or a one-day item
    // silently becomes a two-day bar.
    const laid = layoutItem(
      item({ startDate: '2026-03-16', dueDate: '2026-03-16T00:00:00Z' }),
      march
    );
    expect(laid.shape).toBe('chip');
  });

  it('should still render nothing for an item with no dates at all', () => {
    expect(layoutItem(item(), march).shape).toBe('none');
  });
});
