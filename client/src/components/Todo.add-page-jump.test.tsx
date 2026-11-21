import { describe, it, expect } from 'vitest';
import { screen, fireEvent, waitFor } from '@testing-library/react';
import {
  addBarRail,
  addInput,
  api,
  item as baseItem,
  counter,
  nestedUnder,
  nextPage as next,
  railOf,
  renderTodo,
  topLevel,
} from '@/test/todoHarness';

/**
 * FS-0008 R15–R18: paging and the add bar meet. Tab resolves against the
 * current page, and an add takes the view to the page its item lands on so
 * the arrival is seen — without disturbing what is typed or nested.
 */

const { createChecklistItem } = api;
const item = (id: string, type: 'task' | 'note' = 'task', parentId?: string) =>
  baseItem(id, { type, parentId: parentId ?? null });

// topLevel(45) is t1..t45: three pages.
async function renderList(items: unknown[] = topLevel(45)) {
  await renderTodo(items, { view: 'list' });
  const input = addInput();
  input.focus();
  return input;
}

describe('Todo add bar over a paged list (FS-0008 R15–R18)', () => {
  it('should nest under the last eligible task of the page being looked at', async () => {
    const input = await renderList();
    fireEvent.click(next());
    expect(counter().textContent).toBe('2 of 3');

    fireEvent.keyDown(input, { key: 'Tab' });

    // Page 2 ends at t40; t45 is the last of the whole set but out of sight.
    expect(nestedUnder()).toBe('t40');
    // The bar sits directly under t40, so the rail runs between them.
    expect(addBarRail()).not.toBeNull();
    expect(railOf('t40')).toBe('parent');
  });

  it('should take a top-level add to the last page, where the row lands', async () => {
    createChecklistItem.mockImplementation(async () => item('NEW'));
    const input = await renderList();
    expect(counter().textContent).toBe('1 of 3');

    fireEvent.change(input, { target: { value: 'item NEW' } });
    fireEvent.submit(input.closest('form')!);

    // 46 items is still three pages, and the new row closes the last one.
    await screen.findByText('item NEW');
    expect(counter().textContent).toBe('3 of 3');
    await waitFor(() => expect(addInput().readOnly).toBe(false));
    expect(addInput().value).toBe('');
    expect(document.activeElement).toBe(addInput());
  });

  it('should go back to the family the child joins when it is on another page', async () => {
    createChecklistItem.mockImplementation(async () => item('C', 'task', 't20'));
    const input = await renderList();
    // Nested on page 1, then paged away before anything is typed.
    fireEvent.keyDown(input, { key: 'Tab' });
    expect(nestedUnder()).toBe('t20');
    fireEvent.click(next());
    fireEvent.click(next());
    expect(counter().textContent).toBe('3 of 3');

    fireEvent.change(addInput(), { target: { value: 'item C' } });
    fireEvent.submit(addInput().closest('form')!);

    await screen.findByText('item C');
    expect(counter().textContent).toBe('1 of 3');
    // The bar is still aimed at the family it just fed (R18).
    expect(nestedUnder()).toBe('t20');
    await waitFor(() => expect(addInput().readOnly).toBe(false));
    expect(document.activeElement).toBe(addInput());
  });

  it('should carry half-typed text and its nesting across a page change', async () => {
    const input = await renderList();
    fireEvent.keyDown(input, { key: 'Tab' });
    fireEvent.change(addInput(), { target: { value: 'half typed' } });

    fireEvent.click(next());

    // The target is paged away, but it is still a live top-level task, so the
    // bar keeps it rather than dropping to top level (R18, not R8.11).
    expect(counter().textContent).toBe('2 of 3');
    expect(nestedUnder()).toBe('t20');
    expect(addInput().value).toBe('half typed');
    expect(document.activeElement).toBe(addInput());
  });

  it('should still open a collapsed target that the jump brings back into view', async () => {
    const items = [...topLevel(45), item('c1', 'task', 't20')];
    createChecklistItem.mockImplementation(async () => item('C', 'task', 't20'));
    const input = await renderList(items);
    fireEvent.keyDown(input, { key: 'Tab' });
    expect(nestedUnder()).toBe('t20');
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item t20' }));
    expect(screen.queryByText('item c1')).toBeNull();
    fireEvent.click(next());

    fireEvent.change(addInput(), { target: { value: 'item C' } });
    fireEvent.submit(addInput().closest('form')!);

    // Back on page 1 with the family open, so the new child isn't folded away
    // the moment it arrives (FS-0007 R8.9).
    await screen.findByText('item C');
    expect(counter().textContent).toBe('1 of 3');
    expect(screen.getByText('item c1')).toBeTruthy();
    expect(
      screen.getByRole('button', { name: 'Collapse item t20' })
    ).toBeTruthy();
  });
});
