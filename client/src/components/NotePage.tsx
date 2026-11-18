'use client';

import {
  useCallback,
  useEffect,
  useImperativeHandle,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type Ref,
} from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { formatDistanceToNowStrict } from 'date-fns';
import {
  deleteChecklistItem,
  fetchChecklist,
  updateChecklistItem,
} from '@/services/api';
import { ApiError, errorCode, NOTE_HAS_COMMENTS_MESSAGE } from '@/api/client';
import type { ChecklistItem } from '@/api/checklists';
import ItemActions from '@/components/ItemActions';
import DeleteConfirm from '@/components/DeleteConfirm';
import BackLink from '@/components/BackLink';
import NoteSummaryBlock, { type NoteSummaryHandle } from '@/components/NoteSummaryBlock';
import NoteLeavePrompt from '@/components/NoteLeavePrompt';
import NoteComments from '@/components/NoteComments';
import { cn } from '@/lib/utils';
import LinkedText from '@/components/LinkedText';
import {
  cardText,
  firstLine,
  itemName,
  NOTE_EDIT_PARAM,
} from '@/lib/noteText';
import { useDocumentTitle } from '@/lib/useDocumentTitle';
import { useSummarizeNote, type SummarySaveResult } from '@/lib/useSummarizeNote';
import { useLeaveGuard } from '@/lib/useLeaveGuard';
import { useNoteLeave, type BodySaveResult, type LeavingBody } from '@/lib/useNoteLeave';

/**
 * A note's own page (FS-PGG2N): the plan's middle card, given over to one
 * note, the way the milestone page is (FS-043WP). Read out of the plan's
 * long-term list, the list the grid reads (R18), and written only through the
 * plan's checklist routes, so it serves whoever those routes serve (R22).
 *
 * In the order R19 gives: the header, the body on paper, the meta line, then
 * the "On the card" block and the comments area.
 */

export interface NotePageProps {
  planId: string;
  noteId: string;
}

type Load =
  | { state: 'loading' }
  | { state: 'ready'; items: ChecklistItem[] }
  | { state: 'missing' }
  | { state: 'failed' }
  /**
   * A save answered 404 while the page was open (R37). Back still goes where
   * it went while the note was here.
   */
  | { state: 'deleted'; back: Back };

/**
 * The plan can't be reached (no access, deleted) or its id is malformed: to
 * the person, the note simply isn't here (R36). Anything else may pass, so it
 * is offered again rather than called gone.
 */
const NOT_HERE_STATUSES = new Set([403, 404, 422]);
const isStatus = (err: unknown, statuses: Set<number>) =>
  err instanceof ApiError && statuses.has(err.status);
const GONE = new Set([404]);

/** Where Back goes: the note's milestone if it has one, else the plan (R17). */
interface Back {
  href: string;
  label: string;
}

/** A live note of this plan, or nothing: archived, a task, another plan's. */
function findNote(items: ChecklistItem[], noteId: string, planId: string) {
  const note = items.find((i) => i.id === noteId);
  if (!note || note.archived || note.type !== 'note' || note.planId !== planId) {
    return null;
  }
  return note;
}

const planBack = (planId: string): Back => ({
  href: `/plan/${planId}`,
  label: 'Back to plan',
});

function backFor(items: ChecklistItem[], note: ChecklistItem, planId: string): Back {
  if (!note.parentId) return planBack(planId);
  const parent = items.find((i) => i.id === note.parentId);
  return {
    href: `/plan/${planId}/m/${note.parentId}`,
    label: parent ? `Back to ${parent.description}` : 'Back to milestone',
  };
}

export type SaveState = 'idle' | 'saving' | 'failed';

/**
 * The page's note: loaded from the plan's long-term list and kept there. The
 * body saves run one at a time, each after the one before, so typing never
 * races two writes; a text overtaken while it waited is skipped for the newer
 * one, and text the server already has is not sent again (R24, R25).
 */
