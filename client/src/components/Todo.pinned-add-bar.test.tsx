import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';

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
// Sticky is inert without a scrolling ancestor, so the bar's own parent is
// the bounded list area that scrolls under it.
const listArea = () => pinned().parentElement!;

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
    expect(area.className).toMatch(/\bmax-h-\[/);
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
    // The row is created and appended, but this fixture is 24 items, so it
    // lands on the last page while the view sits on page 1. Jumping to the
    // page an add lands on is R17 (I-0051); until then the arrival is simply
    // off-page, so this slice asserts the create and the focus hand-back.
    await waitFor(() =>
      expect(createChecklistItem).toHaveBeenCalledWith(
        'item NEW',
        'plan-1',
        'longterm',
        expect.objectContaining({ type: 'task' })
      )
    );
    await waitFor(() => expect(addInput().readOnly).toBe(false));
    expect(document.activeElement).toBe(addInput());
    expect(pinned().className).toMatch(/\bsticky\b/);
  });

  it.each([
    [
      'the daily side is AI-only',
      async () => {
        fetchChecklist.mockImplementation(async () =>
          LONG.map((i) => ({ ...i, scope: 'daily' }))
        );
        const Todo = (await import('./Todo')).default;
        render(<Todo fixedTaskType="daily" dailyAIOnly />);
        await screen.findByText('item L0');
      },
    ],
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
  ])('should leave no pinned strip and no scroller when %s', async (_case, mount) => {
    await mount();
    expect(screen.queryByRole('textbox', { name: /^New (task|note)/ })).toBeNull();
    expect(document.querySelector('.sticky')).toBeNull();
    expect(document.querySelector('[data-add-bar-fade]')).toBeNull();
    expect(document.querySelector('.overflow-y-auto')).toBeNull();
  });
});
