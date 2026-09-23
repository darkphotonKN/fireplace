import { describe, it, expect } from 'vitest';
import { applyOrder, moveOneStep, moveOnto } from '@/lib/reorder';
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

describe('moveOnto', () => {
  it('lands the dragged item in the place of the sibling it was dropped on', () => {
    const items = [item('a'), item('b'), item('c')];

    // c picked up, dropped on a's row: c takes the top, a and b shuffle down.
    expect(moveOnto(items, 'c', 'a')).toEqual(['c', 'a', 'b']);
    // The other direction reads the same way: a goes where c was.
    expect(moveOnto(items, 'a', 'c')).toEqual(['b', 'c', 'a']);
  });

  it('refuses a drop outside the dragged item\'s own sibling set (R5.1)', () => {
    const items = [item('p'), item('p1', 'p'), item('q'), item('q1', 'q')];

    // A drag never re-parents: dropping p1 on q's row, or on q's child, is
    // not a reorder of anything, so there is no order to write.
    expect(moveOnto(items, 'p1', 'q')).toBeNull();
    expect(moveOnto(items, 'p1', 'q1')).toBeNull();
  });

  it('has nothing to write when the drop is where the drag began (R3.4)', () => {
    const items = [item('a'), item('b')];

    expect(moveOnto(items, 'a', 'a')).toBeNull();
  });

  it('lands before the visible neighbour, leaving hidden siblings in their slots (R7.3)', () => {
    // The Notes filter: only N1 and N2 are on screen, so the drop is read
    // against them — while the order that comes back still names T1 and T2
    // in the slots they already held (R7.2).
    const items = [item('T1'), item('N1'), item('T2'), item('N2')];
    const visible = new Set(['N1', 'N2']);

    expect(moveOnto(items, 'N2', 'N1', visible)).toEqual([
      'T1',
      'N2',
      'T2',
      'N1',
    ]);
  });
});