function useNote(planId: string, noteId: string) {
  const [load, setLoad] = useState<Load>({ state: 'loading' });
  const [error, setError] = useState<string | null>(null);
  const [save, setSave] = useState<SaveState>('idle');
  const [attempt, setAttempt] = useState(0);
  // The body as the server last took it, the end of the line of saves, the
  // newest save asked for, and how the last one went. Refs: the editor calls
  // in from timers, its own unmount and the leave guard.
  const lastSent = useRef<string | null>(null);
  const tail = useRef<Promise<unknown>>(Promise.resolve());
  const newest = useRef(0);
  const outcome = useRef<BodySaveResult>('saved');
  const queuedSaves = useRef(0);

  useEffect(() => {
    let cancelled = false;
    setLoad({ state: 'loading' });
    fetchChecklist(planId, 'longterm').then(
      (items) => {
        if (cancelled) return;
        lastSent.current = items.find((i) => i.id === noteId)?.description ?? null;
        setLoad({ state: 'ready', items });
      },
      (err) => {
        if (cancelled) return;
        const missing = isStatus(err, NOT_HERE_STATUSES);
        if (!missing) console.error('Failed to load note:', err);
        setLoad({ state: missing ? 'missing' : 'failed' });
      }
    );
    return () => {
      cancelled = true;
    };
  }, [planId, noteId, attempt]);

  const note = useMemo(
    () => (load.state === 'ready' ? findNote(load.items, noteId, planId) : null),
    [load, noteId, planId]
  );
  const back = useMemo(
    () =>
      load.state === 'ready' && note
        ? backFor(load.items, note, planId)
        : load.state === 'deleted'
          ? load.back
          : planBack(planId),
    [load, note, planId]
  );

  const patch = (fields: Partial<ChecklistItem>) =>
    setLoad((prev) =>
      prev.state === 'ready'
        ? {
            state: 'ready',
            items: prev.items.map((i) => (i.id === noteId ? { ...i, ...fields } : i)),
          }
        : prev
    );

  /** The note went while the page was open: say so, keeping Back (R37). */
  const markDeleted = () =>
    setLoad((prev) => {
      if (prev.state !== 'ready') return { state: 'deleted', back: planBack(planId) };
      const gone = findNote(prev.items, noteId, planId);
      return {
        state: 'deleted',
        back: gone ? backFor(prev.items, gone, planId) : planBack(planId),
      };
    });

  /**
   * Sends the body unless it is empty, whitespace or what the server already
   * has (R24, R25). `force` is Try again: it sends even what was sent before.
   */
  const saveBody = useCallback(
    (text: string, force = false): Promise<BodySaveResult> => {
      // An emptied body is never sent (R25): settle on the saves before it.
      if (!text.trim()) return tail.current.then(() => outcome.current);
      const ask = ++newest.current;
      const send = async (): Promise<BodySaveResult> => {
        // Overtaken while it waited: the newer text goes instead.
        if (ask !== newest.current && !force) return outcome.current;
        if (!force && text === lastSent.current) return outcome.current;
        setSave('saving');
        try {
          const saved = await updateChecklistItem(
            noteId,
            { description: text },
            planId,
            'longterm'
          );
          lastSent.current = text;
          patch({
            description: saved?.description ?? text,
            updatedAt: saved?.updatedAt ?? new Date().toISOString(),
          });
          setSave('idle');
          outcome.current = 'saved';
        } catch (err) {
          if (isStatus(err, GONE)) {
            markDeleted();
            outcome.current = 'gone';
          } else {
            console.error('Error saving note:', err);
            setSave('failed');
            outcome.current = 'failed';
          }
        }
        return outcome.current;
      };
      // With none ahead, it starts now: an unmount's last save must not wait.
      const run = (queuedSaves.current === 0 ? send() : tail.current.then(send)).finally(
        () => queuedSaves.current--
      );
      queuedSaves.current++;
      tail.current = run;
      return run;
    },
    // patch and markDeleted only close over setLoad, noteId and planId.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [noteId, planId]
  );

  /**
   * The summary, by hand (R29): optimistic, and put back with the existing
   * error if the save fails. A 404 means the note is gone (R37).
   */
  const saveSummary = async (summary: string): Promise<SummarySaveResult> => {
    const before = note?.summary;
    setError(null);
    patch({ summary: summary || undefined });
    try {
      // Sent trimmed; the server clears it on empty (R2).
      await updateChecklistItem(noteId, { summary }, planId, 'longterm');
      return 'saved';
    } catch (err) {
      if (isStatus(err, GONE)) {
        markDeleted();
        return 'gone';
      }
      console.error('Error saving summary:', err);
      patch({ summary: before });
      setError('Failed to update task. Please try again.');
      return 'failed';
    }
  };

  /**
   * Make a task (R20): not optimistic, since the page leaves once it lands. A
   * note with comments stays a note, and says why (FS-65B3N R31).
   */
  const makeTask = async () => {
    setError(null);
    try {
      await updateChecklistItem(noteId, { type: 'task' }, planId, 'longterm');
      return true;
    } catch (err) {
      console.error('Error changing item type:', err);
      setError(
        errorCode(err) === 'NOTE_HAS_COMMENTS'
          ? NOTE_HAS_COMMENTS_MESSAGE
          : 'Failed to change item type. Please try again.'
      );
      return false;
    }
  };

  const remove = async () => {
    setError(null);
    try {
      await deleteChecklistItem(noteId, planId);
      return true;
    } catch (err) {
      console.error('Error deleting note:', err);
      setError('Failed to delete task');
      return false;
    }
  };

  return {
    load,
    note,
    back,
    error,
    save,
    saveBody,
    saveSummary,
    markDeleted,
    makeTask,
    remove,
    retry: () => setAttempt((n) => n + 1),
  };
}

