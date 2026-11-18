import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, within } from '@testing-library/react';

/**
 * Note rows in the checklist reading (FS-PGG2N R11, R48): the card text,
 * upright in a paper chip and clamped to two lines, where a slanted grey line
 * used to be. Tasks keep their look.
 */

const fetchChecklist = vi.fn();
const fetchArchivedChecklist = vi.fn();

const push = vi.fn();
vi.mock('next/navigation', () => ({
  useParams: () => ({ planId: 'plan-1' }),
  useRouter: () => ({ push, replace: vi.fn(), refresh: vi.fn() }),
}));

vi.mock('@/components/ui/use-toast', () => ({
  useToast: () => ({ toast: vi.fn() }),
}));

vi.mock('@/services/api', () => ({
  fetchChecklist: (...a: unknown[]) => fetchChecklist(...a),
  fetchArchivedChecklist: (...a: unknown[]) => fetchArchivedChecklist(...a),
  createChecklistItem: vi.fn(),
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
}));

vi.mock('@/api/plans', () => ({
  getPlan: vi.fn(async () => ({ id: 'plan-1', dailyReset: false })),
}));

const item = (id: string, description: string, extra: Record<string, unknown> = {}) => ({
  id,
  description,
  done: false,
  scope: 'longterm',
  type: 'task',
  parentId: null,
  planId: 'plan-1',
  ...extra,
});

const ITEMS = [
  item('T0', 'a task'),
  item('N0', 'the long body\nwith lines', { type: 'note', summary: 'the gist' }),
  item('N1', 'first\nsecond', { type: 'note' }),
];

async function renderList() {
  fetchChecklist.mockImplementation(async () => ITEMS);
  const Todo = (await import('./Todo')).default;
  render(<Todo fixedTaskType="longterm" />);
  await screen.findByText('a task');
  fireEvent.click(
    within(screen.getByRole('group', { name: 'View' })).getByRole('button', {
      name: 'Checklist',
    })
  );
}

const noteText = (id: string) =>
  document.querySelector(`[data-note-text="${id}"]`) as HTMLElement;

beforeEach(() => {
  vi.clearAllMocks();
  window.localStorage.clear();
  fetchArchivedChecklist.mockResolvedValue([]);
});

describe('Todo — note rows on paper (FS-PGG2N R11, R48)', () => {
  it('should show a note row its summary rather than its body', async () => {
    await renderList();
    expect(noteText('N0').textContent).toBe('the gist');
    expect(screen.queryByText(/the long body/)).toBeNull();
  });

  it('should set the card text upright in a paper chip, clamped to 2 lines', async () => {
    await renderList();
    const chip = noteText('N1');
    expect(chip.textContent).toBe('first\nsecond');
    for (const cls of ['bg-note-paper', 'rounded-md', 'px-2', 'py-0.5', 'line-clamp-2', 'whitespace-pre-line', 'break-words']) {
      expect(chip.className).toContain(cls);
    }
    expect(document.querySelector('.italic:not(input)')).toBeNull();
  });

  it("should name a note row's menu by its first line", async () => {
    await renderList();
    expect(screen.getByRole('button', { name: 'Actions for first' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Actions for the gist' })).toBeInTheDocument();
  });

  it("should name a note row's reorder grip by its first line (R11, G7)", async () => {
    await renderList();
    expect(screen.getByRole('button', { name: 'Reorder first' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Reorder the gist' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Reorder the long body/ })).toBeNull();
    expect(screen.getByRole('button', { name: 'Reorder a task' })).toBeInTheDocument();
  });

  it('should leave a task row as it was', async () => {
    await renderList();
    const task = screen.getByText('a task');
    expect(task.className).not.toContain('bg-note-paper');
    expect(task.tagName).toBe('LABEL');
  });
});

describe('Todo — the delete confirmation names a note by its first line (R11)', () => {
  it('should ask about the first line, not the whole body', async () => {
    fetchArchivedChecklist.mockResolvedValue([
      item('A0', 'Groceries\nmilk\neggs', { type: 'note', archived: true }),
    ]);
    await renderList();
    fireEvent.click(screen.getByTitle('Settings'));
    fireEvent.click(screen.getByRole('button', { name: /view archived tasks/i }));
    await screen.findByText(/Groceries/);
    fireEvent.click(screen.getByTitle('Delete task permanently'));
    expect(
      screen.getByText(/permanently delete "Groceries"\?/)
    ).toBeInTheDocument();
  });
});

describe('Todo — ways into a note (FS-PGG2N R12–R15)', () => {
  it("should make a live list note row's text a link to its page", async () => {
    await renderList();
    const link = screen.getByRole('link', { name: /^first\s+second$/ });
    expect(link.getAttribute('href')).toBe('/plan/plan-1/n/N1');
    expect(screen.getByRole('link', { name: 'the gist' }).getAttribute('href')).toBe(
      '/plan/plan-1/n/N0'
    );
  });

  it('should leave a task row as it was, not a link', async () => {
    await renderList();
    expect(screen.getByText('a task').closest('a')).toBeNull();
  });

  it("should open a note's page in writing mode from its menu Rename", async () => {
    await renderList();
    fireEvent.click(screen.getByRole('button', { name: 'Actions for first' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Rename' }));
    expect(push).toHaveBeenCalledWith('/plan/plan-1/n/N1?edit=1');
    expect(screen.queryByRole('textbox', { name: /rename/i })).toBeNull();
  });

  it("should keep a task's menu Rename inline", async () => {
    await renderList();
    fireEvent.click(screen.getByRole('button', { name: 'Actions for a task' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Rename' }));
    expect(push).not.toHaveBeenCalled();
    expect(screen.getByDisplayValue('a task')).toBeInTheDocument();
  });

  it('should make a note card in the milestone reading a link to its page', async () => {
    fetchChecklist.mockImplementation(async () => ITEMS);
    const Todo = (await import('./Todo')).default;
    render(<Todo fixedTaskType="longterm" />);
    const link = await screen.findByRole('link', { name: 'Open note: first' });
    expect(link.getAttribute('href')).toBe('/plan/plan-1/n/N1');
  });

  it('should not link archived notes in the archived list (R15)', async () => {
    fetchArchivedChecklist.mockResolvedValue([
      item('A0', 'Groceries\nmilk', { type: 'note', archived: true }),
    ]);
    await renderList();
    fireEvent.click(screen.getByTitle('Settings'));
    fireEvent.click(screen.getByRole('button', { name: /view archived tasks/i }));
    const text = await screen.findByText(/Groceries/);
    expect(text.closest('a')).toBeNull();
  });
});
