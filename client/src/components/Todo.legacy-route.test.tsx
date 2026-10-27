import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';

/**
 * FS-02JBZ R91: the daily scope is retired. The legacy /plans/[planId] route
 * renders a bare <Todo />; it no longer offers the Daily | Long-term tab buttons
 * and shows the long-term list.
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

vi.mock('@/api/client', () => ({
  api: { GET: vi.fn(async () => ({ data: '', response: { status: 200 } })) },
  apiErrorFrom: (e: unknown) => e,
  ApiError: class extends Error {},
}));

vi.mock('@/api/plans', () => ({
  getPlan: vi.fn(async () => ({ id: 'plan-1', dailyReset: false })),
}));

import Todo from './Todo';

const item = (id: string, scope: string, description: string) => ({
  id,
  description,
  done: false,
  scope,
  type: 'task',
  planId: 'plan-1',
});

beforeEach(() => {
  fetchChecklist.mockReset().mockImplementation(async (_p: string, scope: string) =>
    scope === 'longterm'
      ? [item('L1', 'longterm', 'Ship the beta')]
      : [item('D1', 'daily', 'Inbox zero')]
  );
});

describe('Todo on the legacy route (no fixedTaskType)', () => {
  it('should show the long-term list', async () => {
    render(<Todo />);

    expect(await screen.findByText('Ship the beta')).toBeTruthy();
    expect(screen.queryByText('Inbox zero')).toBeNull();
    const scopes = fetchChecklist.mock.calls.map((c) => c[1]);
    expect(scopes).not.toContain('daily');
  });

  it('should offer no Daily | Long-term tab buttons', async () => {
    render(<Todo />);
    await screen.findByText('Ship the beta');

    expect(screen.queryByRole('button', { name: 'Daily' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Long-term' })).toBeNull();
  });
});