export default function NotePage({ planId, noteId }: NotePageProps) {
  const {
    load,
    note,
    back,
    error,
    save,
    saveBody,
    saveSummary,
    markDeleted,
    makeTask,
    remove,
    retry,
  } = useNote(planId, noteId);
  const router = useRouter();
  const view = useRef<NoteViewHandle>(null);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const planHref = `/plan/${planId}`;

  if (load.state === 'deleted') {
    return (
      <div className="space-y-4">
        <BackLink href={back.href} label={back.label} />
        <div className="space-y-2 py-6 text-center">
          <p className="text-base font-medium">This note was deleted</p>
          <p className="text-sm text-muted-foreground">
            Someone removed it while it was open here.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* The header: Back, and the page's menu, which is Make a task and
          Delete and nothing else (R17, R20). Both leave for Back once done. */}
      <header className="flex items-center justify-between gap-3">
        <BackLink href={back.href} label={back.label} />
        {note && (
          <ItemActions
            item={note}
            onToggleType={async () => {
              if (!(await makeTask())) return;
              // The body's last words go first, as on any leave (R27).
              if (view.current) await view.current.leave(back.href);
              else router.push(back.href);
            }}
            onDelete={() => setConfirmingDelete(true)}
            triggerClassName="opacity-100"
          />
        )}
      </header>
      {note && confirmingDelete && (
        <DeleteConfirm
          description={itemName(note)}
          onCancel={() => setConfirmingDelete(false)}
          onConfirm={async () => {
            setConfirmingDelete(false);
            // A save still waiting would only write to a note that is gone.
            view.current?.dropPendingSave();
            if (await remove()) router.push(back.href);
            else view.current?.flush();
          }}
        />
      )}
      {error && (
        <div className="rounded-md bg-destructive/10 p-2 text-base text-destructive">
          {error}
        </div>
      )}
      {load.state === 'loading' ? (
        <p className="py-4 text-sm text-muted-foreground">Loading note…</p>
      ) : load.state === 'failed' ? (
        <LoadFailed onRetry={retry} />
      ) : note ? (
        <NoteView
          ref={view}
          key={note.id}
          planId={planId}
          note={note}
          save={save}
          onSave={saveBody}
          onSaveSummary={saveSummary}
          onGone={markDeleted}
          onReload={retry}
        />
      ) : (
        <NotHere planHref={planHref} />
      )}
    </div>
  );
}

