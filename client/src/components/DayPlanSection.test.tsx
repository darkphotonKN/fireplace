import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import DayPlanSection from './DayPlanSection';
import { toDateOnly } from '@/lib/itemDates';
import { ApiError } from '@/api/client';
import { deferred, settle } from '@/test/async';

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
const setDayPlanSettings = vi.fn();

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
  setDayPlanSettings: (...a: unknown[]) => setDayPlanSettings(...a),
}));

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

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
  settings: { asked: true },
  ...extra,
});

const notPlanned = (extra: Record<string, unknown> = {}) => ({
  planId: 'plan-1',
  date: today(),
  status: 'not_planned',
  refreshesRemaining: 3,
  groups: [],
  // Asked already, so the card is the plain one; the first ask overrides it.
  settings: { asked: true },
  ...extra,
});

const PLAN = planned([
  { label: 'Release', entries: [STEP, NEW] },
  { label: 'Bugs', entries: [DONE] },
]);

/** PLAN, sized to `dailyMinutes` a day, or to a typical day when omitted. */
const withMinutes = (dailyMinutes?: number) =>
  planned(PLAN.groups, {
    settings: dailyMinutes ? { asked: true, dailyMinutes } : { asked: true },
  });

/** A failure the section must read by its code; `detail` is prose it ignores. */
const apiError = (code: string, status: number, detail?: string) =>
  new ApiError({ code, status, ...(detail ? { detail } : {}) } as never, status);
const unavailable = (detail?: string) => apiError('SERVICE_UNAVAILABLE', 503, detail);
const notFound = (detail?: string) => apiError('NOT_FOUND', 404, detail);
const limitReached = () =>
  apiError('BRIEF_REFRESH_LIMIT_REACHED', 429, 'prose that may change');

/** Add on the demo-slot new step creates S9 under M1. */
const createsDemoSlot = () =>
  createChecklistItem.mockResolvedValue(
    item('S9', { parentId: 'M1', description: 'Book a demo slot' })
  );

// ---------------------------------------------------------------------------
// Rendering and queries
// ---------------------------------------------------------------------------

const mount = () => render(<DayPlanSection planId="plan-1" />);

/** Mounts the section and waits until the day plan shows `ready`. */
const renderSection = async (ready = 'Write the release notes') => {
  const view = mount();
  await screen.findByText(ready);
  return view;
};

const section = () => screen.getByRole('region', { name: 'Day plan' });
const planAgain = () => screen.getByRole('button', { name: /^(Plan again|Planning)/ });
const tip = () => screen.getByRole('tooltip');
const cog = () => screen.queryByRole('button', { name: 'Day plan settings' });
const QUESTION = 'How much time do you have for this plan each day?';

/** The tab regains focus, which is when the section re-reads. */
const refocus = () =>
  act(async () => {
    window.dispatchEvent(new Event('focus'));
  });

/** Fakes only the date, so the clock reads `now` while timers stay real. */
const freezeDate = (now: Date) => {
  vi.useFakeTimers({ toFake: ['Date'] });
  vi.setSystemTime(now);
};

