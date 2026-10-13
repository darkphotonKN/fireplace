'use client';

import { useCallback, useEffect, useState } from 'react';
import { X } from 'lucide-react';
import { cn } from '@/lib/utils';
import { isComplete, type Milestone } from '@/lib/milestones';

/**
 * The completion nudge (FS-FSFEC R10–R14): every task step done and the
 * milestone still open, so say so and offer to close it. One behaviour, drawn
 * on a grid card and on a milestone's page alike (FS-043WP R21, C7).
 *
 * Only ever an offer. Close goes through the caller's own done toggle, the
 * checkbox's path, so it is optimistic and reverts the way a tick does;
 * nothing here sets done on its own (R11, R13).
 */

export interface CompletionNudgeProps {
  /** The milestone's title, naming the two buttons. */
  title: string;
  onClose: () => void;
  onDismiss: () => void;
  className?: string;
}

export default function CompletionNudge({
  title,
  onClose,
  onDismiss,
  className,
}: CompletionNudgeProps) {
  return (
    <div
      data-completion-nudge
      className={cn(
        'flex items-center gap-2 rounded-lg bg-primary/[0.08] px-3 py-1.5 text-sm',
        className
      )}
    >
      <span className="min-w-0 flex-1 text-foreground/70">
        All steps done. Close this milestone?
      </span>
      <button
        type="button"
        onClick={onClose}
        aria-label={`Close ${title}`}
        className="shrink-0 rounded-md px-2 py-0.5 font-medium text-primary outline-none transition-colors hover:bg-primary/10 focus-visible:bg-primary/10"
      >
        Close
      </button>
      <button
        type="button"
        onClick={onDismiss}
        aria-label={`Not now, ${title}`}
        title="Not now"
        className="-mr-1 grid h-6 w-6 shrink-0 place-items-center rounded-md text-foreground/40 outline-none transition-colors hover:text-foreground/70 focus-visible:text-foreground/70"
      >
        <X aria-hidden className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}

/** A milestone's progress as one comparable value: what a dismissal holds to. */
const pairOf = (m: Milestone) => `${m.steps.done}/${m.steps.total}`;

/**
 * Which milestones to nudge, and the memory of the ones waved off.
 *
 * Each dismissal holds the done/total it was waved off at. Memory only, so a
 * reload or remount offers again (R12). It lasts while that pair holds: once
 * the progress moves it is forgotten, so unticking and re-ticking a step
 * brings the offer back even at the same pair. Milestones not in `milestones`
 * keep theirs, since nothing on screen can tick them.
 *
 * Pass every milestone on screen, each read through `readMilestones`.
 */
export function useCompletionNudges(milestones: readonly Milestone[]) {
  const [dismissed, setDismissed] = useState<Record<string, string>>({});

  // Keyed on the pairs, not the array, so a caller may build the list inline
  // without the forgetting below re-running every render.
  const pairsKey = milestones.map((m) => `${m.id}=${pairOf(m)}`).join('|');
  useEffect(() => {
    const pairs = new Map(milestones.map((m) => [m.id, pairOf(m)]));
    setDismissed((prev) => {
      const kept = Object.entries(prev).filter(([id, at]) => {
        const pair = pairs.get(id);
        return pair === undefined || pair === at;
      });
      return kept.length === Object.keys(prev).length
        ? prev
        : Object.fromEntries(kept);
    });
    // pairsKey encodes every id and pair read from `milestones` here.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pairsKey]);

  const shows = useCallback(
    (m: Milestone | undefined): m is Milestone =>
      !!m && isComplete(m) && dismissed[m.id] !== pairOf(m),
    [dismissed]
  );
  const dismiss = useCallback(
    (m: Milestone) => setDismissed((prev) => ({ ...prev, [m.id]: pairOf(m) })),
    []
  );
  return { shows, dismiss };
}
