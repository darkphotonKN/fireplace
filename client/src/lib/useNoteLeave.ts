'use client';

import { useRef, useState, type RefObject } from 'react';
import { NOTE_SUMMARY_THRESHOLD } from '@/lib/noteText';
import type { SummarizeFailure, SummarizeOutcome } from '@/lib/useSummarizeNote';

/**
 * Leaving a note's page (FS-PGG2N R27, R32–R35): the decision, apart from the
 * page that draws it. One mount is one visit to the note.
 *
 * - Any pending body save goes first. A failed one is tried once more; if that
 *   fails too the leave stops, holding where it was going for Leave anyway.
 * - Then a note whose trimmed body is past `NOTE_SUMMARY_THRESHOLD` and that
 *   has no summary asks for one, once per visit, whatever is chosen.
 *
 * `decide` is the leave guard's answer to an in-app link; `leave` is the
 * page's own leave (after Make a task), which flushes but never asks.
 */

/**
 * How the body's saves stand once a save call settles: the server has the
 * text, the save failed, or the note is gone (R37).
 */
export type BodySaveResult = 'saved' | 'failed' | 'gone';

/** What a leave asks of the body before the page goes. */
export interface LeavingBody {
  /** Sends a save still waiting on its timer, and settles on every save. */
  flush: () => Promise<BodySaveResult>;
  /** Sends the text again, even if it was sent before (Try again). */
  retry: () => Promise<BodySaveResult>;
  /** The body as it stands: the text being written, or the stored one. */
  text: () => string;
}

/** What went wrong with the prompt's Summarize with AI: the call, or saving its text. */
export type LeavePromptProblem = SummarizeFailure | 'unsaved';

export interface UseNoteLeaveArgs {
  body: RefObject<LeavingBody | null>;
  /** The stored body, for when there is no body to ask. */
  description: string;
  /** The note's summary as it stands now. */
  summary: string | null | undefined;
  /** Summarize with AI, as the "On the card" block presses it (R33, G12). */
  summarize: () => Promise<SummarizeOutcome>;
  /** Opens the summary editor with focus, on a proposal when given one. */
  openSummary: (prefill?: string) => void;
  /** Goes to a destination the guard has already let through. */
  go: (href: string) => void;
}

export function useNoteLeave({
  body,
  description,
  summary,
  summarize,
  openSummary,
  go,
}: UseNoteLeaveArgs) {
  /** The prompt, while it is open, and how its held leave is answered. */
  const [asking, setAsking] = useState<{
    problem: LeavePromptProblem | null;
    answer: (go: boolean) => void;
  } | null>(null);
  /** A leave the body's save stopped: where it was going (R27). */
  const [unsaved, setUnsaved] = useState<string | null>(null);
  const prompted = useRef(false);
  // A leave reads these after its awaits, so from here, not a closure.
  const latest = useRef({ description, summary });
  latest.current = { description, summary };

  const flushBeforeLeaving = async (href: string): Promise<BodySaveResult | 'stopped'> => {
    setUnsaved(null);
    const b = body.current;
    if (!b) return 'saved';
    let saved = await b.flush();
    // One more try before the leave is stopped (R27, G6).
    if (saved === 'failed') saved = await b.retry();
    if (saved !== 'failed') return saved;
    setUnsaved(href);
    return 'stopped';
  };

  const decide = async (href: string): Promise<boolean> => {
    const saved = await flushBeforeLeaving(href);
    if (saved === 'stopped') return false;
    if (saved === 'gone') return true;

    const text = body.current?.text() ?? latest.current.description;
    const asks =
      !prompted.current &&
      !latest.current.summary &&
      text.trim().length > NOTE_SUMMARY_THRESHOLD;
    if (!asks) return true;
    prompted.current = true;
    return new Promise<boolean>((answer) => setAsking({ problem: null, answer }));
  };

  const leave = async (href: string) => {
    if ((await flushBeforeLeaving(href)) !== 'stopped') go(href);
  };

  const answer = (goOn: boolean) => {
    asking?.answer(goOn);
    setAsking(null);
  };

  /** The prompt's Summarize with AI (R33, R34, G12): save it, then go on. */
  const summarizeAndLeave = async () => {
    const outcome = await summarize();
    switch (outcome.kind) {
      case 'saved':
      case 'gone':
        return answer(true);
      case 'failed':
        return setAsking((a) => a && { ...a, problem: outcome.reason });
      case 'saveFailed':
        return setAsking((a) => a && { ...a, problem: 'unsaved' });
      case 'proposed':
        // A summary arrived meanwhile: it is the person's to keep or not.
        answer(false);
        return openSummary(outcome.text);
    }
  };

  return {
    decide,
    leave,
    unsaved,
    /** Leave anyway, after a stopped flush: go without saving. */
    leaveUnsaved: () => {
      const href = unsaved;
      setUnsaved(null);
      if (href) go(href);
    },
    prompt: asking && { problem: asking.problem },
    choose: {
      summarize: summarizeAndLeave,
      write: () => {
        answer(false);
        openSummary();
      },
      leave: () => answer(true),
      stay: () => answer(false),
    },
  };
}
