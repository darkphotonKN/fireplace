'use client';

import { useEffect, useState } from 'react';

/**
 * True once the page has scrolled past `fraction` of the viewport height.
 *
 * Starts `false`, so the server render and the first client render agree and
 * nothing appears over the hero before hydration settles. Reads only
 * `window.scrollY` per frame — no layout, same rule as the parallax layer.
 */
export function useScrolledPast(fraction: number): boolean {
  const [past, setPast] = useState(false);

  useEffect(() => {
    let frame = 0;

    const evaluate = () => {
      frame = 0;
      setPast(window.scrollY > window.innerHeight * fraction);
    };

    const schedule = () => {
      if (frame) return;
      frame = requestAnimationFrame(evaluate);
    };

    evaluate();
    window.addEventListener('scroll', schedule, { passive: true });
    window.addEventListener('resize', schedule);

    return () => {
      window.removeEventListener('scroll', schedule);
      window.removeEventListener('resize', schedule);
      if (frame) cancelAnimationFrame(frame);
    };
  }, [fraction]);

  return past;
}
