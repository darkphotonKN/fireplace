import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import ChecklistGrid from './ChecklistGrid';
import type { ChecklistItem } from '@/services/api';

/**
 * The blocks view: one card per top-level item, its steps inside it, and a
 * meter over the task children. A prototype, so what is pinned here is what
 * the card is FOR — progress at a glance and a step composer that needs no
 * Tab gesture — not its styling.
 */

const mk = (
  id: string,
  description: string,
  extra: Partial<ChecklistItem> = {}
) =>
  ({
    id,
    description,
    done: false,
    scope: 'longterm',
    type: 'task',
    parentId: null,
    planId: 'plan-1',
    ...extra,
  }) as ChecklistItem;

const GROUPS = [
  {
    item: mk('p1', 'Learn Go'),
    children: [
      mk('c1', 'goroutines', { done: true, parentId: 'p1' }),
      mk('c2', 'channels', { parentId: 'p1' }),
      mk('c3', 'a thought', { type: 'note', parentId: 'p1' }),
    ],
  },
  { item: mk('p2', 'Book flights'), children: [] },
  {
    item: mk('p3', 'Reading list', { type: 'note' }),
    children: [
      mk('c4', 'DDIA', { type: 'note', parentId: 'p3' }),
      mk('c5', 'APOSD', { type: 'note', parentId: 'p3' }),
    ],
  },
];

const handlers = () => ({
  onToggleCollapsed: vi.fn(),
  onToggleDone: vi.fn(),
  onRename: vi.fn(),
  onDelete: vi.fn(),
  onAddChild: vi.fn(async () => {}),
  onToggleType: vi.fn(),
  onArchive: vi.fn(),
  onSetDates: vi.fn(),
  onOutdent: vi.fn(),
  // The grid is handed its move actions rather than working them out: the set
  // it can see is one page of one filter, and what counts as an end of the set
  // belongs to whoever holds every item.
  moveProps: vi.fn(() => ({})),
});

const card = (id: string) =>
  document.querySelector(`[data-checklist-card="${id}"]`) as HTMLElement;

function renderGrid(
  props: Partial<React.ComponentProps<typeof ChecklistGrid>> = {}
) {
  const h = handlers();
  const view = render(
    <ChecklistGrid
      groups={GROUPS}
      collapsedIds={new Set()}
      {...h}
      {...props}
    />
  );
  return { ...view, ...h };
}

describe('ChecklistGrid', () => {
  beforeEach(() => vi.clearAllMocks());

  it('should meter the task children only, leaving notes out of the count', () => {
    renderGrid();
    // One of two tasks done; the note child is not progress.
    expect(within(card('p1')).getByText('1/2')).toBeTruthy();
    expect(
      card('p1').querySelector('[data-progress-fill]')!.getAttribute('style')
    ).toContain('50%');
  });

  it('should say what a notes-only block holds, having no progress to show', () => {
    renderGrid();
    expect(within(card('p3')).getByText('2 notes')).toBeTruthy();
    expect(card('p3').querySelector('[data-progress-fill]')).toBeNull();
  });

  it('should give a block with no steps no meter and nothing to fold', () => {
    renderGrid();
    expect(card('p2').querySelector('[data-progress-fill]')).toBeNull();
    expect(
      within(card('p2')).queryByRole('button', { name: /^(Collapse|Expand)/ })
    ).toBeNull();
  });

  it('should fold a block to its head and meter when collapsed', () => {
    renderGrid({ collapsedIds: new Set(['p1']) });
    expect(screen.queryByText('goroutines')).toBeNull();
    // The meter stays: a folded block still says how far along it is.
    expect(within(card('p1')).getByText('1/2')).toBeTruthy();
  });

  it('should ask its parent to fold when the chevron is used', () => {
    const { onToggleCollapsed } = renderGrid();
    fireEvent.click(screen.getByRole('button', { name: 'Collapse Learn Go' }));
    expect(onToggleCollapsed).toHaveBeenCalledWith('p1');
  });

  it('should add a step into the block it was typed in, no Tab needed', async () => {
    const { onAddChild } = renderGrid();
    fireEvent.click(within(card('p2')).getByRole('button', { name: /add step/i }));
    const input = within(card('p2')).getByRole('textbox', {
      name: 'Add a step to Book flights',
    });
    fireEvent.change(input, { target: { value: 'check passport' } });
    fireEvent.keyDown(input, { key: 'Enter' });

    await waitFor(() =>
      expect(onAddChild).toHaveBeenCalledWith('p2', 'check passport')
    );
    // Cleared but still open, so steps can be entered back to back.
    await waitFor(() => expect((input as HTMLInputElement).value).toBe(''));
  });

  it('should offer no step composer on a note block', () => {
    renderGrid();
    expect(
      within(card('p3')).queryByRole('button', { name: /add step/i })
    ).toBeNull();
  });

  it('should rename an item in place on Enter', () => {
    const { onRename } = renderGrid();
    fireEvent.click(screen.getByText('channels'));
    const input = screen.getByRole('textbox', { name: 'Rename channels' });
    fireEvent.change(input, { target: { value: 'channels, buffered' } });
    fireEvent.keyDown(input, { key: 'Enter' });
    expect(onRename).toHaveBeenCalledWith('c2', 'channels, buffered');
  });

  it('should drop a rename on Escape', () => {
    const { onRename } = renderGrid();
    fireEvent.click(screen.getByText('channels'));
    const input = screen.getByRole('textbox', { name: 'Rename channels' });
    fireEvent.change(input, { target: { value: 'nope' } });
    fireEvent.keyDown(input, { key: 'Escape' });
    expect(onRename).not.toHaveBeenCalled();
  });

  it('should check a step off through the same handler the list uses', () => {
    const { onToggleDone } = renderGrid();
    fireEvent.click(screen.getByLabelText('Mark channels done'));
    expect(onToggleDone).toHaveBeenCalledWith('c2');
  });
});
