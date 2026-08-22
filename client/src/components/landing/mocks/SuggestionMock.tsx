'use client';

import Mock from './Mock';
import { GATEWAY_TASK } from './content';

/**
 * One suggestion. Names the task the plan card and the daily list already
 * showed, so the nudge reads as something the product noticed rather than a
 * generic prompt.
 */
export default function SuggestionMock() {
  return (
    <Mock className="w-full max-w-lg">
      <p className="text-xs font-bold uppercase tracking-widest text-muted-foreground">
        Noticed
      </p>
      <p className="mt-3 text-base leading-relaxed text-foreground">
        <span className="font-bold">&ldquo;{GATEWAY_TASK}&rdquo;</span> has slipped three days
        running. Put it first tomorrow?
      </p>
    </Mock>
  );
}
