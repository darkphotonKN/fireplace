import { describe, it, expect, vi, beforeEach } from 'vitest';
import { act, fireEvent, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ApiError } from '@/api/client';
import type { ChecklistItem } from '@/api/checklists';

/**
 * The milestone page (FS-043WP): one milestone read out of the plan's
 * long-term list, the same list the grid reads, so the page and the card can
 * never disagree about it (R10). Pinned here: what the page shows for a
 * milestone, and the calm states for everything that is not one.
 */

const fetchChecklist = vi.fn();
const updateChecklistItem = vi.fn();
const createChecklistItem = vi.fn();
const deleteChecklistItem = vi.fn();
const archiveChecklistItem = vi.fn();
const updateChecklistDates = vi.fn();
vi.mock('@/services/api', () => ({
  fetchChecklist: (...a: unknown[]) => fetchChecklist(...a),
  updateChecklistItem: (...a: unknown[]) => updateChecklistItem(...a),
  createChecklistItem: (...a: unknown[]) => createChecklistItem(...a),
  deleteChecklistItem: (...a: unknown[]) => deleteChecklistItem(...a),
  archiveChecklistItem: (...a: unknown[]) => archiveChecklistItem(...a),
  updateChecklistDates: (...a: unknown[]) => updateChecklistDates(...a),
}));

const duplicateChecklistItem = vi.fn();
vi.mock('@/api/checklists', () => ({
  duplicateChecklistItem: (...a: unknown[]) => duplicateChecklistItem(...a),
}));

const push = vi.fn();
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push, replace: vi.fn(), refresh: vi.fn() }),
}));

import MilestonePage from './MilestonePage';

const mk = (id: string, description: string, extra: Partial<ChecklistItem> = {}) =>
  ({
    id,
    description,
    done: false,
    scope: 'longterm',
    type: 'task',
    planId: 'plan-1',
    archived: false,
    ...extra,
  }) as ChecklistItem;

const ITEMS = [
  mk('m1', 'Learn Go'),
  mk('s1', 'goroutines', { parentId: 'm1', done: true }),
  mk('n1', 'a thought', { parentId: 'm1', type: 'note' }),
  mk('s2', 'channels', { parentId: 'm1' }),
  mk('s3', 'old step', { parentId: 'm1', archived: true }),
];

const renderPage = (milestoneId = 'm1', planId = 'plan-1') =>
  render(<MilestonePage planId={planId} milestoneId={milestoneId} />);

