import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import DayPlanSection from './DayPlanSection';
import { toDateOnly } from '@/lib/itemDates';
import { ApiError } from '@/api/client';

/**
 * The Day plan section on the plan page (FS-02JBZ R62–R74). It reads the day
 * plan for the browser-local date (never generating it), resolves it against
 * the plan's live long-term items, and acts on those items through the
 * existing endpoints.
 */

const fetchChecklist = vi.fn();
const createChecklistItem = vi.fn();
const updateChecklistItem = vi.fn();
const getDayPlan = vi.fn();
const planMyDay = vi.fn();
const tickDayPlanEntry = vi.fn();
const untickDayPlanEntry = vi.fn();
const setItemRevisions = vi.fn();

vi.mock('@/services/api', () => ({
  fetchChecklist: (...a: unknown[]) => fetchChecklist(...a),
  createChecklistItem: (...a: unknown[]) => createChecklistItem(...a),
  updateChecklistItem: (...a: unknown[]) => updateChecklistItem(...a),
}));

vi.mock('@/api/insights', () => ({
  getDayPlan: (...a: unknown[]) => getDayPlan(...a),
  planMyDay: (...a: unknown[]) => planMyDay(...a),
  tickDayPlanEntry: (...a: unknown[]) => tickDayPlanEntry(...a),
  untickDayPlanEntry: (...a: unknown[]) => untickDayPlanEntry(...a),
  setItemRevisions: (...a: unknown[]) => setItemRevisions(...a),
}));

const item = (id: string, extra: Partial<Record<string, unknown>> = {}) => ({
  id,
  description: `item ${id}`,
  done: false,
  archived: false,
  scope: 'longterm',
  type: 'task',
  planId: 'plan-1',
  ...extra,
});

const ITEMS = [
  item('M1', { description: 'Ship the beta' }),
  item('S1', { parentId: 'M1', description: 'Write the release notes' }),
  item('S2', { parentId: 'M1', description: 'Fix the login bug', done: true }),
  item('S3', { parentId: 'M1', description: 'Old idea', archived: true }),
];

const DRAWN_AT = '2026-10-25T07:00:00Z';
const today = () => toDateOnly(new Date());

const STEP = {
  id: 'e-step',
  kind: 'existing_item',
  itemId: 'S1',
  parentItemId: 'M1',
  text: 'Write the release notes',
  reason: 'The beta needs notes before it ships',
  minutes: 25,
  ticked: false,
};
const NEW = {
  id: 'e-new',
  kind: 'new_step',
  parentItemId: 'M1',
  text: 'Book a demo slot',
  reason: 'Nothing is scheduled yet',
  ticked: false,
};
const DONE = {
  id: 'e-done',
  kind: 'existing_item',
  itemId: 'S2',
  parentItemId: 'M1',
  text: 'Fix the login bug',
  reason: 'It blocks sign-ups',
  ticked: false,
};

const planned = (
  groups: { label: string; entries: unknown[] }[],
  extra: Record<string, unknown> = {}
) => ({
  planId: 'plan-1',
  date: today(),
  status: 'planned',
  source: 'ai',
  generatedAt: DRAWN_AT,
  refreshesRemaining: 3,
  groups,
  ...extra,
});

const notPlanned = (extra: Record<string, unknown> = {}) => ({
  planId: 'plan-1',
  date: today(),
  status: 'not_planned',
  refreshesRemaining: 3,
  groups: [],
  ...extra,
});

const PLAN = planned([
  { label: 'Release', entries: [STEP, NEW] },
  { label: 'Bugs', entries: [DONE] },
]);

const limitReached = () =>
  new ApiError(
    {
      code: 'BRIEF_REFRESH_LIMIT_REACHED',
      status: 429,
      detail: 'prose that may change',
    } as never,
    429
  );

const mount = () => render(<DayPlanSection planId="plan-1" />);
const section = () => screen.getByRole('region', { name: 'Day plan' });
const planAgain = () => screen.getByRole('button', { name: /^(Plan again|Planning)/ });

beforeEach(() => {
  window.localStorage.clear();
  fetchChecklist.mockReset().mockResolvedValue(ITEMS);
  getDayPlan.mockReset().mockResolvedValue(PLAN);
  planMyDay.mockReset();
  createChecklistItem.mockReset();
  updateChecklistItem.mockReset();
  // The server echoes the entry back, ticked (R25).
  tickDayPlanEntry
    .mockReset()
    .mockImplementation(async (id: string, itemId?: string) => ({
      ...[STEP, NEW, DONE].find((e) => e.id === id),
      ...(itemId ? { kind: 'existing_item', itemId } : {}),
      ticked: true,
    }));
  untickDayPlanEntry.mockReset().mockResolvedValue(undefined);
});

