"use client";

import { format } from "date-fns";
import type { HomeWork } from "@/lib/homeWork";

/**
 * The dashboard's first line (FS-KSJFR R16–R17, D8).
 *
 * Not a greeting — the focus prompt owns the name — and not a card. One line
 * of prose that says what day it is, how much is due, and **how far home
 * looked** to find it. That last clause is ADR-0013's three-plan cap surfaced
 * in the product: a due item in a fourth, dormant plan is then an explainable
 * absence rather than a bug report (R17).
 *
 * It reads the same `HomeWork` the block below it renders, so the sentence and
 * the list can never disagree.
 *
 * Neutral by construction: R49 spends the page's one coral moment on the
 * Today block, and a status line in coral would spend it twice.
 *
 * It is a plain `<p>` carrying `text-muted-foreground`, which is all R50 ever
 * wanted: the global `p` rule now lives in `@layer base`, so the utility wins
 * in both themes (I-0061). Until it did, this line had to be a `<div>` to dodge
 * `.dark p`.
 */
export default function StatusLine({ work }: { work: HomeWork }) {
  const weekday = format(new Date(), "EEEE");
  const detail = describe(work);

  return (
    <p className="text-lg text-muted-foreground">
      {detail ? `${weekday} · ${detail}` : weekday}
    </p>
  );
}

/**
 * The clause after the weekday, or nothing at all.
 *
 * Nothing is the honest answer three times over: while the fan-out is still in
 * flight, when every item fetch failed, and when no plan was read at all.
 * "Nothing due" would be a claim about a day we haven't read yet, so the line
 * stays a weekday until it can say more (§Edge States: a failed
 * `GET /api/plans` leaves the neutral form, and so does losing all three item
 * fetches).
 */
function describe(work: HomeWork): string | null {
  if (work.status !== "ready" || work.planCount === 0) return null;

  const plans = `across ${count(work.planCount, "plan")}`;
  // Ticking the last row leaves the block in `today` mode (R35 — the heading
  // does not swap mid-visit), so `today` has to be able to say zero. "0 due" is
  // phrasing that appears nowhere else in the product, and this is the sentence
  // the user is left looking at for the rest of the visit.
  if (work.mode === "today") {
    return work.dueCount === 0
      ? `nothing due ${plans}`
      : `${work.dueCount} due ${plans}`;
  }
  if (work.waitingCount === 0) return `nothing due ${plans}`;
  return `nothing due — ${work.waitingCount} waiting ${plans}`;
}

function count(n: number, noun: string): string {
  return `${n} ${noun}${n === 1 ? "" : "s"}`;
}
