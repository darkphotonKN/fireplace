import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, within } from '@testing-library/react';

/**
 * The header's list / blocks toggle. What it changes is the drawing and the
 * page size; what it must NOT change is which items are on screen, the type
 * filter or the remembered folds — those stay shared between the two views.
 */

const fetchChecklist = vi.fn();

const push = vi.fn();
vi.mock('next/navigation', () => ({
  useParams: () => ({ planId: 'plan-1' }),
  useRouter: () => ({ push, replace: vi.fn(), refresh: vi.fn() }),
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
}));

vi.mock('@/api/plans', () => ({
  getPlan: vi.fn(async () => ({ id: 'plan-1', dailyReset: false })),
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

// Fourteen top-level items: one page of rows (twenty), two pages of blocks
// (twelve, then two).
const FOURTEEN = [
  item('P0'),
  item('C0', { parentId: 'P0', done: true }),
  ...Array.from({ length: 13 }, (_, i) => item(`P${i + 1}`)),
];

const cards = () => document.querySelectorAll('[data-checklist-card]');
// The toggle's "Checklist" shares its name with the type filter's tab; the
// "View" group is what tells them apart (FS-FSFEC R3).
const toggle = (name: 'Checklist' | 'Milestones') =>
  within(screen.getByRole('group', { name: 'View' })).getByRole('button', {
    name,
  });

async function renderTodo(items: unknown[] = FOURTEEN) {
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
    // Twelve blocks, not fourteen: a block is taller than a row, so a page
    // holds fewer of them.
    expect(cards().length).toBe(12);
    expect(toggle('Milestones').getAttribute('aria-pressed')).toBe('true');
    // The guide rail belongs to the rows; a card carries its own structure.
    expect(document.querySelector('[data-guide-rail]')).toBeNull();

    fireEvent.click(toggle('Checklist'));

    expect(cards().length).toBe(0);
    expect(document.querySelectorAll('li[tabindex]').length).toBe(15);
  });

  it("should open a milestone's page from its card menu (FS-043WP R6)", async () => {
    await renderTodo();
    fireEvent.click(screen.getByRole('button', { name: 'Actions for item P0' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Open' }));
    expect(push).toHaveBeenCalledWith('/plan/plan-1/m/P0');
  });

  it('should name the two readings Milestones and Checklist, inside a View group', async () => {
    await renderTodo();
    expect(toggle('Milestones').getAttribute('title')).toBe('Milestones');
    expect(toggle('Checklist').getAttribute('title')).toBe('Checklist');
    // The type filter keeps its own "Checklist", outside the group.
    expect(screen.getAllByRole('button', { name: 'Checklist' }).length).toBe(2);
    expect(screen.queryByRole('button', { name: 'Card view' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'List view' })).toBeNull();
  });

  it('should page the blocks at twelve where the same list needs no pages', async () => {
    await renderTodo();
    expect(screen.getByTestId('page-counter').textContent).toBe('1 of 2');
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    expect(cards().length).toBe(2);

    fireEvent.click(toggle('Checklist'));

    // Fourteen top-level items fit one page of rows.
    expect(screen.queryByTestId('page-counter')).toBeNull();
  });

  it('should come back to page one when the view changes under a paged list', async () => {
    await renderTodo();
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    expect(screen.getByTestId('page-counter').textContent).toBe('2 of 2');

    fireEvent.click(toggle('Checklist'));
    fireEvent.click(toggle('Milestones'));

    // Twenty rows a page and twelve blocks a page disagree about where anything
    // is, so the view starts again at the top rather than somewhere random.
    expect(screen.getByTestId('page-counter').textContent).toBe('1 of 2');
    expect(screen.getByText('item P0')).toBeTruthy();
  });

  it('should fit exactly twelve blocks on one page and open a second at the thirteenth', async () => {
    const blocks = (n: number) =>
      Array.from({ length: n }, (_, i) => item(`P${i}`));
    const first = await renderTodo(blocks(12));
    expect(cards().length).toBe(12);
    expect(screen.queryByTestId('page-counter')).toBeNull();
    first.unmount();

    await renderTodo(blocks(13));
    expect(cards().length).toBe(12);
    expect(screen.getByTestId('page-counter').textContent).toBe('1 of 2');
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
    fireEvent.click(toggle('Checklist'));
    expect(window.localStorage.getItem('checklistView')).toBe('list');
    first.unmount();

    await renderTodo();

    expect(cards().length).toBe(0);
  });

  it('should honour the type filter in blocks as in rows', async () => {
    await renderTodo([...FOURTEEN, item('N1', { type: 'note' })]);
    fireEvent.click(screen.getByRole('button', { name: 'Notes' }));

    expect(cards().length).toBe(1);
    expect(screen.getByText('item N1')).toBeTruthy();
  });
});
