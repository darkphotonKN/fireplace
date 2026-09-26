"use client";

import Link from "next/link";
import { Circle, CheckCircle2 } from "lucide-react";
import type { HomeWork, WorkRow } from "@/lib/homeWork";

/**
 * Today — or Next up when nothing is due (FS-KSJFR R23–R32, D5, D14).
 *
 * One block with two headings, never two blocks: the heading always describes
 * what is actually listed, so "Today" never sits above rows that aren't due
 * (R25). Which rows those are is `selectHomeWork`'s decision; this file only
 * renders them.
 *
 * **The page's one coral moment (R49), and why it is an `h1`.** The dashboard's
 * accent is spent here and nowhere else: this heading, and the date on a row
 * whose last day has gone — the same `primary/70` past-date treatment
 * `ItemDateChip` already uses on plan pages. Continue cards, plan rows and the
 * status line stay neutral.
 *
 * The heading is an `h1` because `client/docs/design-guideline.md` reserves
 * coral for the `h1` — one per view — and gives `h2`/`h3` the foreground,
 * which R47 makes binding over this spec. `globals.css` still hands `h1` the
 * coral unconditionally, in both themes, which is the whole point of spending
 * the accent here. It is also the dashboard's subject and the only heading on a
 * page that otherwise has no `h1` at all. (The focus prompt's greeting is the
 * other state of `/` and never renders beside this one, so there is no second
 * `h1`.) That choice was made on the guideline's terms and outlives I-0061's
 * cascade fix.
 *
 * Muted lines are plain `<p>`s carrying `text-muted-foreground`. They used to
 * be `<div>`s, because `.dark p` outranked the utility and R50 forbids patching
 * that per component; I-0061 moved the global element rules into `@layer base`,
 * so the token wins in both themes and the semantic element is back.
 *
 * A row's marker is the tick (I-KSJFR-5, R33). This file only reports the
 * click: whose row it was, and that it happened. Whether the row now reads done
 * is `work`'s business, and `work` is a derived value — see `withTicks` for why
 * the ticked row stays put instead of vanishing.
 */
export default function TodayBlock({
  work,
  onRetry,
  onTick,
}: {
  work: HomeWork;
  onRetry?: () => void;
  onTick: (row: WorkRow) => void;
}) {
  // Still fetching: a quiet line rather than a heading that would have to be
  // rewritten a moment later, or a spinner (R22).
  if (work.status === "loading") {
    return (
      <section className="space-y-6">
        <p className="text-base text-muted-foreground">
          Gathering today&apos;s work…
        </p>
      </section>
    );
  }

  // Every item fetch failed. Home knows nothing about today, which is not the
  // same as knowing today is clear — so the block says so and offers the way
  // back, rather than an empty list the reader would take as fact
  // (§Edge States).
  if (work.status === "failed") {
    return (
      <section className="space-y-4">
        <p className="text-base text-muted-foreground">
          We couldn&apos;t load today&apos;s work.
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

  // Nothing was read at all — no plans, or the plans fetch failed. The shell
  // owns those screens (the invitation, the retry); this block steps aside.
  if (work.planCount === 0) return null;

  if (work.rows.length === 0) {
    return (
      <section className="space-y-6">
        <p className="text-base text-muted-foreground">
          Nothing waiting yet — open a plan and add what&apos;s next.
        </p>
      </section>
    );
  }

  return (
    <section className="space-y-6" aria-labelledby="home-work-heading">
      <h1 id="home-work-heading" className="text-primary">
        {work.mode === "today" ? "Today" : "Next up"}
      </h1>
      <ul className="divide-y divide-border">
        {work.rows.map((row) => (
          <WorkRowItem key={row.id} row={row} onTick={onTick} />
        ))}
      </ul>
    </section>
  );
}

/**
 * One line of work, readable out of context (R29–R30): what it is, whose child
 * it is, which plan it came from, and when it runs out.
 *
 * The parent is the immediate one only. A grandchild reads "in call them
 * back", not a chain from the top of the tree — a breadcrumb that grows with
 * depth stops being context and becomes noise (§Edge States).
 *
 * **The done state costs no layout (R51).** Both markers are the same 4x4 box
 * and the text keeps its weight and size, so a tick settles in colour and
 * opacity alone — nothing under the cursor moves. Unticking from home is not a
 * thing the spec asks for, and a done row's marker is inert accordingly: it also
 * means a second click cannot fire a second write for a row already ticked.
 */
function WorkRowItem({
  row,
  onTick,
}: {
  row: WorkRow;
  onTick: (row: WorkRow) => void;
}) {
  const Marker = row.done ? CheckCircle2 : Circle;
  const marker = <Marker aria-hidden strokeWidth={1.75} className="h-4 w-4" />;

  return (
    <li className="flex items-start gap-3 py-3">
      {/*
        `aria-disabled`, deliberately not `disabled`. A real `disabled` blurs the
        control the instant it is activated, so a keyboard user is thrown back to
        the top of the document after every tick — Tab from a just-ticked row
        should land on the next row's tick, not restart. The second-write guard
        that `disabled` would give us is already held in `useHomeTicks` (it
        returns early on an already-done row), so nothing is lost by keeping
        focus where the user put it.
      */}
      <button
        type="button"
        role="checkbox"
        aria-checked={row.done}
        aria-disabled={row.done}
        aria-label={row.description}
        onClick={() => {
          // The component refuses the second write itself rather than relying on
          // the hook to ignore it — dropping `disabled` (see above) removed the
          // browser's own guard, and a control that reports `aria-disabled`
          // should behave like it.
          if (!row.done) onTick(row);
        }}
        className="mt-1.5 shrink-0 rounded-full text-muted-foreground/60 transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring aria-disabled:cursor-default aria-disabled:hover:text-muted-foreground/60"
      >
        {marker}
      </button>
      <div className="min-w-0 space-y-1">
        <p
          className={`text-base transition-opacity ${
            row.done ? "line-through opacity-60" : "opacity-100"
          }`}
        >
          {row.description}
        </p>
        <p className="text-xs text-muted-foreground">
          {row.parentDescription ? <>in {row.parentDescription} · </> : null}
          <Link
            href={`/plan/${row.planId}`}
            className="rounded-sm underline-offset-4 transition-colors hover:text-foreground hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            {row.planName}
          </Link>
          {row.dateLabel ? (
            <>
              {" · "}
              <span className={row.overdue ? "text-primary/70" : undefined}>
                {row.dateLabel}
              </span>
            </>
          ) : null}
        </p>
      </div>
    </li>
  );
}
