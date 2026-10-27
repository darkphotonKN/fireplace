import { describe, it, expect } from 'vitest';
import {
  selectDayPlanRollup,
  type RollupDayPlanState,
  type RollupItem,
  type RollupItemsState,
  type RollupPlan,
} from './dayPlanRollup';

/**
 * FS-02JBZ R85–R89, D10: the home roll-up is model-free and read-only. A
 * stored day plan gives its first resolvable entries; a plan with none today
 * (or whose read failed) gives rule picks marked basic; two per plan, up to
 * six, and a single focused plan fills the block.
 */

const item = (id: string, extra: Partial<RollupItem> = {}): RollupItem => ({
  id,
  description: `item ${id}`,
  done: false,
  archived: false,
  scope: 'longterm',
  type: 'task',
  parentId: null,
  updatedAt: '2026-10-01T00:00:00Z',
  ...extra,
});

const ready = (items: RollupItem[]): RollupItemsState => ({ status: 'ready', items });

const entry = (id: string, itemId: string, ticked = false) => ({
  id,
  kind: 'existing_item' as const,
  itemId,
  text: `drawn ${id}`,
  reason: `why ${id}`,
  ticked,
});

const planned = (
  entries: ReturnType<typeof entry>[],
  source: 'ai' | 'rules' = 'ai',
): RollupDayPlanState => ({
  status: 'ready',
  dayPlan: { status: 'planned', source, groups: [{ label: 'G', entries }] },
});

const notPlanned: RollupDayPlanState = {
  status: 'ready',
  dayPlan: { status: 'not_planned', groups: [] },
};

const failed: RollupDayPlanState = { status: 'failed' };

const plan = (id: string): RollupPlan => ({ id, name: `Plan ${id}` });

// A plan whose day plan has three live entries, a hidden one first.
const A_ITEMS = [
  item('a1', { description: 'Ship the beta' }),
  item('a2', { parentId: 'a1', description: 'Write notes' }),
  item('a3', { parentId: 'a1', description: 'Fix login' }),
  item('a4', { parentId: 'a1', description: 'Tag release' }),
  item('aX', { archived: true }),
];
const A_PLAN = planned([
  entry('eX', 'aX'), // archived since drawing: hidden, so it costs no slot
  entry('e2', 'a2', true),
  entry('e3', 'a3'),
  entry('e4', 'a4'),
]);

// A plan with dated work and milestones, for the rule picks.
const B_ITEMS = [
  item('m1', { description: 'Old milestone', updatedAt: '2026-10-01T00:00:00Z' }),
  item('m1s', { parentId: 'm1', description: 'Step of old' }),
  item('m2', { description: 'Fresh milestone', updatedAt: '2026-10-20T00:00:00Z' }),
  item('m2done', { parentId: 'm2', done: true, description: 'Done step' }),
  item('m2s', { parentId: 'm2', description: 'Step of fresh' }),
  item('due', { parentId: 'm1', description: 'Due soon', dueDate: '2026-10-28T00:00:00Z' }),
  item('late', { parentId: 'm1', description: 'Late', dueDate: '2026-10-20T00:00:00Z' }),
  item('later', { parentId: 'm1', description: 'Later', dueDate: '2026-11-05T00:00:00Z' }),
  item('lateDone', { done: true, dueDate: '2026-10-01T00:00:00Z' }),
  item('note', { type: 'note', dueDate: '2026-10-02T00:00:00Z' }),
];

const C_ITEMS = [
  item('c1', { description: 'Only milestone', updatedAt: '2026-10-10T00:00:00Z' }),
  item('c1s', { parentId: 'c1', description: 'Only step' }),
];

type Row = { planId: string; text: string; ticked: boolean; basic: boolean };
const view = (rows: Row[]) =>
  rows.map(({ planId, text, ticked, basic }) => ({ planId, text, ticked, basic }));

