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
  // Same reason: whether an item has anywhere to be dragged is a question
  // about its whole sibling set, which the grid never sees.
  canDrag: vi.fn(() => true),
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

/**
 * The completion nudge (FS-FSFEC R10–R14): a milestone whose task steps are
 * all done says so and offers to close itself. It is only ever an offer; the
 * milestone's `done` is set by a person, through the checkbox's own handler.
 */
describe('ChecklistGrid completion nudge', () => {
  beforeEach(() => vi.clearAllMocks());

  const milestone = (
    done: boolean[],
    extra: Partial<ChecklistItem> = {},
    notes = 0
  ) => ({
    item: mk('m1', 'Ship v1', extra),
    children: [
      ...done.map((d, i) =>
        mk(`s${i}`, `step ${i}`, { done: d, parentId: 'm1' })
      ),
      ...Array.from({ length: notes }, (_, i) =>
        mk(`n${i}`, `note ${i}`, { type: 'note', parentId: 'm1' })
      ),
    ],
  });

  const nudge = () =>
    card('m1').querySelector('[data-completion-nudge]') as HTMLElement | null;

  it('should offer to close a milestone whose task steps are all done', () => {
    const { onToggleDone } = renderGrid({
      groups: [milestone([true, true], {}, 1)],
    });

    // Full meter, a glow, and the offer.
    expect(within(card('m1')).getByText('2/2')).toBeTruthy();
    expect(
      card('m1').querySelector('[data-progress-fill]')!.getAttribute('style')
    ).toContain('100%');
    expect(card('m1').hasAttribute('data-complete')).toBe(true);
    expect(nudge()).not.toBeNull();

    fireEvent.click(
      within(nudge()!).getByRole('button', { name: 'Close Ship v1' })
    );
    // The checkbox's own path: optimistic, reverting on failure upstream.
    expect(onToggleDone).toHaveBeenCalledTimes(1);
    expect(onToggleDone).toHaveBeenCalledWith('m1');
  });

  it('should hold a dismissal until the milestone\'s progress changes', () => {
    const { rerender, onToggleDone, ...h } = renderGrid({
      groups: [milestone([true, true])],
    });
    const draw = (groups: ReturnType<typeof milestone>[]) =>
      rerender(
        <ChecklistGrid
          groups={groups}
          collapsedIds={new Set()}
          {...h}
          onToggleDone={onToggleDone}
        />
      );

    fireEvent.click(
      within(nudge()!).getByRole('button', { name: 'Not now, Ship v1' })
    );
    expect(nudge()).toBeNull();
    // Still complete, just not nagging.
    expect(card('m1').hasAttribute('data-complete')).toBe(true);

    // Closed and reopened at the same progress: still dismissed.
    draw([milestone([true, true], { done: true })]);
    draw([milestone([true, true])]);
    expect(nudge()).toBeNull();

    // Untick, re-tick: the progress moved, so the offer comes back.
    draw([milestone([true, false])]);
    expect(nudge()).toBeNull();
    draw([milestone([true, true])]);
    expect(nudge()).not.toBeNull();
    expect(onToggleDone).not.toHaveBeenCalled();
  });

  it('should show a dismissed nudge again after a remount: dismissal is memory only', () => {
    const first = renderGrid({ groups: [milestone([true, true])] });
    fireEvent.click(
      within(nudge()!).getByRole('button', { name: 'Not now, Ship v1' })
    );
    expect(nudge()).toBeNull();
    first.unmount();

    renderGrid({ groups: [milestone([true, true])] });
    expect(nudge()).not.toBeNull();
  });

  it('should never nudge a milestone with no task steps, which keeps its manual tick', () => {
    // No children at all, then only note children (R7).
    for (const groups of [[milestone([])], [milestone([], {}, 2)]]) {
      const view = renderGrid({ groups });
      expect(nudge()).toBeNull();
      expect(card('m1').hasAttribute('data-complete')).toBe(false);
      expect(screen.getByLabelText('Mark Ship v1 done')).toBeTruthy();
      view.unmount();
    }
  });

  it('should never nudge a note card, nor give it a checkbox', () => {
    renderGrid({
      groups: [
        {
          item: mk('m1', 'Ideas', { type: 'note' }),
          children: [mk('s0', 'done task', { done: true, parentId: 'm1' })],
        },
      ],
    });
    expect(nudge()).toBeNull();
    expect(screen.queryByLabelText('Mark Ideas done')).toBeNull();
  });

  it('should end the complete state when a step is added, and never close on its own', () => {
    const { rerender, onToggleDone, ...h } = renderGrid({
      groups: [milestone([true, true])],
    });
    expect(nudge()).not.toBeNull();

    rerender(
      <ChecklistGrid
        groups={[milestone([true, true, false])]}
        collapsedIds={new Set()}
        {...h}
        onToggleDone={onToggleDone}
      />
    );
    expect(nudge()).toBeNull();
    expect(card('m1').hasAttribute('data-complete')).toBe(false);
    expect(within(card('m1')).getByText('2/3')).toBeTruthy();
    expect(onToggleDone).not.toHaveBeenCalled();
  });

  it('should keep a closed milestone in place, faded, with no offer left to make', () => {
    renderGrid({
      groups: [
        { item: mk('a', 'First'), children: [] },
        milestone([true, true], { done: true }),
        { item: mk('z', 'Last'), children: [] },
      ],
    });
    const order = Array.from(
      document.querySelectorAll('[data-checklist-card]')
    ).map((el) => el.getAttribute('data-checklist-card'));
    expect(order).toEqual(['a', 'm1', 'z']);
    expect(card('m1').className).toContain('opacity-60');
    expect(nudge()).toBeNull();
    // Still hand-settable: reopening goes through the same checkbox (R9).
    expect(
      (screen.getByLabelText('Mark Ship v1 done') as HTMLElement).getAttribute(
        'data-state'
      )
    ).toBe('checked');
  });

  it('should still nudge a folded milestone: the offer is about the card, not its steps', () => {
    renderGrid({
      groups: [milestone([true, true])],
      collapsedIds: new Set(['m1']),
    });
    expect(nudge()).not.toBeNull();
  });
});

describe('ChecklistGrid traced milestone (FS-FSFEC R33)', () => {
  const traced = () =>
    Array.from(document.querySelectorAll('[data-traced]')).map((el) =>
      el.getAttribute('data-checklist-card')
    );

  it('should mark only the traced card, with a ring and a star', () => {
    renderGrid({ tracedId: 'p2' });

    expect(traced()).toEqual(['p2']);
    expect(card('p2').className).toContain('ring-2');
    expect(card('p2').querySelector('[data-traced-mark]')?.textContent).toBe('✦');
    expect(card('p1').querySelector('[data-traced-mark]')).toBeNull();
  });

  it('should mark nothing when nothing is traced', () => {
    renderGrid();

    expect(traced()).toEqual([]);
    expect(document.querySelector('[data-traced-mark]')).toBeNull();
  });
});
