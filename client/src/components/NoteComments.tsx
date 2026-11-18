'use client';

import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import type { NoteComment } from '@/api/noteComments';
import DeleteConfirm from '@/components/DeleteConfirm';
import LinkedText from '@/components/LinkedText';
import { cn } from '@/lib/utils';
import { charCount, firstLine } from '@/lib/noteText';
import { fullDateTime, relativeTime } from '@/lib/relativeTime';
import { useNoteComments, type CommentChange } from '@/lib/useNoteComments';

/**
 * A note's comments, on its page below the body and the summary (FS-65B3N
 * R24–R30, R33): oldest first, then the box to add one, with Edit and Delete
 * on the comments the caller may change. They load on their own, so a slow or
 * failed load never stands in the way of the body.
 *
 * What a caller may do with a comment comes from the server (`canEdit`,
 * `canDelete`). The load, the list and what each refusal means live in
 * `useNoteComments`; this only draws them.
 */

/** A body's limit once trimmed, in code points, as the server counts (R9). */
export const COMMENT_MAX = 2000;
/** The count shows once the text passes this (R28). */
export const COMMENT_COUNT_FROM = 1800;

/** Length in code points, so an emoji counts once (R9). */
export const commentLength = (text: string) => charCount(text.trim());
/** 1 to 2000 code points once trimmed (R9, R28). */
export const isSendable = (text: string) => {
  const n = commentLength(text);
  return n >= 1 && n <= COMMENT_MAX;
};

export default function NoteComments({
  planId,
  noteId,
  onGone,
  onNotANote,
}: {
  planId: string;
  noteId: string;
  /** The note went while the page was open (FS-PGG2N R37). */
  onGone: () => void;
  /** The item became a task elsewhere: the page loads again (R30). */
  onNotANote: () => void;
}) {
  const { load, notice, retry, add, edit, remove } = useNoteComments(planId, noteId, {
    onGone,
    onNotANote,
  });
  const [confirming, setConfirming] = useState<NoteComment | null>(null);
  const [deleteFailed, setDeleteFailed] = useState<string | null>(null);

  const confirmRemove = async (comment: NoteComment) => {
    setConfirming(null);
    setDeleteFailed(null);
    if ((await remove(comment.id)) === 'failed') setDeleteFailed(comment.id);
  };

  return (
    <section aria-label="Comments" className="space-y-4">
      <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        Comments
      </p>
      {load.state === 'loading' ? (
        <p className="text-sm text-muted-foreground">Loading comments…</p>
      ) : load.state === 'failed' ? (
        <div className="flex flex-wrap items-center gap-3 text-sm">
          <p className="text-muted-foreground">Couldn&apos;t load comments</p>
          <button
            type="button"
            onClick={retry}
            className="text-primary outline-none transition-colors hover:text-primary/80 focus-visible:underline"
          >
            Retry
          </button>
        </div>
      ) : (
        <>
          {notice && (
            <p role="status" className="text-sm text-muted-foreground">
              {notice}
            </p>
          )}
          <ol aria-label="Comments" className="space-y-4">
            {load.comments.map((comment) => (
              <CommentRow
                key={comment.id}
                comment={comment}
                onEdit={(body) => edit(comment.id, body)}
                onDelete={() => {
                  setDeleteFailed(null);
                  setConfirming(comment);
                }}
                deleteFailed={deleteFailed === comment.id}
              />
            ))}
          </ol>
          <Composer onAdd={add} />
        </>
      )}
      {confirming && (
        <DeleteConfirm
          description={firstLine(confirming.body, 60)}
          onCancel={() => setConfirming(null)}
          onConfirm={() => void confirmRemove(confirming)}
        />
      )}
    </section>
  );
}

/** What a comment's author is called on screen (R13). */
const shownName = (comment: NoteComment) => comment.authorName ?? 'Deleted user';

/**
 * One comment: avatar, "name · time · edited", then its body (R25–R27), and
 * Edit and Delete where the server allows them (R29). The controls show on
 * hover and keyboard focus where there is a pointer, and always on touch.
 */
