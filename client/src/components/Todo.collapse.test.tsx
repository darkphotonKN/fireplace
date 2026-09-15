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
  archiveChecklistItem: vi.fn(),
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
    expect(expanded.className).toContain('group-focus-within:opacity-100');
    expect(expanded.className).toContain('[@media(hover:none)]:opacity-40');

    fireEvent.click(expanded);
    const collapsed = screen.getByRole('button', { name: 'Expand item A' });
    expect(collapsed.className).toMatch(/(^|\s)opacity-100(\s|$)/);
    expect(collapsed.className).not.toMatch(/(^|\s)opacity-0(\s|$)/);
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
