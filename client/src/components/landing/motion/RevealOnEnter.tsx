'use client';

import { useEffect, useRef, useState } from 'react';
import { cn } from '@/lib/utils';
import { usePrefersReducedMotion } from './useMediaQuery';
import type { MotionWrapperProps } from './types';

type RevealState =
  /** Untouched: visible, unanimated. What the server renders and what a
   *  no-JS, pre-hydration, or reduced-motion visitor keeps. */
  | 'idle'
  /** Deliberately hidden, waiting to enter the viewport. Only ever entered for
   *  an element that is currently below the fold, so no visitor sees the hide. */
  | 'hidden'
  /** Entered, animated, and permanently settled. */
  | 'revealed';

export default function RevealOnEnter({ children, className, ...domProps }: MotionWrapperProps) {
  const ref = useRef<HTMLDivElement>(null);
  const [state, setState] = useState<RevealState>('idle');
  const prefersReducedMotion = usePrefersReducedMotion();

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    // Media queries resolve in an effect, so the first commit always runs with
    // `prefersReducedMotion === false`. If it then flips true we must actively
    // undo any hiding already applied — returning early would strand the
    // content invisible for exactly the visitor who opted out of motion.
    if (prefersReducedMotion) {
      setState('idle');
      return;
    }

    if (typeof IntersectionObserver === 'undefined') return;

    // Only hide what the visitor cannot currently see. An element already in or
    // above the viewport is revealed where it stands.
    if (el.getBoundingClientRect().top < window.innerHeight) {
      setState('revealed');
      return;
    }

    setState('hidden');

    const observer = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        // `isIntersecting` alone is not enough. A fast scroll or fling can carry
        // an element from below the fold to above the viewport without a single
        // intersecting frame — and a section that never got its callback would
        // stay hidden forever. Treat "now above the viewport" as entered too.
        if (entry.isIntersecting || entry.boundingClientRect.top < 0) {
          setState('revealed');
          observer.unobserve(el);
        }
      }
    });

    observer.observe(el);
    return () => observer.disconnect();
  }, [prefersReducedMotion]);

  const hidden = state === 'hidden';

  return (
    <div
      ref={ref}
      // Hiding with opacity alone is a trap: the content stays in the tab order
      // and the accessibility tree, so a keyboard visitor tabbing before they
      // scroll lands on an invisible link. `inert` removes the subtree from both;
      // `invisible` (visibility: hidden) does the same in browsers without inert,
      // and neither one shifts layout.
      inert={hidden}
      className={cn(
        className,
        hidden && 'invisible opacity-0',
        state === 'revealed' && 'animate-riseIn'
      )}
      {...domProps}
    >
      {children}
    </div>
  );
}
