import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import type { ComponentProps } from 'react';
import { render, screen } from '@testing-library/react';
import TodayBlock from './TodayBlock';
import { selectHomeWork, type HomeWork, type WorkRow } from '@/lib/homeWork';
import type { Plan } from '@/api/plans';
import type { ChecklistItem } from '@/api/checklists';

/**
 * FS-KSJFR slice 4 (I-KSJFR-4): the block that makes home worth opening
 * (R23–R30, R49).
 *
 * What rows qualify is `homeWork.test.ts`'s subject. This file is about what
 * the reader sees: the heading matching the list, a row that still means
 * something out of context, and rows that don't yet do anything.
 */

const TODAY = new Date(2026, 2, 16, 9, 0);

const row = (over: Partial<WorkRow> = {}): WorkRow => ({
  id: 'i1',
  description: 'Draft the migration note',
  planId: 'p1',
  planName: 'Ship it',
  parentDescription: null,
  dateLabel: null,
  overdue: false,
  done: false,
  ...over,
});

const work = (over: Partial<HomeWork> = {}): HomeWork => ({
  status: 'ready',
  mode: 'today',
  rows: [row()],
  dueCount: 1,
  waitingCount: 0,
  planCount: 3,
  ...over,
});

beforeEach(() => vi.setSystemTime(TODAY));
afterEach(() => vi.useRealTimers());

/**
 * `onTick` is required on the real component — Dashboard is its only production
 * caller and always passes one, so an optional handler described a state no user
 * could reach. Tests that care about ticking pass their own; the rest take this.
 */
function Block({
  onTick = () => {},
  ...rest
}: Omit<ComponentProps<typeof TodayBlock>, 'onTick'> & {
  onTick?: (row: WorkRow) => void;
}) {
  return <TodayBlock onTick={onTick} {...rest} />;
}

