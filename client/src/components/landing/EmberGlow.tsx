'use client';

import { cn } from '@/lib/utils';
import ParallaxLayer from './motion/ParallaxLayer';
import { GLOW_SPEED } from './motion/speeds';

/**
 * The hearth light. Purely decorative and the slowest thing on the page, so the
 * copy reads as sitting in front of it rather than on top of it.
 *
 * Colour comes from `--primary` at low alpha, so it warms correctly in both
 * themes without a second token or a hardcoded value.
 */
export default function EmberGlow({ className }: { className?: string }) {
  return (
    <ParallaxLayer
      speed={GLOW_SPEED}
      aria-hidden="true"
      className={cn('pointer-events-none absolute inset-0 select-none', className)}
    >
      <div
        className="absolute left-1/2 top-1/2 aspect-square w-[110vmin] -translate-x-1/2 -translate-y-1/2 rounded-full blur-3xl"
        style={{
          background:
            'radial-gradient(closest-side, hsl(var(--primary) / 0.28), hsl(var(--primary) / 0.08), transparent)',
        }}
      />
    </ParallaxLayer>
  );
}
