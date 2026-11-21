import { describe, it, expect, vi, beforeEach } from 'vitest';
import { act, fireEvent, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ApiError } from '@/api/client';
import type { ChecklistItem } from '@/api/checklists';
import { api, item as mk, quietErrors, router } from '@/test/pageHarness';
import { deferred } from '@/test/async';

/**
 * The milestone page (FS-043WP): one milestone read out of the plan's
 * long-term list, the same list the grid reads, so the page and the card can
 * never disagree about it (R10). Pinned here: what the page shows for a
 * milestone, and the calm states for everything that is not one.
 */

const duplicateChecklistItem = vi.fn();
vi.mock('@/api/checklists', () => ({
  duplicateChecklistItem: (...a: unknown[]) => duplicateChecklistItem(...a),
}));

import MilestonePage from './MilestonePage';

const {
  fetchChecklist,
  updateChecklistItem,
  createChecklistItem,
  deleteChecklistItem,
  archiveChecklistItem,
  updateChecklistDates,
} = api;
const { push } = router;

const ITEMS = [
  mk('m1', 'Learn Go'),
  mk('s1', 'goroutines', { parentId: 'm1', done: true }),
  mk('n1', 'a thought', { parentId: 'm1', type: 'note' }),
  mk('s2', 'channels', { parentId: 'm1' }),
  mk('s3', 'old step', { parentId: 'm1', archived: true }),
];
const STEPS = ['goroutines', 'a thought', 'channels'];

beforeEach(() => {
  fetchChecklist.mockResolvedValue(ITEMS);
  updateChecklistItem.mockResolvedValue({});
  deleteChecklistItem.mockResolvedValue(undefined);
  archiveChecklistItem.mockResolvedValue({});
  updateChecklistDates.mockResolvedValue({});
  duplicateChecklistItem.mockReset();
});

const serve = (items: ChecklistItem[]) => fetchChecklist.mockResolvedValue(items);

const renderPage = (milestoneId = 'm1', planId = 'plan-1') =>
  render(<MilestonePage planId={planId} milestoneId={milestoneId} />);

/** Renders the page and waits for the milestone's heading. */
const open = async (milestoneId = 'm1', title = 'Learn Go') => {
  const view = renderPage(milestoneId);
  await screen.findByRole('heading', { name: title });
  return view;
};

const slot = () => screen.getByRole('region', { name: 'Next step' });
const noSlot = () => screen.queryByRole('region', { name: 'Next step' });
const list = () => screen.getByRole('list', { name: 'Steps' });
/** The page's step rows, top to bottom, by their text. */
const steps = () =>
  within(list())
    .getAllByRole('listitem')
    .map((li) => li.textContent);
const row = (text: string) => within(list()).getByText(text).closest('li') as HTMLElement;
const stepBox = (text: string) =>
  within(list()).getByRole('checkbox', { name: `Mark ${text} done` });
const nudge = () => document.querySelector('[data-completion-nudge]');

const openMenu = () =>
  fireEvent.click(screen.getByRole('button', { name: 'Actions for Learn Go' }));
/** Opens the milestone's menu and picks `name` from it. */
const choose = async (name: string) => {
  openMenu();
  await act(async () => fireEvent.click(screen.getByRole('menuitem', { name })));
};
const confirmDelete = () =>
  act(async () =>
    fireEvent.click(
      within(
        screen.getByText(/permanently delete/i).closest('div') as HTMLElement
      ).getByRole('button', { name: 'Delete' })
    )
  );

