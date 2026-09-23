import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, within } from '@testing-library/react';

/**
 * The header's list / blocks toggle. What it changes is the drawing and the
 * page size; what it must NOT change is which items are on screen, the type
 * filter or the remembered folds — those stay shared between the two views.
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

const item = (
  id: string,
  extra: Partial<Record<string, unknown>> = {}
) => ({
  id,
  description: `item ${id}`,
  done: false,
  scope: 'longterm',
  type: 'task',
  parentId: null,
  planId: 'plan-1',
  ...extra,
});

// Eight top-level items: one page of rows (ten), two pages of blocks (six).
const EIGHT = [
  item('P0'),
  item('C0', { parentId: 'P0', done: true }),
  ...Array.from({ length: 7 }, (_, i) => item(`P${i + 1}`)),
];

const cards = () => document.querySelectorAll('[data-checklist-card]');
const toggle = (name: 'List view' | 'Card view') =>
  screen.getByRole('button', { name });

async function renderTodo(items: unknown[] = EIGHT) {
  fetchChecklist.mockImplementation(async () => items);
  const Todo = (await import('./Todo')).default;
  const view = render(<Todo fixedTaskType="longterm" enableTypeFilter />);
  await screen.findByText('item P0');
  return view;
}

describe('Todo list / blocks toggle', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
  });

  it('should open on the blocks, with the list one click away', async () => {
    await renderTodo();
    // Six blocks, not eight: a block is taller than a row, so a page holds
    // fewer of them.
    expect(cards().length).toBe(6);
    expect(toggle('Card view').getAttribute('aria-pressed')).toBe('true');
    // The guide rail belongs to the rows; a card carries its own structure.
    expect(document.querySelector('[data-guide-rail]')).toBeNull();

    fireEvent.click(toggle('List view'));

    expect(cards().length).toBe(0);
    expect(document.querySelectorAll('li[tabindex]').length).toBe(9);
  });

  it('should page the blocks at six where the same list needs no pages', async () => {
    await renderTodo();
    expect(screen.getByTestId('page-counter').textContent).toBe('1 of 2');
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    expect(cards().length).toBe(2);

    fireEvent.click(toggle('List view'));

    // Eight top-level items fit one page of rows.
    expect(screen.queryByTestId('page-counter')).toBeNull();
  });

  it('should come back to page one when the view changes under a paged list', async () => {
    await renderTodo();
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    expect(screen.getByTestId('page-counter').textContent).toBe('2 of 2');

    fireEvent.click(toggle('List view'));
    fireEvent.click(toggle('Card view'));

    // Ten rows a page and six blocks a page disagree about where anything
    // is, so the view starts again at the top rather than somewhere random.
    expect(screen.getByTestId('page-counter').textContent).toBe('1 of 2');
    expect(screen.getByText('item P0')).toBeTruthy();
  });

  it('should keep a child inside its own block', async () => {
    await renderTodo();
    const card = document.querySelector(
      '[data-checklist-card="P0"]'
    ) as HTMLElement;
    expect(within(card).getByText('item C0')).toBeTruthy();
    expect(within(card).getByText('1/1')).toBeTruthy();
  });

  it('should remember the view for the next visit', async () => {
    const first = await renderTodo();
    fireEvent.click(toggle('List view'));
    expect(window.localStorage.getItem('checklistView')).toBe('list');
    first.unmount();

    await renderTodo();

    expect(cards().length).toBe(0);
  });

  it('should honour the type filter in blocks as in rows', async () => {
    await renderTodo([...EIGHT, item('N1', { type: 'note' })]);
    fireEvent.click(screen.getByRole('button', { name: 'Notes' }));

    expect(cards().length).toBe(1);
    expect(screen.getByText('item N1')).toBeTruthy();
  });

  it('should offer no blocks view on the daily side, which has no nesting', async () => {
    fetchChecklist.mockImplementation(async () =>
      EIGHT.map((i) => ({ ...i, scope: 'daily' }))
    );
    const Todo = (await import('./Todo')).default;
    render(<Todo fixedTaskType="daily" />);
    await screen.findByText('item P0');

    expect(screen.queryByRole('button', { name: 'Card view' })).toBeNull();
  });
});