afterEach(() => {
  vi.useRealTimers();
});

describe('DayPlanSection', () => {
  it('should read the day plan for the browser-local date and the long-term items, never planning on mount', async () => {
    mount();
    await screen.findByText('Write the release notes');
    expect(getDayPlan).toHaveBeenCalledWith('plan-1', today());
    expect(fetchChecklist).toHaveBeenCalledWith('plan-1', 'longterm');
    expect(planMyDay).not.toHaveBeenCalled();
  });

  it('should carry the "i" tooltip explaining daily progression', async () => {
    mount();
    expect(
      screen.getByRole('button', { name: 'Information about the day plan' })
    ).toBeTruthy();
    const tip = screen.getByRole('tooltip');
    expect(tip).toHaveTextContent(/daily progression/i);
    expect(tip).toHaveTextContent(/plan your day/i);
    expect(tip).toHaveTextContent(/tick what you did today/i);
    expect(tip).toHaveTextContent(/revision/i);
    await screen.findByText('Write the release notes');
  });

  describe('not planned (R65)', () => {
    beforeEach(() => {
      getDayPlan.mockResolvedValue(notPlanned());
    });

    it('should offer Plan my day and plan only when it is pressed', async () => {
      planMyDay.mockResolvedValue(PLAN);
      mount();
      const button = await screen.findByRole('button', { name: 'Plan my day' });
      expect(screen.getByText(/haven.t planned today/i)).toBeTruthy();
      expect(planMyDay).not.toHaveBeenCalled();

      await userEvent.click(button);

      expect(planMyDay).toHaveBeenCalledTimes(1);
      expect(planMyDay).toHaveBeenCalledWith('plan-1', today());
      expect(await screen.findByText('Write the release notes')).toBeTruthy();
      expect(screen.queryByRole('button', { name: 'Plan my day' })).toBeNull();
    });

    it('should show a loading state inside the section while planning', async () => {
      let resolve!: (v: unknown) => void;
      planMyDay.mockReturnValue(new Promise((r) => (resolve = r)));
      mount();
      await userEvent.click(await screen.findByRole('button', { name: 'Plan my day' }));

      expect(
        within(section()).getByRole('list', { name: 'Planning your day' })
      ).toBeTruthy();

      await act(async () => resolve(PLAN));
      expect(await screen.findByText('Write the release notes')).toBeTruthy();
    });

    it('should keep the button and offer a quiet retry when planning fails', async () => {
      planMyDay
        .mockRejectedValueOnce(
          new ApiError({ code: 'SERVICE_UNAVAILABLE', status: 503 } as never, 503)
        )
        .mockResolvedValueOnce(PLAN);
      mount();
      await userEvent.click(await screen.findByRole('button', { name: 'Plan my day' }));

      expect(await screen.findByText(/Couldn.t plan your day/)).toBeTruthy();
      expect(screen.queryByRole('alert')).toBeNull();
      await userEvent.click(screen.getByRole('button', { name: 'Try again' }));
      expect(await screen.findByText('Write the release notes')).toBeTruthy();
    });
  });

  describe('planned (R66, R67)', () => {
    it('should render groups in order with labels, milestone names, text, reasons and minutes', async () => {
      mount();
      expect(screen.getByTestId('day-plan-skeleton')).toBeTruthy();
      await screen.findByText('Write the release notes');
      expect(screen.queryByTestId('day-plan-skeleton')).toBeNull();

      const labels = within(section()).getAllByRole('heading', { level: 3 });
      expect(labels.map((h) => h.textContent)).toEqual(['Release', 'Bugs']);

      expect(screen.getAllByText('Ship the beta')).toHaveLength(3);
      expect(screen.getByText('The beta needs notes before it ships')).toBeTruthy();
      expect(screen.getByText('~25 min')).toBeTruthy();
      expect(screen.getByText('Book a demo slot')).toBeTruthy();
      expect(screen.getByText('Nothing is scheduled yet')).toBeTruthy();
    });

    it('should show a done item as completed and keep its row', async () => {
      mount();
      const text = await screen.findByText('Fix the login bug');
      expect(text.className).toMatch(/line-through/);
      expect(screen.getByText('It blocks sign-ups')).toBeTruthy();
    });

    it('should hide entries whose item is archived or deleted, and new steps under a gone or closed parent', async () => {
      getDayPlan.mockResolvedValue(
        planned([
          {
            label: 'Mixed',
            entries: [
              STEP,
              { ...STEP, id: 'arch', itemId: 'S3', text: 'Old idea' },
              { ...STEP, id: 'del', itemId: 'nope', text: 'Deleted thing' },
              { ...NEW, id: 'orphan', parentItemId: 'nope', text: 'Orphan step' },
              { ...NEW, id: 'closed', parentItemId: 'S2', text: 'Under a done parent' },
            ],
          },
          { label: 'All stale', entries: [{ ...NEW, id: 'x', parentItemId: 'nope' }] },
        ])
      );
      mount();
      await screen.findByText('Write the release notes');
      for (const gone of ['Old idea', 'Deleted thing', 'Orphan step', 'Under a done parent']) {
        expect(screen.queryByText(gone)).toBeNull();
      }
      expect(screen.queryByText('All stale')).toBeNull();
      expect(within(section()).getAllByRole('listitem')).toHaveLength(1);
    });

    it('should say there is nothing to plan, and invite a first milestone when there is none', async () => {
      getDayPlan.mockResolvedValue(planned([]));
      fetchChecklist.mockResolvedValue([]);
      mount();
      expect(await screen.findByText(/Start a milestone in your plan/)).toBeTruthy();
    });

    it('should say there is nothing to plan when milestones exist but the plan is empty', async () => {
      getDayPlan.mockResolvedValue(planned([]));
      mount();
      expect(await screen.findByText('Nothing to plan for today.')).toBeTruthy();
    });

    it('should re-read the items on return so an item finished elsewhere renders done', async () => {
      mount();
      const text = await screen.findByText('Write the release notes');
      expect(text.className).not.toMatch(/line-through/);

      fetchChecklist.mockResolvedValue([ITEMS[0], { ...ITEMS[1], done: true }, ITEMS[2]]);
      await act(async () => {
        window.dispatchEvent(new Event('focus'));
      });
      await waitFor(() =>
        expect(screen.getByText('Write the release notes').className).toMatch(/line-through/)
      );
      expect(getDayPlan).toHaveBeenCalledTimes(1);
    });

    it('should read the new date after local midnight and show not planned (R64)', async () => {
      vi.useFakeTimers({ toFake: ['Date'] });
      vi.setSystemTime(new Date(2026, 9, 25, 23, 59));
      mount();
      await screen.findByText('Write the release notes');
      expect(getDayPlan).toHaveBeenLastCalledWith('plan-1', '2026-10-25');

      getDayPlan.mockResolvedValue(notPlanned({ date: '2026-10-26' }));
      vi.setSystemTime(new Date(2026, 9, 26, 0, 1));
      await act(async () => {
        window.dispatchEvent(new Event('focus'));
      });
      await waitFor(() =>
        expect(getDayPlan).toHaveBeenLastCalledWith('plan-1', '2026-10-26')
      );
      expect(await screen.findByRole('button', { name: 'Plan my day' })).toBeTruthy();
      expect(planMyDay).not.toHaveBeenCalled();
    });
  });

  describe('Plan again (R68)', () => {
    it('should show the count and say nothing has changed when the redraw is the plan on screen', async () => {
      planMyDay.mockResolvedValue(PLAN);
      mount();
      await screen.findByText('Write the release notes');
      expect(planAgain()).toHaveTextContent('3 left');

      await userEvent.click(planAgain());

      expect(planMyDay).toHaveBeenCalledWith('plan-1', today());
      expect(await screen.findByText(/Nothing has changed since this plan was made/)).toBeTruthy();
      expect(screen.getByText('Write the release notes')).toBeTruthy();
    });

    it('should retire the action at zero without asking the server', async () => {
      getDayPlan.mockResolvedValue({ ...PLAN, refreshesRemaining: 0 });
      mount();
      await screen.findByText('Write the release notes');
      expect(planAgain()).toHaveTextContent('0 left');
      expect(planAgain()).toBeDisabled();
      await userEvent.click(planAgain());
      expect(planMyDay).not.toHaveBeenCalled();
    });

    it('should retire the action and keep the plan on a 429 by code', async () => {
      getDayPlan.mockResolvedValue({ ...PLAN, refreshesRemaining: 1 });
      planMyDay.mockRejectedValue(limitReached());
      mount();
      await screen.findByText('Write the release notes');

      await userEvent.click(planAgain());

      await waitFor(() => expect(planAgain()).toBeDisabled());
      expect(planAgain()).toHaveTextContent('0 left');
      expect(screen.getByText('Write the release notes')).toBeTruthy();
      expect(screen.queryByRole('button', { name: 'Try again' })).toBeNull();
    });

    it('should keep the plan and offer a quiet retry for any other failure, whatever its detail says', async () => {
      planMyDay
        .mockRejectedValueOnce(
          new ApiError(
            { code: 'SERVICE_UNAVAILABLE', status: 503, detail: 'refresh limit reached' } as never,
            503
          )
        )
        .mockResolvedValueOnce(PLAN);
      mount();
      await screen.findByText('Write the release notes');

      await userEvent.click(planAgain());

      const retry = await screen.findByRole('button', { name: 'Try again' });
      expect(planAgain()).not.toBeDisabled();
      expect(screen.queryByRole('alert')).toBeNull();
      await userEvent.click(retry);
      expect(planMyDay).toHaveBeenCalledTimes(2);
      expect(await screen.findByText(/Nothing has changed/)).toBeTruthy();
    });

    it('should replace the plan after a counted redraw and forget what was done to the old entries', async () => {
      createChecklistItem.mockResolvedValue(
        item('S9', { parentId: 'M1', description: 'Book a demo slot' })
      );
      planMyDay
        .mockResolvedValueOnce(
          planned(
            [{ label: 'Comms', entries: [{ ...NEW, id: 'e-draft', text: 'Draft the announcement' }] }],
            { generatedAt: '2026-10-25T09:00:00Z', refreshesRemaining: 2 }
          )
        )
        .mockResolvedValueOnce(
          planned([{ label: 'Release', entries: [NEW] }], {
            generatedAt: '2026-10-25T10:00:00Z',
            refreshesRemaining: 1,
          })
        );
      mount();
      await userEvent.click(await screen.findByRole('button', { name: 'Add Book a demo slot' }));
      expect(await screen.findByText('Added')).toBeTruthy();

      await userEvent.click(planAgain());
      expect(await screen.findByText('Draft the announcement')).toBeTruthy();
      expect(screen.queryByText('Write the release notes')).toBeNull();
      expect(planAgain()).toHaveTextContent('2 left');
      expect(screen.queryByText(/Nothing has changed/)).toBeNull();

      await userEvent.click(planAgain());
      expect(await screen.findByRole('button', { name: 'Add Book a demo slot' })).toBeTruthy();
      expect(screen.queryByText('Added')).toBeNull();
    });

    it('should read the new day instead of redrawing yesterday after local midnight', async () => {
      vi.useFakeTimers({ toFake: ['Date'] });
      vi.setSystemTime(new Date(2026, 9, 25, 23, 59));
      mount();
      await screen.findByText('Write the release notes');

      vi.setSystemTime(new Date(2026, 9, 26, 0, 1));
      await userEvent.click(planAgain());

      await waitFor(() =>
        expect(getDayPlan).toHaveBeenLastCalledWith('plan-1', '2026-10-26')
      );
      expect(planMyDay).not.toHaveBeenCalled();
    });
  });

  describe('basic plan (R69)', () => {
    it('should label a rules plan as basic, explain it, and still offer Plan my day', async () => {
      getDayPlan.mockResolvedValue({ ...PLAN, source: 'rules' });
      planMyDay.mockResolvedValue({ ...PLAN, generatedAt: '2026-10-25T09:00:00Z' });
      mount();
      await screen.findByText('Write the release notes');

      expect(within(section()).getByText('Basic plan')).toBeTruthy();
      expect(within(section()).getByText(/AI wasn.t available/)).toBeTruthy();
      const button = screen.getByRole('button', { name: /^Plan my day/ });
      await userEvent.click(button);
      expect(planMyDay).toHaveBeenCalledTimes(1);
      await waitFor(() => expect(within(section()).queryByText('Basic plan')).toBeNull());
    });

    it('should not label an AI plan as basic', async () => {
      mount();
      await screen.findByText('Write the release notes');
      expect(within(section()).queryByText('Basic plan')).toBeNull();
    });
  });

  it('should render nothing when the plan is not reachable (R70)', async () => {
    getDayPlan.mockRejectedValue(new ApiError({ code: 'NOT_FOUND', status: 404 } as never, 404));
    const { container } = mount();
    await waitFor(() => expect(container).toBeEmptyDOMElement());
  });

  it('should offer a quiet retry when the read fails, never an alert (R71)', async () => {
    getDayPlan.mockRejectedValueOnce(
      new ApiError({ code: 'BRIEF_DATE_OUT_OF_RANGE', status: 400 } as never, 400)
    );
    mount();
    const retry = await screen.findByRole('button', { name: 'Try again' });
    expect(screen.queryByRole('alert')).toBeNull();

    await userEvent.click(retry);
    expect(await screen.findByText('Write the release notes')).toBeTruthy();
  });

  describe('show / hide (R72)', () => {
    it('should hide and remember the preference under the existing key', async () => {
      mount();
      await screen.findByText('Write the release notes');
      await userEvent.click(screen.getByRole('button', { name: 'Hide' }));
      expect(screen.queryByText('Write the release notes')).toBeNull();
      expect(window.localStorage.getItem('showDailyInsights')).toBe('false');
    });

    it('should start hidden when the preference says so', async () => {
      window.localStorage.setItem('showDailyInsights', 'false');
      mount();
      expect(await screen.findByRole('button', { name: 'Show' })).toBeTruthy();
      expect(screen.queryByText('Write the release notes')).toBeNull();
      fireEvent.click(screen.getByRole('button', { name: 'Show' }));
      expect(await screen.findByText('Write the release notes')).toBeTruthy();
    });
  });

  describe('Add (R73, R84)', () => {
    it('should add a new step as a long-term child of its parent, never a daily item', async () => {
      createChecklistItem.mockResolvedValue(
        item('S9', { parentId: 'M1', description: 'Book a demo slot' })
      );
      mount();
      await userEvent.click(await screen.findByRole('button', { name: 'Add Book a demo slot' }));

      expect(createChecklistItem).toHaveBeenCalledTimes(1);
      expect(createChecklistItem).toHaveBeenCalledWith(
        'Book a demo slot',
        'plan-1',
        'longterm',
        { type: 'task', parentId: 'M1' }
      );
      expect(await screen.findByText('Added')).toBeTruthy();
      expect(screen.queryByRole('button', { name: 'Add Book a demo slot' })).toBeNull();
    });

    it('should keep the entry when the create fails and show the existing error', async () => {
      createChecklistItem.mockRejectedValue(new Error('boom'));
      mount();
      await userEvent.click(await screen.findByRole('button', { name: 'Add Book a demo slot' }));

      expect(
        await screen.findByText('Failed to add suggested task. Please try again.')
      ).toBeTruthy();
      expect(screen.getByRole('button', { name: 'Add Book a demo slot' })).toBeTruthy();
      expect(screen.queryByText('Added')).toBeNull();
    });
  });

  it('should hide a dismissed entry for the session and record nothing (R74)', async () => {
    mount();
    await userEvent.click(await screen.findByRole('button', { name: 'Dismiss Book a demo slot' }));
    expect(screen.queryByText('Book a demo slot')).toBeNull();
    expect(screen.getByText('Write the release notes')).toBeTruthy();
    expect(planMyDay).not.toHaveBeenCalled();
    expect(createChecklistItem).not.toHaveBeenCalled();
  });

  it('should keep what was dismissed through an unchanged redraw', async () => {
    planMyDay.mockResolvedValue(PLAN);
    mount();
    await userEvent.click(await screen.findByRole('button', { name: 'Dismiss Book a demo slot' }));
    await userEvent.click(planAgain());
    await screen.findByText(/Nothing has changed/);
    expect(screen.queryByText('Book a demo slot')).toBeNull();
  });

  describe('day ticks (R75–R79, R82, R83)', () => {
    const dayTick = (text: string) =>
      screen.getByRole('checkbox', { name: `Done for today: ${text}` });
    const alsoAsk = () => screen.queryByText(/^Also mark '.*' done\?$/);

    const notTickable = () =>
      new ApiError(
        {
          code: 'DAY_PLAN_ENTRY_NOT_TICKABLE',
          status: 400,
          detail: 'prose that may change',
        } as never,
        400
      );

    it('should tick at once, call the day tick and never touch the item', async () => {
      let land!: (v: unknown) => void;
      tickDayPlanEntry.mockImplementationOnce(
        () => new Promise((r) => (land = r))
      );
      mount();
      await screen.findByText('Write the release notes');

      fireEvent.click(dayTick('Write the release notes'));

      expect(dayTick('Write the release notes')).toBeChecked();
      expect(tickDayPlanEntry).toHaveBeenCalledWith('e-step', undefined);
      await act(async () => land({ ...STEP, ticked: true }));
      expect(dayTick('Write the release notes')).toBeChecked();
      expect(updateChecklistItem).not.toHaveBeenCalled();
    });

    it('should revert a tick that fails', async () => {
      vi.spyOn(console, 'error').mockImplementation(() => {});
      tickDayPlanEntry.mockRejectedValueOnce(new Error('down'));
      mount();
      await screen.findByText('Write the release notes');

      fireEvent.click(dayTick('Write the release notes'));

      await waitFor(() =>
        expect(dayTick('Write the release notes')).not.toBeChecked()
      );
      expect(alsoAsk()).toBeNull();
      expect(getDayPlan).toHaveBeenCalledTimes(1);
    });

    it('should show a ticked entry ticked, untick it, and revert an untick that fails', async () => {
      vi.spyOn(console, 'error').mockImplementation(() => {});
      getDayPlan.mockResolvedValue(
        planned([{ label: 'Release', entries: [{ ...STEP, ticked: true }] }])
      );
      untickDayPlanEntry
        .mockResolvedValueOnce(undefined)
        .mockRejectedValueOnce(new Error('down'));
      tickDayPlanEntry.mockResolvedValueOnce({ ...STEP, ticked: true });
      mount();
      await screen.findByText('Write the release notes');
      expect(dayTick('Write the release notes')).toBeChecked();

      fireEvent.click(dayTick('Write the release notes'));
      expect(dayTick('Write the release notes')).not.toBeChecked();
      expect(untickDayPlanEntry).toHaveBeenCalledWith('e-step');
      await waitFor(() =>
        expect(dayTick('Write the release notes')).not.toBeDisabled()
      );

      fireEvent.click(dayTick('Write the release notes'));
      await waitFor(() =>
        expect(dayTick('Write the release notes')).not.toBeDisabled()
      );
      fireEvent.click(dayTick('Write the release notes'));
      await waitFor(() =>
        expect(dayTick('Write the release notes')).toBeChecked()
      );
      expect(untickDayPlanEntry).toHaveBeenCalledTimes(2);
      expect(updateChecklistItem).not.toHaveBeenCalled();
    });

    it('should revert quietly and re-read the day plan when the entry is not tickable, by code', async () => {
      tickDayPlanEntry.mockRejectedValueOnce(notTickable());
      mount();
      await screen.findByText('Write the release notes');
      // The fresh read says the item is gone: the entry hides.
      getDayPlan.mockResolvedValue(PLAN);
      fetchChecklist.mockResolvedValue(ITEMS.filter((i) => i.id !== 'S1'));

      fireEvent.click(dayTick('Write the release notes'));

      await waitFor(() => expect(getDayPlan).toHaveBeenCalledTimes(2));
      await waitFor(() =>
        expect(screen.queryByText('Write the release notes')).toBeNull()
      );
      expect(screen.queryByTestId('day-plan-skeleton')).toBeNull();
      expect(screen.queryByText(/Couldn.t/)).toBeNull();
    });

    it('should revert quietly and re-read when another tab replaced the draw (404 NOT_FOUND), by code', async () => {
      const logged = vi.spyOn(console, 'error').mockImplementation(() => {});
      tickDayPlanEntry.mockRejectedValueOnce(
        new ApiError(
          { code: 'NOT_FOUND', status: 404, detail: 'prose that may change' } as never,
          404
        )
      );
      mount();
      await screen.findByText('Write the release notes');
      // The fresh read is the draw the other tab made.
      getDayPlan.mockResolvedValue(
        planned([{ label: 'Elsewhere', entries: [DONE] }], {
          generatedAt: '2026-10-25T09:00:00Z',
          refreshesRemaining: 2,
        })
      );

      fireEvent.click(dayTick('Write the release notes'));

      await waitFor(() => expect(getDayPlan).toHaveBeenCalledTimes(2));
      expect(await screen.findByText('Elsewhere')).toBeTruthy();
      expect(screen.queryByText('Write the release notes')).toBeNull();
      expect(alsoAsk()).toBeNull();
      expect(screen.queryByText(/Couldn.t/)).toBeNull();
      expect(logged).not.toHaveBeenCalledWith(
        'Failed to tick the day plan entry:',
        expect.anything()
      );
    });

    it('should not tick while Plan again is drawing, so a redraw cannot drop a tick', async () => {
      let land!: (v: unknown) => void;
      planMyDay.mockReturnValue(new Promise((r) => (land = r)));
      getDayPlan.mockResolvedValue(
        planned([
          { label: 'Release', entries: [STEP, { ...DONE, ticked: true }] },
        ])
      );
      mount();
      await screen.findByText('Write the release notes');

      await userEvent.click(planAgain());

      expect(planAgain()).toBeDisabled();
      expect(dayTick('Write the release notes')).toBeDisabled();
      expect(dayTick('Fix the login bug')).toBeDisabled();
      fireEvent.click(dayTick('Write the release notes'));
      fireEvent.click(dayTick('Fix the login bug'));
      expect(tickDayPlanEntry).not.toHaveBeenCalled();
      expect(untickDayPlanEntry).not.toHaveBeenCalled();

      await act(async () => land(PLAN));
      await waitFor(() =>
        expect(dayTick('Write the release notes')).not.toBeDisabled()
      );
    });

    it('should ask inline to mark the item done after a tick, and drop the ask after 8 seconds', async () => {
      mount();
      await screen.findByText('Write the release notes');
      vi.useFakeTimers();

      await act(async () => {
        fireEvent.click(dayTick('Write the release notes'));
      });

      expect(alsoAsk()).toHaveTextContent(
        "Also mark 'Write the release notes' done?"
      );
      expect(screen.queryByRole('dialog')).toBeNull();
      act(() => vi.advanceTimersByTime(7_900));
      expect(alsoAsk()).not.toBeNull();
      act(() => vi.advanceTimersByTime(200));
      expect(alsoAsk()).toBeNull();
    });

    it('should set the item done on Yes, and tell the rest of the page', async () => {
      const { onPlanItemsChanged } = await import('@/lib/planItemsChanged');
      const heard = vi.fn();
      const off = onPlanItemsChanged('plan-1', heard);
      updateChecklistItem.mockResolvedValue({ ...ITEMS[1], done: true });
      mount();
      await screen.findByText('Write the release notes');

      fireEvent.click(dayTick('Write the release notes'));
      await userEvent.click(await screen.findByRole('button', { name: 'Yes' }));

      expect(updateChecklistItem).toHaveBeenCalledWith(
        'S1',
        { done: true },
        'plan-1'
      );
      expect(screen.getByText('Write the release notes').className).toMatch(
        /line-through/
      );
      expect(alsoAsk()).toBeNull();
      // Done never changes the day tick (R28).
      expect(dayTick('Write the release notes')).toBeChecked();
      await waitFor(() => expect(heard).toHaveBeenCalled());
      off();
    });

    it('should revert Yes when the update fails', async () => {
      vi.spyOn(console, 'error').mockImplementation(() => {});
      updateChecklistItem.mockRejectedValue(new Error('down'));
      mount();
      await screen.findByText('Write the release notes');

      fireEvent.click(dayTick('Write the release notes'));
      await userEvent.click(await screen.findByRole('button', { name: 'Yes' }));

      await waitFor(() =>
        expect(
          screen.getByText('Write the release notes').className
        ).not.toMatch(/line-through/)
      );
      expect(dayTick('Write the release notes')).toBeChecked();
    });

    it('should not ask when the item is already done, and the done item stays tickable', async () => {
      mount();
      await screen.findByText('Fix the login bug');

      fireEvent.click(dayTick('Fix the login bug'));

      await waitFor(() => expect(tickDayPlanEntry).toHaveBeenCalledWith('e-done', undefined));
      expect(dayTick('Fix the login bug')).toBeChecked();
      expect(alsoAsk()).toBeNull();
    });

    it('should drop the ask on untick and on any other action', async () => {
      mount();
      await screen.findByText('Write the release notes');

      fireEvent.click(dayTick('Write the release notes'));
      expect(await screen.findByText(/^Also mark/)).toBeTruthy();
      fireEvent.click(dayTick('Write the release notes'));
      expect(alsoAsk()).toBeNull();
      await waitFor(() =>
        expect(dayTick('Write the release notes')).not.toBeDisabled()
      );

      fireEvent.click(dayTick('Write the release notes'));
      expect(await screen.findByText(/^Also mark/)).toBeTruthy();
      fireEvent.click(screen.getByRole('button', { name: 'Dismiss Fix the login bug' }));
      expect(alsoAsk()).toBeNull();
    });

    it('should offer no day tick on a new step until it is added, then tick it with the created id', async () => {
      createChecklistItem.mockResolvedValue(
        item('S9', { parentId: 'M1', description: 'Book a demo slot' })
      );
      mount();
      await screen.findByText('Book a demo slot');
      expect(
        screen.queryByRole('checkbox', { name: 'Done for today: Book a demo slot' })
      ).toBeNull();

      await userEvent.click(screen.getByRole('button', { name: 'Add Book a demo slot' }));
      fireEvent.click(dayTick('Book a demo slot'));

      expect(tickDayPlanEntry).toHaveBeenCalledWith('e-new', 'S9');
      expect(await screen.findByText("Also mark 'Book a demo slot' done?")).toBeTruthy();
      await waitFor(() => expect(dayTick('Book a demo slot')).toBeChecked());
    });

    it('should recognise an added step after a reload and tick it with that child', async () => {
      fetchChecklist.mockResolvedValue([
        ...ITEMS,
        item('S7', { parentId: 'M1', description: '  book a DEMO   slot' }),
      ]);
      mount();
      await screen.findByText('Book a demo slot');

      expect(screen.queryByRole('button', { name: 'Add Book a demo slot' })).toBeNull();
      fireEvent.click(dayTick('Book a demo slot'));

      expect(tickDayPlanEntry).toHaveBeenCalledWith('e-new', 'S7');
    });

    it('should keep ticked entries ticked through Plan again', async () => {
      planMyDay.mockResolvedValue(
        planned(
          [{ label: 'Again', entries: [{ ...STEP, ticked: true }, DONE] }],
          { generatedAt: '2026-10-25T09:00:00Z', refreshesRemaining: 2 }
        )
      );
      mount();
      await screen.findByText('Write the release notes');
      fireEvent.click(dayTick('Write the release notes'));
      await waitFor(() => expect(tickDayPlanEntry).toHaveBeenCalled());

      await userEvent.click(planAgain());

      await screen.findByText('Again');
      expect(dayTick('Write the release notes')).toBeChecked();
      expect(alsoAsk()).toBeNull();
    });

    it('should tick a basic plan exactly like an AI plan', async () => {
      getDayPlan.mockResolvedValue({ ...PLAN, source: 'rules' });
      mount();
      await screen.findByText('Write the release notes');

      fireEvent.click(dayTick('Write the release notes'));

      expect(dayTick('Write the release notes')).toBeChecked();
      expect(tickDayPlanEntry).toHaveBeenCalledWith('e-step', undefined);
    });
  });

  describe('revisions (R66, R80, R81)', () => {
    // A learning plan: existing items carry the switch, one entry is stamped.
    const REVISION = { ...STEP, revision: 2, revisionsEnabled: true };
    const FRESH = { ...DONE, revisionsEnabled: true };
    const LEARNING = planned([
      { label: 'Review', entries: [REVISION, FRESH, NEW] },
    ]);
    const dontRevise = (text: string) =>
      screen.getByRole('switch', { name: `Don't revise: ${text}` });
    const revisionLabels = () => screen.queryAllByText(/^Revision \d+$/);

    beforeEach(() => {
      getDayPlan.mockResolvedValue(LEARNING);
      setItemRevisions.mockReset().mockResolvedValue(undefined);
    });

    it('should label a stamped entry "Revision N" and no other', async () => {
      mount();
      await screen.findByText('Write the release notes');

      const labels = revisionLabels();
      expect(labels.map((l) => l.textContent)).toEqual(['Revision 2']);
      expect(
        within(screen.getAllByRole('listitem')[0]).getByText('Revision 2')
      ).toBeInTheDocument();
    });

    it('should offer the switch only on learning entries for existing items', async () => {
      mount();
      await screen.findByText('Write the release notes');

      // On (revising) by default reads as "Don't revise" off.
      expect(dontRevise('Write the release notes')).not.toBeChecked();
      expect(dontRevise('Fix the login bug')).not.toBeChecked();
      expect(
        screen.queryByRole('switch', { name: "Don't revise: Book a demo slot" })
      ).toBeNull();
    });

    it('should offer no switch and no label on a project plan', async () => {
      getDayPlan.mockResolvedValue(PLAN);
      mount();
      await screen.findByText('Write the release notes');

      expect(screen.queryAllByRole('switch')).toHaveLength(0);
      expect(revisionLabels()).toHaveLength(0);
    });

    it('should reflect revisions already switched off, and hide the stale label', async () => {
      getDayPlan.mockResolvedValue(
        planned([
          {
            label: 'Review',
            entries: [{ ...REVISION, revisionsEnabled: false }],
          },
        ])
      );
      mount();
      await screen.findByText('Write the release notes');

      expect(dontRevise('Write the release notes')).toBeChecked();
      expect(revisionLabels()).toHaveLength(0);
    });

    it('should switch revisions off at once, then call the server', async () => {
      let land!: () => void;
      setItemRevisions.mockImplementationOnce(
        () => new Promise<void>((r) => (land = r))
      );
      mount();
      await screen.findByText('Write the release notes');

      fireEvent.click(dontRevise('Write the release notes'));

      expect(dontRevise('Write the release notes')).toBeChecked();
      expect(revisionLabels()).toHaveLength(0);
      expect(setItemRevisions).toHaveBeenCalledWith('S1', 'plan-1', false);
      await act(async () => land());
      expect(dontRevise('Write the release notes')).toBeChecked();
      // Nothing about the day tick or the item moves.
      expect(tickDayPlanEntry).not.toHaveBeenCalled();
      expect(updateChecklistItem).not.toHaveBeenCalled();
    });

    it('should switch revisions back on', async () => {
      getDayPlan.mockResolvedValue(
        planned([{ label: 'Review', entries: [{ ...FRESH, revisionsEnabled: false }] }])
      );
      mount();
      await screen.findByText('Fix the login bug');

      fireEvent.click(dontRevise('Fix the login bug'));

      expect(dontRevise('Fix the login bug')).not.toBeChecked();
      expect(setItemRevisions).toHaveBeenCalledWith('S2', 'plan-1', true);
    });

    it('should revert the switch, and bring the label back, when the call fails', async () => {
      vi.spyOn(console, 'error').mockImplementation(() => {});
      setItemRevisions.mockRejectedValueOnce(new Error('down'));
      mount();
      await screen.findByText('Write the release notes');

      fireEvent.click(dontRevise('Write the release notes'));

      await waitFor(() =>
        expect(dontRevise('Write the release notes')).not.toBeChecked()
      );
      expect(revisionLabels().map((l) => l.textContent)).toEqual(['Revision 2']);
    });

    it('should send one switch request per entry at a time', async () => {
      setItemRevisions.mockImplementationOnce(() => new Promise<void>(() => {}));
      mount();
      await screen.findByText('Write the release notes');

      fireEvent.click(dontRevise('Write the release notes'));
      fireEvent.click(dontRevise('Write the release notes'));

      expect(setItemRevisions).toHaveBeenCalledTimes(1);
    });
  });
});