describe('MilestonePage', () => {
  it('should show the milestone title, its progress and every live step in order', async () => {
    await open();
    expect(fetchChecklist).toHaveBeenCalledWith('plan-1', 'longterm');
    // Same measure as the card: task steps only, the note and the archived
    // step left out (R20).
    expect(screen.getByText('1/2')).toBeInTheDocument();
    expect(steps()).toEqual(STEPS);
    // A quiet room, not a ticket (R13).
    expect(
      screen.queryByText(/assignee|priority|label|estimate|comment|status/i)
    ).toBeNull();
  });

  const notHere = async () => {
    expect(
      await screen.findByText("This milestone isn't here")
    ).toBeInTheDocument();
    expect(
      screen.getByRole('link', { name: /return to the plan/i })
    ).toHaveAttribute('href', '/plan/plan-1');
  };

  it.each<[string, string, ChecklistItem[]?]>([
    ['an unknown id', 'nope'],
    ['a note', 'n1'],
    ['a step', 's1'],
    ['an archived item', 's3'],
    ["another plan's milestone (G4)", 'm9', [mk('m9', 'Elsewhere', { planId: 'plan-2' })]],
  ])('should say %s is not here, with a way back (R33)', async (_why, id, items) => {
    if (items) serve(items);
    renderPage(id);
    await notHere();
    expect(screen.queryByRole('heading')).not.toBeInTheDocument();
  });

  it.each([404, 403, 422])(
    'should say the milestone is not here when the plan list answers %i (R33)',
    async (status) => {
      fetchChecklist.mockRejectedValue(new ApiError(undefined, status));
      renderPage();
      await notHere();
    }
  );

  it.each([
    ['a 503', () => new ApiError({ code: 'SERVICE_UNAVAILABLE' } as never, 503)],
    ['a 500', () => new ApiError(undefined, 500)],
    ['a network failure', () => new TypeError('Failed to fetch')],
  ])('should offer a retry, not "not here", on %s (R34)', async (_why, err) => {
    quietErrors();
    fetchChecklist.mockRejectedValueOnce(err());
    renderPage();

    expect(
      await screen.findByText("Couldn't load this milestone")
    ).toBeInTheDocument();
    expect(screen.queryByText("This milestone isn't here")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
    expect(
      await screen.findByRole('heading', { name: 'Learn Go' })
    ).toBeInTheDocument();
    expect(fetchChecklist).toHaveBeenCalledTimes(2);
  });

  it('should show a quiet loading line while the list loads, never "not here" (R14)', async () => {
    const load = deferred<ChecklistItem[]>();
    fetchChecklist.mockReturnValue(load.promise);
    renderPage();

    expect(screen.getByText(/loading milestone/i)).toBeInTheDocument();
    expect(screen.queryByText("This milestone isn't here")).not.toBeInTheDocument();

    load.resolve(ITEMS);
    expect(
      await screen.findByRole('heading', { name: 'Learn Go' })
    ).toBeInTheDocument();
    expect(screen.queryByText(/loading milestone/i)).not.toBeInTheDocument();
  });

  it.each([
    ['no steps', [mk('m2', 'Book flights')]],
    [
      'only note children (G12)',
      [mk('m2', 'Book flights'), mk('n2', 'window seat', { parentId: 'm2', type: 'note' })],
    ],
  ])(
    'should show no meter for a milestone with %s, and invite the first step, opening the composer (R20, R28)',
    async (_why, items) => {
      serve(items);
      const { container } = await open('m2', 'Book flights');

      expect(container.querySelector('[data-milestone-progress]')).toBeNull();
      expect(screen.queryByText(/\d+\/\d+/)).not.toBeInTheDocument();

      fireEvent.click(
        within(slot()).getByRole('button', { name: /add the first step/i })
      );
      expect(
        screen.getByRole('textbox', { name: 'Add a step to Book flights' })
      ).toHaveFocus();
    }
  );

  it("should name the browser tab after the milestone while it is open (R12)", async () => {
    document.title = 'Fireplace';
    const { unmount } = await open();
    expect(document.title).toBe('Learn Go');

    unmount();
    expect(document.title).toBe('Fireplace');
  });

  it('should lead back to the plan by its URL, not by history (R9, G13)', async () => {
    await open();
    expect(screen.getByRole('link', { name: 'Back to plan' })).toHaveAttribute(
      'href',
      '/plan/plan-1'
    );
  });
});

/**
 * Working the milestone (FS-043WP R21, R26–R32): steps tick, rename and get
 * added on the page; the one next step is shown on its own; and once every
 * step is done the grid's own nudge stands in for it.
 */
describe('MilestonePage steps and next step', () => {
  it('should tick and untick a task step through the item update (R30)', async () => {
    await open();

    await act(async () => fireEvent.click(stepBox('channels')));
    expect(updateChecklistItem).toHaveBeenCalledWith(
      's2',
      { done: true },
      'plan-1',
      'longterm'
    );
    expect(stepBox('channels')).toBeChecked();
    expect(screen.getByText('2/2')).toBeInTheDocument();

    await act(async () => fireEvent.click(stepBox('goroutines')));
    expect(updateChecklistItem).toHaveBeenLastCalledWith(
      's1',
      { done: false },
      'plan-1',
      'longterm'
    );
    expect(stepBox('goroutines')).not.toBeChecked();
  });

  it('should revert a failed tick, keep it the next step and show the existing error (R32)', async () => {
    quietErrors();
    const call = deferred<never>();
    updateChecklistItem.mockReturnValue(call.promise);
    await open();

    fireEvent.click(stepBox('channels'));
    // Optimistic: ticked before the request lands.
    expect(stepBox('channels')).toBeChecked();

    await act(async () => call.reject(new Error('nope')));
    expect(stepBox('channels')).not.toBeChecked();
    expect(
      screen.getByText('Failed to update task status. Please try again.')
    ).toBeInTheDocument();
    expect(within(slot()).getByText('channels')).toBeInTheDocument();
  });

  it('should show the first open task step on its own, marked in the list, and promote the next once ticked (R26, R27)', async () => {
    serve([
      mk('m1', 'Learn Go'),
      mk('s1', 'goroutines', { parentId: 'm1', done: true }),
      mk('n1', 'a thought', { parentId: 'm1', type: 'note' }),
      mk('s2', 'channels', { parentId: 'm1' }),
      mk('s3', 'select', { parentId: 'm1' }),
    ]);
    await open();

    // Done steps and notes are never next.
    expect(within(slot()).getByText('channels')).toBeInTheDocument();
    expect(row('channels')).toHaveAttribute('aria-current', 'step');
    expect(row('select')).not.toHaveAttribute('aria-current');
    expect(row('a thought')).not.toHaveAttribute('aria-current');

    await act(async () =>
      fireEvent.click(
        within(slot()).getByRole('checkbox', { name: 'Mark channels done' })
      )
    );
    expect(updateChecklistItem).toHaveBeenCalledWith(
      's2',
      { done: true },
      'plan-1',
      'longterm'
    );
    expect(within(slot()).getByText('select')).toBeInTheDocument();
    expect(row('select')).toHaveAttribute('aria-current', 'step');
    expect(stepBox('channels')).toBeChecked();
  });

  const allDone = () => [
    mk('m1', 'Learn Go'),
    mk('s1', 'goroutines', { parentId: 'm1', done: true }),
    mk('n1', 'a thought', { parentId: 'm1', type: 'note' }),
    mk('s2', 'channels', { parentId: 'm1', done: true }),
  ];

  it('should stand the completion nudge in the slot once every step is done, and close through it (R21, R28)', async () => {
    serve(allDone());
    await open();

    const text = within(slot()).getByText('All steps done. Close this milestone?');
    expect(text.closest('[data-completion-nudge]')).not.toBeNull();

    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: 'Close Learn Go' }))
    );
    expect(updateChecklistItem).toHaveBeenCalledWith(
      'm1',
      { done: true },
      'plan-1',
      'longterm'
    );
    // Closed: no nudge and no next step; the closed state speaks (G11).
    expect(nudge()).toBeNull();
    expect(noSlot()).toBeNull();
  });

  it('should hold a dismissed nudge until the progress pair changes (R21)', async () => {
    serve(allDone());
    await open();

    fireEvent.click(screen.getByRole('button', { name: 'Not now, Learn Go' }));
    expect(nudge()).toBeNull();

    // Untick, re-tick: the offer comes back.
    await act(async () => fireEvent.click(stepBox('channels')));
    expect(nudge()).toBeNull();
    expect(within(slot()).getByText('channels')).toBeInTheDocument();
    await act(async () => fireEvent.click(stepBox('channels')));
    expect(nudge()).not.toBeNull();
  });

  it('should show no next step for a closed milestone, its steps still tickable (G11)', async () => {
    serve([mk('m1', 'Learn Go', { done: true }), mk('s2', 'channels', { parentId: 'm1' })]);
    await open();

    expect(noSlot()).toBeNull();
    expect(row('channels')).not.toHaveAttribute('aria-current');
    await act(async () => fireEvent.click(stepBox('channels')));
    expect(stepBox('channels')).toBeChecked();
  });

  it('should show a note step by its card text (FS-PGG2N R11)', async () => {
    serve([
      mk('m1', 'Learn Go'),
      mk('n1', 'a long body\nwith lines', {
        parentId: 'm1',
        type: 'note',
        summary: 'the gist',
      }),
      mk('n2', 'first\nsecond', { parentId: 'm1', type: 'note' }),
      mk('s1', 'a task', { parentId: 'm1' }),
    ]);
    await open();

    expect(within(list()).getByText('the gist')).toBeInTheDocument();
    expect(within(list()).queryByText(/a long body/)).toBeNull();
    expect(within(list()).getByText('first second', { normalizer: (t) => t.replace(/\s+/g, ' ') })).toBeInTheDocument();
  });

  describe('rename (R30)', () => {
    const startRename = async (text: string) => {
      await open();
      fireEvent.click(within(list()).getByText(text));
      return screen.getByRole('textbox', { name: `Rename ${text}` });
    };

    it.each([
      ['Enter', (input: HTMLElement) => fireEvent.keyDown(input, { key: 'Enter' })],
      ['blur', (input: HTMLElement) => fireEvent.blur(input)],
    ])('should save trimmed on %s, and the next-step slot shows the new text too', async (_how, commit) => {
      const input = await startRename('channels');
      fireEvent.change(input, { target: { value: '  chans  ' } });
      await act(async () => commit(input));

      expect(updateChecklistItem).toHaveBeenCalledWith(
        's2',
        { description: 'chans' },
        'plan-1',
        'longterm'
      );
      expect(within(list()).getByText('chans')).toBeInTheDocument();
      expect(within(slot()).getByText('chans')).toBeInTheDocument();
    });

    it("should open a note step's page from its text rather than rename it (FS-PGG2N R13)", async () => {
      await open();
      const link = within(list()).getByRole('link', { name: 'a thought' });
      expect(link.getAttribute('href')).toBe('/plan/plan-1/n/n1');
      fireEvent.click(link);
      expect(screen.queryByRole('textbox', { name: 'Rename a thought' })).toBeNull();
    });

    it('should cancel on Esc, and keep the old text on an empty rename', async () => {
      const input = await startRename('channels');
      fireEvent.change(input, { target: { value: 'something else' } });
      fireEvent.keyDown(input, { key: 'Escape' });
      expect(within(list()).getByText('channels')).toBeInTheDocument();

      fireEvent.click(within(list()).getByText('channels'));
      const again = screen.getByRole('textbox', { name: 'Rename channels' });
      fireEvent.change(again, { target: { value: '   ' } });
      fireEvent.keyDown(again, { key: 'Enter' });
      expect(within(list()).getByText('channels')).toBeInTheDocument();
      expect(updateChecklistItem).not.toHaveBeenCalled();
    });

    it('should revert a failed rename and show the existing error (R32)', async () => {
      quietErrors();
      updateChecklistItem.mockRejectedValue(new Error('nope'));
      const input = await startRename('channels');
      fireEvent.change(input, { target: { value: 'chans' } });
      await act(async () => fireEvent.keyDown(input, { key: 'Enter' }));

      expect(within(list()).getByText('channels')).toBeInTheDocument();
      expect(
        screen.getByText('Failed to update task. Please try again.')
      ).toBeInTheDocument();
    });
  });

  describe('add step (R31, G9)', () => {
    const openComposer = async () => {
      await open();
      fireEvent.click(screen.getByRole('button', { name: /add step/i }));
      return screen.getByRole('textbox', { name: 'Add a step to Learn Go' });
    };

    it('should add a task step at the end, clear the input and keep focus', async () => {
      createChecklistItem.mockResolvedValue(
        mk('s9', 'interfaces', { parentId: 'm1' })
      );
      const input = await openComposer();
      expect(input).toHaveFocus();

      fireEvent.change(input, { target: { value: ' interfaces ' } });
      await act(async () => fireEvent.keyDown(input, { key: 'Enter' }));

      expect(createChecklistItem).toHaveBeenCalledWith(
        'interfaces',
        'plan-1',
        'longterm',
        { type: 'task', parentId: 'm1' }
      );
      expect(steps().at(-1)).toBe('interfaces');
      expect(screen.getByText('1/3')).toBeInTheDocument();
      expect(input).toHaveValue('');
      expect(input).toHaveFocus();
    });

    it('should ignore Enter while a create is in flight', async () => {
      const call = deferred<ChecklistItem>();
      createChecklistItem.mockReturnValue(call.promise);
      const input = await openComposer();

      fireEvent.change(input, { target: { value: 'interfaces' } });
      fireEvent.keyDown(input, { key: 'Enter' });
      fireEvent.keyDown(input, { key: 'Enter' });
      expect(createChecklistItem).toHaveBeenCalledTimes(1);

      await act(async () =>
        call.resolve(mk('s9', 'interfaces', { parentId: 'm1' }))
      );
      expect(within(list()).getAllByText('interfaces')).toHaveLength(1);
    });

    it('should keep the text and show the existing error when the create fails', async () => {
      quietErrors();
      createChecklistItem.mockRejectedValue(new Error('nope'));
      const input = await openComposer();

      fireEvent.change(input, { target: { value: 'interfaces' } });
      await act(async () => fireEvent.keyDown(input, { key: 'Enter' }));

      expect(input).toHaveValue('interfaces');
      expect(
        screen.getByText('Failed to add step. Please try again.')
      ).toBeInTheDocument();
    });

    it('should close on Esc', async () => {
      const input = await openComposer();
      fireEvent.change(input, { target: { value: 'half a thought' } });
      fireEvent.keyDown(input, { key: 'Escape' });
      expect(
        screen.queryByRole('textbox', { name: 'Add a step to Learn Go' })
      ).toBeNull();
      expect(screen.getByRole('button', { name: /add step/i })).toBeInTheDocument();
    });
  });
});

