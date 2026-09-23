import { describe, it, expect, vi, beforeEach } from 'vitest';
import { act, fireEvent, render, screen } from '@testing-library/react';

/**
 * FS-0009 R3, R4, R5, R7, R8 — picking an item up and dropping it somewhere
 * else, in the list and in the blocks.
 *
 * The write underneath is the one the actions panel already uses, and
 * `Todo.reorder.test.tsx` covers it; what these exercise is the gesture: the
 * order a drop produces, the drops that produce nothing, and the set the
 * request carries when a filter or a page is only showing part of it.
 *
 * jsdom lays nothing out, so a drag has to be given a geometry to happen in
 * (see `layout` below) and is driven through dnd-kit's mouse sensor. The
 * *feel* of the drag — the lift, the gap, the settle — is not testable here
 * and is checked by hand.
 */

const fetchChecklist = vi.fn();
const reorderChecklistItems = vi.fn();
const toast = vi.fn();

vi.mock('next/navigation', () => ({
  useParams: () => ({ planId: 'plan-1' }),
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), refresh: vi.fn() }),
}));

vi.mock('@/components/ui/use-toast', () => ({
  useToast: () => ({ toast: (...a: unknown[]) => toast(...a) }),
}));

vi.mock('@/services/api', () => ({
  fetchChecklist: (...a: unknown[]) => fetchChecklist(...a),
  fetchArchivedChecklist: vi.fn(async () => []),
  createChecklistItem: vi.fn(),
  updateChecklistItem: vi.fn(async () => ({})),
  deleteChecklistItem: vi.fn(),
  updateChecklistDates: vi.fn(async () => ({})),
  archiveChecklistItem: vi.fn(async () => ({})),
  reorderChecklistItems: (...a: unknown[]) => reorderChecklistItems(...a),
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

const item = (
  id: string,
  type: 'task' | 'note' = 'task',
  parentId: string | null = null
) => ({
  id,
  description: `item ${id}`,
  done: false,
  scope: 'longterm',
  type,
  parentId,
  planId: 'plan-1',
});

const THREE = [item('A'), item('B'), item('C')];

/** Rendered order, read off the one control every item carries in both views. */
const order = () =>
  screen
    .getAllByRole('button', { name: /^Actions for / })
    .map((b) => b.getAttribute('aria-label')!.replace('Actions for ', ''));

// ---------------------------------------------------------------------------
// A geometry to drag in.
//
// Every rect jsdom reports is 0×0, so dnd-kit cannot tell one row from the
// next and every drop lands on the first thing it finds. Each draggable node
// is therefore given a slot of its own, in DOM order — which is the order the
// view renders and does not change mid-drag, since dnd-kit moves items with
// transforms rather than by re-ordering the DOM.
// ---------------------------------------------------------------------------

const SLOT = 40;

const grip = (description: string) =>
  screen.getByRole('button', { name: `Reorder ${description}` });

const slots = () =>
  Array.from(document.querySelectorAll<HTMLElement>('[data-sortable-id]'));

function layout() {
  Element.prototype.getBoundingClientRect = function () {
    const index = slots().indexOf(this as HTMLElement);
    if (index === -1) {
      return new DOMRect(0, 0, 0, 0);
    }
    return new DOMRect(0, index * SLOT, 300, SLOT);
  };
}

// By the item's own grip, not by its text: a card contains every one of its
// steps' descriptions, so matching on text picks the card for a step.
const nodeOf = (description: string) =>
  grip(description).closest<HTMLElement>('[data-sortable-id]')!;

const centreOf = (node: HTMLElement) => slots().indexOf(node) * SLOT + SLOT / 2;

const settle = (ms = 32) =>
  act(async () => {
    // dnd-kit measures on an animation frame, which jsdom runs on a timer.
    await new Promise((resolve) => setTimeout(resolve, ms));
  });

// A drag ends by swallowing the click that would otherwise follow it, and
// dnd-kit keeps that guard on the document for 50ms. Anything a test clicks
// straight after a drop has to outlast it.
const CLICK_GUARD = 60;

/** Pick `from` up by its grip and let it go over `to`'s slot. */
async function drag(from: string, to: string, options: { drop?: boolean } = {}) {
  const handle = grip(from);
  const start = centreOf(nodeOf(from));
  const end = centreOf(nodeOf(to));

  fireEvent.mouseDown(handle, { button: 0, clientX: 10, clientY: start });
  await settle();
  // Far enough to be a drag and not a click.
  fireEvent.mouseMove(document, { clientX: 10, clientY: start + 10 });
  await settle();
  fireEvent.mouseMove(document, { clientX: 10, clientY: end });
  await settle();
  if (options.drop === false) return;
  fireEvent.mouseUp(document);
  await settle(CLICK_GUARD);
}

async function renderView(
  which: 'list' | 'grid',
  items: { description: string }[]
) {
  window.localStorage.setItem('checklistView', which);
  fetchChecklist.mockImplementation(async () => items);
  const Todo = (await import('./Todo')).default;
  const view = render(<Todo fixedTaskType="longterm" enableTypeFilter />);
  await screen.findByText(items[0].description);
  layout();
  return view;
}

const renderList = (items: { description: string }[] = THREE) =>
  renderView('list', items);

const renderGrid = (items: { description: string }[] = THREE) =>
  renderView('grid', items);

describe('Todo drag to reorder', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
    reorderChecklistItems.mockResolvedValue([]);
  });

  it('should drop a row into another row\'s place and send the whole set in that order', async () => {
    await renderList();
    expect(order()).toEqual(['item A', 'item B', 'item C']);

    await drag('item C', 'item A');

    expect(order()).toEqual(['item C', 'item A', 'item B']);
    expect(reorderChecklistItems).toHaveBeenCalledWith('plan-1', {
      scope: 'longterm',
      parentId: null,
      ids: ['C', 'A', 'B'],
    });
  });

  it('should drop a card among the cards, and show it in the list (R4.1, R1.2)', async () => {
    await renderGrid();
    expect(document.querySelectorAll('[data-checklist-card]').length).toBe(3);

    await drag('item C', 'item A');

    expect(order()).toEqual(['item C', 'item A', 'item B']);
    expect(reorderChecklistItems).toHaveBeenCalledWith('plan-1', {
      scope: 'longterm',
      parentId: null,
      ids: ['C', 'A', 'B'],
    });

    // The same order drawn the other way — the order is the item's, not the
    // view's (R1.1, R1.2).
    fireEvent.click(screen.getByRole('button', { name: 'List view' }));
    expect(order()).toEqual(['item C', 'item A', 'item B']);
  });

  it('should drag a step within its own card (R4.2)', async () => {
    await renderGrid([
      item('P'),
      item('P1', 'task', 'P'),
      item('P2', 'task', 'P'),
      item('P3', 'task', 'P'),
    ]);

    await drag('item P3', 'item P1');

    expect(order()).toEqual(['item P', 'item P3', 'item P1', 'item P2']);
    expect(reorderChecklistItems).toHaveBeenCalledWith('plan-1', {
      scope: 'longterm',
      parentId: 'P',
      ids: ['P3', 'P1', 'P2'],
    });
  });

  it('should take a top-level row\'s children with it (R3.2)', async () => {
    await renderList([
      item('A'),
      item('A1', 'task', 'A'),
      item('A2', 'task', 'A'),
      item('B'),
    ]);
    expect(order()).toEqual(['item A', 'item A1', 'item A2', 'item B']);

    await drag('item B', 'item A');

    // B is one of two top-level rows, so the set written is the two of them;
    // A1 and A2 are not named in it and still travel with A.
    expect(reorderChecklistItems).toHaveBeenCalledWith('plan-1', {
      scope: 'longterm',
      parentId: null,
      ids: ['B', 'A'],
    });
    expect(order()).toEqual(['item B', 'item A', 'item A1', 'item A2']);
  });

  it('should take them with it while they are folded away too (R3.2)', async () => {
    await renderList([
      item('A'),
      item('A1', 'task', 'A'),
      item('A2', 'task', 'A'),
      item('B'),
    ]);

    fireEvent.click(screen.getByRole('button', { name: 'Collapse item A' }));
    expect(order()).toEqual(['item A', 'item B']);

    await drag('item B', 'item A');

    expect(order()).toEqual(['item B', 'item A']);
    // Still the two top-level rows: folding a parent hides its children, it
    // does not take them out of the set or put them into this one.
    expect(reorderChecklistItems).toHaveBeenCalledWith('plan-1', {
      scope: 'longterm',
      parentId: null,
      ids: ['B', 'A'],
    });
    fireEvent.click(screen.getByRole('button', { name: 'Expand item A' }));
    expect(order()).toEqual(['item B', 'item A', 'item A1', 'item A2']);
  });

  it('should refuse a drop that would change the item\'s parent (R5.1)', async () => {
    await renderList([
      item('P'),
      item('P1', 'task', 'P'),
      item('P2', 'task', 'P'),
      item('Q'),
      item('Q1', 'task', 'Q'),
      item('Q2', 'task', 'Q'),
    ]);
    const before = order();

    // P1 dropped on one of Q's children: a different family, so the drag
    // re-parents nothing, the row goes back, and no order is written.
    await drag('item P1', 'item Q1');

    expect(order()).toEqual(before);
    expect(reorderChecklistItems).not.toHaveBeenCalled();
  });

  it('should send nothing when the drop is where the drag began (R3.4)', async () => {
    await renderList();

    await drag('item B', 'item B');

    expect(order()).toEqual(['item A', 'item B', 'item C']);
    expect(reorderChecklistItems).not.toHaveBeenCalled();
  });

  it('should abandon the drag on Escape and leave the order alone (R3.5)', async () => {
    await renderList();

    // The same gesture that reorders in the first test, up to the drop.
    await drag('item C', 'item A', { drop: false });
    fireEvent.keyDown(document, { key: 'Escape', code: 'Escape' });
    await settle();
    fireEvent.mouseUp(document);
    await settle(CLICK_GUARD);

    expect(order()).toEqual(['item A', 'item B', 'item C']);
    expect(reorderChecklistItems).not.toHaveBeenCalled();
  });

  it('should still write the whole set while a filter is hiding half of it (R7)', async () => {
    await renderList([
      item('T1'),
      item('N1', 'note'),
      item('T2'),
      item('N2', 'note'),
    ]);

    fireEvent.click(screen.getByRole('button', { name: 'Notes' }));
    expect(order()).toEqual(['item N1', 'item N2']);

    await drag('item N2', 'item N1');

    expect(order()).toEqual(['item N2', 'item N1']);
    // The tasks the filter hides keep the first and third slots they held.
    expect(reorderChecklistItems).toHaveBeenCalledWith('plan-1', {
      scope: 'longterm',
      parentId: null,
      ids: ['T1', 'N2', 'T2', 'N1'],
    });
  });

  it('should leave the other pages\' order untouched (R8)', async () => {
    const many = Array.from({ length: 13 }, (_, i) =>
      item(`t${String(i + 1).padStart(2, '0')}`)
    );
    await renderList(many);

    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    layout();
    expect(order()).toEqual(['item t11', 'item t12', 'item t13']);

    await drag('item t13', 'item t11');

    expect(order()).toEqual(['item t13', 'item t11', 'item t12']);
    // The whole set is written, and the first page's ten are in it, in the
    // order they already had (R8.2).
    expect(reorderChecklistItems).toHaveBeenCalledWith('plan-1', {
      scope: 'longterm',
      parentId: null,
      ids: [
        ...many.slice(0, 10).map((i) => i.id),
        't13',
        't11',
        't12',
      ],
    });
  });

  it('should give no grip to an item alone in its set (§Edge States)', async () => {
    await renderGrid([item('P'), item('P1', 'task', 'P')]);

    // The one card has no card to trade places with; its one step has no step.
    expect(screen.queryByRole('button', { name: 'Reorder item P' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Reorder item P1' })).toBeNull();
  });
});
