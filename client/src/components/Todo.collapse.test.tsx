import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

/**
 * FS-0007 R5: a Parent Item with children gets a chevron that folds them away,
 * shows a count while collapsed, and never touches done state or editing.
 */

const fetchChecklist = vi.fn();
const updateChecklistItem = vi.fn();

vi.mock('next/navigation', () => ({
  useParams: () => ({ planId: 'plan-1' }),
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), refresh: vi.fn() }),
}));

vi.mock('@/components/ui/use-toast', () => ({
  useToast: () => ({ toast: vi.fn() }),
}));

vi.mock('@/services/api', () => ({
  fetchChecklist: (...a: unknown[]) => fetchChecklist(...a),
  fetchArchivedChecklist: vi.fn(async () => []),
  createChecklistItem: vi.fn(),
  updateChecklistItem: (...a: unknown[]) => updateChecklistItem(...a),
  deleteChecklistItem: vi.fn(),
  updateChecklistDates: vi.fn(async () => ({})),
  archiveChecklistItem: vi.fn(),
  reorderChecklistItems: vi.fn(async () => []),
  scheduleChecklistItem: vi.fn(),
  scope: { DAILY: 'daily', LONGTERM: 'longterm' },
  ScopeEnum: { DAILY: 'daily', LONGTERM: 'longterm' },
}));

vi.mock('@/api/insights', () => ({
  getChecklistSuggestion: vi.fn(async () => ''),
  getDailyInsights: vi.fn(async () => []),
}));

vi.mock('@/api/plans', () => ({
  getPlan: vi.fn(async () => ({ id: 'plan-1', dailyReset: false })),
  toggleDailyReset: vi.fn(),
}));

const item = (
  id: string,
  type: 'task' | 'note',
  parentId?: string,
  done = false
) => ({
  id,
  description: `item ${id}`,
  done,
  scope: 'longterm',
  type,
  parentId: parentId ?? null,
  planId: 'plan-1',
});

// A(task) ─ B(task, done), C(note), E(task); D(task) alone.
const ITEMS = [
  item('A', 'task'),
  item('B', 'task', 'A', true),
  item('C', 'note', 'A'),
  item('E', 'task', 'A'),
  item('D', 'task'),
];

const rowOf = (id: string) => screen.getByText(`item ${id}`).closest('li')!;

async function renderList() {
  const Todo = (await import('./Todo')).default;
  render(<Todo fixedTaskType="longterm" enableTypeFilter />);
  await screen.findByText('item A');
}

