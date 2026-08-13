'use client';

import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';
import RevealOnEnter from './motion/RevealOnEnter';

interface SectionProps {
  children: ReactNode;
  /**
   * Decorative layers painted behind the content. Rendered OUTSIDE the reveal
   * wrapper so a backdrop is never subject to the entrance animation — ambient
   * light should already be there when the copy arrives.
   */
  backdrop?: ReactNode;
  className?: string;
  contentClassName?: string;
  /**
   * `compact` gives a section noticeably less vertical room. NUDGE uses it so
   * the beat lands as a grace note rather than padding — the proportion between
   * it and REVIEW is a requirement, not a styling whim, so it lives in the
   * shared wrapper where it can be asserted.
   */
  size?: 'default' | 'compact';
}

/**
 * One beat of the tour. Owns the vertical rhythm, the reveal-on-enter behavior,
 * and the container width, so individual sections carry only their content.
 */
export default function Section({
  children,
  backdrop,
  className,
  contentClassName,
  size = 'default',
}: SectionProps) {
  return (
    <section
      data-section-size={size}
      className={cn(
        'relative w-full px-6',
        // Clip ONLY when there is a backdrop to contain. Clipping unconditionally
        // meant a 0.6x art layer (up to ±80px) could be cut off against a
        // section's own edge — which the compact section, at py-12, would have
        // done. Transforms are vertical-only, so not clipping cannot introduce
        // horizontal overflow.
        backdrop && 'overflow-hidden',
        size === 'compact' ? 'py-12 md:py-16' : 'py-24 md:py-32',
        className
      )}
    >
      {backdrop}
      <RevealOnEnter
        data-landing-reveal="true"
        className={cn('relative mx-auto w-full max-w-5xl', contentClassName)}
      >
        {children}
      </RevealOnEnter>
    </section>
  );
}
