import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

/**
 * The grid composer when the plan moves under it (FS-FSFEC R37–R39, R41):
 * paging away while filling, the traced milestone going away, notes, and a
 * switch to the list view and back.
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
  deleteChecklistItem: vi.fn(async () => undefined),
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
  getPlan: vi.fn(async () => ({ id: 'plan-1', dailyReset: false })),
  toggleDailyReset: vi.fn(),
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
const composer = () =>
  document.querySelector('[data-milestone-composer]') as HTMLElement;
const composerState = () =>
  composer()?.getAttribute('data-milestone-composer') ?? null;
// Scoped to the composer: "Add a step to ‹title›" is also a card's own input.
const restingButton = () => within(composer()).getByRole('button');
const namingInput = () =>
  within(composer()).queryByRole('textbox', {
    name: /^(Name a new milestone|New note)$/,
  }) as HTMLInputElement | null;
const fillingInput = (title: string) =>
  within(composer()).queryByRole('textbox', {
    name: `Add a step to ${title}`,
  }) as HTMLInputElement | null;
const typeIcon = () =>
  within(composer()).queryByRole('button', { name: /^Switch to adding a/ });
const counter = () => screen.getByTestId('page-counter');
const viewButton = (label: string) =>
  within(screen.getByRole('group', { name: 'View' })).getByRole('button', {
    name: label,
  });
// The type filter's tabs, not the view toggle's same-named "Checklist".
const filterTab = (label: string) =>
  screen
    .getAllByRole('button', { name: label })
    .find((b) => !b.closest('[role="group"]'))!;
const listInput = () =>
  screen.getByRole('textbox', { name: /^New (task|note)/ }) as HTMLInputElement;
const cardAction = async (
  user: ReturnType<typeof userEvent.setup>,
  title: string,
  action: string
) => {
  await user.click(screen.getByRole('button', { name: `Actions for ${title}` }));
  await user.click(screen.getByRole('menuitem', { name: action }));
};
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

async function startFilling(
  user: ReturnType<typeof userEvent.setup>,
  title = 'Launch'
) {
  await user.click(restingButton());
  await user.type(namingInput()!, `${title}{Enter}`);
  await screen.findByRole('textbox', { name: `Add a step to ${title}` });
}

describe('Todo new-milestone composer, when the plan moves under it', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
    nextId = 0;
    createChecklistItem.mockImplementation(echoCreate);
  });

  describe('paged away while filling (R37)', () => {
    it('should add the step to the traced milestone and jump back to its page', async () => {
      const user = userEvent.setup();
      await renderTodo(topLevel(7));
      await startFilling(user);
      expect(counter().textContent).toBe('2 of 2');

      await user.click(screen.getByRole('button', { name: 'Previous page' }));
      expect(counter().textContent).toBe('1 of 2');
      expect(composerState()).toBe('filling');

      await user.type(fillingInput('Launch')!, 'Write copy{Enter}');

      expect(createChecklistItem).toHaveBeenLastCalledWith(
        'Write copy',
        'plan-1',
        'longterm',
        { type: 'task', parentId: 'new1' }
      );
      expect(counter().textContent).toBe('2 of 2');
      expect(traced()).toEqual(['new1']);
      expect(within(card('new1')!).getByText('Write copy')).toBeTruthy();
      expect(document.activeElement).toBe(fillingInput('Launch'));
    });
  });

  describe('traced milestone lost (R38)', () => {
    it.each([
      ['archived', 'Archive'],
      ['deleted', 'Delete'],
      ['converted to a note', 'Make a note'],
    ])('should drop to naming with the text kept when it is %s', async (_, action) => {
      const user = userEvent.setup();
      await renderTodo();
      await startFilling(user);
      await user.type(fillingInput('Launch')!, 'half typed');

      await cardAction(user, 'Launch', action);

      expect(composerState()).toBe('naming');
      expect(namingInput()!.value).toBe('half typed');
      expect(traced()).toEqual([]);
      // Lost from under it, not moved by anyone: the cursor isn't pulled down.
      expect(composer().contains(document.activeElement)).toBe(false);
    });

    it('should drop to naming when the Notes filter hides it, and stay there once it is back', async () => {
      const user = userEvent.setup();
      await renderTodo([item('N0', { type: 'note' }), item('P0')]);
      await startFilling(user);
      await user.type(fillingInput('Launch')!, 'half typed');

      await user.click(filterTab('Notes'));
      expect(composerState()).toBe('naming');
      expect(namingInput()!.value).toBe('half typed');

      // Forgotten, not held: the milestone coming back doesn't re-trace it.
      await user.click(filterTab('All'));
      expect(composerState()).toBe('naming');
      expect(traced()).toEqual([]);
    });

    it('should name the milestone by its current title after a rename', async () => {
      const user = userEvent.setup();
      await renderTodo();
      await startFilling(user);

      await user.click(within(card('new1')!).getByText('Launch'));
      const rename = screen.getByRole('textbox', { name: 'Rename Launch' });
      await user.clear(rename);
      await user.type(rename, 'Go live{Enter}');

      expect(composerState()).toBe('filling');
      expect(fillingInput('Go live')).not.toBeNull();
      expect(composer().textContent).toContain('↳ steps for Go live');
    });
  });

  describe('notes (R39)', () => {
    it('should rest as "+ New note" under the Notes filter and create top-level notes, staying in naming', async () => {
      const user = userEvent.setup();
      await renderTodo([item('N0', { type: 'note' }), item('P0')]);
      await user.click(filterTab('Notes'));

      expect(restingButton().textContent).toContain('New note');
      await user.click(restingButton());
      expect(namingInput()!.placeholder).toBe('Jot down a note…');

      await user.type(namingInput()!, 'An idea{Enter}');

      expect(createChecklistItem).toHaveBeenLastCalledWith(
        'An idea',
        'plan-1',
        'longterm',
        { type: 'note' }
      );
      expect(card('new1')).not.toBeNull();
      expect(composerState()).toBe('naming');
      expect(namingInput()!.value).toBe('');
      expect(document.activeElement).toBe(namingInput());
      expect(traced()).toEqual([]);
    });

    it('should create notes with the icon on note under All, and milestones again once it is back on task', async () => {
      const user = userEvent.setup();
      await renderTodo();
      await user.click(restingButton());

      await user.click(typeIcon()!);
      expect(namingInput()!.placeholder).toBe('Jot down a note…');
      await user.type(namingInput()!, 'An idea{Enter}');
      expect(createChecklistItem).toHaveBeenLastCalledWith(
        'An idea',
        'plan-1',
        'longterm',
        { type: 'note' }
      );
      expect(composerState()).toBe('naming');

      await user.keyboard('{Escape}');
      expect(restingButton().textContent).toContain('New note');
      await user.click(restingButton());

      await user.click(typeIcon()!);
      expect(namingInput()!.placeholder).toBe('Name a new milestone…');
      await user.type(namingInput()!, 'Launch{Enter}');

      expect(await screen.findByRole('textbox', { name: 'Add a step to Launch' })).toBeTruthy();
      expect(composerState()).toBe('filling');
    });

    it('should let the icon override the Notes filter, a hidden milestone leaving it in naming', async () => {
      const user = userEvent.setup();
      await renderTodo([item('N0', { type: 'note' }), item('P0')]);
      await user.click(filterTab('Notes'));
      await user.click(restingButton());

      await user.click(typeIcon()!);
      // Re-choosing the tab you are on doesn't set the type again (R21).
      await user.click(filterTab('Notes'));
      expect(namingInput()!.placeholder).toBe('Name a new milestone…');

      await user.type(namingInput()!, 'Launch{Enter}');

      expect(createChecklistItem).toHaveBeenLastCalledWith(
        'Launch',
        'plan-1',
        'longterm',
        { type: 'task' }
      );
      expect(composerState()).toBe('naming');
      expect(traced()).toEqual([]);
      expect(namingInput()!.value).toBe('');
    });

    it('should offer the type icon in naming only', async () => {
      const user = userEvent.setup();
      await renderTodo();

      expect(typeIcon()).toBeNull();
      await user.click(restingButton());
      expect(typeIcon()).not.toBeNull();
      await user.type(namingInput()!, 'Launch{Enter}');
      await screen.findByRole('textbox', { name: 'Add a step to Launch' });
      expect(typeIcon()).toBeNull();
    });
  });

  describe('view switch (R41)', () => {
    it('should leave the list bar at top level with the text kept, nothing nested', async () => {
      const user = userEvent.setup();
      window.localStorage.setItem('checklistView', 'list');
      await renderTodo();
      // Nested in the list before ever going to the grid.
      await user.click(listInput());
      await user.keyboard('{Tab}');
      expect(listInput().getAttribute('aria-label')).toContain('under');

      await user.click(viewButton('Milestones'));
      await startFilling(user);
      await user.type(fillingInput('Launch')!, 'half typed');
      await user.click(viewButton('Checklist'));

      expect(listInput().value).toBe('half typed');
      expect(listInput().getAttribute('aria-label')).toBe('New task');

      await user.type(listInput(), '{Enter}');
      expect(createChecklistItem).toHaveBeenLastCalledWith(
        'half typed',
        'plan-1',
        'longterm',
        { type: 'task', parentId: undefined }
      );
    });

    it('should leave a nest alone when the view already showing is chosen again', async () => {
      const user = userEvent.setup();
      window.localStorage.setItem('checklistView', 'list');
      await renderTodo();
      await user.click(listInput());
      await user.keyboard('{Tab}');
      const nested = listInput().getAttribute('aria-label');
      expect(nested).toContain('under');

      // Already in list view: nothing changes, so nothing is reset.
      await user.click(viewButton('Checklist'));

      expect(listInput().getAttribute('aria-label')).toBe(nested);
    });

    it('should come back to naming when text remains, resting when it does not', async () => {
      const user = userEvent.setup();
      await renderTodo();
      await startFilling(user);
      await user.type(fillingInput('Launch')!, 'half typed');

      await user.click(viewButton('Checklist'));
      await user.click(viewButton('Milestones'));

      expect(composerState()).toBe('naming');
      expect(namingInput()!.value).toBe('half typed');
      expect(traced()).toEqual([]);

      await user.click(viewButton('Checklist'));
      await user.clear(listInput());
      await user.click(viewButton('Milestones'));

      expect(composerState()).toBe('resting');
    });
  });
});
