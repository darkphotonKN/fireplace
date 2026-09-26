import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest';
import type { Plan } from '@/api/plans';
import type { ChecklistItem } from '@/api/checklists';
import { selectHomeWork, withTicks, type PlanItems } from './homeWork';

/**
 * FS-KSJFR slice 4 (I-KSJFR-4): what Today lists, and what the status line
 * counts.
 *
 * The qualification rule is the whole slice, so it is tested case per case
 * against one fixed "today" — the alternative is a component test that renders
 * seven items and asserts on text, which tells you *that* something is wrong
 * and never *which* rule broke.
 */

/** The day every fixture below is dated against, as a local date. */
const TODAY = new Date(2026, 2, 16, 9, 0); // Monday 16 March 2026, local
const TOMORROW = '2026-03-17';
const YESTERDAY = '2026-03-15';
const TODAY_STR = '2026-03-16';

const plan = (id: string, extra: Partial<Plan> = {}): Plan =>
  ({
    id,
    name: `Plan ${id}`,
    focus: '',
    description: '',
    planType: 'project',
    dailyReset: false,
    userId: 'u1',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-03-01T00:00:00Z',
    ...extra,
  }) as Plan;

/** `scheduledTime` is not on the generated resource any more; it is still on
 *  the wire for items nobody has re-saved, which is the whole point of R32. */
type ItemFields = Partial<ChecklistItem> & { scheduledTime?: string };

