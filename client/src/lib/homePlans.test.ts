import { describe, it, expect } from 'vitest';
import {
  FOCUSED_PLAN_COUNT,
  byRecentlyUpdated,
  cardProgressOf,
  progressOf,
} from './homePlans';
import type { Plan } from '@/api/plans';
import type { ChecklistItem } from '@/api/checklists';

const plan = (id: string, updatedAt: string): Plan =>
  ({
    id,
    name: `plan ${id}`,
    focus: '',
    description: '',
    planType: 'project',
    dailyReset: false,
    userId: 'u1',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt,
  }) as Plan;

const item = (id: string, done: boolean): ChecklistItem =>
  ({
    id,
    planId: 'p1',
    description: id,
    done,
    archived: false,
    scope: 'longterm',
    type: 'task',
    sequence: '1',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  }) as ChecklistItem;

describe('byRecentlyUpdated', () => {
  it('should order plans by updatedAt, most recent first', () => {
    const ordered = byRecentlyUpdated([
      plan('old', '2026-01-01T00:00:00Z'),
      plan('new', '2026-09-01T00:00:00Z'),
      plan('mid', '2026-05-01T00:00:00Z'),
    ]);

    expect(ordered.map((p) => p.id)).toEqual(['new', 'mid', 'old']);
  });

  it('should not mutate the list it was given', () => {
    const plans = [
      plan('old', '2026-01-01T00:00:00Z'),
      plan('new', '2026-09-01T00:00:00Z'),
    ];

    byRecentlyUpdated(plans);

    expect(plans.map((p) => p.id)).toEqual(['old', 'new']);
  });

  it('should sort an unreadable updatedAt last rather than throwing', () => {
    const ordered = byRecentlyUpdated([
      plan('broken', 'not-a-date'),
      plan('real', '2026-01-01T00:00:00Z'),
    ]);

    expect(ordered.map((p) => p.id)).toEqual(['real', 'broken']);
  });

  it('should tolerate an empty list', () => {
    expect(byRecentlyUpdated([])).toEqual([]);
  });

  it('should cap the fan-out at three plans (ADR-0013)', () => {
    expect(FOCUSED_PLAN_COUNT).toBe(3);
  });
});

describe('progressOf', () => {
  it('should count done against total', () => {
    expect(progressOf([item('a', true), item('b', false), item('c', true)])).toEqual({
      done: 2,
      total: 3,
    });
  });

  it('should report 0/0 for a plan that genuinely has no items', () => {
    expect(progressOf([])).toEqual({ done: 0, total: 0 });
  });

  it('should ignore archived items, which are not part of the plan you see', () => {
    const archived = { ...item('gone', true), archived: true } as ChecklistItem;

    expect(progressOf([item('a', false), archived])).toEqual({ done: 0, total: 1 });
  });

  it('should ignore notes, which are not work (R26)', () => {
    const note = { ...item('jotting', false), type: 'note' } as ChecklistItem;
    const tickedNote = { ...item('ticked', true), type: 'note' } as ChecklistItem;

    expect(
      progressOf([item('a', true), item('b', false), note, tickedNote])
    ).toEqual({ done: 1, total: 2 });
  });

  it('should count an item with no type at all as a task', () => {
    const untyped: Partial<ChecklistItem> = { ...item('legacy', true) };
    delete untyped.type;

    expect(progressOf([untyped as ChecklistItem])).toEqual({
      done: 1,
      total: 1,
    });
  });
});

describe('cardProgressOf (FS-FSFEC R23–R26)', () => {
  const with_ = (base: ChecklistItem, extra: Partial<ChecklistItem>) =>
    ({ ...base, ...extra }) as ChecklistItem;

  it('should read milestones when the plan has any, R1 filter applied', () => {
    const progress = cardProgressOf([
      item('A', true),
      item('B', false),
      with_(item('A1', true), { parentId: 'A' }),
      with_(item('habit', false), { scope: 'daily' }),
      with_(item('gone', false), { archived: true }),
      with_(item('idea', false), { type: 'note' }),
    ]);

    expect(progress.kind).toBe('milestones');
    if (progress.kind !== 'milestones') return;
    expect(progress.reading.milestones.map((m) => m.id)).toEqual(['A', 'B']);
    expect([progress.reading.closed, progress.reading.total]).toEqual([1, 2]);
  });

  it('should fall back to done/total, unchanged, when the plan has no milestones', () => {
    const items = [
      with_(item('habit', true), { scope: 'daily' }),
      with_(item('stretch', false), { scope: 'daily' }),
      with_(item('idea', false), { type: 'note' }),
    ];

    expect(cardProgressOf(items)).toEqual({ kind: 'count', ...progressOf(items) });
    expect(cardProgressOf([])).toEqual({ kind: 'count', done: 0, total: 0 });
  });

  it('should count this visit\'s ticks from Today in the strip, as in done/total', () => {
    const items = [
      item('A', false),
      with_(item('A1', false), { parentId: 'A' }),
      with_(item('A2', true), { parentId: 'A' }),
      item('B', false),
    ];

    const before = cardProgressOf(items);
    const after = cardProgressOf(items, ['A1', 'B']);

    if (before.kind !== 'milestones' || after.kind !== 'milestones') {
      throw new Error('expected milestones');
    }
    expect(before.reading.milestones.map((m) => m.progress)).toEqual([0.5, 0]);
    expect(after.reading.milestones.map((m) => m.progress)).toEqual([1, 1]);
    expect(after.reading.closed).toBe(1);
    // The overlay is a reading; the fetched items are left as they were.
    expect(items.map((i) => i.done)).toEqual([false, false, true, false]);
  });
});
