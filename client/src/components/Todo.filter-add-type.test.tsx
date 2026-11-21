import { describe, it, expect } from 'vitest';
import { screen, fireEvent } from '@testing-library/react';
import { addInput, item as baseItem, renderTodo, typeFilter as filter } from '@/test/todoHarness';

/**
 * FS-0008 R19–R22: choosing the Notes or Checklist filter also sets what the
 * add bar creates. All leaves it alone, and a click on the type icon after a
 * filter set it still wins.
 */

const item = (id: string, type: 'task' | 'note') => baseItem(id, { type });

const ITEMS = [item('T', 'task'), item('N', 'note')];

// The add bar's type, read the way a user reads it: off the input's name.
const addType = () =>
  addInput().getAttribute('aria-label')?.startsWith('New note') ? 'note' : 'task';
const flipType = () =>
  fireEvent.click(
    screen.getByRole('button', { name: /^Switch to adding a/ })
  );

// This is the list's add bar; in the grid the bar is the new-milestone
// composer (FS-FSFEC R29), whose notes (R39) are covered with its edges.
const renderList = (items: unknown[] = ITEMS) => renderTodo(items, { view: 'list' });

describe('Todo filter sets the add bar type', () => {
  it('should set the add bar to note on Notes and back to task on Checklist, placeholder following', async () => {
    await renderList();
    expect(addType()).toBe('task');

    fireEvent.click(filter('Notes'));
    expect(addType()).toBe('note');
    expect(addInput().placeholder).toBe('Jot down a note…');

    fireEvent.click(filter('Checklist'));
    expect(addType()).toBe('task');
    expect(addInput().placeholder).toBe('Add something to do…');
  });

  it('should leave the add bar type alone when All is chosen', async () => {
    await renderList();
    flipType();
    expect(addType()).toBe('note');

    fireEvent.click(filter('All'));
    expect(addType()).toBe('note');

    // Still untouched when All follows a typed filter that set the type.
    fireEvent.click(filter('Checklist'));
    expect(addType()).toBe('task');
    flipType();
    fireEvent.click(filter('All'));
    expect(addType()).toBe('note');
  });

  it('should let the type icon override a filter set, and re-choosing that filter keeps the override', async () => {
    await renderList();
    fireEvent.click(filter('Notes'));
    expect(addType()).toBe('note');

    flipType();
    expect(addType()).toBe('task');
    // The tab is still Notes; clicking it again is not a fresh choice.
    fireEvent.click(filter('Notes'));
    expect(addType()).toBe('task');
    expect(addInput().placeholder).toBe('Add something to do…');
  });

  it('should keep filtering the list while setting the add bar type', async () => {
    await renderList();
    fireEvent.click(filter('Notes'));
    expect(screen.queryByText('item T')).toBeNull();
    expect(screen.getByText('item N')).toBeTruthy();

    fireEvent.click(filter('Checklist'));
    expect(screen.getByText('item T')).toBeTruthy();
    expect(screen.queryByText('item N')).toBeNull();
  });
});
