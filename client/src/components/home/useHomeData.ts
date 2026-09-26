"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { listPlans, type Plan } from "@/api/plans";
import { listChecklists, type ChecklistItem } from "@/api/checklists";
import { FOCUSED_PLAN_COUNT, byRecentlyUpdated } from "@/lib/homePlans";

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
  /** Re-runs the whole fan-out. Wired to the retry on the error state. */
  reload: () => void;
};

/**
 * The capped fan-out home is assembled from (ADR-0013, FS-KSJFR R20–R22).
 *
 * `GET /api/plans`, sorted by `updatedAt`, then checklists for the top three —
 * four requests whatever the account holds. The three item fetches are issued
 * together and settled independently: one rejecting marks that plan `failed`
 * and leaves the other two alone, because a single `Promise.all` would throw
 * away two good responses to report one bad one.
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

  useEffect(() => {
    let cancelled = false;
    setPlansStatus("loading");
    setPlans([]);
    setItemsByPlan({});

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
      setPlansStatus("ready");

      // Issued in one pass, settled one at a time. Each result is merged into
      // its own key so a slow plan never gates a fast one.
      const record = (id: string, state: ItemsState) => {
        if (cancelled) return;
        setItemsByPlan((prev) => ({ ...prev, [id]: state }));
      };
      focused.forEach((plan) => {
        listChecklists(plan.id).then(
          (items) => record(plan.id, { status: "ready", items: items ?? [] }),
          () => record(plan.id, { status: "failed" }),
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
    reload,
  };
}
