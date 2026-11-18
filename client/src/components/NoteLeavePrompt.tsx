'use client';

import { useEffect, useId, useRef, type KeyboardEvent } from 'react';
import type { LeavePromptProblem } from '@/lib/useNoteLeave';

/**
 * The leave prompt (FS-PGG2N R32–R34): leaving a long note that has no
 * summary asks for one, once. A modal dialog over a scrim, sized to fit a
 * 375px screen. Esc or a click outside means staying.
 *
 * After a failure it offers only Write a summary and Leave anyway (R34).
 */

export default function NoteLeavePrompt({
  summarizing,
  problem,
  onSummarize,
  onWrite,
  onLeave,
  onStay,
}: {
  summarizing: boolean;
  problem: LeavePromptProblem | null;
  onSummarize: () => void;
  onWrite: () => void;
  onLeave: () => void;
  onStay: () => void;
}) {
  const titleId = useId();
  const textId = useId();
  const dialogRef = useRef<HTMLDivElement>(null);

  // Focus moves in on open, and back to where it was on close.
  useEffect(() => {
    const before = document.activeElement as HTMLElement | null;
    dialogRef.current?.querySelector('button')?.focus();
    return () => {
      if (before?.isConnected) before.focus();
    };
  }, []);

  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Escape') {
      e.preventDefault();
      onStay();
      return;
    }
    if (e.key !== 'Tab') return;
    // Keep Tab inside the dialog while it is open.
    const buttons = Array.from(
      dialogRef.current?.querySelectorAll<HTMLButtonElement>('button:not(:disabled)') ?? []
    );
    if (buttons.length === 0) return;
    const first = buttons[0];
    const last = buttons[buttons.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div data-leave-scrim aria-hidden className="absolute inset-0 bg-scrim" onClick={onStay} />
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={textId}
        onKeyDown={onKeyDown}
        className="relative w-full max-w-sm rounded-lg border border-border bg-card p-6 text-card-foreground shadow-lg"
      >
        <h2 id={titleId} className="text-lg font-bold text-foreground">
          Add a summary for the card?
        </h2>
        <p id={textId} className="mt-2 text-sm text-muted-foreground">
          This note is long, so its card shows only the start. A line or two lets the card say
          what it holds.
        </p>
        {problem && (
          <div role="status" className="mt-3 space-y-0.5 text-sm">
            <p className="text-destructive">
              {problem === 'unsaved' ? "Couldn't save the summary" : "Couldn't summarize"}
            </p>
            {problem === 'paused' && (
              <p className="text-muted-foreground">AI summaries are paused until tomorrow</p>
            )}
          </div>
        )}
        <div className="mt-5 flex flex-col gap-2">
          {!problem && (
            <button
              type="button"
              onClick={onSummarize}
              disabled={summarizing}
              className={PRIMARY}
            >
              {summarizing ? 'Summarizing…' : 'Summarize with AI'}
            </button>
          )}
          <button type="button" onClick={onWrite} disabled={summarizing} className={OUTLINE}>
            Write a summary
          </button>
          <button type="button" onClick={onLeave} disabled={summarizing} className={QUIET}>
            Leave anyway
          </button>
        </div>
      </div>
    </div>
  );
}

const BUTTON =
  'h-9 w-full rounded-md px-4 text-sm font-medium outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-default disabled:opacity-60';
const PRIMARY = `${BUTTON} bg-primary text-primary-foreground hover:bg-primary/90`;
const OUTLINE = `${BUTTON} text-foreground ring-1 ring-border hover:bg-muted`;
const QUIET = `${BUTTON} text-muted-foreground hover:text-foreground`;
