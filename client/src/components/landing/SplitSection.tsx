'use client';

import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';
import Section from './Section';
import ParallaxLayer from './motion/ParallaxLayer';
import { ART_SPEED } from './motion/speeds';

interface SplitSectionProps {
  heading: string;
  body: ReactNode;
  art: ReactNode;
  /**
   * Which side the art takes at `md` and up. Alternating it down the page keeps
   * the scroll from reading as one repeated template. Stacked mobile order puts
   * the words first either way.
   */
  artSide?: 'left' | 'right';
}

/**
 * The tour's recurring beat: a claim, a short elaboration, and a piece of mocked
 * art drifting slightly behind them.
 *
 * Exists so adding a section is an *addition* rather than a re-typing of the
 * grid, the layer speed, and the heading scale — three copies of that had
 * accumulated before this was extracted.
 */
export default function SplitSection({ heading, body, art, artSide = 'right' }: SplitSectionProps) {
  const artLeft = artSide === 'left';

  return (
    <Section>
      <div className="grid items-center gap-12 md:grid-cols-2 md:gap-16">
        <div className={cn(artLeft && 'md:order-last')}>
          <h2 className="text-2xl font-bold md:text-3xl">{heading}</h2>
          <p className="mt-4 text-base leading-relaxed text-muted-foreground">{body}</p>
        </div>

        <ParallaxLayer
          speed={ART_SPEED}
          className={cn(
            'flex justify-center',
            artLeft ? 'md:order-first md:justify-start' : 'md:justify-end'
          )}
        >
          {art}
        </ParallaxLayer>
      </div>
    </Section>
  );
}
