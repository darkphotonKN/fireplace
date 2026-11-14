import type { CSSProperties, ReactNode } from 'react';
import { cn } from '@/lib/utils';
import styles from './bento.module.css';

/**
 * The parts a tile mock reveals with. A mock says WHAT moves; the enclosing
 * BentoTile decides WHEN (hover, focus, play-once, reduced motion), through
 * bento.module.css. A mock that styled its own end state would have to
 * re-implement that whole contract, and would get one trigger wrong.
 *
 * `delay` (ms) staggers a part behind its neighbours.
 */

type Vars = CSSProperties & Record<`--${string}`, string>;

function delayVar(delay?: number): Vars {
  return delay ? { '--reveal-delay': `${delay}ms` } : {};
}

/**
 * A bar fill: scaled from `from` to `to` (0..1) along the start edge. Size and
 * colour come from `className`; the element is a block that fills its track.
 */
export function RevealFill({
  from,
  to,
  delay,
  className,
}: {
  from: number;
  to: number;
  delay?: number;
  className?: string;
}) {
  const style: Vars = {
    '--reveal-from': String(from),
    '--reveal-to': String(to),
    ...delayVar(delay),
  };
  return <span data-reveal-part="fill" style={style} className={cn('block', styles.fill, className)} />;
}

/** Content that arrives on reveal. It holds its space at start, invisibly. */
export function RevealIn({
  children,
  delay,
  className,
  as: Tag = 'div',
}: {
  children: ReactNode;
  delay?: number;
  className?: string;
  as?: 'div' | 'li' | 'span';
}) {
  return (
    <Tag data-reveal-part="enter" style={delayVar(delay)} className={cn(styles.enter, className)}>
      {children}
    </Tag>
  );
}

/**
 * Two pre-rendered states in one cell, cross-faded: a count that goes
 * 1/5 → 2/5, a box that ticks. The cell is as big as the larger of the two.
 */
export function RevealSwap({
  before,
  after,
  delay,
  className,
}: {
  before: ReactNode;
  after: ReactNode;
  delay?: number;
  className?: string;
}) {
  const style = delayVar(delay);
  return (
    <span className={cn(styles.stack, className)}>
      <span data-reveal-part="leave" style={style} className={styles.leave}>
        {before}
      </span>
      <span data-reveal-part="enter" style={style} className={styles.enter}>
        {after}
      </span>
    </span>
  );
}

/**
 * A layer that lights up on reveal: faint and a touch small at start, full at
 * end. A day's highlight, laid over its cell; size and colour come from
 * `className`, and it never takes part in layout (place it absolutely).
 */
export function RevealGlow({ delay, className }: { delay?: number; className?: string }) {
  return <span data-reveal-part="glow" style={delayVar(delay)} className={cn(styles.glow, className)} />;
}
