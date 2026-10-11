import { describe, it, expect } from 'vitest';
import {
  isComplete,
  readMilestones,
  type MilestoneSourceItem,
} from './milestones';

const item = (
  id: string,
  extra: Partial<MilestoneSourceItem> = {}
): MilestoneSourceItem => ({
  id,
  description: `item ${id}`,
  done: false,
  scope: 'longterm',
  type: 'task',
  parentId: null,
  ...extra,
});

// A plan with everything that is not a milestone mixed in among the ones that
// are, in an order the reading has to keep.
const MIXED = [
  item('N0', { type: 'note' }), // top-level note: not a milestone (R5)
  item('M1'),
  item('M1a', { parentId: 'M1', done: true }),
  item('M1b', { parentId: 'M1', done: true }),
  item('M1c', { parentId: 'M1', done: true }),
  item('M1d', { parentId: 'M1' }),
  item('M1n', { parentId: 'M1', type: 'note' }), // note step: never counts (R1)
  item('A0', { archived: true }), // archived: out of the plan
  item('D0', { scope: 'daily' }), // daily: recurs, not a milestone (D12)
  item('M2'),
  item('M2n', { parentId: 'M2', type: 'note' }), // all-notes children (R7)
  item('M3', { done: true }),
  item('M3a', { parentId: 'M3' }), // closed early, steps undone (R8)
  item('M4'),
  item('M4a', { parentId: 'M4', done: true }),
  item('M4x', { parentId: 'M4', archived: true }), // archived step (R6)
];

describe('readMilestones', () => {
  it('should read only top-level, non-archived, long-term tasks, in list order', () => {
    const { milestones } = readMilestones(MIXED);
    expect(milestones.map((m) => m.id)).toEqual(['M1', 'M2', 'M3', 'M4']);
    expect(milestones[0].title).toBe('item M1');
  });

  it.each([
    // id, steps, progress, why
    ['M1', { done: 3, total: 4 }, 0.75, 'task steps only; the note step changes nothing'],
    ['M2', { done: 0, total: 0 }, 0, 'only note children: empty until done (R7)'],
    ['M3', { done: 0, total: 1 }, 1, 'closed early reads full whatever its steps say (R8)'],
    ['M4', { done: 1, total: 1 }, 1, 'an archived step is out of the count (R6)'],
  ])('should read %s at its own progress', (id, steps, progress) => {
    const m = readMilestones(MIXED).milestones.find((x) => x.id === id)!;
    expect(m.steps).toEqual(steps);
    expect(m.progress).toBe(progress);
  });

  it('should read a no-step milestone empty until it is done and full after', () => {
    const open = readMilestones([item('M')]).milestones[0];
    const done = readMilestones([item('M', { done: true })]).milestones[0];
    expect([open.progress, done.progress]).toEqual([0, 1]);
  });

  it('should count closed milestones toward N, a full one not yet closed not', () => {
    // M4's steps are all done but nobody closed it; M3 was closed early.
    expect(readMilestones(MIXED)).toMatchObject({ closed: 1, total: 4 });
  });

  it('should read a plan of only notes as no milestones at all', () => {
    const reading = readMilestones([
      item('N1', { type: 'note' }),
      item('N2', { type: 'note' }),
    ]);
    expect(reading).toEqual({ milestones: [], closed: 0, total: 0 });
  });
});

describe('isComplete', () => {
  // M1 3/4, M2 notes only, M3 closed early, M4 1/1 with an archived step.
  const byId = (id: string) =>
    readMilestones(MIXED).milestones.find((m) => m.id === id)!;

  it('should call a milestone complete once every task step is done and it is still open (R10)', () => {
    expect(isComplete(byId('M4'))).toBe(true);
    expect(isComplete(byId('M1'))).toBe(false);
  });

  it('should never call a milestone with no task steps complete (R7)', () => {
    expect(isComplete(byId('M2'))).toBe(false);
  });

  it('should not call a closed milestone complete: it has nothing left to offer (R10)', () => {
    const closedFull = readMilestones([
      item('M', { done: true }),
      item('Ma', { parentId: 'M', done: true }),
    ]).milestones[0];
    expect(isComplete(closedFull)).toBe(false);
    expect(isComplete(byId('M3'))).toBe(false);
  });
});
