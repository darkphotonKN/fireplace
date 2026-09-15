import { describe, it, expect } from 'vitest';
import type { ChecklistItem } from '@/services/api';
import { groupChecklistItems } from './nesting';

const task = (id: string, parentId?: string): ChecklistItem => ({
  id,
  description: id,
  done: false,
  type: 'task',
  parentId,
});

const note = (id: string, parentId?: string): ChecklistItem => ({
  ...task(id, parentId),
  type: 'note',
});

// Reduce groups to ids so failures read as a shape, not a wall of objects.
const shape = (items: ChecklistItem[]) =>
  groupChecklistItems(items).map((g) => [g.item.id, g.children.map((c) => c.id)]);

describe('groupChecklistItems', () => {
  it('should put each child under its parent, notes included, when the parent is present', () => {
    expect(shape([task('A'), task('B', 'A'), note('C', 'A'), task('D')])).toEqual([
      ['A', ['B', 'C']],
      ['D', []],
    ]);
  });

  it('should render children at top level with no children of their own when the parent is filtered out', () => {
    // The Notes filter removed task A; its note C and task B remain.
    expect(shape([task('B', 'A'), note('C', 'A'), task('D')])).toEqual([
      ['B', []],
      ['C', []],
      ['D', []],
    ]);
  });

  it('should leave a parent with no children when every child is filtered out', () => {
    // The Checklist filter removed note C, A's only child.
    expect(shape([task('A'), task('D')])).toEqual([
      ['A', []],
      ['D', []],
    ]);
  });

  it('should return no groups for an empty list', () => {
    expect(shape([])).toEqual([]);
  });
});
