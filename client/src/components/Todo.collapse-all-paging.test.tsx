import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';

/**
 * FS-0008 R23–R24: collapse all acts on the parents of the current page, and
 * its label reflects that page. Parents on other pages keep the state they
 * were remembered with — the same rule the type filter already follows.
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

const item = (id: string, parentId?: string) => ({
  id,
  description: `item ${id}`,
  done: false,
  scope: 'longterm',
  type: 'task',
  parentId: parentId ?? null,
  planId: 'plan-1',
});

// 12 top-level items over two pages of ten: P0 (page 1) and P10 (page 2) each
// hold a child; every other row is childless.
const PAGED = [
  item('P0'),
  item('C0', 'P0'),
  ...Array.from({ length: 9 }, (_, i) => item(`P${i + 1}`)),
  item('P10'),
  item('C10', 'P10'),
  item('P11'),
];

const nextPage = () => screen.getByRole('button', { name: 'Next page' });
const prevPage = () => screen.getByRole('button', { name: 'Previous page' });
const allToggle = () =>
  screen.queryByRole('button', { name: /^(Collapse|Expand) all$/ });

async function renderPaged(items = PAGED) {
  fetchChecklist.mockImplementation(async () => items);
  const Todo = (await import('./Todo')).default;
  const view = render(<Todo fixedTaskType="longterm" enableTypeFilter />);
  await screen.findByText('item P0');
  return view;
}

describe('Todo collapse all across pages (FS-0008 R23–R24)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
    // Rows are what this file covers, and blocks are the default view,
    // so say which one these expectations are about.
    window.localStorage.setItem('checklistView', 'list');
  });

  it('should label the toggle from the current page, not the whole list', async () => {
    await renderPaged();
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item P0' }));
    expect(allToggle()!.textContent).toContain('Expand all');

    fireEvent.click(nextPage());

    // P10, the only parent here, is expanded — page 1's fold is not this
    // page's business.
    expect(screen.getByText('item C10')).toBeTruthy();
    expect(allToggle()!.textContent).toContain('Collapse all');
  });

  it('should fold only this page when collapse all is used', async () => {
    await renderPaged();
    fireEvent.click(nextPage());
    fireEvent.click(allToggle()!);
    expect(screen.queryByText('item C10')).toBeNull();

    fireEvent.click(prevPage());

    expect(screen.getByText('item C0')).toBeTruthy();
    expect(allToggle()!.textContent).toContain('Collapse all');
  });

  it('should keep an off-page parent collapsed through this page collapse all', async () => {
    const first = await renderPaged();
    // P0 folded on page 1, then page 2 collapsed by the toggle: the page's
    // write must not prune the entry it never saw.
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item P0' }));
    fireEvent.click(nextPage());
    fireEvent.click(allToggle()!);
    first.unmount();

    await renderPaged();

    expect(screen.queryByText('item C0')).toBeNull();
    fireEvent.click(nextPage());
    expect(screen.queryByText('item C10')).toBeNull();
  });

  it('should offer no toggle on a page whose parents have no children', async () => {
    // Only P0 holds a child, so page 2 is all childless rows.
    await renderPaged(PAGED.filter((t) => t.id !== 'C10'));
    expect(allToggle()).toBeTruthy();

    fireEvent.click(nextPage());

    expect(screen.getByText('item P10')).toBeTruthy();
    expect(allToggle()).toBeNull();
  });
});
