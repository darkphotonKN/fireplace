import { describe, it, expect, vi } from 'vitest';
import { cleanup, screen, fireEvent, waitFor, within } from '@testing-library/react';
import {
  addBarRail,
  addInput,
  api,
  item as baseItem,
  nestedUnder,
  railOf,
  renderTodo,
  rowOf,
  toast,
} from '@/test/todoHarness';

/**
 * FS-0007 R8: Tab in the add bar nests it under the nearest top-level task
 * above; Enter then creates a child of that task. Shift+Tab returns to top level.
 */

const { createChecklistItem } = api;
const item = (id: string, type: 'task' | 'note', parentId?: string) =>
  baseItem(id, { type, parentId: parentId ?? null });

// A(task) ─ B(task); D(task) alone at the bottom.
const ITEMS = [item('A', 'task'), item('B', 'task', 'A'), item('D', 'task')];

async function renderList(items: unknown[] = ITEMS) {
  await renderTodo(items, { view: 'list' });
  const input = addInput();
  input.focus();
  return input;
}

describe('Todo add bar Tab to nest', () => {
  it('should nest under the trailing top-level task on Tab, keeping focus and text, and extend its rail', async () => {
    const input = await renderList();
    fireEvent.change(input, { target: { value: 'half typed' } });
    fireEvent.keyDown(input, { key: 'Tab' });
    expect(nestedUnder()).toBe('D');
    expect(addInput().value).toBe('half typed');
    expect(document.activeElement).toBe(addInput());
    expect(addBarRail()).not.toBeNull();
    // A childless target starts a rail so the add bar has one to join.
    expect(railOf('D')).toBe('parent');
  });

  it('should create a child of the target on Enter and stay nested, empty and focused', async () => {
    createChecklistItem.mockImplementation(async () => item('E', 'note', 'A'));
    const input = await renderList([item('A', 'task'), item('B', 'task', 'A')]);
    fireEvent.keyDown(input, { key: 'Tab' });
    fireEvent.click(screen.getByRole('button', { name: 'Switch to adding a note' }));
    fireEvent.change(addInput(), { target: { value: 'item E' } });
    fireEvent.submit(addInput().closest('form')!);

    expect(createChecklistItem).toHaveBeenCalledWith('item E', 'plan-1', 'longterm', {
      type: 'note',
      parentId: 'A',
    });
    await screen.findByText('item E');
    // Appended as A's last child.
    const rows = Array.from(document.querySelectorAll('li')).map((li) => li.textContent);
    expect(rows.findIndex((t) => t?.includes('item E'))).toBe(rows.length - 1);
    expect(railOf('E')).toBe('child');

    await waitFor(() => expect(addInput().readOnly).toBe(false));
    expect(nestedUnder()).toBe('A');
    expect(addInput().value).toBe('');
    expect(document.activeElement).toBe(addInput());
  });

  it('should nest under the parent of a trailing child row', async () => {
    const input = await renderList([item('A', 'task'), item('B', 'task', 'A')]);
    fireEvent.keyDown(input, { key: 'Tab' });
    expect(nestedUnder()).toBe('A');
    expect(addBarRail()).not.toBeNull();
  });

  it('should nest under the task above a trailing note without drawing a rail to it', async () => {
    const input = await renderList([item('A', 'task'), item('N', 'note')]);
    fireEvent.keyDown(input, { key: 'Tab' });
    expect(nestedUnder()).toBe('A');
    expect(addInput().placeholder).toBe('Add under “item A”…');
    // A isn't the last group, so the rail can't run down to the bar.
    expect(addBarRail()).toBeNull();
    expect(railOf('A')).toBeNull();
  });

  it.each([
    ['an empty list', []],
    ['a list of only notes', [item('N', 'note'), item('M', 'note')]],
  ])('should swallow Tab and stay at top level with focus given %s', async (_case, items) => {
    const input = await renderList(items);
    fireEvent.change(input, { target: { value: 'kept' } });
    // fireEvent returns false when the default (focus navigation) was prevented.
    expect(fireEvent.keyDown(input, { key: 'Tab' })).toBe(false);
    expect(nestedUnder()).toBeNull();
    expect(addInput().value).toBe('kept');
    expect(document.activeElement).toBe(addInput());
    expect(addBarRail()).toBeNull();
  });

  it('should return to top level on Shift+Tab, and swallow Shift+Tab at top level', async () => {
    const input = await renderList();
    fireEvent.keyDown(input, { key: 'Tab' });
    expect(fireEvent.keyDown(input, { key: 'Tab', shiftKey: true })).toBe(false);
    expect(nestedUnder()).toBeNull();
    expect(addBarRail()).toBeNull();

    expect(fireEvent.keyDown(input, { key: 'Tab', shiftKey: true })).toBe(false);
    expect(nestedUnder()).toBeNull();
    expect(document.activeElement).toBe(addInput());
  });

  it('should do nothing on Tab while already nested', async () => {
    // Tab once nests under D; after D is no longer last, a second Tab must
    // not re-target, and never go a tier deeper.
    const input = await renderList();
    fireEvent.keyDown(input, { key: 'Tab' });
    expect(fireEvent.keyDown(input, { key: 'Tab' })).toBe(false);
    expect(nestedUnder()).toBe('D');
    expect(document.activeElement).toBe(addInput());
  });

  it('should create at top level with no parentId when not nested', async () => {
    createChecklistItem.mockImplementation(async () => item('E', 'task'));
    const input = await renderList();
    fireEvent.change(input, { target: { value: 'item E' } });
    fireEvent.submit(input.closest('form')!);
    await screen.findByText('item E');
    expect(createChecklistItem.mock.calls[0][3]).toEqual({ type: 'task', parentId: undefined });
  });

  it('should stay nested with text and focus in the input when the create fails', async () => {
    createChecklistItem.mockRejectedValue(new Error('boom'));
    vi.spyOn(console, 'error').mockImplementation(() => {});
    const input = await renderList();
    fireEvent.keyDown(input, { key: 'Tab' });
    fireEvent.change(addInput(), { target: { value: 'not saved' } });
    // Use the Add button, which takes focus away from the input on click.
    const add = screen.getByRole('button', { name: 'Add' });
    add.focus();
    fireEvent.click(add);

    await waitFor(() => expect(addInput().readOnly).toBe(false));
    expect(nestedUnder()).toBe('D');
    expect(addInput().value).toBe('not saved');
    expect(document.activeElement).toBe(addInput());
  });

  // Both live in the row's actions panel now, which opens on the trigger
  // beside the text and renders in a portal.
  const fromMenu = (id: string, label: string) => () => {
    fireEvent.click(
      within(rowOf(id)).getByRole('button', { name: /^Actions for/ })
    );
    fireEvent.click(screen.getByRole('menuitem', { name: label }));
  };

  it.each([
    ['archived', fromMenu('D', 'Archive')],
    ['converted to a note', fromMenu('D', 'Make a note')],
  ])('should return to top level with text intact when the target is %s', async (_case, act) => {
    const input = await renderList();
    fireEvent.keyDown(input, { key: 'Tab' });
    fireEvent.change(addInput(), { target: { value: 'kept' } });
    act();
    await waitFor(() => expect(nestedUnder()).toBeNull());
    expect(addInput().value).toBe('kept');
    expect(addBarRail()).toBeNull();
  });

  it('should return to top level when the target is outdented under another row', async () => {
    const input = await renderList([item('D', 'task'), item('A', 'task')]);
    fireEvent.keyDown(input, { key: 'Tab' });
    expect(nestedUnder()).toBe('A');
    fireEvent.keyDown(rowOf('A'), { key: 'Tab' });
    await waitFor(() => expect(nestedUnder()).toBeNull());
  });

  it('should return to top level when the filter hides the target, and not re-nest when shown again', async () => {
    const input = await renderList();
    fireEvent.keyDown(input, { key: 'Tab' });
    fireEvent.change(addInput(), { target: { value: 'kept' } });
    fireEvent.click(screen.getByRole('button', { name: 'Notes' }));
    await waitFor(() => expect(nestedUnder()).toBeNull());
    expect(addInput().value).toBe('kept');

    fireEvent.click(screen.getByRole('button', { name: 'All' }));
    await screen.findByText('item D');
    expect(nestedUnder()).toBeNull();
  });

  it('should start at top level on a fresh mount', async () => {
    const input = await renderList();
    fireEvent.keyDown(input, { key: 'Tab' });
    expect(nestedUnder()).toBe('D');
    cleanup();
    await renderList();
    expect(nestedUnder()).toBeNull();
  });

  it('should describe add bar nesting in the Tab hint', async () => {
    await renderList();
    expect(toast).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'Tip: Tab to nest',
        description: expect.stringMatching(/Tab nests the add bar under the item above.*Shift\+Tab/),
      })
    );
  });
});

