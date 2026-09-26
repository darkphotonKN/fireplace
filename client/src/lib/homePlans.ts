import type { Plan } from "@/api/plans";
import type { ChecklistItem } from "@/api/checklists";

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
  const live = items
    .filter((item) => !item.archived)
    .filter((item) => (item.type ?? "task") === "task");
  return {
    // A row ticked from Today during this visit counts here too. Without it the
    // Continue card for that plan keeps its pre-tick number while Today shows
    // the row struck through and the status line has already decremented —
    // two blocks on one screen disagreeing about one plan, which is the exact
    // thing the single-derivation rule exists to prevent one level up.
    done: live.filter((item) => item.done || tickedIds.includes(item.id)).length,
    total: live.length,
  };
}
