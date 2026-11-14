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
}: SectionProps) {
  return (
    <section
      className={cn(
        'relative w-full px-6',
        // Clip ONLY when there is a backdrop to contain: the glow is wider than
        // the viewport. Nothing else here is translated, so not clipping cannot
        // introduce horizontal overflow, and an unclipped grid keeps its tiles'
        // focus rings.
        backdrop && 'overflow-hidden',
        'py-24 md:py-32',
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
