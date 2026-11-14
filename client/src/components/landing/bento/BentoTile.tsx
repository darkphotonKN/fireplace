'use client';

import Link from 'next/link';
import { useEffect, useRef, useState, type ReactNode, type RefObject } from 'react';
import { cn } from '@/lib/utils';
import { SIGN_UP_HREF } from '../CtaLink';
import { useNoHover, usePrefersReducedMotion } from '../motion/useMediaQuery';
import styles from './bento.module.css';

/**
 * How long a touch reveal is "playing" before it rests at "end": the module's
 * --reveal-ms (700ms) plus room for the stagger a mock may add behind it.
 */
export const REVEAL_MS = 1200;

/**
 * - `start`: what the server renders, what no-JS and hover-capable devices keep
 *   (their reveal is CSS on :hover / :focus-visible, never this attribute).
 * - `playing`: a touch device's one reveal, under way.
 * - `end`: at rest, revealed, for good.
 */
export type RevealState = 'start' | 'playing' | 'end';

/**
 * Stretches the observed area far above the viewport, so "in or above the
 * viewport" is what counts as intersecting. A browser only calls back when
 * `isIntersecting` flips, and a plain viewport never flips for a tile flung
 * from below the fold to above it in one frame; this margin makes that jump a
 * flip, so the fling rule gets its callback.
 */
const FLING_ROOT_MARGIN = '100000px 0px 0px 0px';

/**
 * The tile's reveal state, by device (FS-XCTMG R17):
 * reduced motion reads `end` from hydration for as long as it is on; a no-hover
 * device plays once on first entering view (or lands at `end` if flung past
 * unseen); anything else stays `start` and leaves the reveal to CSS.
 */
function useRevealState(ref: RefObject<HTMLElement | null>): RevealState {
  const [state, setState] = useState<RevealState>('start');
  const prefersReducedMotion = usePrefersReducedMotion();
  const noHover = useNoHover();

  useEffect(() => {
    // A re-run means reduced motion or hover changed, and the last run's timer
    // went with its cleanup: a reveal under way settles at the end rather than
    // staying "playing" with nothing left to finish it.
    setState((s) => (s === 'playing' ? 'end' : s));
    const el = ref.current;
    if (prefersReducedMotion || !noHover || !el || typeof IntersectionObserver === 'undefined') return;

    let timer: ReturnType<typeof setTimeout> | undefined;
    const observer = new IntersectionObserver(
      (entries) => {
        const entry = entries[entries.length - 1];
        // Still below the fold: keep waiting.
        if (!entry?.isIntersecting) return;
        if (entry.boundingClientRect.bottom <= 0) {
          // The RevealOnEnter fling rule: carried from below the fold to above
          // the viewport without one frame on screen. Nobody saw it play, so it
          // lands at rest rather than waiting forever at the start.
          setState('end');
        } else {
          setState((s) => (s === 'start' ? 'playing' : s));
          timer = setTimeout(() => setState('end'), REVEAL_MS);
        }
        observer.disconnect();
      },
      { rootMargin: FLING_ROOT_MARGIN }
    );
    observer.observe(el);

    return () => {
      observer.disconnect();
      if (timer) clearTimeout(timer);
    };
  }, [prefersReducedMotion, noHover, ref]);

  // Reduced motion is a view over the state, not a write to it: switched off
  // again, a tile that never played is back at `start`, its reveal CSS's again.
  return prefersReducedMotion ? 'end' : state;
}

/** Which breakpoint the tile spans two columns at; one column everywhere else. */
export type TileSpan = 'md' | 'lg';

// Whole class strings, so Tailwind's scanner sees them.
const SPAN_CLASSES: Record<TileSpan, string> = {
  // The md grid's last odd tile fills the row; at lg it is back to one of three.
  md: 'md:col-span-2 lg:col-span-1',
  lg: 'lg:col-span-2',
};

interface BentoTileProps {
  /** Footer name, title case ("Plan"): the tile link's accessible name with "Try it". */
  label: string;
  /** The decorative mock, already wrapped in `Mock`. */
  mock: ReactNode;
  /**
   * The one dark tile (TODAY, R12). A mock on it draws from
   * `secondary-foreground` at reduced opacity, not from the page tokens.
   */
  inverted?: boolean;
  span?: TileSpan;
  /** An in-page anchor on the link: TODAY's `today`, which the hero badge targets. */
  id?: string;
  className?: string;
}

/**
 * The one shell every bento tile uses (FS-XCTMG R11): a single link to sign-up
 * covering the whole tile, the mock on top, label and "Try it →" in the footer.
 * Owns the reveal contract, so a tile slice only supplies a mock and a span.
 */
export default function BentoTile({ label, mock, inverted, span, id, className }: BentoTileProps) {
  const ref = useRef<HTMLAnchorElement>(null);
  const state = useRevealState(ref);

  return (
    <Link
      ref={ref}
      id={id}
      href={SIGN_UP_HREF}
      data-reveal={state}
      className={cn(
        styles.tile,
        // overflow-hidden: nothing a reveal moves can leave the tile's box.
        'relative flex min-w-0 flex-col overflow-hidden rounded-tile border border-border p-6 shadow-tile',
        'outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
        inverted ? 'bg-secondary text-secondary-foreground' : 'bg-card text-card-foreground',
        span && SPAN_CLASSES[span],
        className
      )}
    >
      <div className="flex-1">{mock}</div>

      {/* The literal space keeps the accessible name "Plan Try it", not "PlanTry it". */}
      <div className="mt-6 flex items-center justify-between gap-4 text-sm">
        <span
          className={cn(
            'font-medium uppercase tracking-wide',
            inverted ? 'text-secondary-foreground/70' : 'text-muted-foreground'
          )}
        >
          {label}
        </span>{' '}
        <span className={cn('font-medium', inverted ? 'text-secondary-foreground' : 'text-foreground')}>
          Try it{' '}
          <span aria-hidden="true" className={styles.arrow}>
            →
          </span>
        </span>
      </div>
    </Link>
  );
}
