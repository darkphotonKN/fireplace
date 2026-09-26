"use client";

import Link from "next/link";
import type { Plan } from "@/api/plans";
import { FOCUSED_PLAN_COUNT, OTHER_PLANS_ROW_CAP } from "@/lib/homePlans";

/**
 * Your other plans — everything Continue isn't showing, one line each
 * (FS-KSJFR R39–R42, slice 3).
 *
 * Three things here are decisions, not defaults:
 *
 * The rows are **disjoint** from the Continue cards (R39/D6). The list could
 * have stayed a complete index with the cards as emphasis, but on a four-plan
 * account the repetition reads as a rendering bug, so every plan appears on
 * this page exactly once.
 *
 * Rows carry **name and type only** (R40). Progress is not missing by
 * oversight: `GET /api/plans` carries no counts, so a done/total here would
 * cost one checklist fetch per row and break ADR-0013's four-request cap. That
 * is precisely the cap this component exists to respect — it takes its plans as
 * props and issues no request of its own.
 *
 * At three plans or fewer it renders **nothing at all** (R41) — not a heading
 * over an empty list, not an empty state. Continue is already showing every
 * plan the account has, so there is no "rest" to name.
 *
 * Visually neutral on purpose (R49): the page's one coral moment belongs to
 * Today/Next up, so these rows are border, muted and foreground only.
 */
export default function OtherPlans({ plans }: { plans: Plan[] }) {
  // `plans` arrives most recently updated first, so the first three are exactly
  // the Continue cards and dropping them is the whole disjointness rule.
  const rest = plans.slice(FOCUSED_PLAN_COUNT);
  if (rest.length === 0) return null;

  const rows = rest.slice(0, OTHER_PLANS_ROW_CAP);
  const hidden = rest.length - rows.length;

  return (
    <section className="space-y-4" aria-labelledby="home-other-plans-heading">
      <h2 id="home-other-plans-heading">Your other plans</h2>
      <ul className="divide-y divide-border border-y border-border">
        {rows.map((plan) => (
          <li key={plan.id}>
            <Link
              href={`/plan/${plan.id}`}
              className="flex items-center justify-between gap-4 px-2 py-3 text-base transition-colors hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <span className="truncate">{plan.name}</span>
              {/* Labelled the way `/myplans` labels it — this row links there,
                  and a plan should not change its type's name on the way. A
                  plan with no type (search hits carry none) shows no label at
                  all rather than an empty slot. */}
              {plan.planType && (
                <span className="shrink-0 text-xs text-muted-foreground">
                  {plan.planType === "project" ? "Project" : "Learning"}
                </span>
              )}
            </Link>
          </li>
        ))}
      </ul>
      {/* Only when something is actually hidden: at nine plans the six rows are
          the whole remainder, and a link out would point at nothing new. */}
      {hidden > 0 && (
        <Link
          href="/myplans"
          className="inline-block text-base text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          See all plans
        </Link>
      )}
    </section>
  );
}
