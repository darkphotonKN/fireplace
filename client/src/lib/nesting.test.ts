import { describe, it, expect } from 'vitest';
import type { ChecklistItem } from '@/services/api';
import {
  findAddBarTarget,
  groupChecklistItems,
  resolveAddBarTarget,
} from './nesting';

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

const targetOf = (items: ChecklistItem[]) =>
  findAddBarTarget(groupChecklistItems(items))?.id ?? null;

describe('findAddBarTarget', () => {
  it('should resolve a trailing child row to its parent', () => {
    expect(targetOf([task('A'), task('B', 'A')])).toBe('A');
  });

  it('should skip a trailing top-level note for the task above it', () => {
    expect(targetOf([task('A'), note('N')])).toBe('A');
  });

  it('should find no target in a list of only notes', () => {
    expect(targetOf([note('N'), note('M')])).toBeNull();
  });

  it('should find no target in an empty list', () => {
    expect(targetOf([])).toBeNull();
  });

  it('should never target an optimistic row that has no real id yet', () => {
    expect(targetOf([task('A'), task('insight_1'), task('fallback_2')])).toBe('A');
  });

  it('should never target a child shown at top level because its parent is filtered out', () => {
    // A (task) is filtered out; its child B falls through as a group head.
    expect(targetOf([task('D'), task('B', 'A')])).toBe('D');
  });
});

const resolved = (items: ChecklistItem[], id: string | null) =>
  resolveAddBarTarget(groupChecklistItems(items), id)?.id ?? null;

describe('resolveAddBarTarget', () => {
  it('should keep a target that is still a rendered top-level task, even when not last', () => {
    expect(resolved([task('A'), task('B', 'A'), task('D')], 'A')).toBe('A');
  });

  it.each([
    ['nothing is targeted', [task('A')], null],
    ['the target is gone (archived, deleted or filtered out)', [task('D')], 'A'],
    ['the target became a note', [note('A')], 'A'],
    ['the target was nested under another row', [task('D'), task('A', 'D')], 'A'],
  ])('should drop the target when %s', (_case, items, id) => {
    expect(resolved(items, id)).toBeNull();
  });
});
