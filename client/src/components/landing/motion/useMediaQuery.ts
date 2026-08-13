'use client';

import { useEffect, useState } from 'react';

export const REDUCED_MOTION_QUERY = '(prefers-reduced-motion: reduce)';
/** Tailwind's `sm` breakpoint. Below it the tour runs no depth translation. */
export const WIDE_VIEWPORT_QUERY = '(min-width: 640px)';

/**
 * Subscribes to a media query.
 *
 * Always starts `false`, so the server render and the first client render agree,
 * and so "no motion" is the state until we positively know otherwise. Every
 * caller here treats `false` as the calm, static answer.
 */
export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(false);

  useEffect(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return;

    const mql = window.matchMedia(query);
    const update = () => setMatches(mql.matches);

    update();
    mql.addEventListener('change', update);
    return () => mql.removeEventListener('change', update);
  }, [query]);

  return matches;
}

export function usePrefersReducedMotion(): boolean {
  return useMediaQuery(REDUCED_MOTION_QUERY);
}