describe('TodayBlock', () => {
  it('should head the block Today when the rows are due', () => {
    render(<Block work={work({ mode: 'today' })} />);
    expect(screen.getByRole('heading', { name: 'Today' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Next up' })).toBeNull();
  });

  it('should head the same block Next up when the rows are not due', () => {
    render(<Block work={work({ mode: 'nextUp', dueCount: 0, waitingCount: 4 })} />);
    // One block with two headings, never a second block underneath (R24).
    expect(screen.getByRole('heading', { name: 'Next up' })).toBeInTheDocument();
    expect(screen.getAllByRole('heading')).toHaveLength(1);
  });

  it('should name the plan each row came from and link into it', () => {
    render(<Block work={work()} />);
    expect(screen.getByRole('link', { name: 'Ship it' })).toHaveAttribute(
      'href',
      '/plan/p1'
    );
  });

  it('should carry a sub-item’s parent onto the row as context', () => {
    render(
      <Block
        work={work({ rows: [row({ description: 'Call them back', parentDescription: 'Tuesday review' })] })}
      />
    );
    expect(screen.getByText(/in Tuesday review/)).toBeInTheDocument();
  });

  it('should leave a top-level row without an "in" clause', () => {
    render(<Block work={work()} />);
    expect(screen.queryByText(/^in /)).toBeNull();
  });

  it('should show an item’s dates where it has them', () => {
    render(<Block work={work({ rows: [row({ dateLabel: 'Mar 16' })] })} />);
    expect(screen.getByText('Mar 16')).toBeInTheDocument();
  });

  it('should offer each row a tick, named by the item (I-KSJFR-5, R33)', () => {
    render(<Block work={work()} onTick={vi.fn()} />);
    expect(
      screen.getByRole('checkbox', { name: 'Draft the migration note' })
    ).toHaveAttribute('aria-checked', 'false');
  });

  it('should hand the whole row back on tick, so the caller knows its plan', () => {
    // The mutation is `updateChecklistItem(planId, id, ...)`; a bare id would
    // send the caller looking the plan up again.
    const onTick = vi.fn();
    render(<Block work={work()} onTick={onTick} />);

    screen.getByRole('checkbox', { name: 'Draft the migration note' }).click();

    expect(onTick).toHaveBeenCalledTimes(1);
    expect(onTick.mock.calls[0][0]).toMatchObject({ id: 'i1', planId: 'p1' });
  });

  it('should show a ticked row as done and dimmed, still in its place', () => {
    // D9's refinement: the row does not disappear. It is the second of three
    // and it stays the second of three.
    render(
      <Block
        work={work({
          rows: [
            row({ id: 'a', description: 'first' }),
            row({ id: 'b', description: 'second', done: true }),
            row({ id: 'c', description: 'third' }),
          ],
          dueCount: 2,
        })}
        onTick={vi.fn()}
      />
    );

    expect(screen.getAllByRole('listitem').map((li) => li.textContent)).toEqual([
      expect.stringContaining('first'),
      expect.stringContaining('second'),
      expect.stringContaining('third'),
    ]);
    expect(screen.getByRole('checkbox', { name: 'second' })).toHaveAttribute(
      'aria-checked',
      'true'
    );
    expect(screen.getByText('second').className).toMatch(/opacity-/);
  });

  it('should not fire a second write for a row already ticked', () => {
    const onTick = vi.fn();
    render(
      <Block work={work({ rows: [row({ done: true })] })} onTick={onTick} />
    );

    screen.getByRole('checkbox', { name: 'Draft the migration note' }).click();

    expect(onTick).not.toHaveBeenCalled();
  });


  it('should claim no heading while the fan-out is still in flight', () => {
    render(<Block work={work({ status: 'loading', rows: [] })} />);
    expect(screen.queryAllByRole('heading')).toHaveLength(0);
    expect(screen.getByText(/Gathering today/)).toBeInTheDocument();
  });

  it('should invite rather than head an empty list', () => {
    render(
      <Block work={work({ mode: 'nextUp', rows: [], dueCount: 0, planCount: 2 })} />
    );
    expect(screen.queryAllByRole('heading')).toHaveLength(0);
    expect(screen.getByText(/Nothing waiting yet/)).toBeInTheDocument();
  });

  it('should offer a retry when every item fetch failed', () => {
    // §Edge States: all three item fetches fail. Saying nothing would read as
    // "you have nothing to do today", which is a claim about a day home never
    // managed to read.
    const onRetry = vi.fn();
    render(<Block work={work({ status: 'failed', rows: [], planCount: 0 })} onRetry={onRetry} />);

    expect(screen.getByText(/couldn.t load today/i)).toBeInTheDocument();
    screen.getByRole('button', { name: /try again/i }).click();
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('should render nothing at all when no plan was read', () => {
    // Zero plans (the invitation, I-KSJFR-7) or a failed plans fetch (the
    // shell's retry) — neither is this block's screen to own.
    const { container } = render(<Block work={work({ rows: [], planCount: 0 })} />);
    expect(container).toBeEmptyDOMElement();
  });
});

describe('TodayBlock — the coral moment (R49)', () => {
  it('should spend the accent on the heading', () => {
    render(<Block work={work()} />);
    expect(screen.getByRole('heading', { name: 'Today' }).className).toContain(
      'text-primary'
    );
  });

  it('should spend it on an h1, which is coral in both themes', () => {
    // The design guideline reserves coral for the `h1` — one per view — and
    // gives h2/h3 the foreground, so the accent belongs to an h1 or to nothing.
    // globals.css hands `h1` that coral unconditionally in both themes, and
    // this block's subject heading is the page's only heading. Decided on the
    // guideline's own terms (R47, R49, R50), not as a cascade workaround: it
    // outlives I-0061.
    render(<Block work={work()} />);
    expect(screen.getByRole('heading', { name: 'Today' }).tagName).toBe('H1');
  });

  it('should carry a row\'s muted meta line on a <p> with the muted token', () => {
    // These were <div>s until I-0061: `.dark p` outranked
    // `text-muted-foreground`, and R50 forbids a per-component `dark:` patch.
    // With the global element rules in `@layer base` the token wins in both
    // themes, so the semantic element is back.
    const { container } = render(
      <Block
        work={work({ rows: [row({ parentDescription: 'Tuesday review', dateLabel: 'Mar 16' })] })}
      />
    );
    expect(container.querySelectorAll('p[class*="text-muted-foreground"]')).toHaveLength(1);
    expect(container.innerHTML).not.toMatch(/dark:/);
  });

  it('should carry the loading, failed and empty lines on <p> too', () => {
    const cases = [
      work({ status: 'loading', rows: [] }),
      work({ status: 'failed', rows: [] }),
      work({ mode: 'nextUp', rows: [], dueCount: 0, planCount: 2 }),
    ];

    for (const state of cases) {
      const { container, unmount } = render(<Block work={state} />);
      expect(
        container.querySelectorAll('p[class*="text-muted-foreground"]')
      ).toHaveLength(1);
      expect(container.innerHTML).not.toMatch(/dark:/);
      unmount();
    }
  });

  it('should dress the tick in tokens, in one theme, with no layout to jump', () => {
    // R48/R50/R51: no hex, no rgb(), no `-gray-`, no `dark:` patch — and the
    // done state differs from the not-done one only in colour and opacity, so
    // the icon box is the same size either way.
    const { container } = render(
      <Block
        work={work({
          rows: [row({ id: 'a' }), row({ id: 'b', done: true })],
        })}
        onTick={vi.fn()}
      />
    );

    expect(container.innerHTML).not.toMatch(/#[0-9a-f]{3,6}|rgb\(|-gray-/i);
    expect(container.innerHTML).not.toMatch(/dark:/);
    const [open, ticked] = screen.getAllByRole('checkbox');
    for (const box of [open, ticked]) {
      expect(box.querySelector('svg')?.getAttribute('class')).toMatch(/h-4 w-4/);
    }
  });

  it('should mark an overdue date and leave a due-today one neutral', () => {
    const { container } = render(
      <Block
        work={work({
          rows: [
            row({ id: 'a', dateLabel: 'Mar 15', overdue: true }),
            row({ id: 'b', dateLabel: 'Mar 16', overdue: false }),
          ],
        })}
      />
    );
    expect(screen.getByText('Mar 15').className).toContain('primary');
    expect(screen.getByText('Mar 16').className).not.toContain('primary');
    // No raw colour anywhere: tokens only (R48).
    expect(container.innerHTML).not.toMatch(/#[0-9a-f]{3,6}|rgb\(|-gray-/i);
  });
});

describe('TodayBlock — over what the fan-out actually returns', () => {
  const plan = (id: string, name: string): Plan =>
    ({
      id,
      name,
      focus: '',
      description: '',
      planType: 'project',
      dailyReset: false,
      userId: 'u1',
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-03-01T00:00:00Z',
    }) as Plan;

  const item = (id: string, extra: Partial<ChecklistItem> = {}): ChecklistItem =>
    ({
      id,
      planId: 'p1',
      description: id,
      done: false,
      archived: false,
      scope: 'longterm',
      type: 'task',
      sequence: '1',
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
      ...extra,
    }) as ChecklistItem;

  it('should list exactly what the selector qualified, from real items', () => {
    // RFC3339, because that is what the API sends (I-0059). A date-only
    // fixture here tested the parser against the format the client writes,
    // never the one it reads.
    const items = [
      item('parent'),
      item('call them back', { parentId: 'parent', dueDate: '2026-03-16T00:00:00Z' }),
      item('overdue thing', { dueDate: '2026-03-15T00:00:00Z' }),
      item('a habit', { scope: 'daily' }),
      item('a note', { dueDate: '2026-03-16T00:00:00Z', type: 'note' }),
      item('done already', { dueDate: '2026-03-16T00:00:00Z', done: true }),
      item('next month', { dueDate: '2026-04-16T00:00:00Z' }),
    ];
    const composed = selectHomeWork([plan('p1', 'Ship it')], {
      p1: { status: 'ready', items },
    });

    render(<Block work={composed} />);

    expect(screen.getByRole('heading', { name: 'Today' })).toBeInTheDocument();
    expect(screen.getAllByRole('listitem')).toHaveLength(3);
    for (const shown of ['call them back', 'overdue thing', 'a habit']) {
      expect(screen.getByText(shown)).toBeInTheDocument();
    }
    for (const hidden of ['a note', 'done already', 'next month']) {
      expect(screen.queryByText(hidden)).toBeNull();
    }
    // The dated child came through with its parent as context, and the parent
    // itself — undated and not daily — did not become a row of its own.
    expect(screen.getByText(/in parent/)).toBeInTheDocument();
  });
});
