import { describe, it, expect } from 'vitest';
import { screen, fireEvent, waitFor } from '@testing-library/react';
import { api, item as baseItem, railOf, renderTodo, rowOf, typeFilter } from '@/test/todoHarness';

/**
 * FS-0007 R1–R4: a Parent Item with visible children draws a guide rail that
 * its child rows continue; childless parents and orphaned children draw none.
 */

const { updateChecklistItem } = api;
const item = (id: string, type: 'task' | 'note', parentId?: string) =>
  baseItem(id, { type, parentId: parentId ?? null });

// A(task) ─ B(task), C(note); D(task) alone.
const ITEMS = [
  item('A', 'task'),
  item('B', 'task', 'A'),
  item('C', 'note', 'A'),
  item('D', 'task'),
];

const renderList = (items: unknown[] = ITEMS) => renderTodo(items, { view: 'list' });

describe('Todo guide rail', () => {
  it('should start a rail on a parent with children and continue it on each child', async () => {
    await renderList();
    expect(railOf('A')).toBe('parent');
    expect(railOf('B')).toBe('child');
    expect(railOf('C')).toBe('child');
  });

  it('should draw no rail on a top-level item without children', async () => {
    await renderList();
    expect(railOf('D')).toBeNull();
  });

  it('should render children at top level with no rail when the filter hides their parent', async () => {
    await renderList();
    fireEvent.click(screen.getByRole('button', { name: 'Notes' }));
    await screen.findByText('item C');
    expect(screen.queryByText('item A')).toBeNull();
    expect(railOf('C')).toBeNull();
  });

  it('should draw no rail on a parent whose children the filter hides', async () => {
    await renderList([item('A', 'task'), item('C', 'note', 'A')]);
    expect(railOf('A')).toBe('parent');
    fireEvent.click(typeFilter('Checklist'));
    await screen.findByText('item A');
    expect(screen.queryByText('item C')).toBeNull();
    expect(railOf('A')).toBeNull();
  });
});

describe('Todo row Tab / Shift+Tab (regression)', () => {
  it('should nest a row under the top-level row above on Tab and join its rail', async () => {
    await renderList();
    fireEvent.keyDown(rowOf('D'), { key: 'Tab' });
    expect(updateChecklistItem).toHaveBeenCalledWith(
      'D',
      { parentId: 'A' },
      expect.anything(),
      'longterm'
    );
    await waitFor(() => expect(railOf('D')).toBe('child'));
  });

  it('should move a child to top level on Shift+Tab and take it off the rail', async () => {
    await renderList();
    fireEvent.keyDown(rowOf('B'), { key: 'Tab', shiftKey: true });
    expect(updateChecklistItem).toHaveBeenCalledWith(
      'B',
      { parentId: null },
      expect.anything(),
      'longterm'
    );
    await waitFor(() => expect(railOf('B')).toBeNull());
    expect(railOf('C')).toBe('child');
  });
});
