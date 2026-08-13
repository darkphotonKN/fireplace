import type { HTMLAttributes } from 'react';

/**
 * Wrapper props shared by the landing motion primitives.
 *
 * Extending `HTMLAttributes` matters beyond convenience: the mocked visuals in
 * later slices are decorative and must carry `aria-hidden`, and a wrapper that
 * swallowed it would make them unreachable to fix.
 */
export interface MotionWrapperProps extends HTMLAttributes<HTMLDivElement> {
  [key: `data-${string}`]: string | undefined;
}