describe('Todo add bar nesting into a collapsed parent (FS-0007 R8.9)', () => {
  // A is the last group and has a child, so it is both the Tab target and
  // collapsible.
  const NESTED = [item('A', 'task'), item('B', 'task', 'A')];

  it('should open a collapsed target when the add bar nests under it', async () => {
    const input = await renderList(NESTED);
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item A' }));
    expect(screen.queryByText('item B')).toBeNull();

    fireEvent.keyDown(input, { key: 'Tab' });

    expect(nestedUnder()).toBe('A');
    expect(screen.getByText('item B')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Collapse item A' })).toBeTruthy();
    // The parent's own rail is back, so the bar's piece isn't left hanging.
    expect(railOf('A')).toBe('parent');
    expect(addBarRail()).not.toBeNull();
  });

  it('should open the target on a nested add when it was collapsed after nesting', async () => {
    createChecklistItem.mockImplementation(async () => item('E', 'task', 'A'));
    const input = await renderList(NESTED);
    fireEvent.keyDown(input, { key: 'Tab' });
    // Collapsed while the bar is already nested (by chevron here, the same
    // way "Collapse all" would).
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item A' }));
    expect(screen.queryByText('item B')).toBeNull();
    expect(nestedUnder()).toBe('A');

    fireEvent.change(addInput(), { target: { value: 'item E' } });
    fireEvent.submit(addInput().closest('form')!);

    await screen.findByText('item E');
    expect(screen.getByText('item B')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Collapse item A' })).toBeTruthy();
  });

  it('should remember that nesting opened the target', async () => {
    const input = await renderList(NESTED);
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item A' }));
    fireEvent.keyDown(input, { key: 'Tab' });
    expect(screen.getByText('item B')).toBeTruthy();

    cleanup();
    await renderList(NESTED);

    expect(screen.getByText('item B')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Collapse item A' })).toBeTruthy();
  });

  it('should not write collapse state when the target is already open', async () => {
    const input = await renderList(NESTED);
    // Spy after mount: the component's other preferences write on load.
    const setItem = vi.spyOn(Storage.prototype, 'setItem');

    fireEvent.keyDown(input, { key: 'Tab' });

    expect(nestedUnder()).toBe('A');
    expect(
      setItem.mock.calls.filter(([key]) =>
        String(key).startsWith('collapsedGroups:')
      )
    ).toEqual([]);
  });
});