describe('MilestonePage', () => {
  beforeEach(() => {
    fetchChecklist.mockReset();
    fetchChecklist.mockResolvedValue(ITEMS);
  });

  it('should show the milestone title, its progress and every live step in order', async () => {
    renderPage();

    expect(
      await screen.findByRole('heading', { name: 'Learn Go' })
    ).toBeInTheDocument();
    expect(fetchChecklist).toHaveBeenCalledWith('plan-1', 'longterm');
    // Same measure as the card: task steps only, the note and the archived
    // step left out (R20).
    expect(screen.getByText('1/2')).toBeInTheDocument();

    const steps = within(screen.getByRole('list', { name: 'Steps' }))
      .getAllByRole('listitem')
      .map((li) => li.textContent);
    expect(steps).toEqual(['goroutines', 'a thought', 'channels']);
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

  it.each([
    ['an unknown id', 'nope'],
    ['a note', 'n1'],
    ['a step', 's1'],
    ['an archived item', 's3'],
  ])('should say %s is not here, with a way back (R33)', async (_why, id) => {
    renderPage(id);
    await notHere();
    expect(screen.queryByRole('heading')).not.toBeInTheDocument();
  });

  it("should say another plan's milestone is not here under this plan (G4)", async () => {
    fetchChecklist.mockResolvedValue([mk('m9', 'Elsewhere', { planId: 'plan-2' })]);
    renderPage('m9');
    await notHere();
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
    vi.spyOn(console, 'error').mockImplementation(() => {});
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
    let resolve!: (items: ChecklistItem[]) => void;
    fetchChecklist.mockReturnValue(new Promise((r) => (resolve = r)));
    renderPage();

    expect(screen.getByText(/loading milestone/i)).toBeInTheDocument();
    expect(screen.queryByText("This milestone isn't here")).not.toBeInTheDocument();

    resolve(ITEMS);
    expect(
      await screen.findByRole('heading', { name: 'Learn Go' })
    ).toBeInTheDocument();
    expect(screen.queryByText(/loading milestone/i)).not.toBeInTheDocument();
  });

  it.each([
    ['no steps', [mk('m2', 'Book flights')]],
    [
      'only note children',
      [mk('m2', 'Book flights'), mk('n2', 'window seat', { parentId: 'm2', type: 'note' })],
    ],
  ])('should show no meter for a milestone with %s (R20)', async (_why, items) => {
    fetchChecklist.mockResolvedValue(items);
    const { container } = renderPage('m2');

    expect(
      await screen.findByRole('heading', { name: 'Book flights' })
    ).toBeInTheDocument();
    expect(container.querySelector('[data-milestone-progress]')).toBeNull();
    expect(screen.queryByText(/\d+\/\d+/)).not.toBeInTheDocument();
  });

  it("should name the browser tab after the milestone while it is open (R12)", async () => {
    document.title = 'Fireplace';
    const { unmount } = renderPage();

    await screen.findByRole('heading', { name: 'Learn Go' });
    expect(document.title).toBe('Learn Go');

    unmount();
    expect(document.title).toBe('Fireplace');
  });

  it('should lead back to the plan by its URL, not by history (R9, G13)', async () => {
    renderPage();
    await screen.findByRole('heading', { name: 'Learn Go' });
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
  beforeEach(() => {
    vi.restoreAllMocks();
    fetchChecklist.mockReset();
    updateChecklistItem.mockReset();
    createChecklistItem.mockReset();
    fetchChecklist.mockResolvedValue(ITEMS);
    updateChecklistItem.mockResolvedValue({});
  });

  const ready = (title = 'Learn Go') => screen.findByRole('heading', { name: title });
  const slot = () => screen.getByRole('region', { name: 'Next step' });
  const list = () => screen.getByRole('list', { name: 'Steps' });
  const row = (text: string) =>
    within(list()).getByText(text).closest('li') as HTMLElement;
  const stepBox = (text: string) =>
    within(list()).getByRole('checkbox', { name: `Mark ${text} done` });
  const deferred = <T,>() => {
    let resolve!: (v: T) => void;
    let reject!: (e: unknown) => void;
    const promise = new Promise<T>((res, rej) => {
      resolve = res;
      reject = rej;
    });
    return { promise, resolve, reject };
  };

  it('should tick and untick a task step through the item update (R30)', async () => {
    renderPage();
    await ready();

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
    vi.spyOn(console, 'error').mockImplementation(() => {});
    const call = deferred<never>();
    updateChecklistItem.mockReturnValue(call.promise);
    renderPage();
    await ready();

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
    fetchChecklist.mockResolvedValue([
      mk('m1', 'Learn Go'),
      mk('s1', 'goroutines', { parentId: 'm1', done: true }),
      mk('n1', 'a thought', { parentId: 'm1', type: 'note' }),
      mk('s2', 'channels', { parentId: 'm1' }),
      mk('s3', 'select', { parentId: 'm1' }),
    ]);
    renderPage();
    await ready();

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

  it.each([
    ['no steps', [mk('m2', 'Book flights')]],
    [
      'only note children (G12)',
      [
        mk('m2', 'Book flights'),
        mk('n2', 'window seat', { parentId: 'm2', type: 'note' }),
      ],
    ],
  ])('should invite the first step when there are %s, opening the composer (R28)', async (_why, items) => {
    fetchChecklist.mockResolvedValue(items);
    renderPage('m2');
    await ready('Book flights');

    fireEvent.click(
      within(slot()).getByRole('button', { name: /add the first step/i })
    );
    expect(
      screen.getByRole('textbox', { name: 'Add a step to Book flights' })
    ).toHaveFocus();
  });

  const allDone = () => [
    mk('m1', 'Learn Go'),
    mk('s1', 'goroutines', { parentId: 'm1', done: true }),
    mk('n1', 'a thought', { parentId: 'm1', type: 'note' }),
    mk('s2', 'channels', { parentId: 'm1', done: true }),
  ];

  it('should stand the completion nudge in the slot once every step is done, and close through it (R21, R28)', async () => {
    fetchChecklist.mockResolvedValue(allDone());
    const { container } = renderPage();
    await ready();

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
    expect(container.querySelector('[data-completion-nudge]')).toBeNull();
    expect(screen.queryByRole('region', { name: 'Next step' })).toBeNull();
  });

  it('should hold a dismissed nudge until the progress pair changes (R21)', async () => {
    fetchChecklist.mockResolvedValue(allDone());
    const { container } = renderPage();
    await ready();
    const nudge = () => container.querySelector('[data-completion-nudge]');

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
    fetchChecklist.mockResolvedValue([
      mk('m1', 'Learn Go', { done: true }),
      mk('s2', 'channels', { parentId: 'm1' }),
    ]);
    renderPage();
    await ready();

    expect(screen.queryByRole('region', { name: 'Next step' })).toBeNull();
    expect(row('channels')).not.toHaveAttribute('aria-current');
    await act(async () => fireEvent.click(stepBox('channels')));
    expect(stepBox('channels')).toBeChecked();
  });

  describe('rename (R30)', () => {
    const startRename = async (text: string) => {
      renderPage();
      await ready();
      fireEvent.click(within(list()).getByText(text));
      return screen.getByRole('textbox', { name: `Rename ${text}` });
    };

    it('should save on Enter, and the next-step slot shows the new text too', async () => {
      const input = await startRename('channels');
      fireEvent.change(input, { target: { value: '  chans  ' } });
      await act(async () => fireEvent.keyDown(input, { key: 'Enter' }));

      expect(updateChecklistItem).toHaveBeenCalledWith(
        's2',
        { description: 'chans' },
        'plan-1',
        'longterm'
      );
      expect(within(list()).getByText('chans')).toBeInTheDocument();
      expect(within(slot()).getByText('chans')).toBeInTheDocument();
    });

    it('should save on blur, notes included', async () => {
      const input = await startRename('a thought');
      fireEvent.change(input, { target: { value: 'a better thought' } });
      await act(async () => fireEvent.blur(input));

      expect(updateChecklistItem).toHaveBeenCalledWith(
        'n1',
        { description: 'a better thought' },
        'plan-1',
        'longterm'
      );
      expect(within(list()).getByText('a better thought')).toBeInTheDocument();
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
      vi.spyOn(console, 'error').mockImplementation(() => {});
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
      renderPage();
      await ready();
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
      const steps = within(list())
        .getAllByRole('listitem')
        .map((li) => li.textContent);
      expect(steps.at(-1)).toBe('interfaces');
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
      vi.spyOn(console, 'error').mockImplementation(() => {});
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
  beforeEach(() => {
    vi.restoreAllMocks();
    for (const fn of [
      fetchChecklist,
      updateChecklistItem,
      deleteChecklistItem,
      archiveChecklistItem,
      updateChecklistDates,
      push,
    ])
      fn.mockReset();
    fetchChecklist.mockResolvedValue(ITEMS);
    updateChecklistItem.mockResolvedValue({});
    deleteChecklistItem.mockResolvedValue(undefined);
    archiveChecklistItem.mockResolvedValue({});
    updateChecklistDates.mockResolvedValue({});
  });

  const ready = (title = 'Learn Go') => screen.findByRole('heading', { name: title });
  const openMenu = () =>
    fireEvent.click(screen.getByRole('button', { name: 'Actions for Learn Go' }));

  it('should offer exactly Rename, Dates, Duplicate, Archive and Delete in its menu (R17, G6; FS-F6AEE R14)', async () => {
    renderPage();
    await ready();
    openMenu();
    expect(
      screen.getAllByRole('menuitem').map((r) => r.textContent)
    ).toEqual(['Set dates', 'Rename', 'Duplicate', 'Archive', 'Delete']);
  });

  describe('duplicate (FS-F6AEE R14, R16, R17, R20)', () => {
    /** The page's step rows, top to bottom, by their text. */
    const steps = () =>
      within(screen.getByRole('list', { name: 'Steps' }))
        .getAllByRole('listitem')
        .map((li) => li.textContent);
    const STEPS = ['goroutines', 'a thought', 'channels'];

    beforeEach(() => duplicateChecklistItem.mockReset());

    it('should duplicate the milestone and stay on it, its steps untouched', async () => {
      duplicateChecklistItem.mockResolvedValue([
        mk('d1', 'Learn Go'),
        mk('d2', 'goroutines', { parentId: 'd1' }),
        mk('d3', 'channels', { parentId: 'd1' }),
      ]);
      renderPage();
      await ready();
      openMenu();
      await act(async () =>
        fireEvent.click(screen.getByRole('menuitem', { name: 'Duplicate' }))
      );

      expect(duplicateChecklistItem).toHaveBeenCalledWith('plan-1', 'm1');
      expect(push).not.toHaveBeenCalled();
      expect(screen.getAllByRole('heading', { name: 'Learn Go' })).toHaveLength(1);
      // The copy's steps belong to the copy: this milestone's are as they were.
      expect(steps()).toEqual(STEPS);
      expect(screen.queryByText(/failed to duplicate/i)).toBeNull();
    });

    it("should stay on the page with the page's error when it fails, inserting nothing", async () => {
      vi.spyOn(console, 'error').mockImplementation(() => {});
      duplicateChecklistItem.mockRejectedValue(new Error('nope'));
      renderPage();
      await ready();
      openMenu();
      await act(async () =>
        fireEvent.click(screen.getByRole('menuitem', { name: 'Duplicate' }))
      );

      expect(
        screen.getByText('Failed to duplicate item. Please try again.')
      ).toBeInTheDocument();
      expect(push).not.toHaveBeenCalled();
      expect(screen.getByRole('heading', { name: 'Learn Go' })).toBeInTheDocument();
      expect(steps()).toEqual(STEPS);
    });

    it('should add no menu to the steps on the page', async () => {
      renderPage();
      await ready();
      expect(
        screen.queryByRole('button', { name: /^Actions for (goroutines|channels|a thought)$/ })
      ).toBeNull();
    });
  });

  it('should confirm a delete, then leave for the plan (R18, C11)', async () => {
    renderPage();
    await ready();
    openMenu();
    fireEvent.click(screen.getByRole('menuitem', { name: 'Delete' }));

    // Nothing goes until it is confirmed.
    expect(deleteChecklistItem).not.toHaveBeenCalled();
    const modal = within(
      screen.getByText(/permanently delete "Learn Go"/i).closest('div') as HTMLElement
    );
    await act(async () =>
      fireEvent.click(modal.getByRole('button', { name: 'Delete' }))
    );
    expect(deleteChecklistItem).toHaveBeenCalledWith('m1', 'plan-1');
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });

  it('should delete nothing when the confirmation is cancelled', async () => {
    renderPage();
    await ready();
    openMenu();
    fireEvent.click(screen.getByRole('menuitem', { name: 'Delete' }));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(screen.queryByText(/permanently delete/i)).toBeNull();
    expect(deleteChecklistItem).not.toHaveBeenCalled();
    expect(push).not.toHaveBeenCalled();
  });

  it('should archive, then leave for the plan (R18, G7)', async () => {
    renderPage();
    await ready();
    openMenu();
    await act(async () =>
      fireEvent.click(screen.getByRole('menuitem', { name: 'Archive' }))
    );
    expect(archiveChecklistItem).toHaveBeenCalledWith('m1', 'plan-1', 'longterm');
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });

  it.each([
    ['delete', 'Failed to delete task'],
    ['archive', 'Failed to archive task. Please try again.'],
  ])('should stay on the page with the existing error when a %s fails (R18)', async (what, message) => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    deleteChecklistItem.mockRejectedValue(new Error('nope'));
    archiveChecklistItem.mockRejectedValue(new Error('nope'));
    renderPage();
    await ready();
    openMenu();

    if (what === 'delete') {
      fireEvent.click(screen.getByRole('menuitem', { name: 'Delete' }));
      const modal = within(
        screen.getByText(/permanently delete/i).closest('div') as HTMLElement
      );
      await act(async () =>
        fireEvent.click(modal.getByRole('button', { name: 'Delete' }))
      );
    } else {
      await act(async () =>
        fireEvent.click(screen.getByRole('menuitem', { name: 'Archive' }))
      );
    }

    expect(push).not.toHaveBeenCalled();
    expect(screen.getByText(message)).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Learn Go' })).toBeInTheDocument();
    expect(screen.queryByText(/permanently delete/i)).toBeNull();
  });

  describe('closing (R16, C10)', () => {
    const box = () => screen.getByRole('checkbox', { name: 'Mark Learn Go done' });
    const heading = () => screen.getByRole('heading', { name: 'Learn Go' });

    it('should close by hand at any progress, staying on the page shown closed, and reopen', async () => {
      renderPage();
      await ready();
      expect(box()).toBeEnabled();

      await act(async () => fireEvent.click(box()));
      expect(updateChecklistItem).toHaveBeenCalledWith(
        'm1',
        { done: true },
        'plan-1',
        'longterm'
      );
      expect(box()).toBeChecked();
      expect(heading().className).toMatch(/line-through/);
      expect(push).not.toHaveBeenCalled();
      // Steps open, yet closed: no next step (G11).
      expect(screen.queryByRole('region', { name: 'Next step' })).toBeNull();

      await act(async () => fireEvent.click(box()));
      expect(updateChecklistItem).toHaveBeenLastCalledWith(
        'm1',
        { done: false },
        'plan-1',
        'longterm'
      );
      expect(box()).not.toBeChecked();
      expect(heading().className).not.toMatch(/line-through/);
    });

    it('should revert a failed close and show the existing error (R32)', async () => {
      vi.spyOn(console, 'error').mockImplementation(() => {});
      updateChecklistItem.mockRejectedValue(new Error('nope'));
      renderPage();
      await ready();

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

    it('should rename on click and Enter, and the tab follows', async () => {
      renderPage();
      await ready();
      fireEvent.click(within(screen.getByRole('heading')).getByText('Learn Go'));
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
      renderPage();
      await ready();
      fireEvent.click(within(screen.getByRole('heading')).getByText('Learn Go'));
      fireEvent.change(titleInput(), { target: { value: '   ' } });
      fireEvent.blur(titleInput());
      expect(screen.getByRole('heading', { name: 'Learn Go' })).toBeInTheDocument();

      fireEvent.click(within(screen.getByRole('heading')).getByText('Learn Go'));
      fireEvent.change(titleInput(), { target: { value: 'Other' } });
      fireEvent.keyDown(titleInput(), { key: 'Escape' });
      expect(screen.getByRole('heading', { name: 'Learn Go' })).toBeInTheDocument();
      expect(updateChecklistItem).not.toHaveBeenCalled();
    });

    it("should start the same rename from the menu's Rename", async () => {
      renderPage();
      await ready();
      openMenu();
      fireEvent.click(screen.getByRole('menuitem', { name: 'Rename' }));
      expect(titleInput()).toHaveFocus();
      expect(titleInput()).toHaveValue('Learn Go');
    });
  });

  describe('dates (R19)', () => {
    const picker = () => screen.queryByText('Pick a day, or a start and an end.');
    const the15th = () =>
      document.querySelector(
        '.react-datepicker__day--015:not(.react-datepicker__day--outside-month)'
      ) as HTMLElement;

    it('should offer "Add dates" with none, open the picker, and save a day through the dates call', async () => {
      const { container } = renderPage();
      await ready();
      expect(container.querySelector('[data-item-dates]')).toBeNull();

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
      expect(container.querySelector('[data-item-dates]')).not.toBeNull();
      expect(screen.queryByRole('button', { name: 'Add dates' })).toBeNull();
    });

    it('should open the picker from the chip and clear the dates', async () => {
      fetchChecklist.mockResolvedValue([
        mk('m1', 'Learn Go', { startDate: '2026-03-02', dueDate: '2026-03-09' }),
      ]);
      const { container } = renderPage();
      await ready();

      const chip = container.querySelector('[data-item-dates]') as HTMLElement;
      fireEvent.click(chip);
      expect(picker()).toBeInTheDocument();
      await act(async () =>
        fireEvent.click(screen.getByRole('button', { name: 'Clear' }))
      );

      expect(updateChecklistDates).toHaveBeenCalledWith('plan-1', 'm1', {
        startDate: null,
        dueDate: null,
      });
      expect(container.querySelector('[data-item-dates]')).toBeNull();
      expect(screen.getByRole('button', { name: 'Add dates' })).toBeInTheDocument();
    });

    it("should open the same picker from the menu's dates row", async () => {
      renderPage();
      await ready();
      openMenu();
      fireEvent.click(screen.getByRole('menuitem', { name: 'Set dates' }));
      expect(picker()).toBeInTheDocument();
    });

    it('should revert only the dates when they fail, keeping a rename made meanwhile (R32)', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    let fail!: (e: unknown) => void;
    updateChecklistDates.mockImplementation(
      () => new Promise((_, rej) => (fail = rej))
    );
    fetchChecklist.mockResolvedValue([
      mk('m1', 'Learn Go', { startDate: '2026-03-02', dueDate: '2026-03-09' }),
    ]);
    const { container } = renderPage();
    await ready();

    fireEvent.click(container.querySelector('[data-item-dates]') as HTMLElement);
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: 'Clear' }))
    );
    // A rename lands while the dates are still in flight.
    fireEvent.click(within(screen.getByRole('heading')).getByText('Learn Go'));
    const input = screen.getByRole('textbox', { name: 'Rename Learn Go' });
    fireEvent.change(input, { target: { value: 'Master Go' } });
    await act(async () => fireEvent.keyDown(input, { key: 'Enter' }));
    await act(async () => fail(new Error('nope')));

    expect(container.querySelector('[data-item-dates]')).not.toBeNull();
    expect(screen.getByRole('heading', { name: 'Master Go' })).toBeInTheDocument();
  });

  it('should revert failed dates and show the existing error (R32)', async () => {
      vi.spyOn(console, 'error').mockImplementation(() => {});
      updateChecklistDates.mockRejectedValue(new Error('nope'));
      fetchChecklist.mockResolvedValue([
        mk('m1', 'Learn Go', { startDate: '2026-03-02', dueDate: '2026-03-09' }),
      ]);
      const { container } = renderPage();
      await ready();

      fireEvent.click(container.querySelector('[data-item-dates]') as HTMLElement);
      await act(async () =>
        fireEvent.click(screen.getByRole('button', { name: 'Clear' }))
      );
      expect(container.querySelector('[data-item-dates]')).not.toBeNull();
      expect(
        screen.getByText('Failed to save the dates. Please try again.')
      ).toBeInTheDocument();
    });
  });
});

describe('MilestonePage why it matters (R22–R25, G10)', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    fetchChecklist.mockReset();
    updateChecklistItem.mockReset();
    fetchChecklist.mockResolvedValue(ITEMS);
    updateChecklistItem.mockResolvedValue({});
  });

  const ready = () => screen.findByRole('heading', { name: 'Learn Go' });
  const why = () => screen.getByRole('region', { name: 'Why it matters' });
  const PROMPT = 'Why does this matter?';
  const field = () => within(why()).getByRole('textbox', { name: PROMPT });

  it('should ask "Why does this matter?" when empty, and save the trimmed text on Enter (R22, R23)', async () => {
    renderPage();
    await ready();

    fireEvent.click(within(why()).getByText(PROMPT));
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

  const withIntention = (intention: string) =>
    fetchChecklist.mockResolvedValue([
      mk('m1', 'Learn Go', { intention }),
      ...ITEMS.slice(1),
    ]);

  it('should show set text, edit it on click and save on blur (R22, R23)', async () => {
    withIntention('ship the API');
    renderPage();
    await ready();

    fireEvent.click(within(why()).getByText('ship the API'));
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
    withIntention('ship the API');
    renderPage();
    await ready();

    fireEvent.click(within(why()).getByText('ship the API'));
    fireEvent.change(field(), { target: { value: 'something else' } });
    fireEvent.keyDown(field(), { key: 'Escape' });
    expect(within(why()).getByText('ship the API')).toBeInTheDocument();

    fireEvent.click(within(why()).getByText('ship the API'));
    fireEvent.change(field(), { target: { value: '  ship the API ' } });
    await act(async () => fireEvent.keyDown(field(), { key: 'Enter' }));
    expect(updateChecklistItem).not.toHaveBeenCalled();
    expect(within(why()).getByText('ship the API')).toBeInTheDocument();
  });

  it('should stop input at 280 characters (R23, G10)', async () => {
    renderPage();
    await ready();
    fireEvent.click(within(why()).getByText(PROMPT));
    expect(field()).toHaveAttribute('maxLength', '280');
  });

  it.each([
    ['empty', ''],
    ['whitespace', '   '],
  ])('should clear it on %s text and bring the prompt back (R23)', async (_why, typed) => {
    withIntention('ship the API');
    renderPage();
    await ready();

    fireEvent.click(within(why()).getByText('ship the API'));
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
    vi.spyOn(console, 'error').mockImplementation(() => {});
    updateChecklistItem.mockRejectedValue(new Error('nope'));
    withIntention('ship the API');
    renderPage();
    await ready();

    fireEvent.click(within(why()).getByText('ship the API'));
    fireEvent.change(field(), { target: { value: 'something else' } });
    await act(async () => fireEvent.keyDown(field(), { key: 'Enter' }));

    expect(within(why()).getByText('ship the API')).toBeInTheDocument();
    expect(
      screen.getByText('Failed to update task. Please try again.')
    ).toBeInTheDocument();
  });

  it('should let a keyboard reach the prompt and the set text, and start editing (US29)', async () => {
    const user = userEvent.setup();
    renderPage();
    await ready();

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
    renderPage();
    await ready();

    within(screen.getByRole('heading')).getByRole('button', { name: 'Learn Go' }).focus();
    await user.keyboard('{Enter}');
    expect(screen.getByRole('textbox', { name: 'Rename Learn Go' })).toHaveFocus();
    await user.keyboard('{Escape}');

    within(screen.getByRole('list', { name: 'Steps' }))
      .getByRole('button', { name: 'channels' })
      .focus();
    await user.keyboard('{Enter}');
    expect(screen.getByRole('textbox', { name: 'Rename channels' })).toHaveFocus();
    // The heading still reads as the title alone.
    expect(screen.getByRole('heading', { name: 'Learn Go' })).toBeInTheDocument();
  });

  it('should sit after the progress and above the next step (R11)', async () => {
    const { container } = renderPage();
    await ready();

    const progress = container.querySelector('[data-milestone-progress]')!;
    const next = screen.getByRole('region', { name: 'Next step' });
    const follows = (a: Node, b: Node) =>
      Boolean(a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING);
    expect(follows(progress, why())).toBe(true);
    expect(follows(why(), next)).toBe(true);
  });
});
