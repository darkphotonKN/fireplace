"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { listPlans, type Plan } from "@/api/plans";
import { listChecklists, type ChecklistItem } from "@/api/checklists";
import { getDayPlan, type DayPlan } from "@/api/insights";
import { FOCUSED_PLAN_COUNT, byRecentlyUpdated } from "@/lib/homePlans";
import { toDateOnly } from "@/lib/itemDates";

/**
 * One focused plan's items, as three states rather than a nullable array.
 *
 * "Still loading" and "failed" have to stay distinguishable from "loaded and
 * empty", because the three render differently on a Continue card: nothing yet,
 * nothing ever, and `0/0`. A `ChecklistItem[] | null` collapses two of those.
 */
export type ItemsState =
  | { status: "loading" }
  | { status: "ready"; items: ChecklistItem[] }
  | { status: "failed" };

/**
 * One focused plan's day plan read (FS-02JBZ R85). `failed` stays distinct
 * from a `not_planned` day: both get rule picks on home (R87), but only one
 * of them is a fact about the user's day.
 */
export type DayPlanState =
  | { status: "loading" }
  | { status: "ready"; dayPlan: DayPlan }
  | { status: "failed" };

export type HomeData = {
  /** Whether `GET /api/plans` — the fetch everything else waits on — has landed. */
  plansStatus: "loading" | "ready" | "error";
  /** Every plan the account holds, most recently updated first. */
  plans: Plan[];
  /**
   * The top three: the only plans whose items home fetches (ADR-0013). The
   * array keeps its identity until `plans` changes, so callers can use it as a
   * `useMemo` dependency.
   */
  focusedPlans: Plan[];
  /** Items keyed by plan id, present only for the focused plans. */
  itemsByPlan: Record<string, ItemsState>;
  /** Cached day plans keyed by plan id, present only for the focused plans. */
  dayPlansByPlan: Record<string, DayPlanState>;
  /** Re-runs the whole fan-out. Wired to the retry on the error state. */
  reload: () => void;
};

/**
 * The capped fan-out home is assembled from (ADR-0013, FS-KSJFR R20–R22,
 * FS-02JBZ R85/R90).
 *
 * `GET /api/plans`, sorted by `updatedAt`, then for the top three their
 * checklists and their cached day plans for the browser-local date — **seven
 * requests** whatever the account holds. ADR-0013's bound rose from 4 to 7
 * with the day plan roll-up (FS-02JBZ R90); its decision holds: the same three
 * plans, only shipped endpoints, no home aggregate. The day plan read is a
 * `GET`, which never generates (ADR-0019): home NEVER calls `planMyDay`.
 *
 * The six per-plan requests are issued together and settled independently:
 * one rejecting marks that plan's slot `failed` and leaves the others alone,
 * because a single `Promise.all` would throw away good responses to report
 * one bad one. A failed day plan read costs that plan nothing worse than rule
 * picks (R87).
 *
 * The plan-derived state lands first and on its own, which is what makes the
 * page paint progressively instead of behind a full-page spinner: callers can
 * render every plan fact the moment `plansStatus` is `ready`, with the
 * item-derived parts settling in after.
 */
export function useHomeData(): HomeData {
  const [attempt, setAttempt] = useState(0);
  const [plansStatus, setPlansStatus] = useState<HomeData["plansStatus"]>("loading");
  const [plans, setPlans] = useState<Plan[]>([]);
  const [itemsByPlan, setItemsByPlan] = useState<Record<string, ItemsState>>({});
  const [dayPlansByPlan, setDayPlansByPlan] = useState<Record<string, DayPlanState>>({});

  useEffect(() => {
    let cancelled = false;
    setPlansStatus("loading");
    setPlans([]);
    setItemsByPlan({});
    setDayPlansByPlan({});

    const run = async () => {
      let ordered: Plan[];
      try {
        ordered = byRecentlyUpdated((await listPlans()) ?? []);
      } catch {
        if (!cancelled) setPlansStatus("error");
        return;
      }
      if (cancelled) return;

      const focused = ordered.slice(0, FOCUSED_PLAN_COUNT);
      setPlans(ordered);
      setItemsByPlan(
        Object.fromEntries(
          focused.map((plan) => [plan.id, { status: "loading" } as ItemsState]),
        ),
      );
      setDayPlansByPlan(
        Object.fromEntries(
          focused.map((plan) => [plan.id, { status: "loading" } as DayPlanState]),
        ),
      );
      setPlansStatus("ready");

      // Issued in one pass, settled one at a time. Each result is merged into
      // its own key so a slow plan never gates a fast one.
      const record = (id: string, state: ItemsState) => {
        if (cancelled) return;
        setItemsByPlan((prev) => ({ ...prev, [id]: state }));
      };
      const recordDayPlan = (id: string, state: DayPlanState) => {
        if (cancelled) return;
        setDayPlansByPlan((prev) => ({ ...prev, [id]: state }));
      };
      // The browser-local day, the same date the plan page's section reads.
      const today = toDateOnly(new Date())!;
      focused.forEach((plan) => {
        listChecklists(plan.id).then(
          (items) => record(plan.id, { status: "ready", items: items ?? [] }),
          () => record(plan.id, { status: "failed" }),
        );
        getDayPlan(plan.id, today).then(
          (dayPlan) => recordDayPlan(plan.id, { status: "ready", dayPlan }),
          () => recordDayPlan(plan.id, { status: "failed" }),
        );
      });
    };

    run();
    return () => {
      cancelled = true;
    };
  }, [attempt]);

  const reload = useCallback(() => setAttempt((n) => n + 1), []);

  // Memoised because callers use it as a dependency: `plans.slice(...)` inline
  // would hand back a new array on every render, and a `useMemo` keyed on that
  // recomputes every time — which is exactly the derivation Dashboard is trying
  // to do once per data change and share between the status line and the block.
  const focusedPlans = useMemo(
    () => plans.slice(0, FOCUSED_PLAN_COUNT),
    [plans],
  );

  return {
    plansStatus,
    plans,
    focusedPlans,
    itemsByPlan,
    dayPlansByPlan,
    reload,
  };
}
