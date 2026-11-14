'use client';

import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';

/**
 * Base wrapper for every mocked visual in the tour. Read this before adding one.
 *
 * The mocks are **styled like production, still bespoke and decorative**
 * (FS-XCTMG R13). Same tokens, Lora, radii and density as the real screens they
 * stand for (the milestone cards, the Day plan, the progress strip), so a tile
 * reads as the product. But hand-built: a few rows rather than a full screen,
 * no window chrome, no toolbars, and never the real components themselves.
 *
 * Rules this wrapper enforces or assumes, binding on every mock:
 *
 * 1. **Decorative.** `aria-hidden` here, and nothing inside may be focusable or
 *    carry an interactive role. Fake checkboxes are never real `<input>`s.
 * 2. **Bespoke and isolated.** No imports from real app components, no context,
 *    no service-layer calls, no fetches. Importing the real checklist would drag
 *    in AuthContext and the service layer, coupling marketing to product
 *    internals and breaking the landing on unrelated refactors.
 * 3. **Token-only colour.** No raw hex, no `*-gray-*`. Warm neutrals only.
 * 4. **Static.** Nothing here changes as a function of scroll position.
 *
 * `data-landing-mock` is the handle the tests assert the contract through.
 */
export default function Mock({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div
      data-landing-mock
      aria-hidden="true"
      className={cn(
        'pointer-events-none select-none rounded-lg border border-border bg-card p-6 shadow-sm',
        className
      )}
    >
      {children}
    </div>
  );
}
