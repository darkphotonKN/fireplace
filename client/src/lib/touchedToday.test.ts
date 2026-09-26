import { describe, it, expect, vi, afterEach } from 'vitest';
import {
  markTouchedToday,
  wasTouchedToday,
  markGateFiredToday,
  hasGateFiredToday,
} from './touchedToday';

/**
 * FS-KSJFR R11–R15: the two day-stamps the gate reads. Days are the browser's
 * local calendar days, and a storage that is absent, garbled or hostile reads
 * as "no" rather than throwing into a render path.
 * (vitest.setup restores storage mocks, then clears storage, after each test.)
 */

const TOUCHED_KEY = 'touchedToday';
const GATE_KEY = 'gateFiredToday';

afterEach(() => {
  vi.useRealTimers();
});

describe('touchedToday', () => {
  it('should read false before anything is stamped and true after', () => {
    expect(wasTouchedToday()).toBe(false);
    expect(hasGateFiredToday()).toBe(false);

    markTouchedToday();
    markGateFiredToday();

    expect(wasTouchedToday()).toBe(true);
    expect(hasGateFiredToday()).toBe(true);
  });

  it('should keep the two stamps independent, so firing the gate is not a touch', () => {
    markGateFiredToday();

    expect(hasGateFiredToday()).toBe(true);
    expect(wasTouchedToday()).toBe(false);
  });

  it('should store the local calendar day, not the UTC one', () => {
    vi.useFakeTimers();
    // Constructed from local parts, so this is the 9th locally whatever the
    // runner's zone is. toISOString() would say the 8th or the 10th in most.
    vi.setSystemTime(new Date(2026, 2, 9, 23, 30));

    markTouchedToday();

    expect(window.localStorage.getItem(TOUCHED_KEY)).toBe('2026-03-09');
  });

  it('should turn over at local midnight, on both sides of the stamp', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2026, 2, 9, 23, 30));
    markTouchedToday();

    // Half an hour later is a new local day.
    vi.setSystemTime(new Date(2026, 2, 10, 0, 30));
    expect(wasTouchedToday()).toBe(false);

    // The small hours of the stamped day are still that day.
    vi.setSystemTime(new Date(2026, 2, 9, 0, 30));
    expect(wasTouchedToday()).toBe(true);
  });

  it('should read a stamp written yesterday as not touched', () => {
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    const [y, m, d] = [
      yesterday.getFullYear(),
      `${yesterday.getMonth() + 1}`.padStart(2, '0'),
      `${yesterday.getDate()}`.padStart(2, '0'),
    ];
    window.localStorage.setItem(TOUCHED_KEY, `${y}-${m}-${d}`);

    expect(wasTouchedToday()).toBe(false);
  });

  it.each([
    ['empty', ''],
    ['not a date at all', 'yes'],
    ['an impossible date', '2026-13-45'],
    ['JSON where a date belongs', '{"day":"2026-03-09"}'],
    ['a full ISO timestamp for today', new Date().toISOString()],
  ])('should read stored %s as not touched', (_case, raw) => {
    window.localStorage.setItem(TOUCHED_KEY, raw);

    expect(wasTouchedToday()).toBe(false);
  });

  it('should read false, without throwing, when storage throws on read', () => {
    // Private mode / blocked site data.
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new DOMException('blocked', 'SecurityError');
    });

    expect(wasTouchedToday()).toBe(false);
    expect(hasGateFiredToday()).toBe(false);
  });

  it('should not throw when storage throws on write', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new DOMException('full', 'QuotaExceededError');
    });

    expect(() => markTouchedToday()).not.toThrow();
    expect(() => markGateFiredToday()).not.toThrow();
  });

  it('should degrade to false when the storage object itself is inaccessible', () => {
    // Some browsers throw on the window.localStorage getter, not on its methods.
    vi.spyOn(window, 'localStorage', 'get').mockImplementation(() => {
      throw new DOMException('blocked', 'SecurityError');
    });

    expect(() => markTouchedToday()).not.toThrow();
    expect(() => markGateFiredToday()).not.toThrow();
    expect(wasTouchedToday()).toBe(false);
    expect(hasGateFiredToday()).toBe(false);
  });

  it('should leave the gate stamp under its own key', () => {
    markGateFiredToday();

    expect(window.localStorage.getItem(GATE_KEY)).not.toBeNull();
    expect(window.localStorage.getItem(TOUCHED_KEY)).toBeNull();
  });
});
