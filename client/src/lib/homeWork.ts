import type { Plan } from "@/api/plans";
import type { ChecklistItem } from "@/api/checklists";
import {
  formatDateRange,
  isDueToday,
  isRangePast,
  itemStartDate,
} from "./itemDates";

/**
 * What home has to show you, derived once from the capped fan-out
 * (FS-KSJFR R23–R32, ADR-0013).
 *
 * This is the one place the Today/Next up question is answered, because two
 * surfaces ask it: the block that lists the rows and the status line that
 * counts them (R16). Deriving it twice would let the sentence and the list
 * disagree the moment either rule moved.
 *
 * Pure on purpose — it takes what `useHomeData` already fetched and returns a
 * plain value. Nothing here fetches, and nothing here knows about React.
 */

/** How many rows Next up shows: "the first few", fixed (R24). */
export const NEXT_UP_ROWS = 5;

/**
 * One focused plan's items as the fan-out publishes them — structurally the
 * `ItemsState` of `useHomeData`, restated here so a pure module doesn't depend
 * on a hook. The three states stay three: "still loading" and "failed" are not
 * "no work" (see the note on `ItemsState`).
 */
export type PlanItems = {
  status: "loading" | "ready" | "failed";
  items?: ChecklistItem[];
};

/** One line of work, with everything the row needs to be readable out of context. */
export type WorkRow = {
  id: string;
  description: string;
  planId: string;
  /** The plan's name, which the row links into (R30). */
  planName: string;
  /** The immediate parent's text, so a sub-item isn't orphaned (R29). */
  parentDescription: string | null;
  /** The item's dates as a label, when it has any. */
  dateLabel: string | null;
  /** Whether its last day has gone — the only urgency the row carries. */
  overdue: boolean;
  /**
   * Ticked off during this visit (R33). Always false out of the selector, which
   * only ever emits unfinished work: it is `withTicks` below that turns it on,
   * and only ever for the visit.
   */
  done: boolean;
};

export type HomeWork = {
  /**
   * `loading` until every focused plan has landed one way or the other. The
   * counts are an aggregate over all three, and an aggregate that grows under
   * the reader is worse than one that arrives a moment late (R22).
   *
   * `failed` is the third outcome, and it is not the same as an empty day:
   * plans were fetched for and **every one** of them came back a failure, so
   * home knows nothing about today rather than knowing today is clear
   * (§Edge States, all three item fetches fail). An account with no plans at
   * all is `ready` — there was nothing to fail.
   */
  status: "loading" | "ready" | "failed";
  /** Which question the block is answering, and therefore its heading (R25). */
  mode: "today" | "nextUp";
  rows: WorkRow[];
  /** Rows in the Today case; zero otherwise. */
  dueCount: number;
  /** Everything unchecked across the plans that were read, for the Next-up line. */
  waitingCount: number;
  /** How many plans the counts were drawn from — ADR-0013's cap, made visible (R17). */
  planCount: number;
};

/** Notes are not work (R26), and archived items are nowhere on home (R27). */
function isWork(item: ChecklistItem): boolean {
  return !item.archived && (item.type ?? "task") !== "note";
}

/** Unfinished work — the pool both modes draw from (R28). */
function isWaiting(item: ChecklistItem): boolean {
  return isWork(item) && !item.done;
}

/**
 * Today's rule, in one place: due today, overdue, or a daily-scope habit
 * (R23). The three are a single filter rather than three concatenated lists,
 * which is why an item that is both due today and daily appears once.
 */
function isTodays(item: ChecklistItem): boolean {
  const start = itemStartDate(item);
  return (
    isDueToday(start, item.dueDate) ||
    isRangePast(start, item.dueDate) ||
    item.scope === "daily"
  );
}

/** One plan's items, ready or not. The array order is the server's `sequence` order. */
type ReadyPlan = { plan: Plan; items: ChecklistItem[] };

function toRow(item: ChecklistItem, { plan, items }: ReadyPlan): WorkRow {
  // The immediate parent only: a grandchild says "in child", never
  // "in parent › child" (R29, §Edge States).
  const parent = item.parentId
    ? (items.find((candidate) => candidate.id === item.parentId) ?? null)
    : null;
  const start = itemStartDate(item);

  return {
    id: item.id,
    description: item.description,
    planId: plan.id,
    planName: plan.name,
    parentDescription: parent?.description ?? null,
    dateLabel: formatDateRange(start, item.dueDate),
    overdue: isRangePast(start, item.dueDate),
    done: false,
  };
}

