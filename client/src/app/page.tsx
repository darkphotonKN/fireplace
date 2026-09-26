"use client";

import { useEffect, useState } from "react";
import { useAuth } from "@/context/AuthContext";
import LandingTour from "@/components/landing/LandingTour";
import Dashboard from "@/components/home/Dashboard";
import FocusPrompt from "@/components/home/FocusPrompt";
import { hasGateFiredToday, markGateFiredToday, wasTouchedToday } from "@/lib/touchedToday";

/**
 * The day gate's rule, whole (FS-KSJFR R3): the prompt opens `/` only on the
 * first visit of the browser-local day, and only if nothing has been touched
 * yet. The idle half is not decoration — the prompt *creates a plan*, so a
 * plain daily cadence would ask a user three weeks into a project to start
 * something new every morning (D2).
 *
 * Both reads swallow a hostile storage and answer `false` (R15), so absent,
 * malformed and throwing all land here as "fires": the gate errs toward one
 * extra skip click, never toward an error in a render path.
 */
const dayGateFires = (): boolean => !hasGateFiredToday() && !wasTouchedToday();

/**
 * `/` signed in: one route in two states (FS-KSJFR D1, R2).
 *
 * Skipping the prompt swaps the dashboard in beneath it — React state, not a
 * `router.push`, so there is no URL change, no redirect flash and no back-button
 * trap. The cost of that choice is here in plain sight: the dashboard's fan-out
 * doesn't start until the state flips, which is exactly why the prompt is not
 * overlaid on a live dashboard (a visit where the prompt is the only thing
 * touched pays for no fetch at all).
 *
 * THE DECISION IS MADE IN THE RENDER PHASE, and the lazy initialiser below is
 * the whole of why (R4). An effect would paint one state and then correct it —
 * the flash the spec forbids, invisible to any test that only reads the settled
 * DOM. `useState` with no setter is the shape that says it: read once per mount,
 * from `localStorage` alone, never again. Which also satisfies R6 for free — the
 * gate cannot re-evaluate on focus or visibility, because there is no code path
 * that re-reads it. A tab left open across midnight keeps the state it has until
 * it reloads; that is an accepted limit, not an oversight.
 *
 * Recording the fire is the one part that belongs in an effect: it is a write,
 * and the decision no longer depends on it. It happens on mount rather than on
 * skip, because R5 counts a gate that *fired* — closing the tab on the prompt
 * spends the day's turn just as skipping does. It is not a touch (R10): nothing
 * here writes the touched stamp, which only a successful mutation earns.
 */
function SignedInHome() {
  const [gateFired] = useState(dayGateFires);
  // Which surface is showing, seeded from the gate's one-time decision. It is a
  // surface rather than a `skipped` flag because the traffic runs both ways:
  // skipping reveals the dashboard, and the first-run invitation (R43) puts the
  // prompt back. Neither direction navigates — one route, two states (D1).
  const [surface, setSurface] = useState<"prompt" | "dashboard">(
    gateFired ? "prompt" : "dashboard",
  );

  useEffect(() => {
    if (gateFired) markGateFiredToday();
  }, [gateFired]);

  // Reopening the prompt does not re-fire the gate: the day's turn was spent on
  // mount and the effect above does not run again.
  if (surface === "dashboard") {
    return <Dashboard onStartPlan={() => setSurface("prompt")} />;
  }
  return <FocusPrompt onSkip={() => setSurface("dashboard")} />;
}

export default function Home() {
  const { isAuthenticated } = useAuth();

  // Deliberately does NOT wait on `isLoading`. The tour is static and identical
  // for everyone, so it paints immediately rather than showing a loading string
  // on the page that is both first impression and SEO surface. An authenticated
  // visitor may see a brief flash of the hero before the dashboard replaces it —
  // that is accepted (FS-0003 R19), not a bug to gate away.
  if (isAuthenticated) {
    return <SignedInHome />;
  }

  return <LandingTour />;
}
