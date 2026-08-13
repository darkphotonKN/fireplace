'use client';

import { useEffect, useRef } from 'react';
import { useMediaQuery, usePrefersReducedMotion, WIDE_VIEWPORT_QUERY } from './useMediaQuery';
import type { MotionWrapperProps } from './types';

/**
 * Depth is deliberately bounded. Without a clamp a layer far from the viewport
 * accumulates a large offset and gets dragged into view early, which reads as a
 * glitch rather than as depth — and "restrained" is the whole brief.
 */
export const MAX_OFFSET_PX = 80;

interface ParallaxLayerProps extends MotionWrapperProps {
  /**
   * Apparent scroll speed relative to the page. `1` is natural document flow;
   * `0` would pin the layer. The tour uses ~0.2 for the ambient glow and ~0.6
   * for mock art.
   */
  speed: number;
}

export default function ParallaxLayer({
  speed,
  children,
  className,
  ...domProps
}: ParallaxLayerProps) {
  const ref = useRef<HTMLDivElement>(null);
  const isWide = useMediaQuery(WIDE_VIEWPORT_QUERY);
  const prefersReducedMotion = usePrefersReducedMotion();
  const enabled = isWide && !prefersReducedMotion;

  useEffect(() => {
    const el = ref.current;
    if (!el || !enabled) return;

    // Geometry is measured here and on resize only — never inside the scroll
    // handler — so scrolling reads no layout. The scroll path touches
    // `window.scrollY` and writes `transform`, nothing else.
    let center = 0;
    const measure = () => {
      const rect = el.getBoundingClientRect();
      center = rect.top + window.scrollY + rect.height / 2;
    };

    let frame = 0;
    const apply = () => {
      frame = 0;
      const viewportCenter = window.scrollY + window.innerHeight / 2;
      const raw = (viewportCenter - center) * (1 - speed);
      const offset = Math.min(MAX_OFFSET_PX, Math.max(-MAX_OFFSET_PX, raw));
      el.style.transform = `translate3d(0px, ${offset.toFixed(2)}px, 0px)`;
    };

    const schedule = () => {
      if (frame) return;
      frame = requestAnimationFrame(apply);
    };

    const onResize = () => {
      measure();
      schedule();
    };

    measure();
    apply();

    window.addEventListener('scroll', schedule, { passive: true });
    window.addEventListener('resize', onResize);

    // A `resize` listener alone is not enough. The page reflows for reasons the
    // window never hears about — most reliably the web font swapping in, which
    // shifts every section below it and silently invalidates the cached centre
    // for the rest of the session. Watch the document itself instead.
    let observer: ResizeObserver | undefined;
    if (typeof ResizeObserver !== 'undefined') {
      observer = new ResizeObserver(onResize);
      observer.observe(document.documentElement);
    }
    document.fonts?.ready.then(onResize).catch(() => {});

    return () => {
      window.removeEventListener('scroll', schedule);
      window.removeEventListener('resize', onResize);
      observer?.disconnect();
      if (frame) cancelAnimationFrame(frame);
      el.style.transform = '';
    };
  }, [enabled, speed]);

  return (
    <div ref={ref} className={className} {...domProps}>
      {children}
    </div>
  );
}
