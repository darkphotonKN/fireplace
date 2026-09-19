import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';

/**
 * FS-0007 R1–R4: a Parent Item with visible children draws a guide rail that
 * its child rows continue; childless parents and orphaned children draw none.
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
  parentId?: string
) => ({
  id,
  description: `item ${id}`,
  done: false,
  scope: 'longterm',
  type,
  parentId: parentId ?? null,
  planId: 'plan-1',
});

// A(task) ─ B(task), C(note); D(task) alone.
const ITEMS = [
  item('A', 'task'),
  item('B', 'task', 'A'),
  item('C', 'note', 'A'),
  item('D', 'task'),
];

const rowOf = (id: string) => screen.getByText(`item ${id}`).closest('li')!;
const railOf = (id: string) =>
  rowOf(id).querySelector('[data-guide-rail]')?.getAttribute('data-guide-rail') ??
  null;

async function renderList() {
  const Todo = (await import('./Todo')).default;
  render(<Todo fixedTaskType="longterm" enableTypeFilter />);
  await screen.findByText('item A');
}

describe('Todo guide rail', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Rows are what this file covers, and blocks are the default view,
    // so say which one these expectations are about.
    window.localStorage.setItem('checklistView', 'list');
    fetchChecklist.mockImplementation(async () => ITEMS);
  });

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
    expect(rowOf('C').className).not.toMatch(/\bml-/);
  });

  it('should draw no rail on a parent whose children the filter hides', async () => {
    fetchChecklist.mockImplementation(async () => [
      item('A', 'task'),
      item('C', 'note', 'A'),
    ]);
    await renderList();
    expect(railOf('A')).toBe('parent');
    fireEvent.click(screen.getByRole('button', { name: 'Checklist' }));
    await screen.findByText('item A');
    expect(screen.queryByText('item C')).toBeNull();
    expect(railOf('A')).toBeNull();
  });
});

describe('Todo row Tab / Shift+Tab (regression)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Rows are what this file covers, and blocks are the default view,
    // so say which one these expectations are about.
    window.localStorage.setItem('checklistView', 'list');
    fetchChecklist.mockImplementation(async () => ITEMS);
  });

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
