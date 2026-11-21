import { describe, it, expect } from 'vitest';
import { screen, fireEvent, waitFor } from '@testing-library/react';
import {
  api,
  item as baseItem,
  chooseAction as move,
  order,
  renderTodo,
  toast,
  viewButton,
} from '@/test/todoHarness';

/**
 * FS-0009 R1, R6, R7.2, R9 — moving an item one place from the actions panel.
 *
 * The panel is shared by both views, so these exercise the whole client write
 * path once: the order the request carries, the order the screen shows before
 * the request answers, and what is left behind when it fails.
 */

const { reorderChecklistItems } = api;
const item = (id: string, type: 'task' | 'note' = 'task', parentId: string | null = null) =>
  baseItem(id, { type, parentId });

const THREE = [item('A'), item('B'), item('C')];

const renderList = (items: unknown[] = THREE) => renderTodo(items, { view: 'list' });

describe('Todo reorder from the actions panel', () => {
  it('should move an item down and send the whole set in its new order', async () => {
    await renderList();
    expect(order()).toEqual(['item A', 'item B', 'item C']);

    move('item A', 'Move down');

    expect(order()).toEqual(['item B', 'item A', 'item C']);
    expect(reorderChecklistItems).toHaveBeenCalledWith('plan-1', {
      scope: 'longterm',
      parentId: null,
      ids: ['B', 'A', 'C'],
    });
  });

  it('should round-trip an item back where it started, and offer no move off an end', async () => {
    await renderList();

    move('item A', 'Move down');
    expect(order()).toEqual(['item B', 'item A', 'item C']);
    move('item A', 'Move up');
    expect(order()).toEqual(['item A', 'item B', 'item C']);

    // Back at the top, the panel no longer offers the move that can't happen.
    fireEvent.click(screen.getByRole('button', { name: 'Actions for item A' }));
    expect(screen.queryByRole('menuitem', { name: 'Move up' })).toBeNull();
    expect(screen.getByRole('menuitem', { name: 'Move down' })).toBeTruthy();
  });

  it('should offer neither move to an item alone in its set', async () => {
    await renderList([item('A')]);

    fireEvent.click(screen.getByRole('button', { name: 'Actions for item A' }));
    expect(screen.queryByRole('menuitem', { name: /^Move (up|down)$/ })).toBeNull();
  });

  it('should send the tasks the Notes filter hides, in their existing places (R7.2)', async () => {
    // Interleaved so a set computed from the visible rows alone would both
    // drop the tasks and reorder what is left against the wrong neighbours.
    await renderList([
      item('T1'),
      item('N1', 'note'),
      item('T2'),
      item('N2', 'note'),
    ]);

    fireEvent.click(screen.getByRole('button', { name: 'Notes' }));
    expect(order()).toEqual(['item N1', 'item N2']);

    move('item N1', 'Move down');

    // The whole set, with T1 and T2 still first and third.
    expect(reorderChecklistItems).toHaveBeenCalledWith('plan-1', {
      scope: 'longterm',
      parentId: null,
      ids: ['T1', 'N2', 'T2', 'N1'],
    });
  });

  it('should put the order back and say so when the save fails (R9.2)', async () => {
    reorderChecklistItems.mockRejectedValue(new Error('nope'));
    await renderList();

    move('item A', 'Move down');
    // Shown first, taken back after (R9.1).
    expect(order()).toEqual(['item B', 'item A', 'item C']);

    await waitFor(() =>
      expect(order()).toEqual(['item A', 'item B', 'item C'])
    );
    expect(toast).toHaveBeenCalledWith(
      expect.objectContaining({ title: expect.stringMatching(/move/i) })
    );
  });

  it('should reorder a child within its parent, leaving other families alone', async () => {
    await renderList([
      item('P'),
      item('P1', 'task', 'P'),
      item('Q'),
      item('Q1', 'task', 'Q'),
      item('P2', 'task', 'P'),
    ]);

    move('item P2', 'Move up');

    expect(reorderChecklistItems).toHaveBeenCalledWith('plan-1', {
      scope: 'longterm',
      parentId: 'P',
      ids: ['P2', 'P1'],
    });
  });

  it('should move in the blocks view too, and show it back in the list (R1.2)', async () => {
    window.localStorage.setItem('checklistView', 'card');
    await renderTodo(THREE);
    expect(document.querySelectorAll('[data-checklist-card]').length).toBe(3);

    move('item A', 'Move down');

    expect(order()).toEqual(['item B', 'item A', 'item C']);
    expect(reorderChecklistItems).toHaveBeenCalledWith('plan-1', {
      scope: 'longterm',
      parentId: null,
      ids: ['B', 'A', 'C'],
    });

    // The same order, drawn the other way — no per-view ordering state (R1.1).
    fireEvent.click(viewButton('Checklist'));
    expect(order()).toEqual(['item B', 'item A', 'item C']);
  });

  it('should take its order from the server on a remount, not from what it did last (R1.1)', async () => {
    const first = await renderList();
    move('item A', 'Move down');
    expect(order()).toEqual(['item B', 'item A', 'item C']);
    first.unmount();

    // The server is the only thing that says what the order is: it answers
    // with C first, and that is what comes back up, not the local swap.
    await renderList([item('C'), item('B'), item('A')]);

    expect(order()).toEqual(['item C', 'item B', 'item A']);
  });
});
