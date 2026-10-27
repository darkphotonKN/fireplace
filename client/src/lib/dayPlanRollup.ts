/**
 * Home's day plan roll-up (FS-02JBZ R85–R89, D10; ADR-0019 clause 2).
 *
 * Model-free by construction: it reads only what home already holds, the
 * focused plans' items and their cached day plans (a `GET`, which never
 * generates). Per plan:
 *
 * - **Stored day plan** (R86): its first entries in plan order that still
 *   resolve against the plan's items, through the plan page's own resolver
 *   (`resolveDayPlan`, R67), so home and the section hide the same entries.
 *   A ticked entry is shown ticked. A stored `rules` plan is the basic plan
 *   and its rows carry the basic mark.
 * - **No day plan today, or the read failed** (R87): rule picks from the
 *   items, D9 without revisions (the client holds no revision data): overdue
 *   (most overdue first) → due soonest → the next open step of each
 *   milestone, most recently updated first. Marked basic.
 * - **Items failed, or nothing to show** (R89): the plan contributes nothing.
 *
 * Two rows per plan; a single focused plan fills the block, up to six (R88).
 * A plan that comes up short leaves its slots empty rather than handing them
 * to another plan: "two per plan" is what makes the block a roll-up.
 *
 * Pure: no React, no API module. `useHomeData` fetches, this decides.
 */
import { resolveDayPlan, type DayPlanEntrySource } from './dayPlan';
import { parseDateOnly } from './itemDates';
import type { MilestoneSourceItem } from './milestones';

/** Rows a plan contributes when it shares the block (R88). */
export const ROLLUP_ROWS_PER_PLAN = 2;
/** Rows the block holds at most; also what a single focused plan may fill. */
export const ROLLUP_MAX_ROWS = 6;

/** The item fields the roll-up reads. Structural, so the API type fits. */
export interface RollupItem extends MilestoneSourceItem {
  dueDate?: string | null;
  updatedAt?: string;
}

/** One plan's items, as `useHomeData` publishes them. */
export type RollupItemsState = {
  status: 'loading' | 'ready' | 'failed';
  items?: readonly RollupItem[];
};

/** The day plan fields the roll-up reads. Structural, so the API type fits. */
export interface RollupDayPlanSource {
  status: 'planned' | 'not_planned';
  source?: 'ai' | 'rules';
  groups: readonly {
    label: string;
    entries: readonly DayPlanEntrySource[];
  }[];
}

/** One plan's day plan read: in flight, landed, or failed. */
export type RollupDayPlanState =
  | { status: 'loading' }
  | { status: 'ready'; dayPlan: RollupDayPlanSource }
  | { status: 'failed' };

export interface RollupPlan {
  id: string;
  name: string;
}

export interface RollupRow {
  /** Unique across the block: plan id plus entry or item id. */
  key: string;
  planId: string;
  planName: string;
  text: string;
  /** The top-level ancestor's title, or null when the row is itself one. */
  milestoneTitle: string | null;
  /** The entry's day tick; always false on a rule pick. */
  ticked: boolean;
  /** Drawn by rules, by the server's basic plan or by the client (CONTEXT.md). */
  basic: boolean;
}

export interface DayPlanRollup {
  /** `loading` until every focused plan has its items and day plan settled. */
  status: 'loading' | 'ready';
  rows: RollupRow[];
}

const isOpenTask = (item: RollupItem) =>
  !item.archived && !item.done && (item.type ?? 'task') === 'task';

/** Ties keep plan order: `Array.prototype.sort` is stable. */
function rulePickIds(items: readonly RollupItem[]): string[] {
  const dated = items
    .filter(isOpenTask)
    .map((item) => ({ item, due: parseDateOnly(item.dueDate) }))
    .filter((d): d is { item: RollupItem; due: Date } => d.due !== null)
    .sort((a, b) => a.due.getTime() - b.due.getTime());
  // Overdue (most overdue first) then due soonest is one ascending sort on
  // the due date: everything past today already sorts ahead of today, so no
  // clock is needed to tell the two rules apart.
  const picks = dated.map((d) => d.item.id);

  const milestones = items
    .filter((item) => isOpenTask(item) && !item.parentId)
    .sort((a, b) => (b.updatedAt ?? '').localeCompare(a.updatedAt ?? ''));
  for (const milestone of milestones) {
    const step = items.find(
      (item) => item.parentId === milestone.id && isOpenTask(item),
    );
    picks.push((step ?? milestone).id);
  }

  return [...new Set(picks)];
}

function planRows(
  plan: RollupPlan,
  items: readonly RollupItem[],
  read: RollupDayPlanState,
  quota: number,
): RollupRow[] {
  const stored = read.status === 'ready' && read.dayPlan.status === 'planned';

  if (stored) {
    const { dayPlan } = read;
    return resolveDayPlan(dayPlan.groups, items)
      .flatMap((g) => g.entries)
      .slice(0, quota)
      .map((entry) => ({
        key: `${plan.id}:${entry.id}`,
        planId: plan.id,
        planName: plan.name,
        text: entry.text,
        milestoneTitle: entry.milestoneTitle,
        ticked: entry.ticked,
        basic: dayPlan.source === 'rules',
      }));
  }

  // Rule picks go through the same resolver as stored entries, so a pick's
  // milestone name is read exactly the way the section reads it.
  const picks = rulePickIds(items)
    .slice(0, quota)
    .map((id) => ({ id, kind: 'existing_item' as const, itemId: id, text: '', reason: '' }));
  return resolveDayPlan([{ label: '', entries: picks }], items)
    .flatMap((g) => g.entries)
    .map((entry) => ({
      key: `${plan.id}:rule:${entry.itemId}`,
      planId: plan.id,
      planName: plan.name,
      text: entry.text,
      milestoneTitle: entry.milestoneTitle,
      ticked: false,
      basic: true,
    }));
}

/**
 * The block's rows, plan by plan in focused order. `plans` is ADR-0013's
 * focused set, most recently updated first.
 */
export function selectDayPlanRollup(
  plans: readonly RollupPlan[],
  itemsByPlan: Readonly<Record<string, RollupItemsState | undefined>>,
  dayPlansByPlan: Readonly<Record<string, RollupDayPlanState | undefined>>,
): DayPlanRollup {
  // All or nothing, like Today: a block that grows plan by plan under the
  // reader is worse than one that arrives a moment late (FS-KSJFR R22).
  const pending = plans.some((plan) => {
    const items = itemsByPlan[plan.id];
    const read = dayPlansByPlan[plan.id];
    return !items || items.status === 'loading' || !read || read.status === 'loading';
  });
  if (pending) return { status: 'loading', rows: [] };

  const quota = plans.length === 1 ? ROLLUP_MAX_ROWS : ROLLUP_ROWS_PER_PLAN;
  const rows = plans.flatMap((plan) => {
    const items = itemsByPlan[plan.id]!;
    if (items.status !== 'ready') return [];
    return planRows(plan, items.items ?? [], dayPlansByPlan[plan.id]!, quota);
  });

  return { status: 'ready', rows: rows.slice(0, ROLLUP_MAX_ROWS) };
}
