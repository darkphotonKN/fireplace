'use client';

import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';

/**
 * Base wrapper for every mocked visual in the tour. Read this before adding one.
 *
 * The mocks are **stylized-honest, never replicas**. Same tokens, type scale,
 * radii and coral as the real product, but simplified: a checklist is a few
 * rows, a Gantt is a few soft bars, no window chrome, no toolbars, no dense
 * real-app detail. A near-replica silently becomes a lie the first time the real
 * UI changes, because a hand-built mock cannot track the app. A stylized one
 * reads as *a picture of the idea* and stays true.
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
