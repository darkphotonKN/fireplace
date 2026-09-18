import { describe, it, expect, vi } from 'vitest';
import { loadCollapsedIds, saveCollapsedIds } from './collapsedGroups';

/**
 * FS-0007 R6: collapsed parents are remembered on this device, per plan and
 * per list scope, and a broken storage never breaks the list.
 * (vitest.setup restores storage mocks, then clears storage, after each test.)
 */

const ids = (set: ReadonlySet<string>) => [...set].sort();

describe('collapsedGroups', () => {
  it('should load back the ids saved for the same plan and scope, and nothing for another plan or scope', () => {
    saveCollapsedIds('plan-1', 'longterm', new Set(['A']), new Set(['A']));

    expect(ids(loadCollapsedIds('plan-1', 'longterm'))).toEqual(['A']);
    expect(ids(loadCollapsedIds('plan-1', 'daily'))).toEqual([]);
    expect(ids(loadCollapsedIds('plan-2', 'longterm'))).toEqual([]);
  });

  it('should prune ids that are no longer parents when saving', () => {
    // "gone" lost its last child (outdented, archived or deleted).
    saveCollapsedIds('plan-1', 'longterm', new Set(['A', 'gone']), new Set(['A', 'D']));

    expect(ids(loadCollapsedIds('plan-1', 'longterm'))).toEqual(['A']);
  });

  it.each([
    ['not JSON', 'not json'],
    ['JSON that is not an array', '{"a":1}'],
    ['an array holding non-strings', '["A", 2, null]'],
  ])('should treat stored %s as nothing collapsed', (_case, raw) => {
    window.localStorage.setItem('collapsedGroups:plan-1:longterm', raw);

    expect(ids(loadCollapsedIds('plan-1', 'longterm'))).toEqual([]);
  });

  it('should load nothing collapsed, without throwing, when storage throws on read', () => {
    // Private mode / blocked site data. vitest.setup restores mocks after each test.
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new DOMException('blocked', 'SecurityError');
    });

    expect(ids(loadCollapsedIds('plan-1', 'longterm'))).toEqual([]);
  });

  it('should not throw when storage throws on write', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new DOMException('full', 'QuotaExceededError');
    });

    expect(() =>
      saveCollapsedIds('plan-1', 'longterm', new Set(['A']), new Set(['A']))
    ).not.toThrow();
  });

  it('should load nothing collapsed when the storage object itself is inaccessible', () => {
    // Some browsers throw on the window.localStorage getter, not on its methods.
    vi.spyOn(window, 'localStorage', 'get').mockImplementation(() => {
      throw new DOMException('blocked', 'SecurityError');
    });

    expect(ids(loadCollapsedIds('plan-1', 'longterm'))).toEqual([]);
    expect(() =>
      saveCollapsedIds('plan-1', 'longterm', new Set(['A']), new Set(['A']))
    ).not.toThrow();
  });
});