/**
 * The milestone itself, edited on its page (FS-043WP R15–R19): its title, its
 * dates, closing it, and the two ways out, archive and delete, both of which
 * leave for the plan.
 */
describe('MilestonePage title, dates, close and menu', () => {
  it('should offer exactly Rename, Dates, Duplicate, Archive and Delete in its menu (R17, G6; FS-F6AEE R14)', async () => {
    await open();
    openMenu();
    expect(
      screen.getAllByRole('menuitem').map((r) => r.textContent)
    ).toEqual(['Set dates', 'Rename', 'Duplicate', 'Archive', 'Delete']);
  });

  describe('duplicate (FS-F6AEE R14, R16, R17, R20)', () => {
    it('should duplicate the milestone and stay on it, its steps untouched', async () => {
      duplicateChecklistItem.mockResolvedValue([
        mk('d1', 'Learn Go'),
        mk('d2', 'goroutines', { parentId: 'd1' }),
        mk('d3', 'channels', { parentId: 'd1' }),
      ]);
      await open();
      await choose('Duplicate');

      expect(duplicateChecklistItem).toHaveBeenCalledWith('plan-1', 'm1');
      expect(push).not.toHaveBeenCalled();
      expect(screen.getAllByRole('heading', { name: 'Learn Go' })).toHaveLength(1);
      // The copy's steps belong to the copy: this milestone's are as they were.
      expect(steps()).toEqual(STEPS);
      expect(screen.queryByText(/failed to duplicate/i)).toBeNull();
    });

    it("should stay on the page with the page's error when it fails, inserting nothing", async () => {
      quietErrors();
      duplicateChecklistItem.mockRejectedValue(new Error('nope'));
      await open();
      await choose('Duplicate');

      expect(
        screen.getByText('Failed to duplicate item. Please try again.')
      ).toBeInTheDocument();
      expect(push).not.toHaveBeenCalled();
      expect(screen.getByRole('heading', { name: 'Learn Go' })).toBeInTheDocument();
      expect(steps()).toEqual(STEPS);
    });

    it('should add no menu to the steps on the page', async () => {
      await open();
      expect(
        screen.queryByRole('button', { name: /^Actions for (goroutines|channels|a thought)$/ })
      ).toBeNull();
    });
  });

  it('should confirm a delete, then leave for the plan (R18, C11)', async () => {
    await open();
    await choose('Delete');

    // Nothing goes until it is confirmed.
    expect(deleteChecklistItem).not.toHaveBeenCalled();
    expect(screen.getByText(/permanently delete "Learn Go"/i)).toBeInTheDocument();
    await confirmDelete();
    expect(deleteChecklistItem).toHaveBeenCalledWith('m1', 'plan-1');
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });

  it('should delete nothing when the confirmation is cancelled', async () => {
    await open();
    await choose('Delete');
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(screen.queryByText(/permanently delete/i)).toBeNull();
    expect(deleteChecklistItem).not.toHaveBeenCalled();
    expect(push).not.toHaveBeenCalled();
  });

  it('should archive, then leave for the plan (R18, G7)', async () => {
    await open();
    await choose('Archive');
    expect(archiveChecklistItem).toHaveBeenCalledWith('m1', 'plan-1', 'longterm');
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });

  it.each([
    [
      'delete',
      'Failed to delete task',
      async () => {
        await choose('Delete');
        await confirmDelete();
      },
    ],
    ['archive', 'Failed to archive task. Please try again.', () => choose('Archive')],
  ])('should stay on the page with the existing error when a %s fails (R18)', async (_what, message, leave) => {
    quietErrors();
    deleteChecklistItem.mockRejectedValue(new Error('nope'));
    archiveChecklistItem.mockRejectedValue(new Error('nope'));
    await open();
    await leave();

    expect(push).not.toHaveBeenCalled();
    expect(screen.getByText(message)).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Learn Go' })).toBeInTheDocument();
    expect(screen.queryByText(/permanently delete/i)).toBeNull();
  });

  describe('closing (R16, C10)', () => {
    const box = () => screen.getByRole('checkbox', { name: 'Mark Learn Go done' });

    it('should close by hand at any progress, staying on the page shown closed, and reopen', async () => {
      await open();
      expect(box()).toBeEnabled();

      await act(async () => fireEvent.click(box()));
      expect(updateChecklistItem).toHaveBeenCalledWith(
        'm1',
        { done: true },
        'plan-1',
        'longterm'
      );
      expect(box()).toBeChecked();
      expect(push).not.toHaveBeenCalled();
      // Steps open, yet closed: no next step (G11).
      expect(noSlot()).toBeNull();

      await act(async () => fireEvent.click(box()));
      expect(updateChecklistItem).toHaveBeenLastCalledWith(
        'm1',
        { done: false },
        'plan-1',
        'longterm'
      );
      expect(box()).not.toBeChecked();
    });

    it('should revert a failed close and show the existing error (R32)', async () => {
      quietErrors();
      updateChecklistItem.mockRejectedValue(new Error('nope'));
      await open();

      await act(async () => fireEvent.click(box()));
      expect(box()).not.toBeChecked();
      expect(
        screen.getByText('Failed to update task status. Please try again.')
      ).toBeInTheDocument();
    });
  });

  describe('title (R15)', () => {
    const titleInput = () =>
      screen.getByRole('textbox', { name: 'Rename Learn Go' });
    const clickTitle = () =>
      fireEvent.click(within(screen.getByRole('heading')).getByText('Learn Go'));

    it('should rename on click and Enter, and the tab follows', async () => {
      await open();
      clickTitle();
      fireEvent.change(titleInput(), { target: { value: ' Master Go ' } });
      await act(async () => fireEvent.keyDown(titleInput(), { key: 'Enter' }));

      expect(updateChecklistItem).toHaveBeenCalledWith(
        'm1',
        { description: 'Master Go' },
        'plan-1',
        'longterm'
      );
      expect(screen.getByRole('heading', { name: 'Master Go' })).toBeInTheDocument();
      expect(document.title).toBe('Master Go');
    });

    it('should keep the old title on an empty rename and cancel on Esc', async () => {
      await open();
      clickTitle();
      fireEvent.change(titleInput(), { target: { value: '   ' } });
      fireEvent.blur(titleInput());
      expect(screen.getByRole('heading', { name: 'Learn Go' })).toBeInTheDocument();

      clickTitle();
      fireEvent.change(titleInput(), { target: { value: 'Other' } });
      fireEvent.keyDown(titleInput(), { key: 'Escape' });
      expect(screen.getByRole('heading', { name: 'Learn Go' })).toBeInTheDocument();
      expect(updateChecklistItem).not.toHaveBeenCalled();
    });

    it("should start the same rename from the menu's Rename", async () => {
      await open();
      await choose('Rename');
      expect(titleInput()).toHaveFocus();
      expect(titleInput()).toHaveValue('Learn Go');
    });
  });

  describe('dates (R19)', () => {
    const picker = () => screen.queryByText('Pick a day, or a start and an end.');
    const chip = () => document.querySelector('[data-item-dates]') as HTMLElement | null;
    const the15th = () =>
      document.querySelector(
        '.react-datepicker__day--015:not(.react-datepicker__day--outside-month)'
      ) as HTMLElement;
    const withDates = () =>
      serve([mk('m1', 'Learn Go', { startDate: '2026-03-02', dueDate: '2026-03-09' })]);

    it('should offer "Add dates" with none, open the picker, and save a day through the dates call', async () => {
      await open();
      expect(chip()).toBeNull();

      fireEvent.click(screen.getByRole('button', { name: 'Add dates' }));
      expect(picker()).toBeInTheDocument();
      fireEvent.click(the15th());
      await act(async () =>
        fireEvent.click(screen.getByRole('button', { name: 'Save' }))
      );

      const day = expect.stringMatching(/^\d{4}-\d{2}-15$/);
      expect(updateChecklistDates).toHaveBeenCalledWith('plan-1', 'm1', {
        startDate: day,
        dueDate: day,
      });
      expect(chip()).not.toBeNull();
      expect(screen.queryByRole('button', { name: 'Add dates' })).toBeNull();
    });

    it('should open the picker from the chip and clear the dates', async () => {
      withDates();
      await open();

      fireEvent.click(chip()!);
      expect(picker()).toBeInTheDocument();
      await act(async () =>
        fireEvent.click(screen.getByRole('button', { name: 'Clear' }))
      );

      expect(updateChecklistDates).toHaveBeenCalledWith('plan-1', 'm1', {
        startDate: null,
        dueDate: null,
      });
      expect(chip()).toBeNull();
      expect(screen.getByRole('button', { name: 'Add dates' })).toBeInTheDocument();
    });

    it("should open the same picker from the menu's dates row", async () => {
      await open();
      await choose('Set dates');
      expect(picker()).toBeInTheDocument();
    });

    it('should revert only the dates when they fail, show the error, and keep a rename made meanwhile (R32)', async () => {
      quietErrors();
      const call = deferred<never>();
      updateChecklistDates.mockReturnValue(call.promise);
      withDates();
      await open();

      fireEvent.click(chip()!);
      await act(async () =>
        fireEvent.click(screen.getByRole('button', { name: 'Clear' }))
      );
      // A rename lands while the dates are still in flight.
      fireEvent.click(within(screen.getByRole('heading')).getByText('Learn Go'));
      const input = screen.getByRole('textbox', { name: 'Rename Learn Go' });
      fireEvent.change(input, { target: { value: 'Master Go' } });
      await act(async () => fireEvent.keyDown(input, { key: 'Enter' }));
      await act(async () => call.reject(new Error('nope')));

      expect(chip()).not.toBeNull();
      expect(
        screen.getByText('Failed to save the dates. Please try again.')
      ).toBeInTheDocument();
      expect(screen.getByRole('heading', { name: 'Master Go' })).toBeInTheDocument();
    });
  });
});

