import { cn } from '@/lib/utils';
import type { Milestone, MilestoneReading } from '@/lib/milestones';

/**
 * A plan's progress across its milestones (FS-FSFEC R15–R21): the headline
 * "N of M milestones" and one segment per milestone, each filled to that
 * milestone's own progress. Nothing is averaged or weighted, so a big
 * milestone at 9/10 still reads as nearly there and can't swamp small ones.
 *
 * The headline is the text equivalent; the fills are decorative and each
 * segment titles its milestone for anyone hovering (R21).
 */

export interface MilestoneStripProps {
  reading: MilestoneReading;
  /**
   * `accent` in the plan header; `neutral` where the accent colour is spoken
   * for elsewhere on the screen, as on the dashboard (R28).
   */
  tone?: 'accent' | 'neutral';
  /**
   * `stacked` puts the headline above the track, as in the plan header.
   * `inline` puts the track and the headline on one line, so the strip can
   * settle into a row that is already on screen without changing its height,
   * as on a Continue card (R27).
   */
  layout?: 'stacked' | 'inline';
  className?: string;
}

// Past twelve the gaps start to eat the segments, so they tighten (C5).
const ROOMY_UP_TO = 12;
const GAP_PX = { roomy: 4, tight: 2 };
// Tight gaps are still (M−1)×2px whatever the track's width, which on a
// phone-width card overflows somewhere past a hundred milestones. So past
// twelve all the gaps together never take more than this share of the
// track: each is the smaller of its px size and a slice of the share, and
// the segments always keep the rest (R18: no cap, never a scroll).
const GAP_SHARE_PCT = 25;

function columnGap(total: number): string {
  if (total <= ROOMY_UP_TO) return `${GAP_PX.roomy}px`;
  // Rounded down, so the slices together never add up past the share.
  const slice = Math.floor((GAP_SHARE_PCT / (total - 1)) * 1e4) / 1e4;
  return `min(${GAP_PX.tight}px, ${slice}%)`;
}

// Inline sits in someone else's text row, so it takes that row's size and
// colour, and the track takes whatever width the headline leaves (R27). The
// headline stays first in the DOM, the text equivalent read before the
// decorative track, and `row-reverse` draws the track ahead of it.
const LAYOUT = {
  stacked: {
    strip: 'space-y-1.5',
    headline: 'text-sm text-foreground/60',
    track: '',
  },
  inline: {
    strip: 'flex min-w-0 flex-1 flex-row-reverse items-center gap-3',
    headline: 'shrink-0 whitespace-nowrap',
    track: 'min-w-0 flex-1',
  },
};

const FILL = {
  accent: 'bg-primary',
  neutral: 'bg-foreground/45',
};

export default function MilestoneStrip({
  reading,
  tone = 'accent',
  layout = 'stacked',
  className,
}: MilestoneStripProps) {
  const { milestones, closed, total } = reading;
  // No "0 of 0": a plan without milestones has no strip to show (R19).
  if (total === 0) return null;

  return (
    <div className={cn(LAYOUT[layout].strip, className)} data-milestone-strip>
      <p className={cn('tabular-nums', LAYOUT[layout].headline)}>
        {closed} of {total} {total === 1 ? 'milestone' : 'milestones'}
      </p>
      {/* A grid rather than flex: one equal column per milestone that may
          shrink to nothing, so any count fits one line with no scroll (R18).
          Count-dependent, hence inline. */}
      <div
        data-milestone-track
        aria-hidden
        className={cn('grid', LAYOUT[layout].track)}
        style={{
          gridTemplateColumns: `repeat(${total}, minmax(0, 1fr))`,
          columnGap: columnGap(total),
        }}
      >
        {milestones.map((m) => (
          <span
            key={m.id}
            data-milestone-segment
            title={segmentTitle(m)}
            className="h-1.5 overflow-hidden rounded-full bg-foreground/[0.08]"
          >
            <span
              data-milestone-fill
              style={{ width: `${m.progress * 100}%` }}
              className={cn(
                'block h-full rounded-full transition-[width] duration-500 [transition-timing-function:cubic-bezier(0.22,1,0.36,1)]',
                FILL[tone]
              )}
            />
          </span>
        ))}
      </div>
    </div>
  );
}

function segmentTitle(m: Milestone): string {
  if (m.closed) return `${m.title}: done`;
  if (m.steps.total === 0) return `${m.title}: not done`;
  return `${m.title}: ${m.steps.done}/${m.steps.total} steps`;
}
