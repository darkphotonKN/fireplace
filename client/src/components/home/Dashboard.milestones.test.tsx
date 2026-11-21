import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  api,
  item,
  plan,
  plansAged,
  progressSlot,
  renderHome,
} from '@/test/dashboardHarness';
import { deferred, settle } from '@/test/async';
import type { ChecklistItem } from '@/api/checklists';

/**
 * FS-FSFEC slice 2 (I-FSFEC-2): the milestone strip on home's Continue cards
 * (R23–R28, §Edge States).
 *
 * The strip is a second reading of items the fan-out already holds, so the
 * load-bearing assertions are the ones about what does NOT change: the request
 * budget (ADR-0013), the card's height (FS-KSJFR R22), the done/total of a plan
 * with no milestones, and the coral the Today block owns (FS-KSJFR R49).
 */

const { listPlans, listChecklists } = api;

/**
 * Everything R1 has to filter out, around two real milestones: the fetch
 * carries every scope, so the daily rows and the archived one are on the wire.
 */
const mixedPlan = () => [
  item('Ship v1'),
  item('Ship v1 / write docs', { parentId: 'Ship v1' }),
  item('Launch'),
  item('Stretch', { scope: 'daily' }),
  item('Read', { scope: 'daily', done: true }),
  item('Old milestone', { archived: true }),
  item('A thought', { type: 'note' }),
];

const card = (id: string) => progressSlot(id).closest('li') as HTMLElement;

describe('Dashboard — the milestone strip on Continue cards (FS-FSFEC R23–R28)', () => {
  it('should count only long-term, live, top-level tasks as milestones, with no extra request', async () => {
    listPlans.mockResolvedValue([plan('a')]);
    listChecklists.mockResolvedValue(mixedPlan());

    await renderHome();

    expect(await screen.findByText('0 of 2 milestones')).toBeInTheDocument();
    const segments = card('a').querySelectorAll('[data-milestone-segment]');
    expect(segments).toHaveLength(2);
    // The milestone count stands in for done/total; it is not shown beside it.
    expect(within(card('a')).queryByText(/^\d+\/\d+$/)).toBeNull();

    await settle();
    expect(listPlans).toHaveBeenCalledTimes(1);
    expect(listChecklists).toHaveBeenCalledTimes(1);
  });

  it.each([
    { name: 'no items at all', items: [], count: '0/0' },
    {
      name: 'only notes',
      items: [item('n1', { type: 'note' }), item('n2', { type: 'note' })],
      count: '0/0',
    },
    {
      name: 'only daily items',
      items: [item('d1', { scope: 'daily', done: true }), item('d2', { scope: 'daily' })],
      count: '1/2',
    },
    {
      name: 'only archived long-term tasks',
      items: [item('old', { archived: true })],
      count: '0/0',
    },
  ])('should show done/total as before for a plan with $name', async ({ items, count }) => {
    listPlans.mockResolvedValue([plan('a')]);
    listChecklists.mockResolvedValue(items);

    await renderHome();

    await waitFor(() =>
      expect(progressSlot('a')).toHaveTextContent(count),
    );
    expect(document.querySelector('[data-milestone-strip]')).toBeNull();
    expect(screen.queryByText(/milestone/)).toBeNull();
  });

  it('should keep the slot empty while items load, then fill the same slot with the strip', async () => {
    const items = deferred<ChecklistItem[]>();
    listPlans.mockResolvedValue([plan('a')]);
    listChecklists.mockReturnValue(items.promise);

    await renderHome();

    const slot = await screen.findByTestId('continue-progress-a');
    // Never an empty strip or a zero while we don't know yet (R26).
    expect(slot).toHaveTextContent('');
    expect(slot.querySelector('[data-milestone-strip]')).toBeNull();
    const footer = slot.parentElement as HTMLElement;

    items.resolve(mixedPlan());
    await screen.findByText('0 of 2 milestones');

    // Same node, filled in, in the same footer row as the type pill: the strip
    // settles into a row already on screen rather than adding one (R27).
    expect(progressSlot('a')).toBe(slot);
    expect(slot.parentElement).toBe(footer);
    expect(within(footer).getByText('project')).toBeInTheDocument();
    expect(slot.querySelector('[data-milestone-strip]')).not.toBeNull();
  });

  it('should keep the slot empty for good when the items failed', async () => {
    listPlans.mockResolvedValue([plan('a'), plan('b', '2026-08-01T00:00:00Z')]);
    listChecklists.mockImplementation(async (id: string) => {
      if (id === 'b') throw new Error('nope');
      return mixedPlan();
    });

    await renderHome();

    await screen.findByText('0 of 2 milestones');
    const slot = progressSlot('b');
    expect(slot).toHaveTextContent('');
    expect(slot.querySelector('[data-milestone-strip]')).toBeNull();
  });

  it.each([
    { plans: 1, requests: 2 },
    { plans: 3, requests: 4 },
    { plans: 4, requests: 4 },
    { plans: 12, requests: 4 },
  ])(
    'should still issue $requests requests at $plans plans with milestones on every card',
    async ({ plans, requests }) => {
      listPlans.mockResolvedValue(plansAged(plans));
      listChecklists.mockResolvedValue(mixedPlan());

      await renderHome();

      await waitFor(() =>
        expect(screen.getAllByText('0 of 2 milestones')).toHaveLength(
          Math.min(plans, 3),
        ),
      );
      await settle();
      expect(listPlans.mock.calls.length + listChecklists.mock.calls.length).toBe(
        requests,
      );
    },
  );
});

describe('Dashboard — ticks from Today move the strip (FS-FSFEC R25)', () => {
  // A fixed Monday so "due today" is a fact of the fixture, not the machine.
  beforeEach(() => vi.setSystemTime(new Date(2026, 2, 16, 9, 0)));
  afterEach(() => vi.useRealTimers());

  const DUE = '2026-03-16T00:00:00Z';

  it('should close a milestone ticked from Today in the same visit', async () => {
    listPlans.mockResolvedValue([plan('a')]);
    listChecklists.mockResolvedValue([
      item('Ship v1', { dueDate: DUE }),
      item('Launch'),
    ]);

    await renderHome();
    await screen.findByText('0 of 2 milestones');

    await userEvent.click(await screen.findByRole('checkbox', { name: 'Ship v1' }));

    expect(await screen.findByText('1 of 2 milestones')).toBeInTheDocument();
    expect(listChecklists).toHaveBeenCalledTimes(1);
  });

  it('should fill a segment when one of its steps is ticked from Today', async () => {
    listPlans.mockResolvedValue([plan('a')]);
    listChecklists.mockResolvedValue([
      item('Ship v1'),
      item('write docs', { parentId: 'Ship v1', dueDate: DUE }),
      item('cut release', { parentId: 'Ship v1', done: true }),
      item('Launch'),
    ]);

    await renderHome();
    await screen.findByText('0 of 2 milestones');
    const fill = () =>
      card('a').querySelector<HTMLElement>('[data-milestone-fill]')!.style.width;
    expect(fill()).toBe('50%');

    await userEvent.click(await screen.findByRole('checkbox', { name: 'write docs' }));

    await waitFor(() => expect(fill()).toBe('100%'));
    // A full segment is not a closed milestone: only a person closes one (R10).
    expect(screen.getByText('0 of 2 milestones')).toBeInTheDocument();
  });
});
