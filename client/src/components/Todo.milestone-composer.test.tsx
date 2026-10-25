import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

/**
 * The grid's new-milestone composer (FS-FSFEC R29–R36, R40, R42–R45): the
 * pinned bar rests as "+ New milestone", names a new top-level task, then
 * fills it with steps while the grid traces the card it is filling. List view
 * keeps its add bar untouched.
 */

const fetchChecklist = vi.fn();
const createChecklistItem = vi.fn();
const toast = vi.fn();

vi.mock('next/navigation', () => ({
  useParams: () => ({ planId: 'plan-1' }),
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), refresh: vi.fn() }),
}));

vi.mock('@/components/ui/use-toast', () => ({
  useToast: () => ({ toast }),
}));

vi.mock('@/services/api', () => ({
  fetchChecklist: (...a: unknown[]) => fetchChecklist(...a),
  fetchArchivedChecklist: vi.fn(async () => []),
  createChecklistItem: (...a: unknown[]) => createChecklistItem(...a),
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
}));

vi.mock('@/api/plans', () => ({
  getPlan: vi.fn(async () => ({ id: 'plan-1', dailyReset: false })),
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

// The server echoes back what it was asked to create, under a fresh id.
let nextId = 0;
const echoCreate = async (
  text: string,
  _planId: string,
  _scope: string,
  opts: { type?: string; parentId?: string } = {}
) => {
  nextId += 1;
  return item(`new${nextId}`, {
    description: text,
    type: opts.type ?? 'task',
    parentId: opts.parentId ?? null,
  });
};

const card = (id: string) =>
  document.querySelector(`[data-checklist-card="${id}"]`) as HTMLElement | null;
const traced = () =>
  Array.from(document.querySelectorAll('[data-traced]')).map((el) =>
    el.getAttribute('data-checklist-card')
  );
const resting = () => screen.queryByRole('button', { name: 'New milestone' });
const composer = () =>
  document.querySelector('[data-milestone-composer]') as HTMLElement;
const composerState = () =>
  composer()?.getAttribute('data-milestone-composer') ?? null;
const namingInput = () =>
  within(composer()).queryByRole('textbox', { name: 'Name a new milestone' }) as
    | HTMLInputElement
    | null;
const fillingInput = (title: string) =>
  within(composer()).queryByRole('textbox', {
    name: `Add a step to ${title}`,
  }) as HTMLInputElement | null;
const counter = () => screen.getByTestId('page-counter');
const viewButton = (label: string) =>
  within(screen.getByRole('group', { name: 'View' })).getByRole('button', {
    name: label,
  });
/** n top-level tasks named t1..tn; the grid pages them six at a time. */
const topLevel = (n: number) =>
  Array.from({ length: n }, (_, i) => item(`t${i + 1}`));

async function renderTodo(items: unknown[] = [item('P0')]) {
  fetchChecklist.mockImplementation(async () => items);
  const Todo = (await import('./Todo')).default;
  const view = render(<Todo fixedTaskType="longterm" enableTypeFilter />);
  await screen.findByText(
    (items[0] as { description: string } | undefined)?.description ?? 'never'
  );
  return view;
}

/** Rest → naming → filling on a milestone called `title`. */
async function startFilling(
  user: ReturnType<typeof userEvent.setup>,
  title = 'Launch'
) {
  await user.click(resting()!);
  await user.type(namingInput()!, `${title}{Enter}`);
  await screen.findByRole('textbox', { name: `Add a step to ${title}` });
}

describe('Todo new-milestone composer', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
    nextId = 0;
    createChecklistItem.mockImplementation(echoCreate);
  });

  it('should name a milestone, then fill it with a step, tracing its card', async () => {
    const user = userEvent.setup();
    await renderTodo();

    await startFilling(user, 'Launch');
    await user.type(fillingInput('Launch')!, 'Write copy{Enter}');

    expect(createChecklistItem).toHaveBeenCalledTimes(2);
    const [first, second] = createChecklistItem.mock.calls;
    expect(first.slice(0, 3)).toEqual(['Launch', 'plan-1', 'longterm']);
    expect(first[3]?.parentId).toBeUndefined();
    expect(first[3]?.type).toBe('task');
    expect(second).toEqual([
      'Write copy',
      'plan-1',
      'longterm',
      { type: 'task', parentId: 'new1' },
    ]);
    expect(traced()).toEqual(['new1']);
    expect(within(card('new1')!).getByText('Write copy')).toBeTruthy();
  });

  it('should land a new milestone with the same arrival fade the list bar gives a row', async () => {
    const user = userEvent.setup();
    await renderTodo();

    await startFilling(user, 'Launch');

    expect(card('new1')!.className).toContain('animate-fadeIn');
    expect(card('P0')!.className).not.toContain('animate-fadeIn');
  });

  it('should rest as "+ New milestone" in grid view, in place of the add bar', async () => {
    await renderTodo();

    expect(resting()).toBeTruthy();
    expect(composerState()).toBe('resting');
    expect(screen.queryByRole('textbox', { name: /^New (task|note)/ })).toBeNull();
  });

  it('should leave the list view its add bar, with no composer', async () => {
    window.localStorage.setItem('checklistView', 'list');
    await renderTodo();

    expect(composer()).toBeNull();
    expect(resting()).toBeNull();
    const input = screen.getByRole('textbox', { name: 'New task' }) as HTMLInputElement;
    expect(input.placeholder).toBe('Add something to do…');
  });

  it('should open naming with focus on click, Enter or Space', async () => {
    const user = userEvent.setup();
    await renderTodo();

    await user.click(resting()!);
    expect(composerState()).toBe('naming');
    expect(document.activeElement).toBe(namingInput());
    expect(namingInput()!.placeholder).toBe('Name a new milestone…');

    for (const key of ['{Enter}', ' ']) {
      await user.keyboard('{Escape}');
      expect(composerState()).toBe('resting');
      resting()!.focus();
      await user.keyboard(key);
      expect(composerState()).toBe('naming');
      expect(document.activeElement).toBe(namingInput());
    }
  });

  it('should take no focus on load', async () => {
    await renderTodo();
    expect(composer().contains(document.activeElement)).toBe(false);
  });

  it('should follow a new milestone to the last page, traced and named in the input', async () => {
    const user = userEvent.setup();
    await renderTodo(topLevel(12));
    expect(counter().textContent).toBe('1 of 2');

    await startFilling(user, 'Launch');

    // 13 milestones is three pages of six, and the new one opens the third.
    expect(counter().textContent).toBe('3 of 3');
    expect(card('new1')).not.toBeNull();
    expect(traced()).toEqual(['new1']);
    expect(composerState()).toBe('filling');
    expect(fillingInput('Launch')!.placeholder).toBe('Add a step…');
    expect(composer().textContent).toContain('↳ steps for Launch');
    expect(document.activeElement).toBe(fillingInput('Launch'));
  });

  it('should do nothing on an empty Enter in naming', async () => {
    const user = userEvent.setup();
    await renderTodo();
    await user.click(resting()!);

    await user.type(namingInput()!, '   {Enter}');

    expect(createChecklistItem).not.toHaveBeenCalled();
    expect(composerState()).toBe('naming');
  });

  it('should stay naming with the text kept when the create fails', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    createChecklistItem.mockRejectedValueOnce(new Error('nope'));
    const user = userEvent.setup();
    await renderTodo();
    await user.click(resting()!);

    await user.type(namingInput()!, 'Launch{Enter}');

    expect(composerState()).toBe('naming');
    expect(namingInput()!.value).toBe('Launch');
    expect(traced()).toEqual([]);
    expect(
      screen.getByText('Failed to add milestone. Please try again.')
    ).toBeTruthy();
  });

  it('should add a step, clear the input and keep focus, staying in filling', async () => {
    const user = userEvent.setup();
    await renderTodo();
    await startFilling(user);

    await user.type(fillingInput('Launch')!, 'Write copy{Enter}');
    await user.type(fillingInput('Launch')!, 'Ship it{Enter}');

    expect(composerState()).toBe('filling');
    expect(fillingInput('Launch')!.value).toBe('');
    expect(document.activeElement).toBe(fillingInput('Launch'));
    expect(within(card('new1')!).getByText('Write copy')).toBeTruthy();
    expect(within(card('new1')!).getByText('Ship it')).toBeTruthy();
  });

  it('should keep the step text when the step create fails', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    const user = userEvent.setup();
    await renderTodo();
    await startFilling(user);
    createChecklistItem.mockRejectedValueOnce(new Error('nope'));

    await user.type(fillingInput('Launch')!, 'Write copy{Enter}');

    expect(composerState()).toBe('filling');
    expect(fillingInput('Launch')!.value).toBe('Write copy');
    expect(traced()).toEqual(['new1']);
  });

  it('should open a collapsed traced milestone when a step arrives', async () => {
    const user = userEvent.setup();
    await renderTodo();
    await startFilling(user);
    await user.type(fillingInput('Launch')!, 'Write copy{Enter}');

    await user.click(
      within(card('new1')!).getByRole('button', { name: 'Collapse Launch' })
    );
    expect(within(card('new1')!).queryByText('Write copy')).toBeNull();

    await user.type(fillingInput('Launch')!, 'Ship it{Enter}');

    expect(within(card('new1')!).getByText('Write copy')).toBeTruthy();
    expect(within(card('new1')!).getByText('Ship it')).toBeTruthy();
  });

  it('should go from filling to naming on Tab or Shift+Tab, text kept and trace cleared', async () => {
    const user = userEvent.setup();
    await renderTodo();

    for (const key of ['{Tab}', '{Shift>}{Tab}{/Shift}']) {
      await user.keyboard('{Escape}');
      await startFilling(user, `Launch ${key.length}`);
      await user.type(fillingInput(`Launch ${key.length}`)!, 'half typed');

      await user.keyboard(key);

      expect(composerState()).toBe('naming');
      expect(namingInput()!.value).toBe('half typed');
      expect(document.activeElement).toBe(namingInput());
      expect(traced()).toEqual([]);
    }
  });

  it('should swallow Tab and Shift+Tab in naming, focus staying put', async () => {
    const user = userEvent.setup();
    await renderTodo();
    await user.click(resting()!);
    await user.type(namingInput()!, 'Launch');

    await user.keyboard('{Tab}');
    expect(composerState()).toBe('naming');
    expect(document.activeElement).toBe(namingInput());

    await user.keyboard('{Shift>}{Tab}{/Shift}');
    expect(composerState()).toBe('naming');
    expect(document.activeElement).toBe(namingInput());
    expect(namingInput()!.value).toBe('Launch');
  });

  it('should rest on Esc from naming or filling, text gone and nothing traced', async () => {
    const user = userEvent.setup();
    await renderTodo();

    await user.click(resting()!);
    await user.type(namingInput()!, 'Launch');
    await user.keyboard('{Escape}');
    expect(composerState()).toBe('resting');
    expect(document.activeElement).toBe(resting());

    await startFilling(user);
    await user.type(fillingInput('Launch')!, 'half');
    await user.keyboard('{Escape}');
    expect(composerState()).toBe('resting');
    expect(traced()).toEqual([]);

    await user.click(resting()!);
    expect(namingInput()!.value).toBe('');
  });

  it('should keep filling and the trace when a card is clicked or a step ticked', async () => {
    const user = userEvent.setup();
    await renderTodo([
      item('P0'),
      item('S0', { parentId: 'P0' }),
    ]);
    await startFilling(user);

    await user.click(card('P0')!);
    await user.click(
      within(card('P0')!).getByRole('checkbox', { name: 'Mark item S0 done' })
    );

    expect(composerState()).toBe('filling');
    expect(traced()).toEqual(['new1']);
    expect(composer().contains(document.activeElement)).toBe(false);

    await user.type(fillingInput('Launch')!, 'Write copy{Enter}');
    expect(within(card('new1')!).getByText('Write copy')).toBeTruthy();
  });

  it('should start resting again after a reload mid-fill', async () => {
    const user = userEvent.setup();
    const { unmount } = await renderTodo();
    await startFilling(user);
    unmount();

    await renderTodo();

    expect(composerState()).toBe('resting');
    expect(traced()).toEqual([]);
  });

  it('should ignore Enter while a create is in flight, focus kept', async () => {
    let resolve!: (v: unknown) => void;
    createChecklistItem.mockImplementationOnce(
      () => new Promise((r) => (resolve = r))
    );
    const user = userEvent.setup();
    await renderTodo();
    await user.click(resting()!);
    await user.type(namingInput()!, 'Launch{Enter}');

    expect(namingInput()!.readOnly).toBe(true);
    await user.keyboard('{Enter}');
    await user.keyboard('{Tab}');
    await user.keyboard('{Escape}');

    expect(createChecklistItem).toHaveBeenCalledTimes(1);
    expect(composerState()).toBe('naming');
    expect(document.activeElement).toBe(namingInput());

    await act(async () => resolve(item('new1', { description: 'Launch' })));
    expect(composerState()).toBe('filling');
    expect(traced()).toEqual(['new1']);
  });

  it('should never offer the "Tab to nest" tip', async () => {
    const user = userEvent.setup();
    await renderTodo();
    await startFilling(user);
    await user.keyboard('{Tab}');
    await user.keyboard('{Tab}');

    expect(toast).not.toHaveBeenCalledWith(
      expect.objectContaining({ title: 'Tip: Tab to nest' })
    );
  });

  it("should let a card's own add step add to that card, leaving the composer alone", async () => {
    const user = userEvent.setup();
    await renderTodo();
    await startFilling(user);
    await user.type(fillingInput('Launch')!, 'half');

    await user.click(within(card('P0')!).getByRole('button', { name: 'add step' }));
    await user.type(
      within(card('P0')!).getByRole('textbox', { name: 'Add a step to item P0' }),
      'Old step{Enter}'
    );

    expect(createChecklistItem).toHaveBeenLastCalledWith(
      'Old step',
      'plan-1',
      'longterm',
      { type: 'task', parentId: 'P0' }
    );
    expect(within(card('P0')!).getByText('Old step')).toBeTruthy();
    expect(composerState()).toBe('filling');
    expect(traced()).toEqual(['new1']);
    expect(fillingInput('Launch')!.value).toBe('half');
  });

  it('should clear the trace when the view switches to the list', async () => {
    const user = userEvent.setup();
    await renderTodo();
    await startFilling(user);

    await user.click(viewButton('Checklist'));
    await user.click(viewButton('Milestones'));

    expect(composerState()).toBe('resting');
    expect(traced()).toEqual([]);
  });
});
