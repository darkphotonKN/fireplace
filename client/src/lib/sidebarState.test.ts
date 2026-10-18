import { describe, it, expect, vi, beforeEach } from 'vitest';
import { parseSidebarState, SIDEBAR_HEAD_SCRIPT, subscribeSidebarState } from './sidebarState';

/**
 * The remembered open/hidden choice (FS-S6WVY R1, R2, R4, R5).
 *
 * Only an explicit `hidden` hides the sidebar. Anything else, including the
 * legacy `sidebarCollapsed` values that the old home effect wrote for nearly
 * everyone, means "no preference", which is open.
 */
const cases = [
  ['missing', null, 'open'],
  ['open', 'open', 'open'],
  ['hidden', 'hidden', 'hidden'],
  ['a legacy "true"', 'true', 'open'],
  ['a legacy "false"', 'false', 'open'],
  ['garbage', 'sideways', 'open'],
  ['the wrong case', 'Hidden', 'open'],
] as const;

describe('parseSidebarState', () => {
  it.each(cases)('should read %s (%s) as %s', (_label, raw, expected) => {
    expect(parseSidebarState(raw)).toBe(expected);
  });
});

/** Runs the head script the way the browser does, before React exists. */
const runHeadScript = () => new Function(SIDEBAR_HEAD_SCRIPT)();
const paintedState = () => document.documentElement.getAttribute('data-sidebar');

describe('the pre-paint head script (R5)', () => {
  beforeEach(() => document.documentElement.removeAttribute('data-sidebar'));

  it.each(cases)('should paint %s the same way the parse rule reads it', (_label, raw, expected) => {
    if (raw !== null) localStorage.setItem('sidebar', raw);

    runHeadScript();

    expect(paintedState()).toBe(expected);
  });

  it('should ignore the legacy sidebarCollapsed key', () => {
    localStorage.setItem('sidebarCollapsed', 'true');

    runHeadScript();

    expect(paintedState()).toBe('open');
  });

  it('should paint open, not throw, when storage is blocked', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('SecurityError');
    });

    expect(runHeadScript).not.toThrow();
    expect(paintedState()).toBe('open');
  });

  it('should never write storage', () => {
    const setItem = vi.spyOn(Storage.prototype, 'setItem');
    const removeItem = vi.spyOn(Storage.prototype, 'removeItem');
    localStorage.setItem('sidebarCollapsed', 'true');
    setItem.mockClear();

    runHeadScript();

    expect(setItem).not.toHaveBeenCalled();
    expect(removeItem).not.toHaveBeenCalled();
  });
});

describe('following other tabs (R7)', () => {
  const otherTabWrites = (value: string) =>
    window.dispatchEvent(new StorageEvent('storage', { key: 'sidebar', newValue: value }));

  it('should notify each subscriber once per change, however many there are', () => {
    const first = vi.fn();
    const second = vi.fn();
    const unsubscribeFirst = subscribeSidebarState(first);
    const unsubscribeSecond = subscribeSidebarState(second);

    otherTabWrites('hidden');

    expect(first).toHaveBeenCalledTimes(1);
    expect(second).toHaveBeenCalledTimes(1);
    expect(document.documentElement.getAttribute('data-sidebar')).toBe('hidden');
    unsubscribeFirst();
    unsubscribeSecond();
  });

  it('should keep following while any subscriber remains, and stop after the last leaves', () => {
    const first = vi.fn();
    const second = vi.fn();
    const unsubscribeFirst = subscribeSidebarState(first);
    const unsubscribeSecond = subscribeSidebarState(second);

    unsubscribeFirst();
    otherTabWrites('open');
    expect(second).toHaveBeenCalledTimes(1);
    expect(document.documentElement.getAttribute('data-sidebar')).toBe('open');

    unsubscribeSecond();
    otherTabWrites('hidden');
    expect(second).toHaveBeenCalledTimes(1);
    expect(document.documentElement.getAttribute('data-sidebar')).toBe('open');
  });
});
