import { describe, it, expect } from 'vitest';
import { applyOrder, moveOneStep } from '@/lib/reorder';
import type { ChecklistItem } from '@/services/api';

const item = (id: string, parentId: string | null = null): ChecklistItem => ({
  id,
  description: `item ${id}`,
  done: false,
  parentId,
});

describe('moveOneStep', () => {
  it('moves an item one place down within its sibling set', () => {
    const items = [item('a'), item('b'), item('c')];

    expect(moveOneStep(items, 'a', 'down')).toEqual(['b', 'a', 'c']);
  });

  it('refuses a move off either end of the set, so nothing is sent', () => {
    const items = [item('a'), item('b'), item('c')];

    expect(moveOneStep(items, 'c', 'down')).toBeNull();
    expect(moveOneStep(items, 'a', 'up')).toBeNull();
  });

  it('moves a child within its own parent, never across families', () => {
    // Interleaved on purpose: the server returns one flat sequence, so a
    // parent's children are not necessarily adjacent in the array.
    const items = [
      item('p'),
      item('p1', 'p'),
      item('q'),
      item('q1', 'q'),
      item('p2', 'p'),
      item('q2', 'q'),
    ];

    expect(moveOneStep(items, 'p2', 'up')).toEqual(['p2', 'p1']);
  });

  it('steps past what is on screen, and returns the whole set anyway', () => {
    // A filter showing only the notes. Stepping N1 one place in the FULL set
    // would swap it with the hidden T2 and change nothing the user can see,
    // so the step is taken among the visible siblings — while the order that
    // comes back still names every member, hidden ones included (R7.2).
    const items = [item('T1'), item('N1'), item('T2'), item('N2')];
    const visible = new Set(['N1', 'N2']);

    expect(moveOneStep(items, 'N1', 'down', visible)).toEqual([
      'T1',
      'N2',
      'T2',
      'N1',
    ]);
  });

  it('offers no move past the last visible sibling, whatever is hidden below', () => {
    const items = [item('N1'), item('T1')];

    expect(moveOneStep(items, 'N1', 'down', new Set(['N1']))).toBeNull();
  });

  it('has no move to make in a set of one', () => {
    const items = [item('only'), item('child', 'only')];

    expect(moveOneStep(items, 'only', 'up')).toBeNull();
    expect(moveOneStep(items, 'only', 'down')).toBeNull();
    expect(moveOneStep(items, 'child', 'down')).toBeNull();
  });
});

describe('applyOrder', () => {
  it('re-seats the set into the slots it already occupies, leaving the rest alone', () => {
    const items = [
      item('p'),
      item('p1', 'p'),
      item('q'),
      item('q1', 'q'),
      item('p2', 'p'),
    ];

    const next = applyOrder(items, ['p2', 'p1']);

    // p1/p2 swap; the three non-members keep both their identity and their slot.
    expect(next.map((i) => i.id)).toEqual(['p', 'p2', 'q', 'q1', 'p1']);
  });

  it('leaves the input array untouched', () => {
    const items = [item('a'), item('b')];

    applyOrder(items, ['b', 'a']);

    expect(items.map((i) => i.id)).toEqual(['a', 'b']);
  });
});
