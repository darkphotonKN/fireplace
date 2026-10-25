"use client";

import Link from "next/link";
import { Circle, CheckCircle2 } from "lucide-react";
import type { DayPlanRollup as Rollup, RollupRow } from "@/lib/dayPlanRollup";

/**
 * The day plan roll-up (FS-02JBZ R85–R89, S11): a few entries from each
 * focused plan's day, above the Today block.
 *
 * Read-only by design (D10): no tick, no Add, no Plan my day. Each row is a
 * link into its plan, where the day plan can actually be worked. The marker
 * is a picture of the day tick, not a control. Which rows show is
 * `selectDayPlanRollup`'s decision; this file only renders them.
 *
 * Nothing renders while the reads are in flight or when no plan has anything
 * to show (R89): an empty or placeholder block above Today would push the
 * page's one coral moment down for nothing. Neutral tokens only, for the same
 * reason: coral on this page belongs to Today (FS-KSJFR R49).
 */
export default function DayPlanRollup({ rollup }: { rollup: Rollup }) {
  if (rollup.status !== "ready" || rollup.rows.length === 0) return null;

  return (
    <section className="space-y-6" aria-labelledby="home-day-plan-heading">
      <h2 id="home-day-plan-heading">Day plan</h2>
      <ul className="divide-y divide-border">
        {rollup.rows.map((row) => (
          <RollupRowItem key={row.key} row={row} />
        ))}
      </ul>
    </section>
  );
}

function RollupRowItem({ row }: { row: RollupRow }) {
  const Marker = row.ticked ? CheckCircle2 : Circle;

  return (
    <li>
      <Link
        href={`/plan/${row.planId}`}
        className="flex items-start gap-3 rounded-sm py-3 transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <Marker
          aria-hidden
          strokeWidth={1.75}
          className="mt-1.5 h-4 w-4 shrink-0 text-muted-foreground/60"
        />
        <span className="min-w-0 space-y-1">
          <span
            className={`block text-base ${row.ticked ? "line-through opacity-60" : ""}`}
          >
            {row.text}
            {row.ticked ? <span className="sr-only"> (done for today)</span> : null}
          </span>
          <span className="block text-xs text-muted-foreground">
            {row.milestoneTitle ? <>{row.milestoneTitle} · </> : null}
            {row.planName}
            {row.basic ? (
              <>
                {" · "}
                <span
                  title="Picked by simple rules, not the AI"
                  className="rounded-full bg-muted px-2 py-0.5"
                >
                  basic
                </span>
              </>
            ) : null}
          </span>
        </span>
      </Link>
    </li>
  );
}
