import { describe, it, expect } from 'vitest';
import { screen, fireEvent, waitFor, act } from '@testing-library/react';
import {
  api,
  item as baseItem,
  chooseAction,
  counter,
  nextPage as next,
  renderTodo,
  topLevel,
} from '@/test/todoHarness';

/**
 * FS-F6AEE R14–R18: Duplicate from a list row. The view waits for the server,
 * then puts what it returns directly after the source, follows the copy to
 * its page, and reports a failure on the error line it already has.
 */

const { fetchChecklist, duplicateChecklistItem } = api;
const item = (id: string, parentId?: string, extra: object = {}) =>
  baseItem(id, { parentId: parentId ?? null, ...extra });

/** A family in the middle of the list: t1 with steps c1, c2, then t2. */
const family = () => [item('t1'), item('c1', 't1'), item('c2', 't1'), item('t2')];

/** The rows on screen, top to bottom, by item id. */
const renderedIds = () =>
  screen
    .getAllByText(/^item \w+$/)
    .map((el) => el.textContent!.replace('item ', ''));

const openMenu = (id: string) =>
  fireEvent.click(screen.getByRole('button', { name: `Actions for item ${id}` }));

const duplicate = (id: string) => chooseAction(`item ${id}`, 'Duplicate');

const renderList = (items: unknown[] = family()) => renderTodo(items, { view: 'list' });
const renderGrid = (items: unknown[] = family()) => renderTodo(items, { view: 'grid' });

describe('Todo duplicate from a list row (FS-F6AEE R14–R18)', () => {
  it('should put a duplicated item and its steps right after the source family, without a refetch', async () => {
    duplicateChecklistItem.mockResolvedValue([
      item('D'),
      item('Dc1', 'D'),
      item('Dc2', 'D'),
    ]);
    await renderList();

    duplicate('t1');

    await screen.findByText('item D');
    expect(duplicateChecklistItem).toHaveBeenCalledWith('plan-1', 't1');
    expect(renderedIds()).toEqual(['t1', 'c1', 'c2', 'D', 'Dc1', 'Dc2', 't2']);
    expect(fetchChecklist).toHaveBeenCalledTimes(1);
  });

  it('should put a duplicated step directly after it, inside the same family', async () => {
    duplicateChecklistItem.mockResolvedValue([item('Dc', 't1')]);
    await renderList();

    duplicate('c1');

    await screen.findByText('item Dc');
    expect(renderedIds()).toEqual(['t1', 'c1', 'Dc', 'c2', 't2']);
  });

  it('should show nothing until the server answers', async () => {
    let answer!: (items: unknown[]) => void;
    duplicateChecklistItem.mockReturnValue(
      new Promise((resolve) => {
        answer = resolve;
      })
    );
    await renderList();

    duplicate('t2');

    expect(duplicateChecklistItem).toHaveBeenCalledTimes(1);
    expect(screen.queryByText('item D')).toBeNull();
    expect(renderedIds()).toEqual(['t1', 'c1', 'c2', 't2']);

    await act(async () => answer([item('D')]));
    expect(renderedIds()).toEqual(['t1', 'c1', 'c2', 't2', 'D']);
  });

  it('should insert nothing and show the item-action error when it fails', async () => {
    duplicateChecklistItem.mockRejectedValue(new Error('boom'));
    await renderList();

    duplicate('t1');

    await screen.findByText('Failed to duplicate item. Please try again.');
    expect(renderedIds()).toEqual(['t1', 'c1', 'c2', 't2']);
  });

  it('should follow the copy to the page it lands on', async () => {
    duplicateChecklistItem.mockResolvedValue([item('D')]);
    await renderList(topLevel(45));
    fireEvent.click(next());
    expect(counter().textContent).toBe('2 of 3');

    // t40 closes page 2, so its copy opens page 3.
    duplicate('t40');

    await waitFor(() => expect(counter().textContent).toBe('3 of 3'));
    expect(screen.getByText('item D')).toBeTruthy();
    expect(renderedIds()[0]).toBe('D');
  });

  it('should not offer Duplicate on an archived item (R15)', async () => {
    await renderList([item('t1'), item('t2', undefined, { archived: true })]);

    openMenu('t1');
    expect(screen.getByRole('menuitem', { name: 'Duplicate' })).toBeTruthy();
    fireEvent.keyDown(document, { key: 'Escape' });

    openMenu('t2');
    expect(screen.queryByRole('menuitem', { name: 'Duplicate' })).toBeNull();
  });
});

describe('Todo duplicate from the grid (FS-F6AEE R14, R16, R17)', () => {
  /** The cards on screen, left to right, by item id. */
  const cardIds = () =>
    Array.from(document.querySelectorAll('[data-checklist-card]')).map((el) =>
      el.getAttribute('data-checklist-card')
    );

  /** One card's step rows, top to bottom, by item id. */
  const stepIds = (cardId: string) =>
    Array.from(
      document
        .querySelector(`[data-checklist-card="${cardId}"]`)!
        .querySelectorAll('li')
    ).map((li) => li.textContent!.match(/item (\w+)/)![1]);

  it('should put a duplicated milestone card, with its steps not done, right after the source card', async () => {
    duplicateChecklistItem.mockResolvedValue([
      item('D'),
      item('Dc1', 'D'),
      item('Dc2', 'D'),
    ]);
    await renderGrid([
      item('t1'),
      item('c1', 't1', { done: true }),
      item('c2', 't1'),
      item('t2'),
    ]);

    duplicate('t1');

    await waitFor(() => expect(cardIds()).toEqual(['t1', 'D', 't2']));
    expect(duplicateChecklistItem).toHaveBeenCalledWith('plan-1', 't1');
    expect(stepIds('D')).toEqual(['Dc1', 'Dc2']);
    for (const id of ['Dc1', 'Dc2']) {
      expect(
        screen
          .getByRole('checkbox', { name: `Mark item ${id} done` })
          .getAttribute('aria-checked')
      ).toBe('false');
    }
    expect(fetchChecklist).toHaveBeenCalledTimes(1);
  });

  it('should put a duplicated step directly after it, in the same card', async () => {
    duplicateChecklistItem.mockResolvedValue([item('Dc', 't1')]);
    await renderGrid();

    duplicate('c1');

    await screen.findByText('item Dc');
    expect(stepIds('t1')).toEqual(['c1', 'Dc', 'c2']);
    expect(cardIds()).toEqual(['t1', 't2']);
  });

  it("should insert nothing and show Todo's item-action error when it fails", async () => {
    duplicateChecklistItem.mockRejectedValue(new Error('boom'));
    await renderGrid();

    duplicate('t1');

    await screen.findByText('Failed to duplicate item. Please try again.');
    expect(cardIds()).toEqual(['t1', 't2']);
    expect(stepIds('t1')).toEqual(['c1', 'c2']);
  });
});
