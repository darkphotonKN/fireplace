import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';

/**
 * Deleting a checklist item, through the API layer (I-0058).
 *
 * Ten other `Todo.*.test.tsx` files mock `deleteChecklistItem` and never call
 * it: the component deleted with a hand-written `fetch` that carried no
 * `Authorization` header and read an env var that does not exist, so the mocks
 * were green while the product 404'd. These assertions are about the call
 * actually being made — which is also what makes the delete stamp
 * touched-today (FS-KSJFR R11–R12), free once the path is right.
 *
 * `fetch` is deliberately left undefined below. If either call site regresses
 * to a raw request, the test fails on the spot rather than on a mock nobody
 * configured.
 */

const fetchChecklist = vi.fn();
const fetchArchivedChecklist = vi.fn();
const deleteChecklistItem = vi.fn();

vi.mock('next/navigation', () => ({
  useParams: () => ({ planId: 'plan-1' }),
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), refresh: vi.fn() }),
}));

vi.mock('@/components/ui/use-toast', () => ({
  useToast: () => ({ toast: vi.fn() }),
}));

vi.mock('@/services/api', () => ({
  fetchChecklist: (...a: unknown[]) => fetchChecklist(...a),
  fetchArchivedChecklist: (...a: unknown[]) => fetchArchivedChecklist(...a),
  createChecklistItem: vi.fn(),
  updateChecklistItem: vi.fn(async () => ({})),
  deleteChecklistItem: (...a: unknown[]) => deleteChecklistItem(...a),
  updateChecklistDates: vi.fn(async () => ({})),
  archiveChecklistItem: vi.fn(async () => ({})),
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

const item = (id: string, extra: Record<string, unknown> = {}) => ({
  id,
  description: `item ${id}`,
  done: false,
  scope: 'longterm',
  type: 'task',
  parentId: null,
  planId: 'plan-1',
  ...extra,
});

async function renderTodo(items: unknown[] = [item('P0'), item('P1')]) {
  fetchChecklist.mockImplementation(async () => items);
  const Todo = (await import('./Todo')).default;
  const view = render(<Todo fixedTaskType="longterm" />);
  await screen.findByText('item P0');
  return view;
}

/** The list view, where each row carries its own actions menu. */
const openActions = (description: string) => {
  fireEvent.click(screen.getByRole('button', { name: `Actions for ${description}` }));
};

beforeEach(() => {
  vi.clearAllMocks();
  window.localStorage.clear();
  deleteChecklistItem.mockResolvedValue(undefined);
  fetchArchivedChecklist.mockResolvedValue([]);
});

describe('Todo — deleting an item from the list', () => {
  it('should delete through the API layer, not a hand-written fetch', async () => {
    await renderTodo();
    fireEvent.click(screen.getByRole('button', { name: 'List view' }));

    openActions('item P0');
    fireEvent.click(screen.getByRole('menuitem', { name: /delete/i }));

    await waitFor(() => expect(deleteChecklistItem).toHaveBeenCalledTimes(1));
    // Item first, then plan — the adapter's signature (services/api.ts).
    expect(deleteChecklistItem.mock.calls[0].slice(0, 2)).toEqual(['P0', 'plan-1']);
  });

  it('should take the deleted row off the list', async () => {
    await renderTodo();
    fireEvent.click(screen.getByRole('button', { name: 'List view' }));

    openActions('item P0');
    fireEvent.click(screen.getByRole('menuitem', { name: /delete/i }));

    await waitFor(() => expect(screen.queryByText('item P0')).toBeNull());
    expect(screen.getByText('item P1')).toBeInTheDocument();
  });

  it('should keep the row and say so when the delete fails', async () => {
    deleteChecklistItem.mockRejectedValue(new Error('nope'));
    await renderTodo();
    fireEvent.click(screen.getByRole('button', { name: 'List view' }));

    openActions('item P0');
    fireEvent.click(screen.getByRole('menuitem', { name: /delete/i }));

    expect(await screen.findByText(/failed to delete task/i)).toBeInTheDocument();
    expect(screen.getByText('item P0')).toBeInTheDocument();
  });
});

describe('Todo — deleting an archived item from the confirmation modal', () => {
  const openArchived = async () => {
    fetchArchivedChecklist.mockResolvedValue([item('A0'), item('A1')]);
    await renderTodo();
    fireEvent.click(screen.getByTitle('Settings'));
    fireEvent.click(
      screen.getByRole('button', { name: /view archived tasks/i }),
    );
    await screen.findByText('item A0');
  };

  it('should delete through the API layer when the deletion is confirmed', async () => {
    await openArchived();

    fireEvent.click(screen.getAllByTitle('Delete task permanently')[0]);
    const modal = within(
      screen.getByText(/permanently delete/i).closest('div') as HTMLElement,
    );
    fireEvent.click(modal.getByRole('button', { name: 'Delete' }));

    await waitFor(() => expect(deleteChecklistItem).toHaveBeenCalledTimes(1));
    expect(deleteChecklistItem.mock.calls[0].slice(0, 2)).toEqual(['A0', 'plan-1']);
    await waitFor(() => expect(screen.queryByText('item A0')).toBeNull());
    expect(screen.getByText('item A1')).toBeInTheDocument();
  });
});
