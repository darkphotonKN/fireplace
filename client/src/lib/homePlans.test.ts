import { describe, it, expect } from 'vitest';
import {
  FOCUSED_PLAN_COUNT,
  byRecentlyUpdated,
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
