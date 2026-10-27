import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, within, act, waitFor } from '@testing-library/react';

/**
 * The plan header's progress strip (FS-FSFEC R15–R22). It reads the whole
 * long-term list, so neither the page, the type filter nor collapse decides
 * what it counts, and it shows in both readings of the plan.
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

// Eight milestones and a note: two pages of cards. P0 is half done, P1 closed.
const PLAN = [
  item('P0'),
  item('C0', { parentId: 'P0', done: true }),
  item('C1', { parentId: 'P0' }),
  item('P1', { done: true }),
  ...Array.from({ length: 6 }, (_, i) => item(`P${i + 2}`)),
  item('N1', { type: 'note' }),
];

const segments = () =>
  document.querySelectorAll<HTMLElement>('[data-milestone-segment]');
const fillOf = (title: string) =>
  Array.from(segments())
    .find((s) => s.title.startsWith(`${title}:`))!
    .querySelector<HTMLElement>('[data-milestone-fill]')!.style.width;
const viewButton = (name: 'Milestones' | 'Checklist') =>
  within(screen.getByRole('group', { name: 'View' })).getByRole('button', {
    name,
  });

async function renderTodo(items: unknown[] = PLAN, taskType = 'longterm') {
  fetchChecklist.mockImplementation(async () => items);
  const Todo = (await import('./Todo')).default;
  const view = render(
    <Todo fixedTaskType={taskType as 'longterm'} enableTypeFilter />
  );
  await screen.findByText('item P0');
  return view;
}

describe('Todo milestone strip', () => {
  // The Day plan section writes items behind this card's back (FS-02JBZ
  // R77): it announces, and the card re-reads so the strip moves.
  it('should re-read and move the strip when another part of the page changes this plan', async () => {
    await renderTodo();
    expect(fillOf('item P0')).toBe('50%');
    const { announcePlanItemsChanged } = await import('@/lib/planItemsChanged');
    fetchChecklist.mockClear();

    act(() => announcePlanItemsChanged('plan-2'));
    expect(fetchChecklist).not.toHaveBeenCalled();

    fetchChecklist.mockImplementation(async () =>
      PLAN.map((i) => (i.id === 'C1' ? { ...i, done: true } : i))
    );
    await act(async () => announcePlanItemsChanged('plan-1'));

    expect(fetchChecklist).toHaveBeenCalledWith('plan-1', 'longterm');
    await waitFor(() => expect(fillOf('item P0')).toBe('100%'));
  });

  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
    updateChecklistItem.mockResolvedValue({});
  });

  it('should count the whole plan in the milestone reading, whatever the page or filter', async () => {
    await renderTodo();
    // The grid holds six cards a page; the strip holds all eight.
    expect(screen.getByTestId('page-counter').textContent).toBe('1 of 2');
    expect(screen.getByText('1 of 8 milestones')).toBeTruthy();
    expect(segments().length).toBe(8);
    expect(fillOf('item P0')).toBe('50%');

    fireEvent.click(screen.getByRole('button', { name: 'Notes' }));

    expect(screen.getByText('1 of 8 milestones')).toBeTruthy();
    expect(segments().length).toBe(8);
  });

  it('should show the same strip in the checklist reading, through collapse and filter', async () => {
    await renderTodo();
    fireEvent.click(viewButton('Checklist'));

    expect(screen.getByText('1 of 8 milestones')).toBeTruthy();
    expect(segments().length).toBe(8);

    fireEvent.click(screen.getByRole('button', { name: /collapse all/i }));
    fireEvent.click(screen.getByRole('button', { name: 'Notes' }));

    expect(screen.getByText('1 of 8 milestones')).toBeTruthy();
    expect(fillOf('item P0')).toBe('50%');
  });

  it('should show no strip and no headline for a plan of only notes', async () => {
    await renderTodo([
      item('P0', { type: 'note' }),
      item('P1', { type: 'note' }),
    ]);

    expect(document.querySelector('[data-milestone-strip]')).toBeNull();
    // The headline, not the grid composer's "New milestone".
    expect(screen.queryByText(/of \d+ milestones?$/)).toBeNull();
  });

  it('should move a segment on a tick before the request lands, and back if it fails', async () => {
    let reject!: (e: Error) => void;
    updateChecklistItem.mockImplementation(
      () => new Promise((_, r) => (reject = r))
    );
    vi.spyOn(console, 'error').mockImplementation(() => {});
    await renderTodo();

    fireEvent.click(screen.getByRole('checkbox', { name: 'Mark item C1 done' }));

    expect(fillOf('item P0')).toBe('100%');

    await act(async () => reject(new Error('nope')));

    expect(fillOf('item P0')).toBe('50%');
  });

  it('should show no strip on the daily tab', async () => {
    await renderTodo(
      PLAN.map((i) => ({ ...i, scope: 'daily' })),
      'daily'
    );
    expect(document.querySelector('[data-milestone-strip]')).toBeNull();
  });

  it('should show no strip in settings or the archived view', async () => {
    await renderTodo();
    fireEvent.click(screen.getByTitle('Settings'));
    expect(document.querySelector('[data-milestone-strip]')).toBeNull();

    fireEvent.click(screen.getByText('View archived tasks'));
    await screen.findByText('Back to settings');
    expect(document.querySelector('[data-milestone-strip]')).toBeNull();
  });

  it("should link each milestone card to its page under this plan, and no list row (FS-043WP R5, R8)", async () => {
    await renderTodo();
    expect(
      screen.getByRole('link', { name: 'Open item P0' })
    ).toHaveAttribute('href', '/plan/plan-1/m/P0');
    expect(screen.queryByRole('link', { name: 'Open item N1' })).toBeNull();

    fireEvent.click(viewButton('Checklist'));
    expect(screen.queryByRole('link', { name: /^Open / })).toBeNull();
  });
});