function CommentRow({
  comment,
  onEdit,
  onDelete,
  deleteFailed,
}: {
  comment: NoteComment;
  onEdit: (body: string) => Promise<CommentChange>;
  onDelete: () => void;
  deleteFailed: boolean;
}) {
  const name = shownName(comment);
  const deleted = comment.authorName === null;
  const full = fullDateTime(comment.createdAt);
  const [draft, setDraft] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [failed, setFailed] = useState(false);
  const boxRef = useRef<HTMLTextAreaElement>(null);

  const editing = draft !== null;
  useEffect(() => {
    const el = boxRef.current;
    if (!editing || !el) return;
    el.focus();
    el.setSelectionRange(el.value.length, el.value.length);
  }, [editing]);
  useLayoutEffect(() => {
    const el = boxRef.current;
    if (!el) return;
    el.style.height = 'auto';
    el.style.height = `${el.scrollHeight}px`;
  }, [draft]);

  const cancel = () => {
    setDraft(null);
    setFailed(false);
  };
  const save = async () => {
    if (draft === null || !isSendable(draft) || saving) return;
    const body = draft.trim();
    // The same text is no edit at all (R16): nothing to send.
    if (body === comment.body) return cancel();
    setSaving(true);
    setFailed(false);
    const result = await onEdit(body);
    setSaving(false);
    if (result === 'failed') setFailed(true);
    else setDraft(null);
  };

  return (
    <li className="group/comment flex gap-3">
      <span
        data-avatar
        aria-hidden
        className={cn(
          'flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-sm font-medium',
          deleted ? 'bg-muted text-muted-foreground' : 'bg-primary/20 text-foreground'
        )}
      >
        {([...name][0] ?? '?').toUpperCase()}
      </span>
      <div className="min-w-0 flex-1 space-y-0.5">
        <div className="flex items-baseline justify-between gap-3">
          <p data-comment-meta className="text-xs text-muted-foreground">
            <span className={deleted ? undefined : 'font-medium text-foreground'}>{name}</span>
            {' · '}
            <time dateTime={comment.createdAt} title={full} aria-label={full}>
              {relativeTime(comment.createdAt)}
            </time>
            {comment.editedAt && ' · edited'}
          </p>
          {!editing && (comment.canEdit || comment.canDelete) && (
            <div
              data-comment-actions
              className="flex shrink-0 items-center gap-3 opacity-0 transition-opacity group-hover/comment:opacity-100 group-focus-within/comment:opacity-100 [@media(hover:none)]:opacity-100"
            >
              {comment.canEdit && (
                <button type="button" onClick={() => setDraft(comment.body)} className={QUIET}>
                  Edit
                </button>
              )}
              {comment.canDelete && (
                <button type="button" onClick={onDelete} className={QUIET}>
                  Delete
                </button>
              )}
            </div>
          )}
        </div>
        {editing ? (
          <div className="space-y-2 pt-1">
            <textarea
              ref={boxRef}
              aria-label="Edit comment"
              value={draft}
              onChange={(e) => {
                setDraft(e.target.value);
                setFailed(false);
              }}
              onKeyDown={(e) => {
                if (e.key === 'Escape') {
                  e.preventDefault();
                  cancel();
                } else if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
                  e.preventDefault();
                  void save();
                }
              }}
              className={BOX}
            />
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p role="status" className="text-sm text-destructive">
                {failed && "Couldn't save your edit. Try again."}
              </p>
              <div className="flex items-center gap-3">
                <Count length={commentLength(draft)} />
                <button type="button" onClick={cancel} className={QUIET}>
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={() => void save()}
                  disabled={!isSendable(draft) || saving}
                  className={PRIMARY}
                >
                  Save
                </button>
              </div>
            </div>
          </div>
        ) : (
          <p
            data-comment-body
            className="whitespace-pre-wrap break-words text-base leading-relaxed text-foreground/88"
          >
            <LinkedText text={comment.body} rel="noopener noreferrer nofollow" />
          </p>
        )}
        {deleteFailed && (
          <p role="status" className="text-sm text-destructive">
            Couldn&apos;t delete the comment. Try again.
          </p>
        )}
      </div>
    </li>
  );
}

/** The character count, once the text passes 1800 (R28). */
function Count({ length }: { length: number }) {
  if (length <= COMMENT_COUNT_FROM) return null;
  return (
    <span
      className={cn(
        'text-xs tabular-nums',
        length > COMMENT_MAX ? 'text-destructive' : 'text-muted-foreground'
      )}
    >
      {length}/{COMMENT_MAX}
    </span>
  );
}

/**
 * The box at the bottom (R28): it grows with its text, Ctrl/Cmd+Enter sends,
 * Enter is a new line, and the count shows once the text passes 1800. A sent
 * comment clears the box and keeps its focus; a refused one keeps its text.
 */
function Composer({ onAdd }: { onAdd: (body: string) => Promise<boolean> }) {
  const [text, setText] = useState('');
  const [sending, setSending] = useState(false);
  const [failed, setFailed] = useState(false);
  const boxRef = useRef<HTMLTextAreaElement>(null);
  const busy = useRef(false);

  useLayoutEffect(() => {
    const el = boxRef.current;
    if (!el) return;
    el.style.height = 'auto';
    el.style.height = `${el.scrollHeight}px`;
  }, [text]);

  const length = commentLength(text);
  const sendable = isSendable(text);

  const send = async () => {
    if (!sendable || busy.current) return;
    busy.current = true;
    setSending(true);
    setFailed(false);
    const ok = await onAdd(text.trim());
    busy.current = false;
    setSending(false);
    if (ok) {
      setText('');
      boxRef.current?.focus();
    } else {
      setFailed(true);
    }
  };

  return (
    <div className="space-y-2">
      <textarea
        ref={boxRef}
        aria-label="Add a comment"
        placeholder="Add a comment…"
        rows={2}
        value={text}
        onChange={(e) => {
          setText(e.target.value);
          setFailed(false);
        }}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
            e.preventDefault();
            void send();
          }
        }}
        className={BOX}
      />
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p role="status" className="text-sm text-destructive">
          {failed && "Couldn't add your comment. Try again."}
        </p>
        <div className="flex items-center gap-3">
          <Count length={length} />
          <button
            type="button"
            onClick={() => void send()}
            disabled={!sendable || sending}
            className={PRIMARY}
          >
            Comment
          </button>
        </div>
      </div>
    </div>
  );
}

const BOX =
  'block w-full resize-none overflow-hidden rounded-md border border-border bg-background px-3 py-2 text-base leading-6 text-foreground caret-primary outline-none placeholder:text-muted-foreground focus-visible:ring-2 focus-visible:ring-ring/40';
const PRIMARY =
  'h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground outline-none transition-colors hover:bg-primary/90 focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-default disabled:opacity-50';
const QUIET =
  'text-xs text-muted-foreground outline-none transition-colors hover:text-foreground focus-visible:text-foreground focus-visible:underline';
