import { describe, it, expect, vi } from 'vitest';
import { announcePlanItemsChanged, onPlanItemsChanged } from './planItemsChanged';

describe('planItemsChanged', () => {
  it('tells listeners of the same plan, and only them', () => {
    const mine = vi.fn();
    const other = vi.fn();
    const offMine = onPlanItemsChanged('plan-1', mine);
    const offOther = onPlanItemsChanged('plan-2', other);

    announcePlanItemsChanged('plan-1');

    expect(mine).toHaveBeenCalledTimes(1);
    expect(other).not.toHaveBeenCalled();
    offMine();
    offOther();
  });

  it('stops telling a listener once it unsubscribes', () => {
    const listener = vi.fn();
    const off = onPlanItemsChanged('plan-1', listener);
    off();

    announcePlanItemsChanged('plan-1');

    expect(listener).not.toHaveBeenCalled();
  });
});
