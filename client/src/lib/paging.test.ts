import { describe, it, expect } from 'vitest';
import type { ChecklistItem } from '@/services/api';
import type { ChecklistGroup } from './nesting';
import { PAGE_SIZE, clampPage, pageCount, pageOfItem, pageSlice } from './paging';

const task = (id: string, parentId?: string): ChecklistItem => ({
  id,
  description: id,
  done: false,
  type: 'task',
  parentId,
});

// n top-level groups named 1..n, each with `childCount` children.
const groups = (n: number, childCount = 0): ChecklistGroup[] =>
  Array.from({ length: n }, (_, i) => ({
    item: task(`g${i + 1}`),
    children: Array.from({ length: childCount }, (_, c) =>
      task(`g${i + 1}c${c + 1}`, `g${i + 1}`)
    ),
  }));

const heads = (gs: ChecklistGroup[]) => gs.map((g) => g.item.id);

describe('pageCount', () => {
  it('should split 25 top-level items into three pages of ten', () => {
    expect(pageCount(25)).toBe(3);
  });

  it('should keep exactly one page at the page size, and at zero', () => {
    expect(pageCount(PAGE_SIZE)).toBe(1);
    expect(pageCount(0)).toBe(1);
  });
});

describe('clampPage', () => {
  it('should pull a page past the end back to the last page', () => {
    expect(clampPage(4, 25)).toBe(3);
  });

  it('should leave a page inside the range alone and floor below one', () => {
    expect(clampPage(2, 25)).toBe(2);
    expect(clampPage(0, 25)).toBe(1);
  });
});

describe('pageSlice', () => {
  it('should cut ten top-level groups per page, the last page short', () => {
    const all = groups(25);
    expect(heads(pageSlice(all, 1))).toEqual(heads(all).slice(0, 10));
    expect(heads(pageSlice(all, 3))).toEqual(heads(all).slice(20));
  });

  it('should never split a family, however many children it holds', () => {
    // Ten groups of eight children each: page 1 takes all ten groups whole.
    const page = pageSlice(groups(10, 8), 1);
    expect(page).toHaveLength(10);
    expect(page.every((g) => g.children.length === 8)).toBe(true);
  });

  it('should give nothing for a page past the end', () => {
    expect(pageSlice(groups(25), 4)).toEqual([]);
  });
});

describe('pageOfItem', () => {
  it('should find the page of a top-level item', () => {
    expect(pageOfItem(groups(25), 'g1')).toBe(1);
    expect(pageOfItem(groups(25), 'g11')).toBe(2);
    expect(pageOfItem(groups(25), 'g25')).toBe(3);
  });

  it("should find a child on its parent's page", () => {
    expect(pageOfItem(groups(25, 2), 'g11c2')).toBe(2);
  });

  it('should give null for an id that is nowhere in the set', () => {
    expect(pageOfItem(groups(25), 'nope')).toBeNull();
  });
});