beforeEach(() => {
  window.localStorage.clear();
  fetchChecklist.mockReset().mockResolvedValue(ITEMS);
  getDayPlan.mockReset().mockResolvedValue(PLAN);
  planMyDay.mockReset();
  setDayPlanSettings.mockReset();
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
    await renderSection();
    expect(getDayPlan).toHaveBeenCalledWith('plan-1', today());
    expect(fetchChecklist).toHaveBeenCalledWith('plan-1', 'longterm');
    expect(planMyDay).not.toHaveBeenCalled();
  });

  it('should carry the "i" tooltip explaining daily progression', async () => {
    mount();
    expect(
      screen.getByRole('button', { name: 'Information about the day plan' })
    ).toBeTruthy();
    expect(tip()).toHaveTextContent(/daily progression/i);
    expect(tip()).toHaveTextContent(/plan your day/i);
    expect(tip()).toHaveTextContent(/tick what you did today/i);
    expect(tip()).toHaveTextContent(/revision/i);
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
      const draw = deferred<unknown>();
      planMyDay.mockReturnValue(draw.promise);
      mount();
      await userEvent.click(await screen.findByRole('button', { name: 'Plan my day' }));

      expect(
        within(section()).getByRole('list', { name: 'Planning your day' })
      ).toBeTruthy();

      await act(async () => draw.resolve(PLAN));
      expect(await screen.findByText('Write the release notes')).toBeTruthy();
    });

    it('should keep the button and offer a quiet retry when planning fails', async () => {
      planMyDay.mockRejectedValueOnce(unavailable()).mockResolvedValueOnce(PLAN);
      mount();
      await userEvent.click(await screen.findByRole('button', { name: 'Plan my day' }));

      expect(await screen.findByText(/Couldn.t plan your day/)).toBeTruthy();
      expect(screen.queryByRole('alert')).toBeNull();
      await userEvent.click(screen.getByRole('button', { name: 'Try again' }));
      expect(await screen.findByText('Write the release notes')).toBeTruthy();
    });
  });

  describe('the first ask, hours a day (R108–R112)', () => {
    const neverAsked = (extra: Record<string, unknown> = {}) =>
      notPlanned({ settings: { asked: false }, ...extra });
    const chips = () => within(screen.getByRole('group', { name: QUESTION }));

    /** Mounts on the first ask and answers it with the chip named `label`. */
    const answer = async (label: string) => {
      const view = mount();
      await screen.findByRole('group', { name: QUESTION });
      await userEvent.click(chips().getByRole('button', { name: label }));
      return view;
    };

    beforeEach(() => {
      getDayPlan.mockResolvedValue(neverAsked());
    });

    it('should ask with five chips and a quiet skip in place of the line and Plan my day', async () => {
      mount();
      const group = await screen.findByRole('group', { name: QUESTION });
      expect(
        within(group)
          .getAllByRole('button')
          .map((b) => b.textContent)
      ).toEqual(['30m', '1h', '2h', '3h', '4h']);
      // Named in words for a screen reader (R115), shown compact (R108).
      expect(
        within(group)
          .getAllByRole('button')
          .map((b) => b.getAttribute('aria-label'))
      ).toEqual(['30 minutes', '1 hour', '2 hours', '3 hours', '4 hours']);
      expect(screen.getByRole('button', { name: 'Skip, plan anyway' })).toBeTruthy();
      expect(screen.queryByRole('button', { name: 'Plan my day' })).toBeNull();
      expect(screen.queryByText(/haven.t planned today/i)).toBeNull();
      expect(setDayPlanSettings).not.toHaveBeenCalled();
      expect(planMyDay).not.toHaveBeenCalled();
    });

    it('should save the chip then plan, once each and in that order, in one click', async () => {
      setDayPlanSettings.mockResolvedValue({ asked: true, dailyMinutes: 60 });
      planMyDay.mockResolvedValue(withMinutes(60));
      await answer('1 hour');

      expect(setDayPlanSettings).toHaveBeenCalledTimes(1);
      expect(setDayPlanSettings).toHaveBeenCalledWith('plan-1', 60);
      expect(planMyDay).toHaveBeenCalledTimes(1);
      expect(planMyDay).toHaveBeenCalledWith('plan-1', today());
      expect(setDayPlanSettings.mock.invocationCallOrder[0]).toBeLessThan(
        planMyDay.mock.invocationCallOrder[0]
      );
      expect(await screen.findByText('Write the release notes')).toBeTruthy();
      expect(screen.queryByText(QUESTION)).toBeNull();
    });

    it('should show the planning skeleton from the save through the draw', async () => {
      const save = deferred<unknown>();
      setDayPlanSettings.mockReturnValue(save.promise);
      const draw = deferred<unknown>();
      planMyDay.mockReturnValue(draw.promise);
      await answer('2 hours');

      expect(setDayPlanSettings).toHaveBeenCalledWith('plan-1', 120);
      expect(
        within(section()).getByRole('list', { name: 'Planning your day' })
      ).toBeTruthy();

      await act(async () => save.resolve({ asked: true, dailyMinutes: 120 }));
      expect(
        within(section()).getByRole('list', { name: 'Planning your day' })
      ).toBeTruthy();
      expect(planMyDay).toHaveBeenCalledTimes(1);

      await act(async () => draw.resolve(PLAN));
      expect(await screen.findByText('Write the release notes')).toBeTruthy();
    });

    it('should save no budget on Skip, then plan', async () => {
      setDayPlanSettings.mockResolvedValue({ asked: true });
      planMyDay.mockResolvedValue(PLAN);
      mount();
      await userEvent.click(
        await screen.findByRole('button', { name: 'Skip, plan anyway' })
      );

      expect(setDayPlanSettings).toHaveBeenCalledTimes(1);
      expect(setDayPlanSettings).toHaveBeenCalledWith('plan-1', null);
      expect(planMyDay).toHaveBeenCalledTimes(1);
      expect(setDayPlanSettings.mock.invocationCallOrder[0]).toBeLessThan(
        planMyDay.mock.invocationCallOrder[0]
      );
      expect(await screen.findByText('Write the release notes')).toBeTruthy();
    });

    it('should not ask once asked, so a later not-planned read shows the plain card', async () => {
      freezeDate(new Date(2026, 9, 25, 23, 0));
      setDayPlanSettings.mockResolvedValue({ asked: true, dailyMinutes: 120 });
      planMyDay.mockResolvedValue(PLAN);
      await answer('2 hours');
      await screen.findByText('Write the release notes');

      // The next day's read: the server recorded the answer (R94).
      getDayPlan.mockResolvedValue(
        notPlanned({
          date: '2026-10-26',
          settings: { asked: true, dailyMinutes: 120 },
        })
      );
      vi.setSystemTime(new Date(2026, 9, 26, 8, 0));
      await refocus();

      expect(await screen.findByRole('button', { name: 'Plan my day' })).toBeTruthy();
      expect(screen.queryByText(QUESTION)).toBeNull();
    });

    it('should not ask when Plan my day is retired (R112)', async () => {
      getDayPlan.mockResolvedValue(neverAsked({ refreshesRemaining: 0 }));
      mount();
      const button = await screen.findByRole('button', { name: 'Plan my day' });
      expect(button).toBeDisabled();
      expect(screen.queryByText(QUESTION)).toBeNull();
      expect(screen.queryByRole('button', { name: 'Skip, plan anyway' })).toBeNull();
    });

    it('should still plan when the save fails, say so quietly, and match the code not the detail (R110)', async () => {
      setDayPlanSettings.mockRejectedValue(unavailable('not found, prose that may change'));
      planMyDay.mockResolvedValue(PLAN);
      await answer('3 hours');

      expect(planMyDay).toHaveBeenCalledTimes(1);
      expect(await screen.findByText('Write the release notes')).toBeTruthy();
      expect(
        screen.getByText(
          "Couldn't save your hours, so this plan uses a typical day."
        )
      ).toBeTruthy();
      expect(screen.queryByRole('alert')).toBeNull();
    });

    it('should drop the hours notice on the next Plan again', async () => {
      setDayPlanSettings.mockRejectedValue(unavailable());
      planMyDay
        .mockResolvedValueOnce(PLAN)
        .mockResolvedValueOnce(planned(PLAN.groups, { generatedAt: '2026-10-25T09:00:00Z' }));
      await answer('1 hour');
      await screen.findByText(/Couldn.t save your hours/);

      await userEvent.click(planAgain());

      await waitFor(() => expect(planMyDay).toHaveBeenCalledTimes(2));
      expect(setDayPlanSettings).toHaveBeenCalledTimes(1);
      await waitFor(() =>
        expect(screen.queryByText(/Couldn.t save your hours/)).toBeNull()
      );
    });

    it('should ask again when both the save and the draw fail, and send the chosen answer again on retry (R110)', async () => {
      setDayPlanSettings
        .mockRejectedValueOnce(unavailable())
        .mockResolvedValueOnce({ asked: true, dailyMinutes: 60 });
      planMyDay.mockRejectedValueOnce(unavailable()).mockResolvedValueOnce(withMinutes(60));
      await answer('1 hour');
      await screen.findByText(/Couldn.t plan your day/);
      expect(screen.getByRole('group', { name: QUESTION })).toBeTruthy();

      await userEvent.click(screen.getByRole('button', { name: 'Try again' }));

      expect(await screen.findByText('Write the release notes')).toBeTruthy();
      expect(setDayPlanSettings.mock.calls).toEqual([
        ['plan-1', 60],
        ['plan-1', 60],
      ]);
      expect(planMyDay).toHaveBeenCalledTimes(2);
      expect(setDayPlanSettings.mock.invocationCallOrder[1]).toBeLessThan(
        planMyDay.mock.invocationCallOrder[1]
      );
      expect(screen.queryByText(/Couldn.t save your hours/)).toBeNull();
    });

    it('should render nothing when the save says the plan is not reachable (R70)', async () => {
      setDayPlanSettings.mockRejectedValue(notFound());
      const { container } = await answer('1 hour');

      await waitFor(() => expect(container).toBeEmptyDOMElement());
      expect(planMyDay).not.toHaveBeenCalled();
    });

    it('should show the quiet retry and not ask again when the save works but the draw fails (R111)', async () => {
      setDayPlanSettings.mockResolvedValue({ asked: true, dailyMinutes: 60 });
      planMyDay.mockRejectedValueOnce(unavailable()).mockResolvedValueOnce(PLAN);
      await answer('1 hour');

      expect(await screen.findByText(/Couldn.t plan your day/)).toBeTruthy();
      expect(screen.queryByText(QUESTION)).toBeNull();
      expect(screen.getByRole('button', { name: 'Plan my day' })).toBeTruthy();

      await userEvent.click(screen.getByRole('button', { name: 'Try again' }));
      expect(await screen.findByText('Write the release notes')).toBeTruthy();
      // The retry plans; it doesn't save the answer again.
      expect(setDayPlanSettings).toHaveBeenCalledTimes(1);
      expect(planMyDay).toHaveBeenCalledTimes(2);
    });
  });

  describe('the tooltip and the cog, hours a day (R114–R118)', () => {
    const UNSET_LINE = 'Sized for a typical day. Set your hours with the cog.';

    const openSettings = async () => {
      await userEvent.click(
        await screen.findByRole('button', { name: 'Day plan settings' })
      );
    };
    /** Mounts on PLAN sized to `dailyMinutes` (unset when omitted) and opens the cog. */
    const openCogOn = async (dailyMinutes?: number) => {
      getDayPlan.mockResolvedValue(withMinutes(dailyMinutes));
      const view = mount();
      await openSettings();
      return view;
    };
    const more = () => screen.getByRole('button', { name: 'More time' });
    const less = () => screen.getByRole('button', { name: 'Less time' });
    const readout = () => screen.getByRole('status', { name: 'Hours a day' });

    /** Every save lands at once, echoing what it was sent. */
    const echo = () =>
      setDayPlanSettings.mockImplementation(
        async (_: string, dailyMinutes: number | null) =>
          dailyMinutes === null ? { asked: true } : { asked: true, dailyMinutes }
      );
    /** Saves held open until the test settles each one, in order. */
    const heldSaves = () => {
      const pending: { ok: () => void; fail: (err: unknown) => void }[] = [];
      setDayPlanSettings.mockImplementation(
        (_: string, dailyMinutes: number) =>
          new Promise((resolve, reject) =>
            pending.push({
              ok: () => resolve({ asked: true, dailyMinutes }),
              fail: reject,
            })
          )
      );
      return pending;
    };

    it.each<[string, number | undefined, string]>([
      ['name the hours when set', 150, 'Sized for 2½ hours a day. Change it with the cog.'],
      ['say a typical day when unset', undefined, UNSET_LINE],
    ])('should %s in the tooltip', async (_when, dailyMinutes, line) => {
      getDayPlan.mockResolvedValue(withMinutes(dailyMinutes));
      await renderSection();
      expect(tip()).toHaveTextContent(line);
    });

    it('should show the cog beside Hide only once the day plan has loaded, and swap the body for the settings view', async () => {
      const read = deferred<unknown>();
      getDayPlan.mockReturnValue(read.promise);
      mount();
      expect(cog()).toBeNull();

      await act(async () => read.resolve(PLAN));
      await screen.findByText('Write the release notes');
      // In the header's control cluster, next to Hide.
      expect(cog()!.parentElement).toBe(
        screen.getByRole('button', { name: 'Hide' }).parentElement
      );

      await userEvent.click(cog()!);
      expect(screen.queryByText('Write the release notes')).toBeNull();
      expect(screen.getByText('Hours a day')).toBeTruthy();
      expect(screen.getByText('Typical day')).toBeTruthy();
      expect(
        screen.getByText('Applies the next time you plan your day.')
      ).toBeTruthy();
      expect(screen.queryByRole('button', { name: 'Use the default' })).toBeNull();

      await userEvent.click(screen.getByRole('button', { name: 'Back to day plan' }));
      expect(screen.getByText('Write the release notes')).toBeTruthy();
      expect(screen.queryByText('Hours a day')).toBeNull();
    });

    it.each([
      ['+', more],
      ['−', less],
    ])('should set 2 hours on a first %s from unset, save 120, and update the tooltip without planning', async (_key, press) => {
      echo();
      await openCogOn();
      expect(readout()).toHaveTextContent('Typical day');

      await userEvent.click(press());

      expect(readout()).toHaveTextContent('2 hours');
      expect(setDayPlanSettings).toHaveBeenCalledTimes(1);
      expect(setDayPlanSettings).toHaveBeenCalledWith('plan-1', 120);
      await waitFor(() =>
        expect(tip()).toHaveTextContent(
          'Sized for 2 hours a day. Change it with the cog.'
        )
      );
      expect(planMyDay).not.toHaveBeenCalled();
    });

    it.each([
      { bound: '30 minutes', to: 30, from: 60, fromLabel: '1 hour', press: less, back: more },
      { bound: '12 hours', to: 720, from: 690, fromLabel: '11½ hours', press: more, back: less },
    ])(
      'should step by 30 minutes and stop at $bound',
      async ({ bound, to, from, fromLabel, press, back }) => {
        echo();
        await openCogOn(from);
        expect(readout()).toHaveTextContent(fromLabel);

        await userEvent.click(press());
        expect(readout()).toHaveTextContent(bound);
        expect(setDayPlanSettings).toHaveBeenLastCalledWith('plan-1', to);
        expect(press()).toBeDisabled();
        expect(back()).toBeEnabled();

        await userEvent.click(back());
        expect(readout()).toHaveTextContent(fromLabel);
        expect(setDayPlanSettings).toHaveBeenLastCalledWith('plan-1', from);
      }
    );

    it('should clear to the default with "Use the default", saving null', async () => {
      echo();
      await openCogOn(150);
      expect(readout()).toHaveTextContent('2½ hours');

      await userEvent.click(screen.getByRole('button', { name: 'Use the default' }));

      expect(setDayPlanSettings).toHaveBeenCalledWith('plan-1', null);
      expect(readout()).toHaveTextContent('Typical day');
      await waitFor(() =>
        expect(screen.queryByRole('button', { name: 'Use the default' })).toBeNull()
      );
      expect(tip()).toHaveTextContent(UNSET_LINE);
    });

    it('should send fast presses one after another, the last value pressed winning', async () => {
      const pending = heldSaves();
      await openCogOn(120);

      await userEvent.click(more());
      await userEvent.click(more());
      await userEvent.click(more());
      expect(readout()).toHaveTextContent('3½ hours');
      // One request at a time: the next waits for the one before.
      expect(setDayPlanSettings).toHaveBeenCalledTimes(1);

      for (let i = 0; i < 3; i++) {
        await waitFor(() => expect(pending.length).toBe(i + 1));
        await act(async () => pending[i].ok());
      }

      expect(setDayPlanSettings.mock.calls.map((c) => c[1])).toEqual([
        150, 180, 210,
      ]);
      expect(readout()).toHaveTextContent('3½ hours');
      expect(tip()).toHaveTextContent('Sized for 3½ hours a day.');
    });

    it('should revert to the last saved value and say so quietly when a save fails, by code', async () => {
      setDayPlanSettings.mockRejectedValue(unavailable('not found, prose that may change'));
      await openCogOn(120);
      await userEvent.click(more());

      expect(await screen.findByText("Couldn't save. Try again.")).toBeTruthy();
      expect(readout()).toHaveTextContent('2 hours');
      expect(screen.queryByRole('alert')).toBeNull();
      expect(tip()).toHaveTextContent('Sized for 2 hours a day.');

      // The next press clears it.
      echo();
      await userEvent.click(more());
      await waitFor(() =>
        expect(screen.queryByText("Couldn't save. Try again.")).toBeNull()
      );
      expect(readout()).toHaveTextContent('2½ hours');
    });

    it('should settle on the last press when an earlier save in the queue fails', async () => {
      const pending = heldSaves();
      await openCogOn(120);
      await userEvent.click(more());
      await userEvent.click(more());

      await act(async () => pending[0].fail(unavailable()));
      // A later press is queued, so the failure isn't the last word.
      expect(screen.queryByText("Couldn't save. Try again.")).toBeNull();
      expect(readout()).toHaveTextContent('3 hours');

      await waitFor(() => expect(pending.length).toBe(2));
      await act(async () => pending[1].ok());

      expect(setDayPlanSettings.mock.calls.map((c) => c[1])).toEqual([150, 180]);
      expect(readout()).toHaveTextContent('3 hours');
      expect(screen.queryByText("Couldn't save. Try again.")).toBeNull();
      expect(tip()).toHaveTextContent('Sized for 3 hours a day.');
    });

    it('should revert to the last value saved in the queue when the last press fails', async () => {
      const pending = heldSaves();
      await openCogOn(120);
      await userEvent.click(more());
      await userEvent.click(more());

      await act(async () => pending[0].ok());
      await waitFor(() => expect(pending.length).toBe(2));
      await act(async () => pending[1].fail(unavailable()));

      expect(await screen.findByText("Couldn't save. Try again.")).toBeTruthy();
      expect(readout()).toHaveTextContent('2½ hours');
      expect(tip()).toHaveTextContent('Sized for 2½ hours a day.');
    });

    it('should render nothing and drop the presses still queued once a save says the plan is not reachable (R70)', async () => {
      const pending = heldSaves();
      const { container } = await openCogOn(120);
      await userEvent.click(more());
      await userEvent.click(more());
      await userEvent.click(more());

      await act(async () => pending[0].fail(notFound()));
      await waitFor(() => expect(container).toBeEmptyDOMElement());
      // Let anything still queued run, were it going to.
      await settle(0);
      expect(setDayPlanSettings).toHaveBeenCalledTimes(1);
    });

    it('should keep a cog save made while Plan again is drawing over the settings the draw brings back (R114)', async () => {
      echo();
      getDayPlan.mockResolvedValue(withMinutes(120));
      const draw = deferred<unknown>();
      planMyDay.mockReturnValue(draw.promise);
      await renderSection();

      await userEvent.click(planAgain());
      await openSettings();
      await userEvent.click(more());
      await waitFor(() =>
        expect(tip()).toHaveTextContent('Sized for 2½ hours a day.')
      );

      // The draw read the settings before the save landed.
      await act(async () =>
        draw.resolve(
          planned(PLAN.groups, {
            generatedAt: '2026-10-25T09:00:00Z',
            settings: { asked: true, dailyMinutes: 120 },
          })
        )
      );

      expect(tip()).toHaveTextContent('Sized for 2½ hours a day.');
      expect(readout()).toHaveTextContent('2½ hours');
    });

    it('should count a cog save as asked, so the not-planned card asks nothing after (R118)', async () => {
      echo();
      getDayPlan.mockResolvedValue(notPlanned({ settings: { asked: false } }));
      mount();
      await screen.findByRole('group', { name: QUESTION });
      await openSettings();
      await userEvent.click(more());
      await waitFor(() => expect(setDayPlanSettings).toHaveBeenCalledTimes(1));
      await userEvent.click(screen.getByRole('button', { name: 'Back to day plan' }));

      expect(await screen.findByRole('button', { name: 'Plan my day' })).toBeTruthy();
      expect(screen.queryByText(QUESTION)).toBeNull();
      expect(planMyDay).not.toHaveBeenCalled();
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
      // An AI plan wears no basic label (R69).
      expect(within(section()).queryByText('Basic plan')).toBeNull();
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
      await renderSection();
      for (const gone of ['Old idea', 'Deleted thing', 'Orphan step', 'Under a done parent']) {
        expect(screen.queryByText(gone)).toBeNull();
      }
      expect(screen.queryByText('All stale')).toBeNull();
      expect(within(section()).getAllByRole('listitem')).toHaveLength(1);
    });

    it.each([
      { when: 'and invite a first milestone when there is none', items: [], line: /Start a milestone in your plan/ },
      { when: 'when milestones exist but the plan is empty', items: ITEMS, line: 'Nothing to plan for today.' },
    ])('should say there is nothing to plan $when', async ({ items, line }) => {
      getDayPlan.mockResolvedValue(planned([]));
      fetchChecklist.mockResolvedValue(items);
      mount();
      expect(await screen.findByText(line)).toBeTruthy();
    });

    it('should re-read the items on return so an item finished elsewhere renders done', async () => {
      mount();
      const text = await screen.findByText('Write the release notes');
      expect(text.className).not.toMatch(/line-through/);

      fetchChecklist.mockResolvedValue([ITEMS[0], { ...ITEMS[1], done: true }, ITEMS[2]]);
      await refocus();
      await waitFor(() =>
        expect(screen.getByText('Write the release notes').className).toMatch(/line-through/)
      );
      expect(getDayPlan).toHaveBeenCalledTimes(1);
    });

    it('should read the new date after local midnight and show not planned (R64)', async () => {
      freezeDate(new Date(2026, 9, 25, 23, 59));
      await renderSection();
      expect(getDayPlan).toHaveBeenLastCalledWith('plan-1', '2026-10-25');

      getDayPlan.mockResolvedValue(notPlanned({ date: '2026-10-26' }));
      vi.setSystemTime(new Date(2026, 9, 26, 0, 1));
      await refocus();
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
      await renderSection();
      expect(planAgain()).toHaveTextContent('3 left');

      await userEvent.click(planAgain());

      expect(planMyDay).toHaveBeenCalledWith('plan-1', today());
      expect(await screen.findByText(/Nothing has changed since this plan was made/)).toBeTruthy();
      expect(screen.getByText('Write the release notes')).toBeTruthy();
    });

    it('should retire the action at zero without asking the server', async () => {
      getDayPlan.mockResolvedValue({ ...PLAN, refreshesRemaining: 0 });
      await renderSection();
      expect(planAgain()).toHaveTextContent('0 left');
      expect(planAgain()).toBeDisabled();
      await userEvent.click(planAgain());
      expect(planMyDay).not.toHaveBeenCalled();
    });

    it('should retire the action and keep the plan on a 429 by code', async () => {
      getDayPlan.mockResolvedValue({ ...PLAN, refreshesRemaining: 1 });
      planMyDay.mockRejectedValue(limitReached());
      await renderSection();

      await userEvent.click(planAgain());

      await waitFor(() => expect(planAgain()).toBeDisabled());
      expect(planAgain()).toHaveTextContent('0 left');
      expect(screen.getByText('Write the release notes')).toBeTruthy();
      expect(screen.queryByRole('button', { name: 'Try again' })).toBeNull();
    });

    it('should keep the plan and offer a quiet retry for any other failure, whatever its detail says', async () => {
      planMyDay
        .mockRejectedValueOnce(unavailable('refresh limit reached'))
        .mockResolvedValueOnce(PLAN);
      await renderSection();

      await userEvent.click(planAgain());

      const retry = await screen.findByRole('button', { name: 'Try again' });
      expect(planAgain()).not.toBeDisabled();
      expect(screen.queryByRole('alert')).toBeNull();
      await userEvent.click(retry);
      expect(planMyDay).toHaveBeenCalledTimes(2);
      expect(await screen.findByText(/Nothing has changed/)).toBeTruthy();
    });

    it('should replace the plan after a counted redraw and forget what was done to the old entries', async () => {
      createsDemoSlot();
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
      freezeDate(new Date(2026, 9, 25, 23, 59));
      await renderSection();

      vi.setSystemTime(new Date(2026, 9, 26, 0, 1));
      await userEvent.click(planAgain());

      await waitFor(() =>
        expect(getDayPlan).toHaveBeenLastCalledWith('plan-1', '2026-10-26')
      );
      expect(planMyDay).not.toHaveBeenCalled();
    });
  });

  it('should label a rules plan as basic, explain it, and still offer Plan my day (R69)', async () => {
    getDayPlan.mockResolvedValue({ ...PLAN, source: 'rules' });
    planMyDay.mockResolvedValue({ ...PLAN, generatedAt: '2026-10-25T09:00:00Z' });
    await renderSection();

    expect(within(section()).getByText('Basic plan')).toBeTruthy();
    expect(within(section()).getByText(/AI wasn.t available/)).toBeTruthy();
    const button = screen.getByRole('button', { name: /^Plan my day/ });
    await userEvent.click(button);
    expect(planMyDay).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(within(section()).queryByText('Basic plan')).toBeNull());
  });

  it('should render nothing when the plan is not reachable (R70)', async () => {
    getDayPlan.mockRejectedValue(notFound());
    const { container } = mount();
    await waitFor(() => expect(container).toBeEmptyDOMElement());
  });

  it('should offer a quiet retry when the read fails, never an alert, and no cog until it succeeds (R71, R116)', async () => {
    getDayPlan.mockRejectedValueOnce(apiError('BRIEF_DATE_OUT_OF_RANGE', 400));
    mount();
    const retry = await screen.findByRole('button', { name: 'Try again' });
    expect(screen.getByText(/Couldn.t load the day plan/)).toBeTruthy();
    expect(screen.queryByRole('alert')).toBeNull();
    expect(cog()).toBeNull();

    await userEvent.click(retry);
    expect(await screen.findByText('Write the release notes')).toBeTruthy();
    expect(cog()).toBeTruthy();
  });

  describe('show / hide (R72)', () => {
    it('should hide and remember the preference under the existing key', async () => {
      await renderSection();
      await userEvent.click(screen.getByRole('button', { name: 'Hide' }));
      expect(screen.queryByText('Write the release notes')).toBeNull();
      expect(window.localStorage.getItem('showDailyInsights')).toBe('false');
    });

    it('should start hidden when the preference says so, offering no cog (R116)', async () => {
      window.localStorage.setItem('showDailyInsights', 'false');
      mount();
      expect(await screen.findByRole('button', { name: 'Show' })).toBeTruthy();
      expect(screen.queryByText('Write the release notes')).toBeNull();
      // Still no cog once the hidden section has read the day plan.
      await waitFor(() => expect(getDayPlan).toHaveBeenCalled());
      expect(cog()).toBeNull();
      fireEvent.click(screen.getByRole('button', { name: 'Show' }));
      expect(await screen.findByText('Write the release notes')).toBeTruthy();
    });
  });

  describe('Add (R73, R84)', () => {
    it('should add a new step as a long-term child of its parent, never a daily item', async () => {
      createsDemoSlot();
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
      apiError('DAY_PLAN_ENTRY_NOT_TICKABLE', 400, 'prose that may change');

    it.each([
      { kind: 'an AI', source: 'ai' },
      { kind: 'a basic', source: 'rules' },
    ])('should tick $kind plan at once, call the day tick and never touch the item', async ({ source }) => {
      const land = deferred<unknown>();
      tickDayPlanEntry.mockImplementationOnce(() => land.promise);
      getDayPlan.mockResolvedValue({ ...PLAN, source });
      await renderSection();

      fireEvent.click(dayTick('Write the release notes'));

      expect(dayTick('Write the release notes')).toBeChecked();
      expect(tickDayPlanEntry).toHaveBeenCalledWith('e-step', undefined);
      await act(async () => land.resolve({ ...STEP, ticked: true }));
      expect(dayTick('Write the release notes')).toBeChecked();
      expect(updateChecklistItem).not.toHaveBeenCalled();
    });

    it('should revert a tick that fails', async () => {
      vi.spyOn(console, 'error').mockImplementation(() => {});
      tickDayPlanEntry.mockRejectedValueOnce(new Error('down'));
      await renderSection();

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
      await renderSection();
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
      await renderSection();
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
      tickDayPlanEntry.mockRejectedValueOnce(notFound('prose that may change'));
      await renderSection();
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
      const draw = deferred<unknown>();
      planMyDay.mockReturnValue(draw.promise);
      getDayPlan.mockResolvedValue(
        planned([
          { label: 'Release', entries: [STEP, { ...DONE, ticked: true }] },
        ])
      );
      await renderSection();

      await userEvent.click(planAgain());

      expect(planAgain()).toBeDisabled();
      expect(dayTick('Write the release notes')).toBeDisabled();
      expect(dayTick('Fix the login bug')).toBeDisabled();
      fireEvent.click(dayTick('Write the release notes'));
      fireEvent.click(dayTick('Fix the login bug'));
      expect(tickDayPlanEntry).not.toHaveBeenCalled();
      expect(untickDayPlanEntry).not.toHaveBeenCalled();

      await act(async () => draw.resolve(PLAN));
      await waitFor(() =>
        expect(dayTick('Write the release notes')).not.toBeDisabled()
      );
    });

    it('should ask inline to mark the item done after a tick, and drop the ask after 8 seconds', async () => {
      await renderSection();
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
      await renderSection();

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
      await renderSection();

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
      await renderSection('Fix the login bug');

      fireEvent.click(dayTick('Fix the login bug'));

      await waitFor(() => expect(tickDayPlanEntry).toHaveBeenCalledWith('e-done', undefined));
      expect(dayTick('Fix the login bug')).toBeChecked();
      expect(alsoAsk()).toBeNull();
    });

    it('should drop the ask on untick and on any other action', async () => {
      await renderSection();

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
      createsDemoSlot();
      await renderSection('Book a demo slot');
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
      await renderSection('Book a demo slot');

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
      await renderSection();
      fireEvent.click(dayTick('Write the release notes'));
      await waitFor(() => expect(tickDayPlanEntry).toHaveBeenCalled());

      await userEvent.click(planAgain());

      await screen.findByText('Again');
      expect(dayTick('Write the release notes')).toBeChecked();
      expect(alsoAsk()).toBeNull();
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
      await renderSection();

      const labels = revisionLabels();
      expect(labels.map((l) => l.textContent)).toEqual(['Revision 2']);
      expect(
        within(screen.getAllByRole('listitem')[0]).getByText('Revision 2')
      ).toBeInTheDocument();
    });

    it('should offer the switch only on learning entries for existing items', async () => {
      await renderSection();

      // On (revising) by default reads as "Don't revise" off.
      expect(dontRevise('Write the release notes')).not.toBeChecked();
      expect(dontRevise('Fix the login bug')).not.toBeChecked();
      expect(
        screen.queryByRole('switch', { name: "Don't revise: Book a demo slot" })
      ).toBeNull();
    });

    it('should offer no switch and no label on a project plan', async () => {
      getDayPlan.mockResolvedValue(PLAN);
      await renderSection();

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
      await renderSection();

      expect(dontRevise('Write the release notes')).toBeChecked();
      expect(revisionLabels()).toHaveLength(0);
    });

    it('should switch revisions off at once, then call the server', async () => {
      const land = deferred<void>();
      setItemRevisions.mockImplementationOnce(() => land.promise);
      await renderSection();

      fireEvent.click(dontRevise('Write the release notes'));

      expect(dontRevise('Write the release notes')).toBeChecked();
      expect(revisionLabels()).toHaveLength(0);
      expect(setItemRevisions).toHaveBeenCalledWith('S1', 'plan-1', false);
      await act(async () => land.resolve());
      expect(dontRevise('Write the release notes')).toBeChecked();
      // Nothing about the day tick or the item moves.
      expect(tickDayPlanEntry).not.toHaveBeenCalled();
      expect(updateChecklistItem).not.toHaveBeenCalled();
    });

    it('should switch revisions back on', async () => {
      getDayPlan.mockResolvedValue(
        planned([{ label: 'Review', entries: [{ ...FRESH, revisionsEnabled: false }] }])
      );
      await renderSection('Fix the login bug');

      fireEvent.click(dontRevise('Fix the login bug'));

      expect(dontRevise('Fix the login bug')).not.toBeChecked();
      expect(setItemRevisions).toHaveBeenCalledWith('S2', 'plan-1', true);
    });

    it('should revert the switch, and bring the label back, when the call fails', async () => {
      vi.spyOn(console, 'error').mockImplementation(() => {});
      setItemRevisions.mockRejectedValueOnce(new Error('down'));
      await renderSection();

      fireEvent.click(dontRevise('Write the release notes'));

      await waitFor(() =>
        expect(dontRevise('Write the release notes')).not.toBeChecked()
      );
      expect(revisionLabels().map((l) => l.textContent)).toEqual(['Revision 2']);
    });

    it('should send one switch request per entry at a time', async () => {
      setItemRevisions.mockImplementationOnce(() => new Promise<void>(() => {}));
      await renderSection();

      fireEvent.click(dontRevise('Write the release notes'));
      fireEvent.click(dontRevise('Write the release notes'));

      expect(setItemRevisions).toHaveBeenCalledTimes(1);
    });
  });
});
