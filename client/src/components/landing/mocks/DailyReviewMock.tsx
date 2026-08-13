'use client';

import Mock from './Mock';

const STREAK_LENGTH = 5;
const STREAK_KEPT = 4;

export default function DailyReviewMock() {
  return (
    <Mock className="w-full max-w-md">
      <p className="text-xs font-bold uppercase tracking-widest text-muted-foreground">Friday</p>

      <p className="mt-3 text-lg font-bold text-foreground">3 of 4 done</p>

      <div className="mt-5 flex items-center gap-2">
        {Array.from({ length: STREAK_LENGTH }, (_, i) => (
          <span
            key={i}
            className={
              i < STREAK_KEPT ? 'h-2.5 w-2.5 rounded-full bg-primary' : 'h-2.5 w-2.5 rounded-full border border-border'
            }
          />
        ))}
        <span className="ml-2 text-xs text-muted-foreground">4-day streak</span>
      </div>

      {/* The written note is the point of this beat: the day closes with a
          sentence about what actually happened, not just a tally. */}
      <p className="mt-6 border-l-2 border-border pl-4 text-sm italic leading-relaxed text-muted-foreground">
        &ldquo;Gateway took longer than I thought. Moving it to the front tomorrow.&rdquo;
      </p>
    </Mock>
  );
}
