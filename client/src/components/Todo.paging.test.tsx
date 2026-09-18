import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';

/**
 * FS-0008 R1–R12: the list is paged ten top-level items at a time, with quiet
 * controls bottom-right. Paging is a view concern — no request gains a page.
 */

const fetchChecklist = vi.fn();
const archiveChecklistItem = vi.fn();
const createChecklistItem = vi.fn();
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
  createChecklistItem: (...a: unknown[]) => createChecklistItem(...a),
  updateChecklistItem: (...a: unknown[]) => updateChecklistItem(...a),
  deleteChecklistItem: vi.fn(),
  archiveChecklistItem: (...a: unknown[]) => archiveChecklistItem(...a),
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
  type: 'task' | 'note' = 'task',
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

/** n top-level tasks named t1..tn. */
const topLevel = (n: number, prefix = 't') =>
  Array.from({ length: n }, (_, i) => item(`${prefix}${i + 1}`));

const prev = () => screen.getByRole('button', { name: 'Previous page' });
const next = () => screen.getByRole('button', { name: 'Next page' });
const counter = () => screen.queryByTestId('page-counter');

async function renderList(firstItem = 'item t1') {
  const Todo = (await import('./Todo')).default;
  const view = render(<Todo fixedTaskType="longterm" enableTypeFilter />);
  await screen.findByText(firstItem);
  return view;
}