/**
 * Whether the page was opened to write (the menu's Rename on a note, R14).
 * Read once, then dropped from the address, so a reload or a copied link
 * opens reading.
 */
function useEditRequest() {
  const [requested] = useState(
    () =>
      typeof window !== 'undefined' &&
      new URLSearchParams(window.location.search).get(NOTE_EDIT_PARAM) === '1'
  );
  useEffect(() => {
    if (!requested) return;
    const url = new URL(window.location.href);
    url.searchParams.delete(NOTE_EDIT_PARAM);
    window.history.replaceState(window.history.state, '', url.pathname + url.search + url.hash);
  }, [requested]);
  return requested;
}

/** What the page's own menu asks of the note below it (R20, R27). */
interface NoteViewHandle {
  /** Goes to `href` once the body is saved, or stops as any leave does. */
  leave: (href: string) => Promise<void>;
  /** Forgets a body save still waiting on its timer. */
  dropPendingSave: () => void;
  /** Sends the body if it has changed. */
  flush: () => Promise<BodySaveResult>;
}

/**
 * The note itself, below the header: its body, then what follows it (R19).
 * One mount is one visit to this note (R32): leaving it in-app flushes the
 * body first, and a long note with no summary asks for one, once (R35).
 */
function NoteView({
  planId,
  note,
  save,
  onSave,
  onSaveSummary,
  onGone,
  onReload,
  ref,
}: {
  planId: string;
  note: ChecklistItem;
  save: SaveState;
  onSave: (text: string, force?: boolean) => Promise<BodySaveResult>;
  onSaveSummary: (summary: string) => Promise<SummarySaveResult>;
  onGone: () => void;
  /** Loads the note again: it may have become a task elsewhere. */
  onReload: () => void;
  ref?: Ref<NoteViewHandle>;
}) {
  useDocumentTitle(firstLine(cardText(note), 60));
  const writeFirst = useEditRequest();
  const router = useRouter();
  const bodyRef = useRef<NoteBodyHandle>(null);
  const summaryBlock = useRef<NoteSummaryHandle>(null);
  const ai = useSummarizeNote({
    planId,
    noteId: note.id,
    summary: note.summary,
    saveSummary: onSaveSummary,
    onGone,
  });
  /** On the press only (R44): a summary already there gets a proposal (G11). */
  const summarizeHere = async () => {
    const outcome = await ai.summarize();
    if (outcome.kind === 'proposed') summaryBlock.current?.open(outcome.text);
  };

  const leaving = useNoteLeave({
    body: bodyRef,
    description: note.description,
    summary: note.summary,
    summarize: ai.summarize,
    openSummary: (prefill) => summaryBlock.current?.open(prefill),
    go: (href) => router.push(href),
  });
  useLeaveGuard(leaving.decide);

  useImperativeHandle(ref, () => ({
    leave: leaving.leave,
    dropPendingSave: () => bodyRef.current?.drop(),
    flush: () => bodyRef.current?.flush() ?? Promise.resolve('saved'),
  }));

  return (
    <div className="space-y-6">
      <NoteBody
        ref={bodyRef}
        startWriting={writeFirst}
        body={note.description}
        updatedAt={note.updatedAt}
        save={save}
        onSave={onSave}
      />

      {leaving.unsaved && (
        <div
          role="alert"
          className="flex flex-wrap items-center justify-between gap-2 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          <span>Couldn&apos;t save your changes</span>
          <button
            type="button"
            onClick={leaving.leaveUnsaved}
            className="font-medium underline-offset-2 outline-none hover:underline focus-visible:underline"
          >
            Leave anyway
          </button>
        </div>
      )}

      {/* On the card's own surface, below the paper (R49): what the card
          shows, then the comments area, which FS-65B3N fills. */}
      <NoteSummaryBlock
        ref={summaryBlock}
        summary={note.summary}
        onSave={onSaveSummary}
        onSummarize={summarizeHere}
        summarizing={ai.running}
        summarizeFailure={ai.failure}
      />

      <NoteComments
        planId={planId}
        noteId={note.id}
        onGone={onGone}
        onNotANote={onReload}
      />

      {leaving.prompt && (
        <NoteLeavePrompt
          summarizing={ai.running}
          problem={leaving.prompt.problem}
          onSummarize={leaving.choose.summarize}
          onWrite={leaving.choose.write}
          onLeave={leaving.choose.leave}
          onStay={leaving.choose.stay}
        />
      )}
    </div>
  );
}

