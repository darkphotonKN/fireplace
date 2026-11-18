import { describe, it, expect } from 'vitest';
import { fullDateTime, relativeTime } from './relativeTime';

/** A comment's time, in the bands FS-65B3N R26 sets. Local time throughout. */
const NOW = new Date(2026, 10, 19, 15, 30, 0); // Thu Nov 19 2026, 15:30
const ago = (ms: number) => new Date(NOW.getTime() - ms);
const SEC = 1000;
const MIN = 60 * SEC;
const HOUR = 60 * MIN;

describe('relativeTime (FS-65B3N R26)', () => {
  it.each([
    ['now', ago(0), 'just now'],
    ['59 seconds', ago(59 * SEC), 'just now'],
    ['a moment in the future (clock skew)', ago(-30 * SEC), 'just now'],
    ['one minute', ago(MIN), '1 min ago'],
    ['59 minutes', ago(59 * MIN + 59 * SEC), '59 min ago'],
    ['one hour', ago(HOUR), '1 h ago'],
    ['23 hours', ago(23 * HOUR + 59 * MIN), '23 h ago'],
    ['yesterday afternoon', new Date(2026, 10, 18, 14, 0), 'yesterday'],
    ['two days', new Date(2026, 10, 17, 9, 0), '2 days ago'],
    ['six days', new Date(2026, 10, 13, 23, 0), '6 days ago'],
    ['a week', new Date(2026, 10, 12, 15, 30), 'Nov 12'],
    ['earlier this year', new Date(2026, 2, 3, 8, 0), 'Mar 3'],
    ['another year', new Date(2025, 10, 3, 8, 0), 'Nov 3, 2025'],
  ])('should read %s as "%s"', (_why, at, expected) => {
    expect(relativeTime(at, NOW)).toBe(expected);
  });

  it('should take an ISO string as well as a date', () => {
    expect(relativeTime(ago(5 * MIN).toISOString(), NOW)).toBe('5 min ago');
  });
});

describe('fullDateTime (FS-65B3N R26)', () => {
  it('should give the full date and time', () => {
    expect(fullDateTime(new Date(2026, 10, 3, 16, 5))).toBe('Nov 3, 2026, 4:05 PM');
  });
});
