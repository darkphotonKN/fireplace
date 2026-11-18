'use client';

import { useEffect, useImperativeHandle, useRef, useState, type Ref } from 'react';
import { charCount, clipChars, NOTE_SUMMARY_MAX } from '@/lib/noteText';
import type { SummarizeFailure } from '@/lib/useSummarizeNote';

/**
 * The note page's "On the card" block (FS-PGG2N R28–R31): what the note's
 * card shows, the summary written by hand, and Summarize with AI. One short
 * paragraph: Enter or blur saves, Esc cancels, typing stops at the server's
 * limit, counted in characters as the server counts them.
 *
 * An AI proposal (G11, ADR-0007) opens the same editor, but only Save (or
 * Enter) keeps it: blur leaves it waiting, and Cancel or Esc keeps the old one.
 */

/** Opens the editor from outside the block, with focus (R30, R33). */
export interface NoteSummaryHandle {
  /**
   * With no `prefill`, the editor opens on the current summary. A `prefill`
   * is a proposal: it stands in for the summary until Save keeps it.
   */
  open: (prefill?: string) => void;
}

export default function NoteSummaryBlock({
  summary,
  onSave,
  onSummarize,
  summarizing = false,
  summarizeFailure = null,
  ref,
}: {
  summary?: string | null;
  /** The trimmed text, sent only when it differs; empty clears the summary. */
  onSave: (summary: string) => void;
  /** Summarize with AI was pressed (R30). Without it, the button is not offered. */
  onSummarize?: () => void;
  /** A call is running: the button reads "Summarizing…" and takes no press. */
  summarizing?: boolean;
  /** The last call failed, and why (R31). */
  summarizeFailure?: SummarizeFailure | null;
  ref?: Ref<NoteSummaryHandle>;
}) {
  const current = summary ?? '';
  const [draft, setDraft] = useState<string | null>(null);
  const [proposal, setProposal] = useState(false);
  // Esc unmounts the editor; the blur on the way out must not save it.
  const done = useRef(false);
  const boxRef = useRef<HTMLTextAreaElement>(null);

  const open = (prefill?: string) => {
    done.current = false;
    setProposal(prefill !== undefined);
    setDraft(clipChars(prefill ?? current, NOTE_SUMMARY_MAX));
  };
  useImperativeHandle(ref, () => ({ open }));

  const editing = draft !== null;
  useEffect(() => {
    const el = boxRef.current;
    if (!editing || !el) return;
    el.focus();
    el.setSelectionRange(el.value.length, el.value.length);
  }, [editing]);

  const close = () => {
    done.current = true;
    setDraft(null);
  };
  const commit = () => {
    if (done.current || draft === null) return;
    const next = draft.trim();
    close();
    if (next !== current) onSave(next);
  };

  return (
    <section aria-label="On the card" className="space-y-2">
      <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        On the card
      </p>
      {editing ? (
        <div className="space-y-1">
          <textarea
            ref={boxRef}
            aria-label="Summary"
            value={draft}
            rows={3}
            // Stopped here, not by maxLength, which counts UTF-16 units: the
            // server counts characters, so an emoji is one (R29).
            onChange={(e) => setDraft(clipChars(e.target.value, NOTE_SUMMARY_MAX))}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                // One paragraph: Enter saves, and Shift+Enter adds nothing.
                e.preventDefault();
                if (!e.shiftKey) commit();
              }
              if (e.key === 'Escape') close();
            }}
            // A proposal waits for Save; the summary written by hand saves on blur.
            onBlur={proposal ? undefined : commit}
            className="block w-full resize-none rounded-md border border-border bg-background px-3 py-2 text-base leading-6 text-foreground caret-primary outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
          />
          <div className="flex items-center justify-between gap-3">
            {proposal ? (
              <div className="flex items-center gap-3">
                <button type="button" onClick={commit} className={ACTION}>
                  Save
                </button>
                <button type="button" onClick={close} className={QUIET}>
                  Cancel
                </button>
              </div>
            ) : (
              <span />
            )}
            <p className="text-xs tabular-nums text-muted-foreground">
              {charCount(draft)}/{NOTE_SUMMARY_MAX}
            </p>
          </div>
        </div>
      ) : (
        <>
          <p className={current ? 'text-base leading-6 text-foreground/88' : 'text-sm text-muted-foreground'}>
            {current || 'No summary yet. The card shows the start of the note.'}
          </p>
          {summarizeFailure && (
            <div role="status" className="space-y-0.5 text-sm">
              <p className="text-destructive">Couldn&apos;t summarize</p>
              {summarizeFailure === 'paused' && (
                <p className="text-muted-foreground">AI summaries are paused until tomorrow</p>
              )}
            </div>
          )}
          <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
            <button type="button" onClick={() => open()} className={ACTION}>
              {current ? 'Edit summary' : 'Write a summary'}
            </button>
            {onSummarize && (
              <button
                type="button"
                onClick={onSummarize}
                disabled={summarizing}
                className={ACTION}
              >
                {summarizing ? 'Summarizing…' : 'Summarize with AI'}
              </button>
            )}
          </div>
        </>
      )}
    </section>
  );
}

const ACTION =
  'text-sm text-primary outline-none transition-colors hover:text-primary/80 focus-visible:underline disabled:cursor-default disabled:text-muted-foreground';
const QUIET =
  'text-sm text-muted-foreground outline-none transition-colors hover:text-foreground focus-visible:underline';