describe('Todo collapse and expand', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Rows are what this file covers, and blocks are the default view,
    // so say which one these expectations are about.
    window.localStorage.setItem('checklistView', 'list');
    fetchChecklist.mockImplementation(async () => ITEMS);
  });

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
    fetchChecklist.mockImplementation(async () => [
      item('A', 'task'),
      item('B', 'note', 'A'),
      item('C', 'note', 'A'),
      item('D', 'task'),
      item('E', 'note', 'D'),
    ]);
    await renderList();
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

  it('should keep the chevron hidden until hover/focus while expanded, quiet on touch, and always visible while collapsed', async () => {
    await renderList();
    const expanded = screen.getByRole('button', { name: 'Collapse item A' });
    expect(expanded.className).toMatch(/(^|\s)opacity-0(\s|$)/);
    expect(expanded.className).toContain('group-hover:opacity-100');
    // A keyboard on the row, or on the chevron itself — but NOT focus-within,
    // which left the chevron on screen after a mouse click had focused it and
    // the pointer had moved on.
    expect(expanded.className).toContain('group-focus-visible:opacity-100');
    expect(expanded.className).toContain('focus-visible:opacity-100');
    expect(expanded.className).not.toMatch(/focus-within/);
    expect(expanded.className).toContain('[@media(hover:none)]:opacity-40');

    fireEvent.click(expanded);
    const collapsed = screen.getByRole('button', { name: 'Expand item A' });
    expect(collapsed.className).toMatch(/(^|\s)opacity-100(\s|$)/);
    expect(collapsed.className).not.toMatch(/(^|\s)opacity-0(\s|$)/);
  });

  it('should keep the chevron in a gutter the list reserves, with no ring circle around it', async () => {
    await renderList();
    const chevron = screen.getByRole('button', { name: 'Collapse item A' });

    // It hangs 32px left of the row, and the list pads that much so the
    // button stays inside the card instead of spilling over its edge.
    expect(chevron.className).toContain('-left-8');
    expect(rowOf('A').closest('ul')!.className).toMatch(/(^|\s)pl-8(\s|$)/);
    expect(chevron.className).not.toContain('rounded-full');
    expect(chevron.className).not.toContain('ring-');
  });

  it('should ring a row only for keyboard focus, never on a click', async () => {
    await renderList();

    // Plain focus: would draw the ring when the row, its chevron or a hover
    // action is clicked.
    expect(rowOf('A').className).not.toMatch(/(^|\s)focus:ring-1(\s|$)/);
    expect(rowOf('A').className).toContain('focus-visible:ring-1');
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
  beforeEach(() => {
    vi.clearAllMocks();
    // Rows are what this file covers, and blocks are the default view,
    // so say which one these expectations are about.
    window.localStorage.setItem('checklistView', 'list');
    fetchChecklist.mockImplementation(async () => ITEMS);
  });

  it('should keep a parent collapsed after the list is unmounted and mounted again, as on reload', async () => {
    const Todo = (await import('./Todo')).default;
    const first = render(<Todo fixedTaskType="longterm" enableTypeFilter />);
    await screen.findByText('item A');
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item A' }));
    first.unmount();

    render(<Todo fixedTaskType="longterm" enableTypeFilter />);
    await screen.findByText('item A');
    expect(screen.queryByText('item B')).toBeNull();
    expect(screen.getByRole('button', { name: 'Expand item A' })).toBeTruthy();
  });

  it('should keep the daily list expanded when the same parent id is collapsed in the long-term list', async () => {
    // Same ids in both scopes, so only the storage key can tell them apart.
    fetchChecklist.mockImplementation(async (_planId: string, scope: string) =>
      ITEMS.map((t) => ({ ...t, scope }))
    );
    const Todo = (await import('./Todo')).default;
    const longterm = render(<Todo fixedTaskType="longterm" enableTypeFilter />);
    await screen.findByText('item A');
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item A' }));
    longterm.unmount();

    render(<Todo fixedTaskType="daily" />);
    await screen.findByText('item A');
    expect(screen.getByText('item B')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Collapse item A' })).toBeTruthy();
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

  beforeEach(() => {
    vi.clearAllMocks();
    // Rows are what this file covers, and blocks are the default view,
    // so say which one these expectations are about.
    window.localStorage.setItem('checklistView', 'list');
    fetchChecklist.mockImplementation(async () => PARENTS);
  });

  it('should offer Collapse all while every parent is expanded', async () => {
    await renderList();

    expect(screen.getByRole('button', { name: 'Collapse all' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Expand all' })).toBeNull();
  });

  it('should fold every parent on Collapse all and then offer Expand all', async () => {
    await renderList();

    fireEvent.click(screen.getByRole('button', { name: 'Collapse all' }));

    expect(screen.queryByText('item B')).toBeNull();
    expect(screen.queryByText('item E')).toBeNull();
    expect(screen.getByRole('button', { name: 'Expand all' })).toBeTruthy();
  });

  it('should offer Expand all when only one parent is collapsed, and open every parent', async () => {
    await renderList();
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item A' }));

    expect(screen.getByRole('button', { name: 'Expand all' })).toBeTruthy();

    fireEvent.click(screen.getByRole('button', { name: 'Expand all' }));

    expect(screen.getByText('item B')).toBeTruthy();
    expect(screen.getByText('item E')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Collapse all' })).toBeTruthy();
  });

  it('should remember what Collapse all folded, as on reload', async () => {
    const Todo = (await import('./Todo')).default;
    const first = render(<Todo fixedTaskType="longterm" enableTypeFilter />);
    await screen.findByText('item A');
    fireEvent.click(screen.getByRole('button', { name: 'Collapse all' }));
    first.unmount();

    render(<Todo fixedTaskType="longterm" enableTypeFilter />);
    await screen.findByText('item A');
    expect(screen.queryByText('item B')).toBeNull();
    expect(screen.queryByText('item E')).toBeNull();
  });

  it('should not offer the toggle when no parent has children', async () => {
    fetchChecklist.mockImplementation(async () => [
      item('A', 'task'),
      item('D', 'task'),
    ]);
    await renderList();

    expect(screen.queryByRole('button', { name: /(Collapse|Expand) all/ })).toBeNull();
  });

  it('should leave the stored state of parents the filter hides untouched', async () => {
    const Todo = (await import('./Todo')).default;
    const first = render(<Todo fixedTaskType="longterm" enableTypeFilter />);
    await screen.findByText('item A');
    fireEvent.click(screen.getByRole('button', { name: 'Collapse all' }));

    // Checklist hides D's only child (a note), so D is no longer a parent here.
    fireEvent.click(screen.getByRole('button', { name: 'Checklist' }));
    fireEvent.click(screen.getByRole('button', { name: 'Expand all' }));
    expect(screen.getByText('item B')).toBeTruthy();

    // Back to All: A was expanded, D kept the state Expand all never saw.
    fireEvent.click(screen.getByRole('button', { name: 'All' }));
    expect(screen.getByText('item B')).toBeTruthy();
    expect(screen.queryByText('item E')).toBeNull();

    first.unmount();
    render(<Todo fixedTaskType="longterm" enableTypeFilter />);
    await screen.findByText('item A');
    expect(screen.getByText('item B')).toBeTruthy();
    expect(screen.queryByText('item E')).toBeNull();
  });
});
