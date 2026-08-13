'use client';

import HeroSection from './sections/HeroSection';
import PlanSection from './sections/PlanSection';

/**
 * The logged-out product tour.
 *
 * Six sections tell the product's own loop; each lands in its own slice. This
 * file is the ONLY thing outside `components/landing/` that the logged-out
 * branch imports — the directory is deliberately sealed, because the landing is
 * the one surface permitted decorative motion (FS-0003 §Known tensions 1) and
 * that exemption must not be reachable from product code.
 */
export default function LandingTour() {
  // `bg-background`, not the app's `.bg-layout` helper: that helper is declared
  // with raw hex in globals.css, and the tour is token-only. The two resolve to
  // the same colour in both themes.
  return (
    <main className="min-h-screen bg-background text-foreground">
      <HeroSection />
      <PlanSection />
    </main>
  );
}
