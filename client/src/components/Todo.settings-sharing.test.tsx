import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';

/**
 * The plan's Settings shift, for the owner and for a member (FS-12ZMF R28,
 * R31). Which one the viewer is comes from `relation` on getPlan, never from
 * comparing user ids. The daily-reset toggle is the owner's; Sharing and the
 * archived view are there for both.
 */

const getPlan = vi.fn();
const listPlanMembers = vi.fn();
const fetchArchivedChecklist = vi.fn();

vi.mock('next/navigation', () => ({
  useParams: () => ({ planId: 'plan-1' }),
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), refresh: vi.fn() }),
}));

vi.mock('@/components/ui/use-toast', () => ({
  useToast: () => ({ toast: vi.fn() }),
}));

vi.mock('@/services/api', () => ({
  fetchChecklist: vi.fn(async () => [
    {
      id: 'P0',
      description: 'item P0',
      done: false,
      scope: 'longterm',
      type: 'task',
      parentId: null,
      planId: 'plan-1',
    },
  ]),
  fetchArchivedChecklist: (...a: unknown[]) => fetchArchivedChecklist(...a),
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
  getDailyInsights: vi.fn(async () => []),
}));

vi.mock('@/api/plans', () => ({
  getPlan: (...a: unknown[]) => getPlan(...a),
  toggleDailyReset: vi.fn(),
  listPlanMembers: (...a: unknown[]) => listPlanMembers(...a),
  addPlanMember: vi.fn(),
  removePlanMember: vi.fn(),
}));

const people = [
  { id: 'u-1', name: 'Kranti', email: 'kranti@example.com', relation: 'owner', since: '2026-10-01T00:00:00Z' },
  { id: 'u-2', name: 'Nick', email: 'nick@example.com', relation: 'member', since: '2026-10-02T00:00:00Z' },
];

async function openSettingsAs(relation: 'owner' | 'member') {
  getPlan.mockResolvedValue({ id: 'plan-1', dailyReset: false, relation });
  const Todo = (await import('./Todo')).default;
  render(<Todo fixedTaskType="longterm" />);
  await screen.findByText('item P0');
  fireEvent.click(screen.getByTitle('Settings'));
  await screen.findByText('Nick');
}

beforeEach(() => {
  vi.clearAllMocks();
  window.localStorage.clear();
  listPlanMembers.mockResolvedValue(people);
  fetchArchivedChecklist.mockResolvedValue([]);
});

describe('Todo settings — as the owner', () => {
  it('should offer the daily-reset toggle and Sharing with an email field', async () => {
    await openSettingsAs('owner');

    expect(screen.getByText('Refresh daily tasks')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Sharing' })).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: /email/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Remove Nick' })).toBeInTheDocument();
  });
});

describe('Todo settings — as a member', () => {
  it('should show who is on the plan with no controls and no daily-reset toggle', async () => {
    await openSettingsAs('member');

    expect(screen.getByRole('heading', { name: 'Sharing' })).toBeInTheDocument();
    expect(screen.getByText('owner')).toBeInTheDocument();
    expect(screen.queryByText('Refresh daily tasks')).not.toBeInTheDocument();
    expect(screen.queryByRole('textbox', { name: /email/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /remove/i })).not.toBeInTheDocument();
  });

  it('should still open the archived view', async () => {
    await openSettingsAs('member');

    fireEvent.click(screen.getByRole('button', { name: /view archived tasks/i }));

    expect(await screen.findByText('Back to settings')).toBeInTheDocument();
    expect(fetchArchivedChecklist).toHaveBeenCalled();
  });
});

describe('Todo settings — relation not known', () => {
  async function openSettingsWithPlan(plan: Promise<unknown>) {
    getPlan.mockReturnValue(plan);
    const Todo = (await import('./Todo')).default;
    render(<Todo fixedTaskType="longterm" />);
    await screen.findByText('item P0');
    fireEvent.click(screen.getByTitle('Settings'));
    await screen.findByRole('button', { name: /view archived tasks/i });
  }

  it('should hide the daily-reset toggle while getPlan has not answered', async () => {
    await openSettingsWithPlan(new Promise(() => {}));

    expect(screen.queryByText('Refresh daily tasks')).not.toBeInTheDocument();
  });

  it('should hide the daily-reset toggle when getPlan fails', async () => {
    await openSettingsWithPlan(Promise.reject(new Error('boom')));

    expect(screen.queryByText('Refresh daily tasks')).not.toBeInTheDocument();
  });

  it('should hide the daily-reset toggle when getPlan carries no relation', async () => {
    await openSettingsWithPlan(Promise.resolve({ id: 'plan-1', dailyReset: false }));

    expect(screen.queryByText('Refresh daily tasks')).not.toBeInTheDocument();
  });
});
