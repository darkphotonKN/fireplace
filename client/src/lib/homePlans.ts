import type { Plan } from "@/api/plans";
import type { ChecklistItem } from "@/api/checklists";
import { readMilestones, type MilestoneReading } from "@/lib/milestones";

/**
 * How far home looks (ADR-0013).
 *
 * Home composes from the endpoints that already ship, fanning out over the
 * three most recently updated plans only: `GET /api/plans` plus three checklist
 * fetches, four requests bounded forever however many plans an account
 * accumulates. Raising this number is not a tuning knob — it is a different
 * contract with the user about what home shows, and ADR-0013 names
 * `GET /api/home` as the upgrade path rather than a bigger cap.
 */
export const FOCUSED_PLAN_COUNT = 3;

/**
 * How many quiet rows "Your other plans" shows before handing off to `/myplans`
 * (FS-KSJFR R40). Unlike `FOCUSED_PLAN_COUNT` this cap is about calm rather
 * than cost — the rows are already-fetched plan facts and cost no request — so
 * it can move without touching ADR-0013's budget. It lives beside the other cap
 * anyway: both answer "how much of the account does home show?", and that is
 * one question, not two.
 */
export const OTHER_PLANS_ROW_CAP = 6;

/**
 * Plans most recently updated first.
 *
 * "Recently updated" is ordering, not a claim about recency (FS-KSJFR R38):
 * three plans dormant for months still sort to the front, and there is
 * deliberately no staleness floor filtering them out.
 *
 * An `updatedAt` that does not parse sorts last rather than poisoning the
 * comparison — `NaN` comparisons are all false, which leaves a sort ordering
 * unspecified rather than merely wrong.
 */
export function byRecentlyUpdated(plans: Plan[]): Plan[] {
  const at = (plan: Plan) => {
    const ms = Date.parse(plan.updatedAt);
    return Number.isNaN(ms) ? -Infinity : ms;
  };
  return [...plans].sort((a, b) => at(b) - at(a));
}

export type Progress = { done: number; total: number };

/**
 * Done/total for a Continue card, derived from the items the fan-out already
 * fetched (R37). `GET /api/plans` carries no counts, so this is the only place
 * the number can come from without a fifth request.
 *
 * Notes are not work (R26), so they count towards neither half of the fraction —
 * the same rule the grid's collapsed count and `ChecklistGrid` already apply. An
 * item with no `type` at all is a task, as everywhere else.
 *
 * A plan with genuinely no items is `0/0` and says so. A plan whose item fetch
 * *failed* has no progress at all — that is the caller's distinction to make by
 * not calling this, because `0/0` there would be a confident lie.
 */
export function progressOf(
  items: ChecklistItem[],
  tickedIds: readonly string[] = [],
): Progress {
  const live = asOfThisVisit(items, tickedIds)
    .filter((item) => !item.archived)
    .filter((item) => (item.type ?? "task") === "task");
  return {
    done: live.filter((item) => item.done).length,
    total: live.length,
  };
}

/**
 * What a Continue card shows in its progress slot: the plan's milestones when
 * it has any (FS-FSFEC R23), else done/total exactly as before (R26).
 *
 * `readMilestones` applies R1 itself, scope included — which matters here more
 * than inside `Todo`, because the fan-out carries every scope and a daily-only
 * plan must not grow a strip (R24).
 */
export type CardProgress =
  | { kind: "milestones"; reading: MilestoneReading }
  | ({ kind: "count" } & Progress);

export function cardProgressOf(
  items: ChecklistItem[],
  tickedIds: readonly string[] = [],
): CardProgress {
  const reading = readMilestones(asOfThisVisit(items, tickedIds));
  if (reading.total > 0) return { kind: "milestones", reading };
  return { kind: "count", ...progressOf(items, tickedIds) };
}

/**
 * The plan's items with this visit's ticks from Today laid over them.
 *
 * A row ticked from Today counts as done on its Continue card too. Without it
 * the card keeps its pre-tick number while Today shows the row struck through
 * and the status line has already decremented — two blocks on one screen
 * disagreeing about one plan, which is the exact thing the single-derivation
 * rule exists to prevent one level up. Both readings of the card go through
 * here, so the strip and the count can't disagree either (FS-FSFEC R25).
 */
function asOfThisVisit(
  items: ChecklistItem[],
  tickedIds: readonly string[],
): ChecklistItem[] {
  if (tickedIds.length === 0) return items;
  return items.map((item) =>
    !item.done && tickedIds.includes(item.id) ? { ...item, done: true } : item,
  );
}
