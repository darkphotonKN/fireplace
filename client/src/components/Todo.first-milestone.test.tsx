import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

/**
 * An empty plan invites its first milestone (FS-FSFEC R46, R47). A plan with
 * nothing in it shows "Start your first milestone" in the grid instead of a
 * lone resting bar under nothing; activating it opens the composer, the one
 * way in. A plan whose items the filter hides keeps its filter-empty text and
 * the composer below it, and the list view's empty state is untouched.
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

const invitation = () =>
  screen.queryByRole('button', { name: 'Start your first milestone' });
const composer = () =>
  document.querySelector('[data-milestone-composer]') as HTMLElement | null;
const composerState = () =>
  composer()?.getAttribute('data-milestone-composer') ?? null;
// The resting bar, as distinct from the invitation: a button inside the bar.
const restingBar = () =>
  composerState() === 'resting'
    ? within(composer()!).queryByRole('button')
    : null;
const namingInput = () =>
  composer() &&
  (within(composer()!).queryByRole('textbox', {
    name: /^(Name a new milestone|New note)$/,
  }) as HTMLInputElement | null);
const card = (id: string) =>
  document.querySelector(`[data-checklist-card="${id}"]`) as HTMLElement | null;
const filterTab = (label: string) =>
  screen
    .getAllByRole('button', { name: label })
    .find((b) => !b.closest('[role="group"]'))!;

async function renderTodo(items: unknown[] = []) {
  fetchChecklist.mockImplementation(async () => items);
  const Todo = (await import('./Todo')).default;
  const view = render(<Todo fixedTaskType="longterm" enableTypeFilter />);
  // Settled once the filter tabs are drawn, whatever the list holds.
  await screen.findByRole('button', { name: 'Notes' });
  return view;
}

describe('Todo first-milestone invitation', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
    nextId = 0;
    createChecklistItem.mockImplementation(echoCreate);
  });

  it('should invite the first milestone in an empty grid, with no resting bar', async () => {
    await renderTodo();

    expect(invitation()).toBeTruthy();
    expect(screen.queryByText('No items yet. Add one below!')).toBeNull();
    expect(restingBar()).toBeNull();
    expect(screen.queryByRole('button', { name: 'New milestone' })).toBeNull();
  });

  it('should take no focus on load', async () => {
    await renderTodo();
    expect(document.activeElement).not.toBe(invitation());
    expect(document.activeElement?.tagName).not.toBe('INPUT');
  });

  it('should open naming with focus on click, Enter or Space', async () => {
    const user = userEvent.setup();
    await renderTodo();

    await user.click(invitation()!);
    expect(composerState()).toBe('naming');
    expect(document.activeElement).toBe(namingInput());
    expect(namingInput()!.placeholder).toBe('Name a new milestone…');

    for (const key of ['{Enter}', ' ']) {
      await user.keyboard('{Escape}');
      // Back to the invitation, which takes the focus Esc hands back.
      expect(composerState()).toBe('resting');
      expect(restingBar()).toBeNull();
      expect(document.activeElement).toBe(invitation());
      await user.keyboard(key);
      expect(composerState()).toBe('naming');
      expect(document.activeElement).toBe(namingInput());
    }
  });

  it('should stand while naming, and take focus back to the input when activated again', async () => {
    const user = userEvent.setup();
    await renderTodo();

    await user.click(invitation()!);
    await user.type(namingInput()!, 'Lau');
    expect(invitation()).toBeTruthy();

    await user.click(invitation()!);
    expect(composerState()).toBe('naming');
    expect(document.activeElement).toBe(namingInput());
    expect(namingInput()!.value).toBe('Lau');
  });

  it('should give way to the grid once the first milestone is named, the composer filling', async () => {
    const user = userEvent.setup();
    await renderTodo();

    await user.click(invitation()!);
    await user.type(namingInput()!, 'Launch{Enter}');

    await screen.findByRole('textbox', { name: 'Add a step to Launch' });
    expect(invitation()).toBeNull();
    expect(card('new1')).toBeTruthy();
    expect(composerState()).toBe('filling');
    expect(createChecklistItem).toHaveBeenCalledTimes(1);
    expect(createChecklistItem.mock.calls[0].slice(0, 3)).toEqual([
      'Launch',
      'plan-1',
      'longterm',
    ]);

    // Esc rests on the bar now the plan has a milestone.
    await user.keyboard('{Escape}');
    expect(restingBar()?.textContent).toContain('New milestone');
  });

  it('should invite under the Checklist filter too', async () => {
    const user = userEvent.setup();
    await renderTodo();
    await user.click(filterTab('Checklist'));

    expect(invitation()).toBeTruthy();
    expect(screen.queryByText('No tasks yet.')).toBeNull();
    expect(restingBar()).toBeNull();
  });

  it('should keep "No notes yet." with the note composer below on an empty plan under Notes', async () => {
    const user = userEvent.setup();
    await renderTodo();
    await user.click(filterTab('Notes'));

    expect(invitation()).toBeNull();
    expect(screen.getByText('No notes yet.')).toBeTruthy();
    expect(restingBar()?.textContent).toContain('New note');
  });

  describe('items the filter hides (R47)', () => {
    it('should keep "No notes yet." with the composer below when only tasks exist', async () => {
      const user = userEvent.setup();
      await renderTodo([item('P0')]);
      await user.click(filterTab('Notes'));

      expect(invitation()).toBeNull();
      expect(screen.getByText('No notes yet.')).toBeTruthy();
      expect(restingBar()?.textContent).toContain('New note');
    });

    it('should keep "No tasks yet." with the composer below when only notes exist', async () => {
      const user = userEvent.setup();
      await renderTodo([item('N0', { type: 'note' })]);
      await user.click(filterTab('Checklist'));

      expect(invitation()).toBeNull();
      expect(screen.getByText('No tasks yet.')).toBeTruthy();
      expect(restingBar()?.textContent).toContain('New milestone');
    });
  });

  it('should leave the list view its empty text and add bar', async () => {
    window.localStorage.setItem('checklistView', 'list');
    await renderTodo();

    expect(invitation()).toBeNull();
    expect(composer()).toBeNull();
    expect(screen.getByText('No items yet. Add one below!')).toBeTruthy();
    expect(screen.getByRole('textbox', { name: 'New task' })).toBeTruthy();
  });

  it('should invite again when the view switches back to an empty grid', async () => {
    const user = userEvent.setup();
    window.localStorage.setItem('checklistView', 'list');
    await renderTodo();

    await user.click(
      within(screen.getByRole('group', { name: 'View' })).getByRole('button', {
        name: 'Milestones',
      })
    );
    expect(invitation()).toBeTruthy();
    expect(restingBar()).toBeNull();
  });
});
