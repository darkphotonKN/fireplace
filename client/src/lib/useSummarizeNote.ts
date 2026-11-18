'use client';

import { useCallback, useRef, useState } from 'react';
import { summarizeNote } from '@/api/insights';
import { ApiError } from '@/api/client';

/**
 * Summarize with AI, pressed (FS-PGG2N R30, R31, R44): one call to insights
 * for this note's summary, then what to do with the text it returns. The one
 * home of that logic, for the "On the card" block and the leave prompt alike.
 *
 * The model runs only when `summarize` is called, and callers call it only on
 * a press (ADR-0019). A press while one is running starts nothing.
 */

/** How the page's summary save went: kept, refused, or the note is gone. */
export type SummarySaveResult = 'saved' | 'failed' | 'gone';

/**
 * Why the AI gave nothing: it failed (worth another try), or the day's AI
 * budget is spent (R31).
 */
export type SummarizeFailure = 'failed' | 'paused';

export type SummarizeOutcome =
  /** There was no summary, and the returned one is saved. */
  | { kind: 'saved' }
  /**
   * There was one already: the returned text is only a proposal, for the
   * person to keep or not (G11, ADR-0007). Nothing was saved.
   */
  | { kind: 'proposed'; text: string }
  | { kind: 'failed'; reason: SummarizeFailure }
  /** The AI answered, but saving its summary failed: the summary is as it was. */
  | { kind: 'saveFailed' }
  /** The note went while the page was open (R37). */
  | { kind: 'gone' }
  /** A call was already running; this press started nothing. */
  | { kind: 'busy' };

export interface UseSummarizeNoteArgs {
  planId: string;
  noteId: string;
  /** The note's summary as it stands now. */
  summary: string | null | undefined;
  /** Saves a summary through the item update, as the page does by hand. */
  saveSummary: (summary: string) => Promise<SummarySaveResult>;
  /** Told when the call answers that the note is gone. */
  onGone: () => void;
}

export function useSummarizeNote({
  planId,
  noteId,
  summary,
  saveSummary,
  onGone,
}: UseSummarizeNoteArgs) {
  const [running, setRunning] = useState(false);
  const [failure, setFailure] = useState<SummarizeFailure | null>(null);
  // Read at the press, which may come from a prompt opened long after render.
  const latest = useRef({ summary, saveSummary, onGone });
  latest.current = { summary, saveSummary, onGone };
  const busy = useRef(false);

  const summarize = useCallback(async (): Promise<SummarizeOutcome> => {
    if (busy.current) return { kind: 'busy' };
    busy.current = true;
    setRunning(true);
    setFailure(null);
    try {
      let text: string;
      try {
        text = await summarizeNote(planId, noteId);
      } catch (err) {
        // On the code, never the detail (ADR-0004).
        const code = err instanceof ApiError ? err.code : undefined;
        if (code === 'NOT_FOUND') {
          latest.current.onGone();
          return { kind: 'gone' };
        }
        if (!(err instanceof ApiError)) console.error('Error summarizing note:', err);
        const reason: SummarizeFailure =
          code === 'LLM_BUDGET_EXHAUSTED' ? 'paused' : 'failed';
        setFailure(reason);
        return { kind: 'failed', reason };
      }
      // A summary already there is the person's: propose, never overwrite.
      if (latest.current.summary) return { kind: 'proposed', text };
      const saved = await latest.current.saveSummary(text);
      if (saved === 'saved') return { kind: 'saved' };
      return saved === 'gone' ? { kind: 'gone' } : { kind: 'saveFailed' };
    } finally {
      busy.current = false;
      setRunning(false);
    }
  }, [planId, noteId]);

  return { running, failure, summarize };
}