describe('MilestonePage why it matters (R22–R25, G10)', () => {
  const why = () => screen.getByRole('region', { name: 'Why it matters' });
  const PROMPT = 'Why does this matter?';
  const field = () => within(why()).getByRole('textbox', { name: PROMPT });
  /** Opens the page on a milestone whose "why" already reads `intention`, and starts editing it. */
  const editIntention = async (intention = 'ship the API') => {
    serve([mk('m1', 'Learn Go', { intention }), ...ITEMS.slice(1)]);
    await open();
    fireEvent.click(within(why()).getByText(intention));
  };

  it('should ask "Why does this matter?" when empty, stop input at 280 characters, and save the trimmed text on Enter (R22, R23, G10)', async () => {
    await open();

    fireEvent.click(within(why()).getByText(PROMPT));
    expect(field()).toHaveAttribute('maxLength', '280');
    fireEvent.change(field(), { target: { value: '  to build the backend  ' } });
    await act(async () => fireEvent.keyDown(field(), { key: 'Enter' }));

    expect(updateChecklistItem).toHaveBeenCalledWith(
      'm1',
      { intention: 'to build the backend' },
      'plan-1',
      'longterm'
    );
    expect(within(why()).getByText('to build the backend')).toBeInTheDocument();
    expect(within(why()).queryByText(PROMPT)).not.toBeInTheDocument();
  });

  it('should show set text, edit it on click and save on blur (R22, R23)', async () => {
    await editIntention();
    expect(field()).toHaveValue('ship the API');
    fireEvent.change(field(), { target: { value: 'ship the API by May' } });
    await act(async () => fireEvent.blur(field()));

    expect(updateChecklistItem).toHaveBeenCalledWith(
      'm1',
      { intention: 'ship the API by May' },
      'plan-1',
      'longterm'
    );
    expect(within(why()).getByText('ship the API by May')).toBeInTheDocument();
  });

  it('should cancel on Esc and send nothing for unchanged text (R23)', async () => {
    await editIntention();
    fireEvent.change(field(), { target: { value: 'something else' } });
    fireEvent.keyDown(field(), { key: 'Escape' });
    expect(within(why()).getByText('ship the API')).toBeInTheDocument();

    fireEvent.click(within(why()).getByText('ship the API'));
    fireEvent.change(field(), { target: { value: '  ship the API ' } });
    await act(async () => fireEvent.keyDown(field(), { key: 'Enter' }));
    expect(updateChecklistItem).not.toHaveBeenCalled();
    expect(within(why()).getByText('ship the API')).toBeInTheDocument();
  });

  it.each([
    ['empty', ''],
    ['whitespace', '   '],
  ])('should clear it on %s text and bring the prompt back (R23)', async (_why, typed) => {
    await editIntention();
    fireEvent.change(field(), { target: { value: typed } });
    await act(async () => fireEvent.keyDown(field(), { key: 'Enter' }));

    expect(updateChecklistItem).toHaveBeenCalledWith(
      'm1',
      { intention: '' },
      'plan-1',
      'longterm'
    );
    expect(within(why()).getByText(PROMPT)).toBeInTheDocument();
  });

  it('should restore the old text and show the existing error when the save fails (R24)', async () => {
    quietErrors();
    updateChecklistItem.mockRejectedValue(new Error('nope'));
    await editIntention();
    fireEvent.change(field(), { target: { value: 'something else' } });
    await act(async () => fireEvent.keyDown(field(), { key: 'Enter' }));

    expect(within(why()).getByText('ship the API')).toBeInTheDocument();
    expect(
      screen.getByText('Failed to update task. Please try again.')
    ).toBeInTheDocument();
  });

  it('should let a keyboard reach the prompt and the set text, and start editing (US29)', async () => {
    const user = userEvent.setup();
    await open();

    const prompt = within(why()).getByRole('button', { name: PROMPT });
    prompt.focus();
    await user.keyboard('{Enter}');
    expect(field()).toHaveFocus();
    await user.keyboard('to build the backend');
    await act(async () => user.keyboard('{Enter}'));

    expect(updateChecklistItem).toHaveBeenCalledWith(
      'm1',
      { intention: 'to build the backend' },
      'plan-1',
      'longterm'
    );
    // Back on the text, ready to edit again with Space.
    const set = within(why()).getByRole('button', { name: 'to build the backend' });
    expect(set).toHaveFocus();
    await user.keyboard(' ');
    expect(field()).toHaveValue('to build the backend');
    await user.keyboard('{Escape}');
    expect(within(why()).getByRole('button', { name: 'to build the backend' })).toHaveFocus();
  });

  it('should let a keyboard start a title or step rename, each still named "Rename …" (US29)', async () => {
    const user = userEvent.setup();
    await open();

    within(screen.getByRole('heading')).getByRole('button', { name: 'Learn Go' }).focus();
    await user.keyboard('{Enter}');
    expect(screen.getByRole('textbox', { name: 'Rename Learn Go' })).toHaveFocus();
    await user.keyboard('{Escape}');

    within(list()).getByRole('button', { name: 'channels' }).focus();
    await user.keyboard('{Enter}');
    expect(screen.getByRole('textbox', { name: 'Rename channels' })).toHaveFocus();
    // The heading still reads as the title alone.
    expect(screen.getByRole('heading', { name: 'Learn Go' })).toBeInTheDocument();
  });

  it('should sit after the progress and above the next step (R11)', async () => {
    const { container } = await open();

    const progress = container.querySelector('[data-milestone-progress]')!;
    const follows = (a: Node, b: Node) =>
      Boolean(a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING);
    expect(follows(progress, why())).toBe(true);
    expect(follows(why(), slot())).toBe(true);
  });
});
