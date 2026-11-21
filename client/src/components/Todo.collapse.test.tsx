import { describe, it, expect, vi } from 'vitest';
import { screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { api, item as baseItem, renderTodo, rowOf, typeFilter } from '@/test/todoHarness';

/**
 * FS-0007 R5: a Parent Item with children gets a chevron that folds them away,
 * shows a count while collapsed, and never touches done state or editing.
 */

const { fetchChecklist, updateChecklistItem } = api;
const item = (id: string, type: 'task' | 'note', parentId?: string, done = false) =>
  baseItem(id, { type, parentId: parentId ?? null, done });

// A(task) ─ B(task, done), C(note), E(task); D(task) alone.
const ITEMS = [
  item('A', 'task'),
  item('B', 'task', 'A', true),
  item('C', 'note', 'A'),
  item('E', 'task', 'A'),
  item('D', 'task'),
];

const renderList = (items: unknown[] = ITEMS) =>
  renderTodo(items, { view: 'list', ready: 'item A' });

describe('Todo collapse and expand', () => {
  it('should hide a parent\'s children on chevron click and show them again on a second click', async () => {
    await renderList();
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item A' }));
    expect(screen.queryByText('item B')).toBeNull();
    expect(screen.queryByText('item C')).toBeNull();
    expect(screen.getByText('item D')).toBeTruthy();

    const expand = screen.getByRole('button', { name: 'Expand item A' });
    expect(expand.getAttribute('aria-expanded')).toBe('false');
    fireEvent.click(expand);
    expect(screen.getByText('item B')).toBeTruthy();
    expect(
      screen.getByRole('button', { name: 'Collapse item A' }).getAttribute('aria-expanded')
    ).toBe('true');

    // Toggling never changes done state or opens the editor.
    expect(updateChecklistItem).not.toHaveBeenCalled();
    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();
  });

  it('should show done/total over task children only while collapsed, and no count while expanded', async () => {
    await renderList();
    expect(screen.queryByText('1/2')).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item A' }));
    expect(rowOf('A').textContent).toContain('1/2');
  });

  it('should count notes when every child is a note', async () => {
    await renderList([
      item('A', 'task'),
      item('B', 'note', 'A'),
      item('C', 'note', 'A'),
      item('D', 'task'),
      item('E', 'note', 'D'),
    ]);
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item A' }));
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item D' }));
    expect(screen.getByText('2 notes')).toBeTruthy();
    expect(screen.getByText('1 note')).toBeTruthy();
  });

  it('should hide the parent\'s guide rail with its children and bring it back on expand', async () => {
    await renderList();
    const rail = () => rowOf('A').querySelector('[data-guide-rail]');
    expect(rail()).not.toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item A' }));
    expect(rail()).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Expand item A' }));
    expect(rail()).not.toBeNull();
  });

  it('should give a chevron only to parents with children', async () => {
    await renderList();
    expect(screen.queryByRole('button', { name: /(Collapse|Expand) item D/ })).toBeNull();
    expect(screen.queryByRole('button', { name: /(Collapse|Expand) item B/ })).toBeNull();
  });

  it('should toggle from the keyboard with Enter and Space, and let Tab leave the chevron without indenting', async () => {
    const user = userEvent.setup();
    await renderList();
    screen.getByRole('button', { name: 'Collapse item A' }).focus();

    await user.keyboard('{Enter}');
    expect(screen.queryByText('item B')).toBeNull();
    expect(document.activeElement?.getAttribute('aria-label')).toBe('Expand item A');

    await user.keyboard(' ');
    expect(screen.getByText('item B')).toBeTruthy();

    // The row's Tab handler would preventDefault and trap focus here.
    const chevron = screen.getByRole('button', { name: 'Collapse item A' });
    await user.keyboard('{Tab}');
    expect(document.activeElement).not.toBe(chevron);
    expect(updateChecklistItem).not.toHaveBeenCalled();
  });

  it('should expand a collapsed parent when a row is indented into it, so the row stays in view', async () => {
    await renderList();
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item A' }));
    fireEvent.keyDown(rowOf('D'), { key: 'Tab' });
    expect(updateChecklistItem).toHaveBeenCalledWith(
      'D',
      { parentId: 'A' },
      expect.anything(),
      'longterm'
    );
    await waitFor(() => expect(screen.getByText('item D')).toBeTruthy());
    expect(screen.getByText('item B')).toBeTruthy();
    expect(
      screen.getByRole('button', { name: 'Collapse item A' }).getAttribute('aria-expanded')
    ).toBe('true');
  });
});

describe('Todo remembers collapsed groups (FS-0007 R6)', () => {
  it('should keep a parent collapsed after the list is unmounted and mounted again, as on reload', async () => {
    const first = await renderList();
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item A' }));
    first.unmount();

    await renderList();
    expect(screen.queryByText('item B')).toBeNull();
    expect(screen.getByRole('button', { name: 'Expand item A' })).toBeTruthy();
  });

  it('should not touch the network when collapsing or expanding', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch');
    await renderList();
    const loads = fetchChecklist.mock.calls.length;

    fireEvent.click(screen.getByRole('button', { name: 'Collapse item A' }));
    fireEvent.click(screen.getByRole('button', { name: 'Expand item A' }));

    expect(fetchChecklist.mock.calls.length).toBe(loads);
    expect(updateChecklistItem).not.toHaveBeenCalled();
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it('should render fully expanded and still collapse for the session when collapse storage is blocked', async () => {
    // Block only this feature's keys: the component's older preference reads
    // (showDailyInsights, refreshDailyTasks) are unguarded and out of scope.
    const blocked = (key: string) => key.startsWith('collapsedGroups:');
    const realGet = Storage.prototype.getItem;
    const realSet = Storage.prototype.setItem;
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(function (
      this: Storage,
      key: string
    ) {
      if (blocked(key)) throw new DOMException('blocked', 'SecurityError');
      return realGet.call(this, key);
    });
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(function (
      this: Storage,
      key: string,
      value: string
    ) {
      if (blocked(key)) throw new DOMException('blocked', 'SecurityError');
      return realSet.call(this, key, value);
    });

    await renderList();
    expect(screen.getByText('item B')).toBeTruthy();

    fireEvent.click(screen.getByRole('button', { name: 'Collapse item A' }));
    expect(screen.queryByText('item B')).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Expand item A' }));
    expect(screen.getByText('item B')).toBeTruthy();
  });
});

describe('Todo collapse and expand all (FS-0007 R7)', () => {
  // Two parents: A holds a task, D holds a note. The Checklist filter leaves
  // D without visible children, so only A counts as a rendered parent there.
  const PARENTS = [
    item('A', 'task'),
    item('B', 'task', 'A'),
    item('D', 'task'),
    item('E', 'note', 'D'),
  ];

  it('should offer Collapse all while every parent is expanded', async () => {
    await renderList(PARENTS);

    expect(screen.getByRole('button', { name: 'Collapse all' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Expand all' })).toBeNull();
  });

  it('should fold every parent on Collapse all and then offer Expand all', async () => {
    await renderList(PARENTS);

    fireEvent.click(screen.getByRole('button', { name: 'Collapse all' }));

    expect(screen.queryByText('item B')).toBeNull();
    expect(screen.queryByText('item E')).toBeNull();
    expect(screen.getByRole('button', { name: 'Expand all' })).toBeTruthy();
  });

  it('should offer Expand all when only one parent is collapsed, and open every parent', async () => {
    await renderList(PARENTS);
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item A' }));

    expect(screen.getByRole('button', { name: 'Expand all' })).toBeTruthy();

    fireEvent.click(screen.getByRole('button', { name: 'Expand all' }));

    expect(screen.getByText('item B')).toBeTruthy();
    expect(screen.getByText('item E')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Collapse all' })).toBeTruthy();
  });

  it('should remember what Collapse all folded, as on reload', async () => {
    const first = await renderList(PARENTS);
    fireEvent.click(screen.getByRole('button', { name: 'Collapse all' }));
    first.unmount();

    await renderList(PARENTS);
    expect(screen.queryByText('item B')).toBeNull();
    expect(screen.queryByText('item E')).toBeNull();
  });

  it('should not offer the toggle when no parent has children', async () => {
    await renderList([item('A', 'task'), item('D', 'task')]);

    expect(screen.queryByRole('button', { name: /(Collapse|Expand) all/ })).toBeNull();
  });

  it('should leave the stored state of parents the filter hides untouched', async () => {
    const first = await renderList(PARENTS);
    fireEvent.click(screen.getByRole('button', { name: 'Collapse all' }));

    // Checklist hides D's only child (a note), so D is no longer a parent here.
    fireEvent.click(typeFilter('Checklist'));
    fireEvent.click(screen.getByRole('button', { name: 'Expand all' }));
    expect(screen.getByText('item B')).toBeTruthy();

    // Back to All: A was expanded, D kept the state Expand all never saw.
    fireEvent.click(screen.getByRole('button', { name: 'All' }));
    expect(screen.getByText('item B')).toBeTruthy();
    expect(screen.queryByText('item E')).toBeNull();

    first.unmount();
    await renderList(PARENTS);
    expect(screen.getByText('item B')).toBeTruthy();
    expect(screen.queryByText('item E')).toBeNull();
  });
});
