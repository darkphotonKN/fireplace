"use client";

/**
 * What a brand-new account sees on home (FS-KSJFR R43–R44, D13).
 *
 * Without this, the worst screen in the product is the first one after signing
 * up: the day gate fires because nothing has been touched, the user skips, and
 * three hollow blocks appear — a Today with nothing due, a Continue with
 * nothing to continue, and a plan list with no plans. One invitation replaces
 * all three.
 *
 * It says what a plan *is*, because at this moment the user has no way to know,
 * and it says what home becomes once they have one — the payoff is the reason
 * to start, and an empty dashboard cannot show it.
 *
 * It does not navigate (R44): the action puts the focus prompt back on screen,
 * in place, which is the same one-route-two-states shape as D1. And it does not
 * restate the logged-out tour — that surface sells the product to someone who
 * has not signed up; this one is for someone who already did.
 */
export default function FirstRun({ onStart }: { onStart: () => void }) {
  return (
    <section
      aria-labelledby="first-run-heading"
      className="max-w-xl space-y-4 py-8"
    >
      {/* Coral by the global `h1` rule. It is the only heading on this screen,
          which is also how the page keeps exactly one `h1`. */}
      <h1 id="first-run-heading">Your first plan</h1>
      <p className="text-lg">
        A plan is one thing you&apos;re building or learning, kept as the steps
        that get you there.
      </p>
      <p className="text-base text-muted-foreground">
        Start one and this page fills in — what&apos;s due today, and where you
        left off.
      </p>
      <button
        type="button"
        onClick={onStart}
        className="mt-2 rounded-lg bg-primary px-6 py-3 text-base font-semibold text-primary-foreground transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        Start a plan
      </button>
    </section>
  );
}
