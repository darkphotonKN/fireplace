import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';

/**
 * FS-0008 R13–R14: the add bar is pinned to the bottom of the list card and
 * stays in view while the list area scrolls, without moving off the end of
 * the list. jsdom computes no geometry, so these assert the structure and the
 * classes that produce the pinning — the visual result is HITL.
 */

const fetchChecklist = vi.fn();
const createChecklistItem = vi.fn();
const toast = vi.fn();

vi.mock('next/navigation', () => ({
  useParams: () => ({ planId: 'plan-1' }),
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), refresh: vi.fn() }),
}));

vi.mock('@/components/ui/use-toast', () => ({
  useToast: () => ({ toast }),
}));

vi.mock('@/services/api', () => ({
  fetchChecklist: (...a: unknown[]) => fetchChecklist(...a),
  fetchArchivedChecklist: vi.fn(async () => []),
  createChecklistItem: (...a: unknown[]) => createChecklistItem(...a),
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

const item = (id: string, scope: 'daily' | 'longterm' = 'longterm') => ({
  id,
  description: `item ${id}`,
  done: false,
  scope,
  type: 'task',
  parentId: null,
  planId: 'plan-1',
});

// Long enough that the card would run off the screen without pinning.
const LONG = Array.from({ length: 24 }, (_, i) => item(`L${i}`));

const addInput = () =>
  screen.getByRole('textbox', { name: /^New (task|note)/ }) as HTMLInputElement;
// The add bar's wrapper is what pins; the form inside keeps its own look.
const pinned = () => addInput().closest('form')!.parentElement!;
// Sticky is inert without a scrolling ancestor: this is the bounded list area
// the rows scroll in and the add bar pins to, in list and grid alike.
const listArea = () => document.querySelector<HTMLElement>('[data-list-area]')!;

// One cap for both states (FS-0008 R27): a single page is bounded by it, a
// paged list is fixed at it.
const HUG = /(^|\s)max-h-\[min\(80vh,60rem\)\](\s|$)/;
const PAGED = /(^|\s)h-\[min\(80vh,60rem\)\](\s|$)/;

async function renderLong() {
  fetchChecklist.mockImplementation(async () => LONG);
  const Todo = (await import('./Todo')).default;
  render(<Todo fixedTaskType="longterm" enableTypeFilter />);
  await screen.findByText('item L0');
}

describe('Todo pinned add bar (FS-0008 R13–R14)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
    // Rows are what this file covers, and blocks are the default view,
    // so say which one these expectations are about.
    window.localStorage.setItem('checklistView', 'list');
  });

  it('should pin the add bar to the bottom of the list area', async () => {
    await renderLong();
    expect(pinned().className).toMatch(/\bsticky\b/);
    expect(pinned().className).toMatch(/\bbottom-0\b/);
  });

  it('should scroll the list under the bar in a bounded area holding both', async () => {
    await renderLong();
    const area = listArea();
    expect(area.className).toMatch(/\boverflow-y-auto\b/);
    // Bounded either way: a single page caps with max-h, a paged list gets the
    // fixed window so every page is the same height (FS-0008 R5).
    expect(area.className).toMatch(/\b(max-)?h-\[/);
    // The rows must scroll inside the same box the bar sticks to, and the bar
    // must still come last in the DOM so Tab-to-nest keeps its meaning (R13).
    const rows = area.querySelector('ul')!;
    expect(rows).toBeTruthy();
    expect(rows.compareDocumentPosition(pinned()) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it('should carry the card background, a soft top fade and sit above the row menus', async () => {
    await renderLong();
    // The card's own token, so the strip reads as part of the card rather
    // than a new colour laid over it.
    expect(pinned().className).toMatch(/\bbg-card\b/);
    expect(pinned().querySelector('[data-add-bar-fade]')).toBeTruthy();
    // Row hover menus are z-10; the bar has to win or it gets drawn through.
    expect(pinned().className).toMatch(/\bz-20\b/);
  });

  it('should keep the bar looking the same and sharing the list gutter', async () => {
    await renderLong();
    const form = addInput().closest('form')!;
    // Same pl-8 gutter as the rows, so the type icon stays in the checkbox
    // column and the chevron still has its 32px to hang in.
    expect(pinned().className).toMatch(/\bpl-8\b/);
    expect(document.querySelector('ul')!.className).toMatch(/\bpl-8\b/);
    // Hairline plus the ember underline that draws in on focus.
    expect(form.className).toMatch(/\bborder-b\b/);
    expect(form.querySelector('.group-focus-within\\/add\\:scale-x-100')).toBeTruthy();
    expect(pinned().className).toMatch(/\bpt-3\b/);
    expect(pinned().className).toMatch(/\bpb-6\b/);
    // The add bar's rail piece reaches -top-7 = this gap plus pt-3, so the
    // list area has to carry it now that the bar sits inside the scroller.
    expect(listArea().className).toMatch(/\bspace-y-4\b/);
  });

  it('should still add through the pinned bar and hand focus back', async () => {
    createChecklistItem.mockImplementation(async () => item('NEW'));
    await renderLong();
    const input = addInput();
    input.focus();
    fireEvent.change(input, { target: { value: 'item NEW' } });
    fireEvent.submit(input.closest('form')!);
    // 24 items is two pages, so the appended row lands on the last one and
    // the view goes with it (R17) — the arrival is seen, not left off-page.
    expect(await screen.findByText('item NEW')).toBeTruthy();
    await waitFor(() => expect(addInput().readOnly).toBe(false));
    expect(document.activeElement).toBe(addInput());
    expect(pinned().className).toMatch(/\bsticky\b/);
  });

  it.each([
    [
      'the archived view is open',
      async () => {
        fetchChecklist.mockImplementation(async () => LONG);
        const Todo = (await import('./Todo')).default;
        render(<Todo fixedTaskType="longterm" enableTypeFilter />);
        await screen.findByText('item L0');
        fireEvent.click(screen.getByTitle('Settings'));
        fireEvent.click(screen.getByText('View archived tasks'));
        // The back link naming settings is the archived view's own marker.
        await screen.findByText('Back to settings');
      },
    ],
  ])('should leave no pinned strip when %s', async (_case, mount) => {
    await mount();
    expect(screen.queryByRole('textbox', { name: /^New (task|note)/ })).toBeNull();
    expect(document.querySelector('.sticky')).toBeNull();
    expect(document.querySelector('[data-add-bar-fade]')).toBeNull();
  });

  it('should hold the list area at the cap on every page once the list pages', async () => {
    // 24 items is two pages. Without a fixed height the section would
    // collapse from twenty rows to four when page 2 holds the rest, moving
    // the controls and the bar under the cursor (R29).
    await renderLong();
    const area = listArea();
    expect(area.className).toMatch(PAGED);
    expect(area.className).not.toMatch(/\bmax-h-\[/);
    // The rows region grows, so a short page still leaves the bar at the
    // window's bottom rather than floating under the last row.
    expect(area.firstElementChild!.className).toMatch(/\bflex-1\b/);

    // Page 2 is the short last page: four rows, same height.
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    expect(listArea().className).toMatch(PAGED);
  });

  it('should hold the cap on a middle page as well', async () => {
    fetchChecklist.mockImplementation(async () =>
      Array.from({ length: 45 }, (_, i) => item(`L${i}`))
    );
    const Todo = (await import('./Todo')).default;
    render(<Todo fixedTaskType="longterm" enableTypeFilter />);
    await screen.findByText('item L0');
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    expect(screen.getByTestId('page-counter').textContent).toBe('2 of 3');
    expect(listArea().className).toMatch(PAGED);
  });

  it('should hug its content while everything fits on one page', async () => {
    fetchChecklist.mockImplementation(async () => LONG.slice(0, 4));
    const Todo = (await import('./Todo')).default;
    render(<Todo fixedTaskType="longterm" enableTypeFilter />);
    await screen.findByText('item L0');

    // A four-item plan shouldn't sit in a mostly empty box. Only the cap
    // applies, for the one-huge-family case, and it scrolls inside (R28).
    const area = listArea();
    expect(area.className).toMatch(HUG);
    expect(area.className).toMatch(/\boverflow-y-auto\b/);
    expect(area.className).not.toMatch(/(^|\s)h-\[/);
  });

  it('should bound one page and fix many pages at the same cap in the grid', async () => {
    // Twelve cards is one page, thirteen is two (R26). Same cap as the list,
    // so the board is never taller in one view than in the other (R27).
    window.localStorage.setItem('checklistView', 'grid');
    fetchChecklist.mockImplementation(async () => LONG.slice(0, 12));
    const Todo = (await import('./Todo')).default;
    const { unmount } = render(<Todo fixedTaskType="longterm" enableTypeFilter />);
    await screen.findByText('item L0');
    expect(listArea().className).toMatch(HUG);
    expect(listArea().className).not.toMatch(/(^|\s)h-\[/);
    unmount();

    fetchChecklist.mockImplementation(async () => LONG.slice(0, 13));
    render(<Todo fixedTaskType="longterm" enableTypeFilter />);
    await screen.findByText('item L0');
    expect(listArea().className).toMatch(PAGED);
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    expect(listArea().className).toMatch(PAGED);
  });

  it('should use one cap expression for both states, so page 2 never shrinks the board', async () => {
    fetchChecklist.mockImplementation(async () => LONG.slice(0, 20));
    const Todo = (await import('./Todo')).default;
    const { unmount } = render(<Todo fixedTaskType="longterm" enableTypeFilter />);
    await screen.findByText('item L0');
    const hugCap = listArea().className.match(/\bmax-h-\[([^\]]+)\]/)![1];
    unmount();

    fetchChecklist.mockImplementation(async () => LONG.slice(0, 21));
    render(<Todo fixedTaskType="longterm" enableTypeFilter />);
    await screen.findByText('item L0');
    const pagedCap = listArea().className.match(/(?:^|\s)h-\[([^\]]+)\]/)![1];
    expect(pagedCap).toBe(hugCap);
    // Neither of the old, mismatched bounds survives on the list area.
    expect(listArea().className).not.toMatch(/70vh|40rem/);
  });

  it.each([
    [
      'archiving one',
      async () => {
        const row = screen.getByText('item L0').closest('li')!;
        fireEvent.click(within(row).getByRole('button', { name: /^Actions for/ }));
        fireEvent.click(screen.getByRole('menuitem', { name: 'Archive' }));
      },
    ],
    [
      'filtering to the notes',
      async () => {
        fireEvent.click(screen.getByRole('button', { name: 'Notes' }));
      },
    ],
  ])('should go back to hugging when %s drops the list to one page', async (_case, drop) => {
    // 21 items pages: twenty tasks and one note, so archiving any one leaves
    // twenty and the Notes filter leaves one.
    fetchChecklist.mockImplementation(async () => [
      ...LONG.slice(0, 20),
      { ...item('N0'), type: 'note' },
    ]);
    const Todo = (await import('./Todo')).default;
    render(<Todo fixedTaskType="longterm" enableTypeFilter />);
    await screen.findByText('item L0');
    expect(listArea().className).toMatch(PAGED);

    await drop();
    await waitFor(() => expect(screen.queryByTestId('page-counter')).toBeNull());
    expect(listArea().className).toMatch(HUG);
    expect(listArea().className).not.toMatch(/(^|\s)h-\[/);
  });
});
