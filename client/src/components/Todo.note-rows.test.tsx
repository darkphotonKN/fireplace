import { describe, it, expect } from 'vitest';
import { screen, fireEvent } from '@testing-library/react';
import { api, item as baseItem, renderTodo, router, viewButton } from '@/test/todoHarness';

/**
 * Note rows in the checklist reading (FS-PGG2N R11, R48): the card text,
 * upright in a paper chip and clamped to two lines, where a slanted grey line
 * used to be. Tasks keep their look.
 */

const { fetchArchivedChecklist } = api;
const { push } = router;
const item = (id: string, description: string, extra: Record<string, unknown> = {}) =>
  baseItem(id, { description, ...extra });

const ITEMS = [
  item('T0', 'a task'),
  item('N0', 'the long body\nwith lines', { type: 'note', summary: 'the gist' }),
  item('N1', 'first\nsecond', { type: 'note' }),
];

async function renderList() {
  await renderTodo(ITEMS, { props: { enableTypeFilter: false } });
  fireEvent.click(viewButton('Checklist'));
}

const noteText = (id: string) =>
  document.querySelector(`[data-note-text="${id}"]`) as HTMLElement;

describe('Todo — note rows on paper (FS-PGG2N R11, R48)', () => {
  it('should show a note row its summary rather than its body', async () => {
    await renderList();
    expect(noteText('N0').textContent).toBe('the gist');
    expect(screen.queryByText(/the long body/)).toBeNull();
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

  it('should leave a task row as it was, a label and not a link', async () => {
    await renderList();
    const task = screen.getByText('a task');
    expect(task.tagName).toBe('LABEL');
    expect(task.closest('a')).toBeNull();
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
    await renderTodo(ITEMS, { props: { enableTypeFilter: false } });
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