/** How long the body waits after the last keystroke before it saves (R24, G5). */
export const NOTE_AUTOSAVE_MS = 800;

/**
 * The body on its sheet of paper (R23–R27, R49). Reading, it keeps its line
 * breaks and links http(s) URLs. Clicking the text, or Enter on the sheet,
 * turns it into a plain textarea that grows with it; blur turns it back.
 * The text it holds is the page's own from then on: a save that fails keeps
 * it here rather than snapping back.
 */
/** What a leave asks of the body (R27, R32), and what Delete does (R20). */
interface NoteBodyHandle extends LeavingBody {
  /** Forgets a save still waiting on its timer, sending nothing. */
  drop: () => void;
}

function NoteBody({
  startWriting = false,
  body,
  updatedAt,
  save,
  onSave,
  ref,
}: {
  /** Open writing, as the menu's Rename asks (R14). */
  startWriting?: boolean;
  body: string;
  updatedAt: string;
  save: SaveState;
  onSave: (text: string, force?: boolean) => Promise<BodySaveResult>;
  ref?: Ref<NoteBodyHandle>;
}) {
  const [draft, setDraft] = useState(body);
  const [writing, setWriting] = useState(startWriting);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Read by the unmount below, which sees only the first render's values.
  const draftRef = useRef(draft);
  draftRef.current = draft;
  const onSaveRef = useRef(onSave);
  onSaveRef.current = onSave;

  const flush = () => {
    if (timer.current) clearTimeout(timer.current);
    timer.current = null;
    return onSaveRef.current(draftRef.current);
  };
  useImperativeHandle(ref, () => ({
    flush,
    retry: () => onSaveRef.current(draftRef.current, true),
    drop: () => {
      if (timer.current) clearTimeout(timer.current);
      timer.current = null;
    },
    // An emptied body is never saved (R25): the stored one still stands.
    text: () => (draftRef.current.trim() ? draftRef.current : body),
  }));

  // Leaving the page with a save still waiting sends it, best effort (R24).
  useEffect(
    () => () => {
      if (timer.current) flush();
    },
    []
  );

  // Writing: the caret goes to the end of the text, and the box grows to fit.
  useEffect(() => {
    const el = textareaRef.current;
    if (!writing || !el) return;
    el.focus();
    el.setSelectionRange(el.value.length, el.value.length);
  }, [writing]);
  useLayoutEffect(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = 'auto';
    el.style.height = `${el.scrollHeight}px`;
  }, [writing, draft]);

  const sheet =
    'rounded-xl bg-note-paper p-5 font-normal text-note text-foreground/88 md:p-7';

  return (
    <div className="space-y-2">
      {writing ? (
        <textarea
          ref={textareaRef}
          aria-label="Note"
          value={draft}
          onChange={(e) => {
            setDraft(e.target.value);
            if (timer.current) clearTimeout(timer.current);
            const text = e.target.value;
            timer.current = setTimeout(() => {
              timer.current = null;
              onSaveRef.current(text);
            }, NOTE_AUTOSAVE_MS);
          }}
          onBlur={() => {
            flush();
            // An emptied body is never saved (R25); the stored one comes back.
            if (!draftRef.current.trim()) setDraft(body);
            setWriting(false);
          }}
          className={cn(
            sheet,
            'block w-full resize-none overflow-hidden caret-primary outline-none',
            'focus-visible:ring-2 focus-visible:ring-ring/40'
          )}
        />
      ) : (
        <div
          data-note-sheet
          tabIndex={0}
          aria-label="Note, Enter to edit"
          role="group"
          onClick={(e) => {
            // A link opens; anywhere else on the text starts writing.
            if ((e.target as HTMLElement).closest('a')) return;
            setWriting(true);
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && e.target === e.currentTarget) {
              e.preventDefault();
              setWriting(true);
            }
          }}
          className={cn(
            sheet,
            'cursor-text whitespace-pre-wrap break-words outline-none',
            'focus-visible:ring-2 focus-visible:ring-ring/40'
          )}
        >
          <LinkedText text={draft} rel="noopener noreferrer" />
        </div>
      )}
      <MetaLine
        text={draft}
        updatedAt={updatedAt}
        save={save}
        onRetry={() => onSave(draftRef.current, true)}
      />
    </div>
  );
}

