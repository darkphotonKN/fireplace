"use client";

import { useMemo } from "react";
import Link from "next/link";
import type { Plan } from "@/api/plans";
import { progressOf } from "@/lib/homePlans";
import { selectHomeWork } from "@/lib/homeWork";
import FirstRun from "./FirstRun";
import OtherPlans from "./OtherPlans";
import StatusLine from "./StatusLine";
import TodayBlock from "./TodayBlock";
import { useHomeData, type ItemsState } from "./useHomeData";
import { useHomeTicks } from "./useHomeTicks";

/**
 * The signed-in dashboard (FS-KSJFR §The dashboard, slice 2).
 *
 * Block order is R19: the status line, then Today/Next up (I-KSJFR-4), then
 * Continue, then Your other plans (I-KSJFR-3). The status line and the block
 * are handed the same `work` object, so the sentence and the list can never
 * disagree about what is due — and the derivation behind it runs once per data
 * change rather than once per render, because `useHomeData` returns a
 * `focusedPlans` whose identity is stable. R49's one coral moment lives in
 * TodayBlock; everything else on this page stays neutral.
 *
 * Ticking (I-KSJFR-5) keeps that single derivation intact rather than working
 * around it: `useHomeTicks` overlays the visit's ticks onto the same `work`
 * object, so the sentence and the list still cannot disagree — see `withTicks`
 * for why the tick is an overlay and not a re-selection.
 *
 * Still owned by a later slice: the zero-plan invitation (I-KSJFR-7).
 *
 * No header card either (R18): the `backdrop-blur-sm rounded-2xl p-8 shadow-lg`
 * block stays the convention on /myplans and /plan/[planId], which are siblings
 * and should rhyme. Home's job is orientation, not content.
 */
export default function Dashboard({ onStartPlan }: { onStartPlan: () => void }) {
  const { plansStatus, plans, focusedPlans, itemsByPlan, reload } = useHomeData();
  const selected = useMemo(
    () => selectHomeWork(focusedPlans, itemsByPlan),
    [focusedPlans, itemsByPlan],
  );
  const { work, tick, tickedIds } = useHomeTicks(selected);

  return (
    <main className="min-h-screen p-8">
      <div className="mx-auto max-w-5xl space-y-12">
        {/* The status line renders in every state, the failed one included:
            §Edge States asks for its neutral form **and** the error state, and
            with no plan read it is already just the weekday — a fact the
            network cannot take away. Swapping it out for the error left the
            page with no statement of the day at all. */}
        <StatusLine work={work} />
        {plansStatus === "error" ? (
          <PlansUnavailable onRetry={reload} />
        ) : plansStatus === "ready" && plans.length === 0 ? (
          /* R43: one invitation, not three empty blocks. The status line above
             still says what day it is — that much is true before any plan
             exists, and it keeps the screen from opening on a bare heading. */
          <FirstRun onStart={onStartPlan} />
        ) : (
          <>
            <TodayBlock work={work} onRetry={reload} onTick={tick} />
            {focusedPlans.length > 0 && (
              <ContinueBlock
                plans={focusedPlans}
                itemsByPlan={itemsByPlan}
                tickedIds={tickedIds}
              />
            )}
            {/* Takes the FULL ordered list: it slices off the Continue three
                itself, so pre-slicing here would silently drop three plans. */}
            <OtherPlans plans={plans} />
          </>
        )}
      </div>
    </main>
  );
}

/**
 * The plans fetch failing costs the page, not the visit: a calm line and a way
 * to try again, never a blank screen (FS-KSJFR §Edge States). The status line
 * above it stays — this block replaces the content, not the day.
 *
 */
function PlansUnavailable({ onRetry }: { onRetry: () => void }) {
  return (
    <section className="space-y-4">
      <p className="text-base text-muted-foreground">
        We couldn&apos;t reach your plans just now.
      </p>
      <button
        type="button"
        onClick={onRetry}
        className="rounded-md border border-border px-4 py-2 text-base transition-colors hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        Try again
      </button>
    </section>
  );
}

function ContinueBlock({
  plans,
  itemsByPlan,
  tickedIds,
}: {
  plans: Plan[];
  itemsByPlan: Record<string, ItemsState>;
  tickedIds: readonly string[];
}) {
  return (
    <section className="space-y-6">
      <h2>Continue where you left off</h2>
      {/* Whatever exists fills the row — two plans are two cards, never two
          cards and an empty slot. */}
      <ul className="grid grid-cols-1 gap-4 md:grid-cols-3">
        {plans.map((plan) => (
          <ContinueCard
            key={plan.id}
            plan={plan}
            items={itemsByPlan[plan.id] ?? { status: "loading" }}
            tickedIds={tickedIds}
          />
        ))}
      </ul>
    </section>
  );
}

/**
 * One plan, with the progress the fan-out already paid for.
 *
 * The progress slot renders in every state, empty while the items are in
 * flight, so the number settles into a space the card already reserved rather
 * than growing the row under the cursor (R22). A plan whose items failed keeps
 * that slot empty for good: `0/0` there would claim the plan is empty when all
 * we know is that we couldn't ask.
 */
function ContinueCard({
  plan,
  items,
  tickedIds,
}: {
  plan: Plan;
  items: ItemsState;
  tickedIds: readonly string[];
}) {
  const progress =
    items.status === "ready" ? progressOf(items.items, tickedIds) : null;

  return (
    <li>
      <Link
        href={`/plan/${plan.id}`}
        className="flex h-full flex-col rounded-lg border border-border bg-card p-6 text-card-foreground shadow-sm transition-colors hover:border-foreground/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <span className="text-lg font-bold">{plan.name}</span>
        {plan.focus ? (
          <span className="mt-2 text-base text-muted-foreground">{plan.focus}</span>
        ) : null}
        <span className="mt-6 flex items-center justify-between gap-3 text-xs text-muted-foreground">
          <span className="rounded-full bg-muted px-3 py-1 capitalize">
            {plan.planType}
          </span>
          <span data-testid={`continue-progress-${plan.id}`}>
            {progress ? `${progress.done}/${progress.total}` : ""}
          </span>
        </span>
      </Link>
    </li>
  );
}
