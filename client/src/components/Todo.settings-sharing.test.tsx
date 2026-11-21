import { describe, it, expect, beforeEach } from 'vitest';
import { screen, fireEvent } from '@testing-library/react';
import { api, item, renderTodo } from '@/test/todoHarness';

/**
 * The plan's Settings shift, for the owner and for a member (FS-12ZMF R28,
 * R31). Which one the viewer is comes from `relation` on getPlan, never from
 * comparing user ids. The daily-reset toggle is the owner's; Sharing and the
 * archived view are there for both.
 */

const { getPlan, listPlanMembers, fetchArchivedChecklist } = api;

const people = [
  { id: 'u-1', name: 'Kranti', email: 'kranti@example.com', relation: 'owner', since: '2026-10-01T00:00:00Z' },
  { id: 'u-2', name: 'Nick', email: 'nick@example.com', relation: 'member', since: '2026-10-02T00:00:00Z' },
];

async function openSettingsAs(relation: 'owner' | 'member') {
  getPlan.mockResolvedValue({ id: 'plan-1', dailyReset: false, relation });
  await renderTodo([item('P0')], { props: { enableTypeFilter: false } });
  fireEvent.click(screen.getByTitle('Settings'));
  await screen.findByText('Nick');
}

beforeEach(() => {
  listPlanMembers.mockResolvedValue(people);
});

describe('Todo settings — as the owner', () => {
  // FS-02JBZ R91: the daily scope is retired, so there is no daily reset to
  // configure. Sharing and the archived view are what settings still holds.
  it('should offer Sharing with an email field and no daily-reset toggle', async () => {
    await openSettingsAs('owner');

    expect(screen.queryByText('Refresh daily tasks')).not.toBeInTheDocument();
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Sharing' })).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: /email/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Remove Nick' })).toBeInTheDocument();
  });
});

describe('Todo settings — as a member', () => {
  it('should show who is on the plan with no controls', async () => {
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
    await renderTodo([item('P0')], { props: { enableTypeFilter: false } });
    fireEvent.click(screen.getByTitle('Settings'));
    await screen.findByRole('button', { name: /view archived tasks/i });
  }

  // Factories, so the rejection is made inside its test and not at collect time.
  it.each([
    ['while getPlan has not answered', () => new Promise(() => {})],
    ['when getPlan fails', () => Promise.reject(new Error('boom'))],
    ['when getPlan carries no relation', () => Promise.resolve({ id: 'plan-1', dailyReset: false })],
  ])('should hide Sharing %s', async (_case, plan) => {
    await openSettingsWithPlan(plan());

    expect(screen.queryByRole('heading', { name: 'Sharing' })).not.toBeInTheDocument();
  });
});