const wordCount = (text: string) => text.split(/\s+/).filter(Boolean).length;

/** "edited 3 minutes ago", from the item's `updatedAt` (R26). */
function editedAgo(updatedAt: string | undefined) {
  const at = updatedAt ? new Date(updatedAt) : null;
  if (!at || Number.isNaN(at.getTime())) return 'edited';
  return `edited ${formatDistanceToNowStrict(at, { addSuffix: true })}`;
}

/** "N words · edited ‹time›", or how the save is going (R26, R27). */
function MetaLine({
  text,
  updatedAt,
  save,
  onRetry,
}: {
  text: string;
  updatedAt: string;
  save: SaveState;
  onRetry: () => void;
}) {
  const words = wordCount(text);
  return (
    <p data-note-meta className="px-1 text-xs text-muted-foreground" aria-live="polite">
      {words === 1 ? '1 word' : `${words} words`}
      {' · '}
      {save === 'saving' ? (
        'Saving…'
      ) : save === 'failed' ? (
        <>
          <span className="text-destructive">Couldn&apos;t save</span>
          {' · '}
          <button
            type="button"
            onClick={onRetry}
            className="text-primary underline-offset-2 hover:underline focus-visible:underline focus-visible:outline-none"
          >
            Try again
          </button>
        </>
      ) : (
        editedAgo(updatedAt)
      )}
    </p>
  );
}

/** A blip, not a verdict: say so and offer the load again (R36). */
function LoadFailed({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="space-y-3 py-6 text-center">
      <p className="text-base font-medium">Couldn&apos;t load this note</p>
      <button
        type="button"
        onClick={onRetry}
        className="rounded-full px-3.5 py-1.5 text-sm font-medium text-primary ring-1 ring-primary/30 transition-colors hover:bg-primary/10"
      >
        Try again
      </button>
    </div>
  );
}

/** Calm, never a crash or a blank card: the id is no note of this plan (R36). */
function NotHere({ planHref }: { planHref: string }) {
  return (
    <div className="space-y-2 py-6 text-center">
      <p className="text-base font-medium">This note isn&apos;t here</p>
      <p className="text-sm text-muted-foreground">
        It may have been archived, removed, or turned into a task.
      </p>
      <Link
        href={planHref}
        className="inline-block text-sm text-primary transition-colors hover:text-primary/80"
      >
        Return to the plan
      </Link>
    </div>
  );
}
