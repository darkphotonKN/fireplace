import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';

/**
 * FS-0008 R19–R22: choosing the Notes or Checklist filter also sets what the
 * add bar creates. All leaves it alone, and a click on the type icon after a
 * filter set it still wins.
 */

const fetchChecklist = vi.fn();

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
  updateChecklistItem: vi.fn(async () => ({})),
  deleteChecklistItem: vi.fn(),
  archiveChecklistItem: vi.fn(async () => ({})),
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

const item = (id: string, type: 'task' | 'note') => ({
  id,
  description: `item ${id}`,
  done: false,
  scope: 'longterm',
  type,
  parentId: null,
  planId: 'plan-1',
});

const ITEMS = [item('T', 'task'), item('N', 'note')];

const addInput = () =>
  screen.getByRole('textbox', { name: /^New (task|note)/ }) as HTMLInputElement;
// The add bar's type, read the way a user reads it: off the input's name.
const addType = () =>
  addInput().getAttribute('aria-label')?.startsWith('New note') ? 'note' : 'task';
const filter = (label: string) =>
  screen.getByRole('button', { name: label });
const flipType = () =>
  fireEvent.click(
    screen.getByRole('button', { name: /^Switch to adding a/ })
  );

async function renderList(items: unknown[] = ITEMS) {
  fetchChecklist.mockImplementation(async () => items);
  const Todo = (await import('./Todo')).default;
  render(<Todo fixedTaskType="longterm" enableTypeFilter />);
  await screen.findByText('item T');
}

describe('Todo filter sets the add bar type', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
  });

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