describe('selectDayPlanRollup', () => {
  it('should take two rows per plan: stored entries, rule picks for not planned and failed reads', () => {
    const rollup = selectDayPlanRollup(
      [plan('A'), plan('B'), plan('C')],
      { A: ready(A_ITEMS), B: ready(B_ITEMS), C: ready(C_ITEMS) },
      { A: A_PLAN, B: notPlanned, C: failed }
    );

    expect(rollup.status).toBe('ready');
    expect(view(rollup.rows)).toEqual([
      { planId: 'A', text: 'Write notes', ticked: true, basic: false },
      { planId: 'A', text: 'Fix login', ticked: false, basic: false },
      // overdue first, then due soonest
      { planId: 'B', text: 'Late', ticked: false, basic: true },
      { planId: 'B', text: 'Due soon', ticked: false, basic: true },
      // no dated work: the next open step of the most recently updated milestone
      { planId: 'C', text: 'Only step', ticked: false, basic: true },
    ]);
  });

  it('should let a single focused plan fill the block, up to six', () => {
    const rollup = selectDayPlanRollup(
      [plan('B')],
      { B: ready(B_ITEMS) },
      { B: notPlanned }
    );

    // overdue → due soonest → next step of each milestone, freshest first;
    // done items and notes are never picked.
    expect(rollup.rows.map((r) => r.text)).toEqual([
      'Late',
      'Due soon',
      'Later',
      'Step of fresh',
      'Step of old',
    ]);
  });

  it('should cap a single stored plan at six rows', () => {
    const items = Array.from({ length: 8 }, (_, i) => item(`i${i}`));
    const rollup = selectDayPlanRollup(
      [plan('A')],
      { A: ready(items) },
      { A: planned(items.map((it, i) => entry(`e${i}`, it.id))) }
    );
    expect(rollup.rows).toHaveLength(6);
  });

  it('should mark rows from a stored rules plan basic', () => {
    const rollup = selectDayPlanRollup(
      [plan('A')],
      { A: ready(A_ITEMS) },
      { A: planned([entry('e2', 'a2')], 'rules') }
    );
    expect(view(rollup.rows)).toEqual([
      { planId: 'A', text: 'Write notes', ticked: false, basic: true },
    ]);
  });

  it('should name the milestone of a step and leave a milestone row bare', () => {
    const rollup = selectDayPlanRollup(
      [plan('C')],
      { C: ready(C_ITEMS) },
      { C: planned([entry('e1', 'c1s'), entry('e2', 'c1')]) }
    );
    expect(rollup.rows.map((r) => r.milestoneTitle)).toEqual(['Only milestone', null]);
    expect(rollup.rows.map((r) => r.planName)).toEqual(['Plan C', 'Plan C']);
  });

  it('should fall back to the milestone itself when it has no open step', () => {
    const rollup = selectDayPlanRollup(
      [plan('D')],
      { D: ready([item('d1', { description: 'Bare milestone' })]) },
      { D: notPlanned }
    );
    expect(rollup.rows.map((r) => r.text)).toEqual(['Bare milestone']);
  });

  it.each([
    { name: 'a plan with no items', items: ready([]), dayPlan: notPlanned },
    { name: 'a stored plan with nothing left that resolves', items: ready(A_ITEMS), dayPlan: planned([entry('eX', 'aX')]) },
    { name: 'a plan whose items failed', items: { status: 'failed' } as RollupItemsState, dayPlan: A_PLAN },
  ])('should take nothing from $name', ({ items, dayPlan }) => {
    const rollup = selectDayPlanRollup([plan('A')], { A: items }, { A: dayPlan });
    expect(rollup).toEqual({ status: 'ready', rows: [] });
  });

  it('should stay loading until every focused plan has its items and day plan', () => {
    const plans = [plan('A'), plan('B')];
    expect(
      selectDayPlanRollup(plans, { A: ready(A_ITEMS), B: { status: 'loading' } }, { A: A_PLAN, B: notPlanned })
        .status,
    ).toBe('loading');
    expect(
      selectDayPlanRollup(plans, { A: ready(A_ITEMS), B: ready(B_ITEMS) }, { A: A_PLAN, B: { status: 'loading' } })
        .status,
    ).toBe('loading');
    expect(selectDayPlanRollup(plans, { A: ready(A_ITEMS) }, { A: A_PLAN }).status).toBe('loading');
  });

  it('should give every row a key unique across plans', () => {
    const rollup = selectDayPlanRollup(
      [plan('A'), plan('B')],
      { A: ready(A_ITEMS), B: ready(B_ITEMS) },
      { A: A_PLAN, B: notPlanned }
    );
    const keys = rollup.rows.map((r) => r.key);
    expect(new Set(keys).size).toBe(keys.length);
  });
});