const item = (id: string, extra: ItemFields = {}): ChecklistItem =>
  ({
    id,
    planId: 'a',
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

const ready = (items: ChecklistItem[]): PlanItems => ({ status: 'ready', items });

/**
 * The Today rows for one plan holding `items`.
 *
 * An anchor item that certainly qualifies rides along so the block stays in
 * its Today mode: without it a fixture that fails the rule would reappear
 * through the Next-up fallback and the case would pass for the wrong reason.
 */
const todayRows = (items: ChecklistItem[]) =>
  selectHomeWork([plan('a')], {
    a: ready([...items, item('anchor', { dueDate: TODAY_STR })]),
  }).rows.map((row) => row.description);

beforeEach(() => vi.setSystemTime(TODAY));
afterEach(() => vi.useRealTimers());

describe('selectHomeWork — what qualifies for Today', () => {
  it.each([
    { case: 'due today', fixture: item('x', { dueDate: TODAY_STR }), listed: true },
    { case: 'overdue', fixture: item('x', { dueDate: YESTERDAY }), listed: true },
    { case: 'daily-scope, undated', fixture: item('x', { scope: 'daily' }), listed: true },
    { case: 'due today by startDate alone', fixture: item('x', { startDate: TODAY_STR }), listed: true },
    {
      case: 'due today by the deprecated scheduledTime',
      fixture: item('x', { scheduledTime: '2026-03-16T08:00:00Z' }),
      listed: true,
    },
    { case: 'due tomorrow', fixture: item('x', { dueDate: TOMORROW }), listed: false },
    {
      case: 'mid-range, ending later',
      fixture: item('x', { startDate: YESTERDAY, dueDate: TOMORROW }),
      listed: false,
    },
    { case: 'undated and longterm', fixture: item('x'), listed: false },
    { case: 'a note due today', fixture: item('x', { dueDate: TODAY_STR, type: 'note' }), listed: false },
    { case: 'a daily-scope note', fixture: item('x', { scope: 'daily', type: 'note' }), listed: false },
    {
      case: 'archived and due today',
      fixture: item('x', { dueDate: TODAY_STR, archived: true }),
      listed: false,
    },
    { case: 'archived and daily', fixture: item('x', { scope: 'daily', archived: true }), listed: false },
    { case: 'done and due today', fixture: item('x', { dueDate: TODAY_STR, done: true }), listed: false },
    { case: 'done and overdue', fixture: item('x', { dueDate: YESTERDAY, done: true }), listed: false },
  ])('should list an item $case: $listed', ({ fixture, listed }) => {
    expect(todayRows([fixture]).includes('x')).toBe(listed);
  });

  it('should list an item that is both due today and daily-scope exactly once', () => {
    const rows = todayRows([item('both', { dueDate: TODAY_STR, scope: 'daily' })]);
    expect(rows).toEqual(['both', 'anchor']);
  });

  it('should prefer startDate over the deprecated scheduledTime where both exist', () => {
    // startDate says tomorrow, the stale mirror says today: startDate wins, so
    // this is not due yet.
    const stale = item('x', { startDate: TOMORROW, scheduledTime: '2026-03-16T08:00:00Z' });
    expect(todayRows([stale])).toEqual(['anchor']);
  });

  it('should qualify an item at any depth, and carry only its immediate parent', () => {
    const parent = item('parent');
    const child = item('child', { parentId: 'parent' });
    const grandchild = item('grandchild', { parentId: 'child', dueDate: TODAY_STR });
    const work = selectHomeWork([plan('a')], {
      a: ready([parent, child, grandchild]),
    });

    expect(work.rows).toHaveLength(1);
    // The chain stops at "child": no "parent › child › grandchild".
    expect(work.rows[0]).toMatchObject({
      description: 'grandchild',
      parentDescription: 'child',
    });
  });

  it('should leave a top-level item without parent context', () => {
    const work = selectHomeWork([plan('a')], {
      a: ready([item('x', { dueDate: TODAY_STR })]),
    });
    expect(work.rows[0].parentDescription).toBeNull();
  });

  it('should name the plan each row came from', () => {
    const work = selectHomeWork([plan('a', { name: 'Ship it' })], {
      a: ready([item('x', { dueDate: TODAY_STR })]),
    });
    expect(work.rows[0]).toMatchObject({ planId: 'a', planName: 'Ship it' });
  });

  it('should mark an overdue row as overdue and a due-today row as not', () => {
    const work = selectHomeWork([plan('a')], {
      a: ready([item('late', { dueDate: YESTERDAY }), item('now', { dueDate: TODAY_STR })]),
    });
    expect(work.rows.map((r) => [r.description, r.overdue])).toEqual([
      ['late', true],
      ['now', false],
    ]);
  });
});

describe('selectHomeWork — across the fetched plans', () => {
  it('should draw from every ready plan, in plan order then sequence order', () => {
    const work = selectHomeWork([plan('a'), plan('b')], {
      a: ready([item('a1', { dueDate: TODAY_STR }), item('a2', { scope: 'daily' })]),
      b: ready([item('b1', { dueDate: YESTERDAY })]),
    });
    expect(work.rows.map((r) => r.description)).toEqual(['a1', 'a2', 'b1']);
    expect(work.mode).toBe('today');
    expect(work.dueCount).toBe(3);
    expect(work.planCount).toBe(2);
  });

  it('should omit a plan whose items failed, and count only the plans it read', () => {
    const work = selectHomeWork([plan('a'), plan('b')], {
      a: ready([item('a1', { dueDate: TODAY_STR })]),
      b: { status: 'failed' },
    });
    expect(work.status).toBe('ready');
    expect(work.rows.map((r) => r.description)).toEqual(['a1']);
    expect(work.planCount).toBe(1);
  });

  it('should hold everything at loading while any plan is still in flight', () => {
    const work = selectHomeWork([plan('a'), plan('b')], {
      a: ready([item('a1', { dueDate: TODAY_STR })]),
      b: { status: 'loading' },
    });
    // A count that grows under the reader is worse than one that arrives late.
    expect(work.status).toBe('loading');
    expect(work.rows).toEqual([]);
  });

  it('should treat a plan with no state yet as still in flight', () => {
    expect(selectHomeWork([plan('a')], {}).status).toBe('loading');
  });

  it('should be ready and empty when there are no plans at all', () => {
    const work = selectHomeWork([], {});
    expect(work).toMatchObject({ status: 'ready', rows: [], planCount: 0, dueCount: 0 });
  });

  it('should report failure when every plan it was given failed', () => {
    // Distinct from "no plans at all": something was asked for and nothing came
    // back, so the block owes the reader a retry rather than silence
    // (FS-KSJFR §Edge States, all three item fetches fail).
    const work = selectHomeWork([plan('a'), plan('b')], {
      a: { status: 'failed' },
      b: { status: 'failed' },
    });
    expect(work.status).toBe('failed');
    expect(work.planCount).toBe(0);
    expect(work.rows).toEqual([]);
  });

  it('should not call an empty account a failure', () => {
    expect(selectHomeWork([], {}).status).toBe('ready');
  });
});

describe('selectHomeWork — the Next up fallback', () => {
  const undated = (id: string, extra: ItemFields = {}) => item(id, extra);

  it('should fall back to the most recent plan when nothing qualifies', () => {
    const work = selectHomeWork([plan('a'), plan('b')], {
      a: ready([undated('a1'), undated('a2')]),
      b: ready([undated('b1')]),
    });
    expect(work.mode).toBe('nextUp');
    // Plans arrive most-recently-updated first, so "the most recent plan" is
    // the first one — and only that one's items are listed.
    expect(work.rows.map((r) => r.description)).toEqual(['a1', 'a2']);
  });

  it('should count everything waiting across the plans it read, not just what it lists', () => {
    const work = selectHomeWork([plan('a'), plan('b')], {
      a: ready([undated('a1'), undated('a2')]),
      b: ready([undated('b1')]),
    });
    expect(work.waitingCount).toBe(3);
    expect(work.planCount).toBe(2);
  });

  it('should list only the first few, however many are waiting', () => {
    const many = Array.from({ length: 12 }, (_, i) => undated(`i${i}`));
    const work = selectHomeWork([plan('a')], { a: ready(many) });
    expect(work.rows).toHaveLength(5);
    expect(work.rows.map((r) => r.description)).toEqual(['i0', 'i1', 'i2', 'i3', 'i4']);
    expect(work.waitingCount).toBe(12);
  });

  it('should skip a finished plan rather than show an empty Next up', () => {
    const work = selectHomeWork([plan('a'), plan('b')], {
      a: ready([undated('a1', { done: true })]),
      b: ready([undated('b1')]),
    });
    expect(work.rows.map((r) => r.description)).toEqual(['b1']);
  });

  it('should exclude notes, archived and done items from Next up too', () => {
    const work = selectHomeWork([plan('a')], {
      a: ready([
        undated('note', { type: 'note' }),
        undated('archived', { archived: true }),
        undated('done', { done: true }),
        undated('real'),
      ]),
    });
    expect(work.rows.map((r) => r.description)).toEqual(['real']);
    expect(work.waitingCount).toBe(1);
  });

  it('should stay in Next up with nothing to show when the plans are empty', () => {
    const work = selectHomeWork([plan('a')], { a: ready([]) });
    expect(work).toMatchObject({
      status: 'ready',
      mode: 'nextUp',
      rows: [],
      waitingCount: 0,
      planCount: 1,
    });
  });

  it('should carry parent context and the plan name into Next up rows as well', () => {
    const work = selectHomeWork([plan('a', { name: 'Ship it' })], {
      a: ready([item('parent'), item('child', { parentId: 'parent' })]),
    });
    expect(work.rows.map((r) => [r.description, r.parentDescription])).toEqual([
      ['parent', null],
      ['child', 'parent'],
    ]);
    expect(work.rows[0].planName).toBe('Ship it');
  });
});

describe('selectHomeWork — over the wire format the API really sends (I-0059)', () => {
  // Every fixture above is date-only, which is what the client *sends*. The
  // server stores a *time.Time and returns RFC3339, so these are the strings
  // the selector meets in production — and until I-0059 the due and overdue
  // arms could not fire against them at all.
  const wire = (day: string) => `${day}T00:00:00Z`;

  it('should qualify due-today and overdue items given RFC3339 dates', () => {
    const work = selectHomeWork([plan('a')], {
      a: ready([
        item('due today', { dueDate: wire(TODAY_STR) }),
        item('overdue', { dueDate: wire(YESTERDAY) }),
        item('later', { dueDate: wire(TOMORROW) }),
      ]),
    });

    expect(work.mode).toBe('today');
    expect(work.rows.map((r) => r.description)).toEqual(['due today', 'overdue']);
    expect(work.dueCount).toBe(2);
    expect(work.rows.map((r) => r.overdue)).toEqual([false, true]);
    // The chip label comes through too, which is what ItemDateChip renders.
    expect(work.rows[0].dateLabel).toBe('Mar 16');
  });

  it('should take the date portion of a deprecated scheduledTime the same way', () => {
    const work = selectHomeWork([plan('a')], {
      a: ready([item('scheduled', { scheduledTime: `${TODAY_STR}T22:00:00Z` })]),
    });
    expect(work.mode).toBe('today');
    expect(work.rows.map((r) => r.description)).toEqual(['scheduled']);
  });
});

describe('withTicks — a row ticked during the visit (I-KSJFR-5, R33–R35, D9)', () => {
  const today = () =>
    selectHomeWork([plan('a', { name: 'Ship it' })], {
      a: ready([
        item('first', { dueDate: TODAY_STR }),
        item('second', { dueDate: TODAY_STR }),
        item('third', { dueDate: TODAY_STR }),
      ]),
    });

  it('should mark the ticked row done and leave it exactly where it was', () => {
    // D9's refinement, and the whole reason the overlay exists rather than a
    // re-run of the selection: a vanishing row collapses the list under the
    // cursor and turns the next tick into a mis-click.
    const ticked = withTicks(today(), ['second']);

    expect(ticked.rows.map((r) => r.description)).toEqual([
      'first',
      'second',
      'third',
    ]);
    expect(ticked.rows.map((r) => r.done)).toEqual([false, true, false]);
  });

  it('should decrement the count the status line reads', () => {
    expect(today().dueCount).toBe(3);
    expect(withTicks(today(), ['second']).dueCount).toBe(2);
    expect(withTicks(today(), ['second', 'third']).dueCount).toBe(1);
  });

  it('should hold the heading and pull in no rows when every row is ticked', () => {
    // R35. The block settles into a done state until the next load; it does not
    // become Next up, and it does not reach into another plan for filler.
    const all = withTicks(today(), ['first', 'second', 'third']);

    expect(all.mode).toBe('today');
    expect(all.rows).toHaveLength(3);
    expect(all.rows.every((r) => r.done)).toBe(true);
    expect(all.dueCount).toBe(0);
  });

  it('should take the waiting count down too, so Next up counts honestly', () => {
    const nextUp = selectHomeWork([plan('a')], {
      a: ready([item('x'), item('y')]),
    });
    expect(nextUp.mode).toBe('nextUp');
    expect(nextUp.waitingCount).toBe(2);
    expect(withTicks(nextUp, ['x']).waitingCount).toBe(1);
  });

  it('should never take a count below zero', () => {
    // An id that matches no row cannot spend a count, and a row already done
    // cannot be spent twice.
    const once = withTicks(today(), ['first']);
    expect(withTicks(once, ['first', 'not-a-row']).dueCount).toBe(2);
  });

  it('should hand back the same object when nothing has been ticked', () => {
    // Identity matters: this sits inside a `useMemo` whose result both the
    // status line and the block read, and an untouched visit should not churn.
    const work = today();
    expect(withTicks(work, [])).toBe(work);
    expect(withTicks(work, ['not-a-row'])).toBe(work);
  });

  it('should emit rows not-done out of the selector', () => {
    expect(today().rows.every((r) => r.done)).toBe(false);
  });
});
