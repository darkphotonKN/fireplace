import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';

/**
 * FS-0008 R15–R18: paging and the add bar meet. Tab resolves against the
 * current page, and an add takes the view to the page its item lands on so
 * the arrival is seen — without disturbing what is typed or nested.
 */

const fetchChecklist = vi.fn();
const createChecklistItem = vi.fn();

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

/** n top-level tasks named t1..tn — 25 of them is three pages. */
const topLevel = (n: number) =>
  Array.from({ length: n }, (_, i) => item(`t${i + 1}`));

const rowOf = (id: string) => screen.getByText(`item ${id}`).closest('li')!;
const addInput = () =>
  screen.getByRole('textbox', { name: /^New (task|note)/ }) as HTMLInputElement;
// The input names its target while nested; null means top level.
const nestedUnder = () =>
  addInput().getAttribute('aria-label')?.match(/under “item (\w+)”/)?.[1] ??
  null;
const railOf = (id: string) =>
  rowOf(id)
    .querySelector('[data-guide-rail]')
    ?.getAttribute('data-guide-rail') ?? null;
const addBarRail = () => document.querySelector('[data-guide-rail="add-bar"]');
const counter = () => screen.getByTestId('page-counter');
const next = () => screen.getByRole('button', { name: 'Next page' });

async function renderList(items: unknown[] = topLevel(25)) {
  fetchChecklist.mockImplementation(async () => items);
  const Todo = (await import('./Todo')).default;
  render(<Todo fixedTaskType="longterm" enableTypeFilter />);
  await screen.findByText('item t1');
  const input = addInput();
  input.focus();
  return input;
}

describe('Todo add bar over a paged list (FS-0008 R15–R18)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
  });

  it('should nest under the last eligible task of the page being looked at', async () => {
    const input = await renderList();
    fireEvent.click(next());
    expect(counter().textContent).toBe('2 of 3');

    fireEvent.keyDown(input, { key: 'Tab' });

    // Page 2 ends at t20; t25 is the last of the whole set but out of sight.
    expect(nestedUnder()).toBe('t20');
    // The bar sits directly under t20, so the rail runs between them.
    expect(addBarRail()).not.toBeNull();
    expect(railOf('t20')).toBe('parent');
  });

  it('should take a top-level add to the last page, where the row lands', async () => {
    createChecklistItem.mockImplementation(async () => item('NEW'));
    const input = await renderList();
    expect(counter().textContent).toBe('1 of 3');

    fireEvent.change(input, { target: { value: 'item NEW' } });
    fireEvent.submit(input.closest('form')!);

    // 26 items is still three pages, and the new row closes the last one.
    await screen.findByText('item NEW');
    expect(counter().textContent).toBe('3 of 3');
    expect(rowOf('NEW').className).toMatch(/animate-fadeIn/);
    await waitFor(() => expect(addInput().readOnly).toBe(false));
    expect(addInput().value).toBe('');
    expect(document.activeElement).toBe(addInput());
  });

  it('should go back to the family the child joins when it is on another page', async () => {
    createChecklistItem.mockImplementation(async () => item('C', 'task', 't10'));
    const input = await renderList();
    // Nested on page 1, then paged away before anything is typed.
    fireEvent.keyDown(input, { key: 'Tab' });
    expect(nestedUnder()).toBe('t10');
    fireEvent.click(next());
    fireEvent.click(next());
    expect(counter().textContent).toBe('3 of 3');

    fireEvent.change(addInput(), { target: { value: 'item C' } });
    fireEvent.submit(addInput().closest('form')!);

    await screen.findByText('item C');
    expect(counter().textContent).toBe('1 of 3');
    expect(rowOf('C').className).toMatch(/animate-fadeIn/);
    // The bar is still aimed at the family it just fed (R18).
    expect(nestedUnder()).toBe('t10');
    await waitFor(() => expect(addInput().readOnly).toBe(false));
    expect(document.activeElement).toBe(addInput());
  });

  it('should carry half-typed text and its nesting across a page change', async () => {
    const input = await renderList();
    fireEvent.keyDown(input, { key: 'Tab' });
    fireEvent.change(addInput(), { target: { value: 'half typed' } });

    fireEvent.click(next());

    // The target is paged away, but it is still a live top-level task, so the
    // bar keeps it rather than dropping to top level (R18, not R8.11).
    expect(counter().textContent).toBe('2 of 3');
    expect(nestedUnder()).toBe('t10');
    expect(addInput().value).toBe('half typed');
    expect(document.activeElement).toBe(addInput());
  });

  it('should still open a collapsed target that the jump brings back into view', async () => {
    const items = [...topLevel(25), item('c1', 'task', 't10')];
    createChecklistItem.mockImplementation(async () => item('C', 'task', 't10'));
    const input = await renderList(items);
    fireEvent.keyDown(input, { key: 'Tab' });
    expect(nestedUnder()).toBe('t10');
    fireEvent.click(screen.getByRole('button', { name: 'Collapse item t10' }));
    expect(screen.queryByText('item c1')).toBeNull();
    fireEvent.click(next());

    fireEvent.change(addInput(), { target: { value: 'item C' } });
    fireEvent.submit(addInput().closest('form')!);

    // Back on page 1 with the family open, so the new child isn't folded away
    // the moment it arrives (FS-0007 R8.9).
    await screen.findByText('item C');
    expect(counter().textContent).toBe('1 of 3');
    expect(screen.getByText('item c1')).toBeTruthy();
    expect(
      screen.getByRole('button', { name: 'Collapse item t10' })
    ).toBeTruthy();
    expect(rowOf('C').className).toMatch(/animate-fadeIn/);
  });
});
