import { cn } from '@/lib/utils';

/**
 * A milestone's meter and its done/total, as a grid card and a milestone's
 * page both draw it, so the two read the same (FS-043WP R20). Two inline
 * pieces for the caller's own flex row; `className` restyles the track.
 * Callers draw nothing at all when there are no task steps.
 */
export default function ProgressMeter({
  done,
  total,
  className,
}: {
  done: number;
  total: number;
  className?: string;
}) {
  return (
    <>
      <span
        className={cn(
          'h-1 flex-1 overflow-hidden rounded-full bg-foreground/[0.08]',
          className
        )}
      >
        <span
          data-progress-fill
          style={{ width: `${(done / total) * 100}%` }}
          className="block h-full rounded-full bg-primary transition-[width] duration-500 [transition-timing-function:cubic-bezier(0.22,1,0.36,1)]"
        />
      </span>
      <span
        className={cn(
          'shrink-0 text-sm tabular-nums transition-colors duration-500',
          done === total ? 'text-primary/80' : 'text-foreground/45'
        )}
      >
        {done}/{total}
      </span>
    </>
  );
}
