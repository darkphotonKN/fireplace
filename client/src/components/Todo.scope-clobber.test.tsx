import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';

/**
 * Repro for: an item created on the LONG-TERM panel disappears after a reload.
 *
 * The plan page mounts two <Todo> instances (daily + longterm). Every instance
 * runs two effects that both call setTodos:
 *
 *   A) the taskType effect  -> fetchChecklist(planId, taskType)
 *   B) loadPlanDetails      -> fetchChecklist(planId, 'daily')   <-- hardcoded
 *
 * B ignores taskType, so on the longterm instance it overwrites the longterm
 * list with the DAILY list. B resolves later than A in practice because it
 * awaits Promise.all([fetchChecklist, getPlan]) -- two round trips, not one.
 *
 * The test forces that ordering deterministically rather than racing.
 */

const fetchChecklist = vi.fn();
const getPlan = vi.fn();

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
  updateChecklistItem: vi.fn(),
  deleteChecklistItem: vi.fn(),
  archiveChecklistItem: vi.fn(),
  scheduleChecklistItem: vi.fn(),
  scope: { DAILY: 'daily', LONGTERM: 'longterm' },
  ScopeEnum: { DAILY: 'daily', LONGTERM: 'longterm' },
}));

vi.mock('@/api/insights', () => ({
  getChecklistSuggestion: vi.fn(async () => ''),
  getDailyInsights: vi.fn(async () => []),
}));

vi.mock('@/api/plans', () => ({
  getPlan: (...a: unknown[]) => getPlan(...a),
  toggleDailyReset: vi.fn(),
}));

const DAILY_ITEM = {
  id: 'd1',
  description: 'a daily item',
  done: false,
  scope: 'daily',
  type: 'task',
  archived: false,
  planId: 'plan-1',
};

const LONGTERM_ITEM = {
  id: 'l1',
  description: 'the long-term item I just created',
  done: false,
  scope: 'longterm',
  type: 'task',
  archived: false,
  planId: 'plan-1',
};

describe('Todo — longterm list is not clobbered by the daily fetch', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    // Scope-correct backend: each call returns exactly what was asked for.
    // (Verified against the live gateway: both scopes persist and filter.)
    fetchChecklist.mockImplementation(async (_planId: string, s: string) =>
      s === 'daily' ? [DAILY_ITEM] : [LONGTERM_ITEM]
    );

    // loadPlanDetails awaits getPlan too, so it settles after the scoped fetch.
    getPlan.mockImplementation(
      async () =>
        new Promise((resolve) =>
          setTimeout(() => resolve({ id: 'plan-1', dailyReset: false }), 10)
        )
    );
  });

  it('keeps showing the long-term item after mount', async () => {
    const Todo = (await import('./Todo')).default;
    render(<Todo fixedTaskType="longterm" />);

    // It appears first -- effect A resolved.
    expect(
      await screen.findByText('the long-term item I just created')
    ).toBeTruthy();

    // ...and must still be there once loadPlanDetails settles.
    await waitFor(() => expect(getPlan).toHaveBeenCalled());

    await waitFor(() => {
      expect(
        screen.queryByText('the long-term item I just created')
      ).not.toBeNull();
    });

    // And the daily item must never leak into the long-term panel.
    expect(screen.queryByText('a daily item')).toBeNull();
  });
});