describe('Todo paging', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    fetchChecklist.mockImplementation(async () => topLevel(25));
    archiveChecklistItem.mockImplementation(async () => ({}));
  });

  it('should show the first ten top-level items with controls reading 1 of 3', async () => {
    await renderList();
    expect(screen.getByText('item t10')).toBeTruthy();
    expect(screen.queryByText('item t11')).toBeNull();
    expect(counter()!.textContent).toBe('1 of 3');
  });

  it('should count a parent with eight children as one of the ten and keep the family together', async () => {
    fetchChecklist.mockImplementation(async () => [
      item('t1'),
      ...Array.from({ length: 8 }, (_, i) => item(`c${i + 1}`, 'task', 't1')),
      ...topLevel(24).slice(1),
    ]);
    await renderList();
    for (let i = 1; i <= 8; i++) {
      expect(screen.getByText(`item c${i}`)).toBeTruthy();
    }
    // t1 + nine more top-level rows fill the page; t11 waits for page 2.
    expect(screen.getByText('item t10')).toBeTruthy();
    expect(screen.queryByText('item t11')).toBeNull();
    expect(counter()!.textContent).toBe('1 of 3');
  });

  it('should render no page controls when ten or fewer top-level items fit', async () => {
    fetchChecklist.mockImplementation(async () => topLevel(10));
    await renderList();
    expect(counter()).toBeNull();
    expect(screen.queryByRole('button', { name: 'Previous page' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Next page' })).toBeNull();
  });

  it('should walk pages with next and previous, disabling each at its end', async () => {
    await renderList();
    expect(prev()).toBeDisabled();
    expect(next()).toBeEnabled();

    fireEvent.click(next());
    expect(screen.getByText('item t11')).toBeTruthy();
    expect(screen.queryByText('item t10')).toBeNull();
    expect(counter()!.textContent).toBe('2 of 3');
    expect(prev()).toBeEnabled();

    fireEvent.click(next());
    expect(screen.getByText('item t25')).toBeTruthy();
    expect(counter()!.textContent).toBe('3 of 3');
    expect(next()).toBeDisabled();

    fireEvent.click(prev());
    expect(counter()!.textContent).toBe('2 of 3');
  });

  it('should keep a disabled control out of the tab order', async () => {
    await renderList();
    // A disabled <button> is unfocusable and reads as disabled without any
    // tabIndex of our own; assert both rather than the styling.
    expect(prev()).toHaveAttribute('disabled');
    prev().focus();
    expect(document.activeElement).not.toBe(prev());
  });

  it('should announce the current page politely', async () => {
    await renderList();
    const live = screen.getByRole('status');
    expect(live.getAttribute('aria-live')).toBe('polite');
    expect(live.textContent).toBe('Page 1 of 3');
    fireEvent.click(next());
    expect(screen.getByRole('status').textContent).toBe('Page 2 of 3');
  });

  it('should make no request when the page changes', async () => {
    await renderList();
    fetchChecklist.mockClear();
    fireEvent.click(next());
    fireEvent.click(next());
    fireEvent.click(prev());
    expect(fetchChecklist).not.toHaveBeenCalled();
    expect(createChecklistItem).not.toHaveBeenCalled();
    expect(updateChecklistItem).not.toHaveBeenCalled();
    expect(archiveChecklistItem).not.toHaveBeenCalled();
  });

  it('should return to page 1 when the type filter changes', async () => {
    fetchChecklist.mockImplementation(async () => [
      ...topLevel(25),
      ...Array.from({ length: 12 }, (_, i) => item(`n${i + 1}`, 'note')),
    ]);
    await renderList();
    fireEvent.click(next());
    expect(counter()!.textContent).toBe('2 of 4');

    fireEvent.click(screen.getByRole('button', { name: 'Notes' }));
    expect(counter()!.textContent).toBe('1 of 2');
    expect(screen.getByText('item n1')).toBeTruthy();
  });

  it('should return to page 1 when the daily / long-term list changes', async () => {
    const Todo = (await import('./Todo')).default;
    render(<Todo enableTypeFilter />);
    await screen.findByText('item t1');
    fireEvent.click(next());
    expect(counter()!.textContent).toBe('2 of 3');

    fireEvent.click(screen.getByRole('button', { name: 'Long-term' }));
    await waitFor(() => expect(counter()!.textContent).toBe('1 of 3'));
  });

  it('should start on page 1 on a fresh mount while collapse state is restored', async () => {
    fetchChecklist.mockImplementation(async () => [
      ...topLevel(25),
      item('c1', 'task', 't1'),
    ]);
    const { unmount } = await renderList();
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item t1' }));
    fireEvent.click(next());
    expect(counter()!.textContent).toBe('2 of 3');

    unmount();
    await renderList();
    expect(counter()!.textContent).toBe('1 of 3');
    // Collapse is still remembered (FS-0007 R6); only the page is not.
    expect(screen.getByRole('button', { name: 'Expand item t1' })).toBeTruthy();
  });

  it('should clamp to the new last page when the current page is archived away', async () => {
    fetchChecklist.mockImplementation(async () => topLevel(21));
    await renderList();
    expect(counter()!.textContent).toBe('1 of 3');
    fireEvent.click(next());
    fireEvent.click(next());
    expect(counter()!.textContent).toBe('3 of 3');

    // Page 3 holds t21 alone; archiving it leaves two pages, not an empty one.
    const row = screen.getByText('item t21').closest('li')!;
    fireEvent.click(row.querySelector('[title="Archive task"]')!);
    await waitFor(() => expect(counter()!.textContent).toBe('2 of 2'));
    expect(screen.getByText('item t20')).toBeTruthy();
  });

  it('should compute counts, filter tabs and collapse over the whole set while one page shows', async () => {
    fetchChecklist.mockImplementation(async () => [
      ...topLevel(25),
      // t25 sits on page 3 with two children, one done.
      { ...item('c1', 'task', 't25'), done: true },
      item('c2', 'task', 't25'),
      item('n1', 'note'),
    ]);
    await renderList();
    // The filter reaches past page 1: n1 only exists behind the 25 tasks.
    fireEvent.click(screen.getByRole('button', { name: 'Notes' }));
    expect(screen.getByText('item n1')).toBeTruthy();
    expect(counter()).toBeNull();

    fireEvent.click(screen.getByRole('button', { name: 'All' }));
    fireEvent.click(next());
    fireEvent.click(next());
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item t25' }));
    // The count spans the whole family, not the page.
    expect(screen.getByText('item t25').closest('li')!.textContent).toContain(
      '1/2'
    );

    // Collapse state is kept over the whole set: paging away and back leaves
    // the off-page parent folded.
    fireEvent.click(prev());
    fireEvent.click(next());
    expect(
      screen.getByRole('button', { name: 'Expand item t25' })
    ).toBeTruthy();
  });
});
