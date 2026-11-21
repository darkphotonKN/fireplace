import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import type { ChecklistItem } from '@/services/api';

import ChecklistGrid from './ChecklistGrid';

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

  it('should mark only the traced card, with a star', () => {
    renderGrid({ tracedId: 'p2' });

    expect(traced()).toEqual(['p2']);
    expect(card('p2').querySelector('[data-traced-mark]')?.textContent).toBe('✦');
    expect(card('p1').querySelector('[data-traced-mark]')).toBeNull();
  });

  it('should mark nothing when nothing is traced', () => {
    renderGrid();

    expect(traced()).toEqual([]);
    expect(document.querySelector('[data-traced-mark]')).toBeNull();
  });
});

describe('ChecklistGrid the way into a milestone (FS-043WP R5–R7)', () => {
  beforeEach(() => vi.clearAllMocks());

  const href = (id: string) => `/plan/plan-1/m/${id}`;

  it('should give every milestone card a real link to its page, named for it', () => {
    renderGrid({ milestoneHref: href });
    expect(
      within(card('p1')).getByRole('link', { name: 'Open Learn Go' })
    ).toHaveAttribute('href', '/plan/plan-1/m/p1');
    expect(
      within(card('p2')).getByRole('link', { name: 'Open Book flights' })
    ).toHaveAttribute('href', '/plan/plan-1/m/p2');
  });

  it('should give a closed milestone its link too: it is still a milestone', () => {
    renderGrid({
      milestoneHref: href,
      groups: [{ item: mk('p4', 'Done thing', { done: true }), children: [] }],
    });
    expect(
      within(card('p4')).getByRole('link', { name: 'Open Done thing' })
    ).toBeInTheDocument();
  });

  it('should give note cards and steps no milestone link', () => {
    renderGrid({ milestoneHref: href });
    expect(within(card('p3')).queryByRole('link')).toBeNull();
    // p1's only link is its own; its steps carry none.
    expect(within(card('p1')).getAllByRole('link')).toHaveLength(1);
  });

  it("should offer Open in a milestone card menu, handing the milestone to the caller's navigation", () => {
    const onOpenMilestone = vi.fn();
    renderGrid({ milestoneHref: href, onOpenMilestone });
    fireEvent.click(screen.getByRole('button', { name: 'Actions for Learn Go' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Open' }));
    expect(onOpenMilestone).toHaveBeenCalledWith('p1');
  });

  it.each([
    ['a step', 'Actions for goroutines'],
    ['a note card', 'Actions for Reading list'],
  ])('should not offer Open in %s menu', (_why, trigger) => {
    renderGrid({ milestoneHref: href });
    fireEvent.click(screen.getByRole('button', { name: trigger }));
    expect(screen.queryByRole('menuitem', { name: 'Open' })).toBeNull();
  });

  it("should still rename when a card's title is clicked", () => {
    const onOpenMilestone = vi.fn();
    const { onRename } = renderGrid({ milestoneHref: href, onOpenMilestone });
    fireEvent.click(screen.getByText('Learn Go'));
    const input = screen.getByRole('textbox', { name: 'Rename Learn Go' });
    fireEvent.change(input, { target: { value: 'Learn Go well' } });
    fireEvent.keyDown(input, { key: 'Enter' });
    expect(onRename).toHaveBeenCalledWith('p1', 'Learn Go well');
    expect(onOpenMilestone).not.toHaveBeenCalled();
  });

  it('should never show why a milestone matters on its card (FS-043WP R25)', () => {
    renderGrid({
      milestoneHref: href,
      groups: [
        {
          item: mk('p1', 'Learn Go', { intention: 'to build the backend' }),
          children: [],
        },
      ],
    });
    expect(screen.queryByText(/to build the backend/)).toBeNull();
    expect(screen.queryByText(/why does this matter/i)).toBeNull();
  });
});

describe('ChecklistGrid duplicate (FS-F6AEE R14, R16)', () => {
  beforeEach(() => vi.clearAllMocks());

  const choose = (trigger: string) => {
    fireEvent.click(screen.getByRole('button', { name: trigger }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Duplicate' }));
  };

  it.each([
    ['a milestone card', 'Actions for Learn Go', 'p1'],
    ['a note card', 'Actions for Reading list', 'p3'],
    ['a step in a milestone card', 'Actions for channels', 'c2'],
    ['a step in a note card', 'Actions for DDIA', 'c4'],
  ])('should hand %s to the caller to duplicate', (_why, trigger, id) => {
    const onDuplicate = vi.fn();
    renderGrid({ onDuplicate });
    choose(trigger);
    expect(onDuplicate).toHaveBeenCalledWith(id);
  });

  it('should offer no Duplicate when the caller hands none over', () => {
    renderGrid();
    fireEvent.click(screen.getByRole('button', { name: 'Actions for Learn Go' }));
    expect(screen.queryByRole('menuitem', { name: 'Duplicate' })).toBeNull();
  });

  it('should never offer Duplicate on an archived item (R15)', () => {
    renderGrid({
      onDuplicate: vi.fn(),
      groups: [
        {
          item: mk('p1', 'Learn Go', { archived: true }),
          children: [mk('c1', 'goroutines', { parentId: 'p1', archived: true })],
        },
      ],
    });
    fireEvent.click(screen.getByRole('button', { name: 'Actions for Learn Go' }));
    expect(screen.queryByRole('menuitem', { name: 'Duplicate' })).toBeNull();
    fireEvent.keyDown(document, { key: 'Escape' });
    fireEvent.click(screen.getByRole('button', { name: 'Actions for goroutines' }));
    expect(screen.queryByRole('menuitem', { name: 'Duplicate' })).toBeNull();
  });

  it('should leave a copy card untraced: tracing follows only the composer', () => {
    const { rerender } = renderGrid({ tracedId: 'p1', onDuplicate: vi.fn() });
    const copy = {
      item: mk('d1', 'Learn Go'),
      children: [mk('dc1', 'goroutines', { parentId: 'd1' })],
    };
    rerender(
      <ChecklistGrid
        {...handlers()}
        groups={[GROUPS[0], copy, ...GROUPS.slice(1)]}
        collapsedIds={new Set()}
        tracedId="p1"
        onDuplicate={vi.fn()}
      />
    );
    expect(card('p1').hasAttribute('data-traced')).toBe(true);
    expect(card('d1').hasAttribute('data-traced')).toBe(false);
  });
});

describe('ChecklistGrid notes on paper (FS-PGG2N R10, R11, R47, R48)', () => {
  beforeEach(() => vi.clearAllMocks());

  const noteGroups = [
    {
      item: mk('n1', 'The whole long body\nwith a second line', {
        type: 'note',
        summary: 'The gist',
      }),
      children: [],
    },
    {
      item: mk('n2', 'First line\nsecond line', { type: 'note' }),
      children: [],
    },
    {
      item: mk('m1', 'Learn Go'),
      children: [
        mk('s1', 'a task step', { parentId: 'm1' }),
        mk('s2', 'a child note\nover two lines', {
          parentId: 'm1',
          type: 'note',
          summary: 'child gist',
        }),
      ],
    },
  ];

  const textOf = (id: string) =>
    document.querySelector(`[data-note-text="${id}"]`) as HTMLElement;

  it('should show the summary on a note card, not the body', () => {
    renderGrid({ groups: noteGroups });
    expect(within(card('n1')).getByText('The gist')).toBeInTheDocument();
    expect(within(card('n1')).queryByText(/whole long body/)).toBeNull();
  });

  it('should show the body, line breaks kept, when a note has no summary', () => {
    renderGrid({ groups: noteGroups });
    expect(textOf('n2').textContent).toBe('First line\nsecond line');
  });

  it("should show a child note row's summary, not its body", () => {
    renderGrid({ groups: noteGroups });
    expect(textOf('s2').textContent).toBe('child gist');
  });

  it("should name a note's menu by its card text's first line (R11)", () => {
    renderGrid({ groups: noteGroups });
    expect(
      screen.getByRole('button', { name: 'Actions for First line' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Actions for The gist' })
    ).toBeInTheDocument();
  });
});

describe('ChecklistGrid ways into a note (FS-PGG2N R12–R14)', () => {
  beforeEach(() => vi.clearAllMocks());

  const noteHref = (id: string) => `/plan/p/n/${id}`;
  const milestoneHref = (id: string) => `/plan/p/m/${id}`;
  const groups = [
    { item: mk('n1', 'Launch ideas\nA waitlist', { type: 'note' }), children: [] },
    {
      item: mk('m1', 'Learn Go'),
      children: [
        mk('s1', 'a task step', { parentId: 'm1' }),
        mk('s2', 'a child note', { parentId: 'm1', type: 'note', summary: 'child gist' }),
      ],
    },
  ];

  it('should make the whole note card a link named for its first line', () => {
    renderGrid({ groups, noteHref, milestoneHref });
    const link = within(card('n1')).getByRole('link', { name: 'Open note: Launch ideas' });
    expect(link.getAttribute('href')).toBe('/plan/p/n/n1');
  });

  it("should keep the card's menu and its entries clear of the link", () => {
    const onEditNote = vi.fn();
    renderGrid({ groups, noteHref, onEditNote });
    const menu = screen.getByRole('button', { name: 'Actions for Launch ideas' });
    expect(menu.closest('a')).toBeNull();
    fireEvent.click(menu);
    expect(screen.getAllByRole('menuitem').map((r) => r.textContent)).toEqual([
      'Set dates',
      'Rename',
      'Make a task',
      'Archive',
      'Delete',
    ]);
  });

  it("should open the note page in writing mode from a note's menu Rename (R14)", () => {
    const onEditNote = vi.fn();
    const { onRename } = renderGrid({ groups, noteHref, onEditNote });
    fireEvent.click(screen.getByRole('button', { name: 'Actions for Launch ideas' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Rename' }));
    expect(onEditNote).toHaveBeenCalledWith('n1');
    expect(screen.queryByRole('textbox', { name: /^Rename/ })).toBeNull();
    expect(onRename).not.toHaveBeenCalled();
  });

  it('should never rename a note inline when its text is clicked', () => {
    renderGrid({ groups, noteHref });
    fireEvent.click(screen.getByText(/Launch ideas/));
    expect(screen.queryByRole('textbox')).toBeNull();
  });

  it("should make a child note row's text a link to its page (R13)", () => {
    renderGrid({ groups, noteHref, milestoneHref });
    const link = within(card('m1')).getByRole('link', { name: 'child gist' });
    expect(link.getAttribute('href')).toBe('/plan/p/n/s2');
  });

  it('should still rename task steps and milestone titles inline', () => {
    const { onRename } = renderGrid({ groups, noteHref, milestoneHref });
    fireEvent.click(screen.getByText('a task step'));
    const step = screen.getByRole('textbox', { name: 'Rename a task step' });
    fireEvent.change(step, { target: { value: 'a better step' } });
    fireEvent.keyDown(step, { key: 'Enter' });
    expect(onRename).toHaveBeenCalledWith('s1', 'a better step');

    fireEvent.click(screen.getByText('Learn Go'));
    expect(screen.getByRole('textbox', { name: 'Rename Learn Go' })).toBeInTheDocument();
  });

  it("should keep a task's menu Rename inline", () => {
    const onEditNote = vi.fn();
    renderGrid({ groups, noteHref, onEditNote, milestoneHref });
    fireEvent.click(screen.getByRole('button', { name: 'Actions for a task step' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Rename' }));
    expect(onEditNote).not.toHaveBeenCalled();
    expect(screen.getByRole('textbox', { name: 'Rename a task step' })).toBeInTheDocument();
  });
});
