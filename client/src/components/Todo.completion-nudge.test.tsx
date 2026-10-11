import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, within, act } from '@testing-library/react';

/**
 * The completion nudge end to end (FS-FSFEC R8–R14): Close goes through the
 * page's own optimistic toggle, so a failure reverts and shows the existing
 * error, and the closed milestone stays where it was. Grid only.
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
  getDailyInsights: vi.fn(async () => []),
}));

vi.mock('@/api/plans', () => ({
  getPlan: vi.fn(async () => ({ id: 'plan-1', dailyReset: false })),
  toggleDailyReset: vi.fn(),
}));

const item = (id: string, extra: Partial<Record<string, unknown>> = {}) => ({
  id,
  description: `item ${id}`,
  done: false,
  scope: 'longterm',
  type: 'task',
  parentId: null,
  planId: 'plan-1',
  ...extra,
});

// P1 is complete: both task steps done, itself still open.
const PLAN = [
  item('P0'),
  item('P1'),
  item('S1', { parentId: 'P1', done: true }),
  item('S2', { parentId: 'P1', done: true }),
  item('P2'),
];

const card = (id: string) =>
  document.querySelector(`[data-checklist-card="${id}"]`) as HTMLElement;
const nudge = () => document.querySelector('[data-completion-nudge]');
const closeButton = () =>
  within(card('P1')).getByRole('button', { name: 'Close item P1' });

async function renderTodo() {
  fetchChecklist.mockImplementation(async () => PLAN);
  const Todo = (await import('./Todo')).default;
  render(<Todo fixedTaskType="longterm" enableTypeFilter />);
  await screen.findByText('item P0');
}

describe('Todo completion nudge', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
    updateChecklistItem.mockResolvedValue({});
  });

  it('should close the milestone on Close and keep it in place, faded, its segment full', async () => {
    await renderTodo();
    expect(screen.getByText('0 of 3 milestones')).toBeTruthy();

    await act(async () => fireEvent.click(closeButton()));

    expect(updateChecklistItem).toHaveBeenCalledWith(
      'P1',
      { done: true },
      'plan-1',
      'longterm'
    );
    expect(screen.getByText('1 of 3 milestones')).toBeTruthy();
    expect(nudge()).toBeNull();
    expect(card('P1').className).toContain('opacity-60');
    const order = Array.from(
      document.querySelectorAll('[data-checklist-card]')
    ).map((el) => el.getAttribute('data-checklist-card'));
    expect(order).toEqual(['P0', 'P1', 'P2']);
  });

  it('should revert a failed Close and show the existing error', async () => {
    let reject!: (e: Error) => void;
    updateChecklistItem.mockImplementation(
      () => new Promise((_, r) => (reject = r))
    );
    vi.spyOn(console, 'error').mockImplementation(() => {});
    await renderTodo();

    fireEvent.click(closeButton());
    // Optimistic: closed before the request lands.
    expect(card('P1').className).toContain('opacity-60');

    await act(async () => reject(new Error('nope')));

    expect(card('P1').className).not.toContain('opacity-60');
    expect(nudge()).not.toBeNull();
    expect(
      screen.getByText('Failed to update task status. Please try again.')
    ).toBeTruthy();
  });

  it('should show no nudge in the checklist reading', async () => {
    await renderTodo();
    expect(nudge()).not.toBeNull();

    fireEvent.click(
      within(screen.getByRole('group', { name: 'View' })).getByRole('button', {
        name: 'Checklist',
      })
    );

    expect(nudge()).toBeNull();
    expect(screen.queryByRole('button', { name: 'Close item P1' })).toBeNull();
  });
});