/**
 * Today, or Next up when Today is empty (R24, D5).
 *
 * `plans` is the focused set — the three the fan-out actually fetched, most
 * recently updated first. A plan whose items failed is left out rather than
 * counted as empty: the status line would otherwise report a smaller day than
 * the user has and call it fact (§Edge States).
 */
export function selectHomeWork(
  plans: Plan[],
  itemsByPlan: Record<string, PlanItems>,
): HomeWork {
  const states = plans.map((plan) => ({ plan, state: itemsByPlan[plan.id] }));
  const pending = states.some(
    ({ state }) => !state || state.status === "loading",
  );

  const read: ReadyPlan[] = states
    .filter(({ state }) => state?.status === "ready")
    .map(({ plan, state }) => ({ plan, items: state!.items ?? [] }));

  // Asked for, and nothing came back: the block owes the reader a retry, and
  // the status line owes them silence about a day it never read. An account
  // with no plans is not this — `states` is empty and nothing failed.
  const failed = !pending && states.length > 0 && read.length === 0;

  const empty: HomeWork = {
    status: pending ? "loading" : failed ? "failed" : "ready",
    mode: "today",
    rows: [],
    dueCount: 0,
    waitingCount: 0,
    planCount: read.length,
  };
  if (pending || failed) return empty;

  const today = read.flatMap((readyPlan) =>
    readyPlan.items
      .filter((item) => isWaiting(item) && isTodays(item))
      .map((item) => toRow(item, readyPlan)),
  );
  if (today.length > 0) {
    return { ...empty, rows: today, dueCount: today.length };
  }

  // Nothing is due, so the block changes its question. The rows come from one
  // plan — the most recent one with anything waiting — while the count spans
  // every plan that was read, because that is what "6 waiting" means.
  const source = read.find((readyPlan) => readyPlan.items.some(isWaiting));
  const rows = source
    ? source.items
        .filter(isWaiting)
        .slice(0, NEXT_UP_ROWS)
        .map((item) => toRow(item, source))
    : [];

  return {
    ...empty,
    mode: "nextUp",
    rows,
    waitingCount: read.reduce(
      (total, readyPlan) => total + readyPlan.items.filter(isWaiting).length,
      0,
    ),
  };
}

/**
 * The same `HomeWork`, with the rows ticked off during this visit marked done
 * and the counts spent accordingly (R33–R35, D9).
 *
 * **This is an overlay, not a re-selection, and that is the point of the
 * slice.** D9 was refined during spec-writing: a ticked row does *not*
 * disappear. It stays exactly where it is, done and dimmed, for the rest of the
 * visit. Feeding the tick back into `selectHomeWork` would drop the row —
 * `isWaiting` excludes it — collapsing the list under the user's cursor and
 * turning the next tick into a mis-click. Worse, finishing the last Today row
 * would flip `mode` to `nextUp` and yank in unrelated rows from another plan
 * mid-interaction, which R35 forbids outright. So the selection is left alone
 * and the visit's ticks are painted on top of it.
 *
 * Because the done set is the *input*, not accumulated state, concurrency comes
 * free: two ticks in flight put two ids in, and one failing takes only its own
 * id out — restoring its own row and its own share of the count and nothing
 * else (§Edge States, two ticks in flight).
 *
 * Both counts come down, because a ticked row was both due and waiting. In
 * practice only one of them is ever non-zero (`dueCount` in Today mode,
 * `waitingCount` in Next up), and neither goes below zero: an id matching no
 * row spends nothing, and a row already done cannot be spent twice.
 *
 * Returns the argument unchanged when nothing landed, so the object both
 * `StatusLine` and `TodayBlock` read keeps its identity across an untouched
 * visit.
 */
export function withTicks(work: HomeWork, tickedIds: readonly string[]): HomeWork {
  if (tickedIds.length === 0) return work;

  let spent = 0;
  const rows = work.rows.map((row) => {
    if (row.done || !tickedIds.includes(row.id)) return row;
    spent += 1;
    return { ...row, done: true };
  });
  if (spent === 0) return work;

  return {
    ...work,
    rows,
    dueCount: Math.max(0, work.dueCount - spent),
    waitingCount: Math.max(0, work.waitingCount - spent),
  };
}
