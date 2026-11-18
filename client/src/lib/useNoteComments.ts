'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { errorCode } from '@/api/client';
import {
  createNoteComment,
  deleteNoteComment,
  listNoteComments,
  updateNoteComment,
  type NoteComment,
} from '@/api/noteComments';

/**
 * A note's comments as its page holds them (FS-65B3N R24–R30, R33): the load,
 * the list as adds, edits and deletes change it, and what each refusal means,
 * told apart by `code` alone (R30, ADR-0004). The component only draws it.
 *
 * Loading again keeps a list already shown (R28): the flags may have gone
 * stale (FORBIDDEN), but the box being typed in and any edit still open stay
 * mounted, and a failed refresh leaves the list as it was.
 */

export type CommentsLoad =
  | { state: 'loading' }
  | { state: 'ready'; comments: NoteComment[] }
  | { state: 'failed' };

/** How a change to one comment went: kept, refused, or already dealt with. */
export type CommentChange = 'done' | 'failed' | 'handled';

export interface NoteCommentsPage {
  /** The note went while the page was open (FS-PGG2N R37). */
  onGone: () => void;
  /** The item became a task elsewhere: the page loads again (R30). */
  onNotANote: () => void;
}

export function useNoteComments(planId: string, noteId: string, page: NoteCommentsPage) {
  const [load, setLoad] = useState<CommentsLoad>({ state: 'loading' });
  const [attempt, setAttempt] = useState(0);
  const [notice, setNotice] = useState<string | null>(null);
  const latest = useRef(page);
  latest.current = page;
  // Whose list is on screen: only that note's may stay up while it reloads.
  const shown = useRef<string | null>(null);

  /** The page's own answer to a code that is about the note, not a comment. */
  const handledByPage = useCallback((err: unknown) => {
    const code = errorCode(err);
    if (code === 'ITEM_NOT_A_NOTE') latest.current.onNotANote();
    else if (code === 'NOT_FOUND') latest.current.onGone();
    else return false;
    return true;
  }, []);

  useEffect(() => {
    let cancelled = false;
    const key = `${planId}/${noteId}`;
    const keep = shown.current === key;
    if (!keep) setLoad({ state: 'loading' });
    listNoteComments(planId, noteId).then(
      (comments) => {
        if (cancelled) return;
        shown.current = key;
        setLoad({ state: 'ready', comments });
      },
      (err) => {
        if (cancelled || handledByPage(err)) return;
        console.error('Failed to load comments:', err);
        if (!keep) setLoad({ state: 'failed' });
      }
    );
    return () => {
      cancelled = true;
    };
  }, [planId, noteId, attempt, handledByPage]);

  const setComments = (change: (comments: NoteComment[]) => NoteComment[]) =>
    setLoad((prev) =>
      prev.state === 'ready' ? { state: 'ready', comments: change(prev.comments) } : prev
    );
  const reload = () => setAttempt((n) => n + 1);

  /**
   * A refused edit or delete, by code (R30): the comment is gone, the flags
   * were stale, or the item is not a note any more. Anything else is the
   * caller's to show.
   */
  const triage = (err: unknown, id: string): CommentChange => {
    const code = errorCode(err);
    // On an edit or delete, NOT_FOUND is the comment, not the note (R30).
    if (code === 'NOT_FOUND') {
      setComments((all) => all.filter((c) => c.id !== id));
      setNotice('This comment was deleted.');
      return 'handled';
    }
    if (code === 'FORBIDDEN') {
      reload();
      return 'handled';
    }
    if (handledByPage(err)) return 'handled';
    console.error('Failed to change comment:', err);
    return 'failed';
  };

  const edit = async (id: string, body: string): Promise<CommentChange> => {
    setNotice(null);
    try {
      const saved = await updateNoteComment(planId, noteId, id, body);
      setComments((all) => all.map((c) => (c.id === id ? saved : c)));
      return 'done';
    } catch (err) {
      return triage(err, id);
    }
  };

  const remove = async (id: string): Promise<CommentChange> => {
    setNotice(null);
    try {
      await deleteNoteComment(planId, noteId, id);
      setComments((all) => all.filter((c) => c.id !== id));
      return 'done';
    } catch (err) {
      return triage(err, id);
    }
  };

  /** True once the comment is added, last; false with the text the caller's to keep. */
  const add = async (body: string): Promise<boolean> => {
    try {
      const made = await createNoteComment(planId, noteId, body);
      setComments((all) => [...all, made]);
      return true;
    } catch (err) {
      // The note itself is gone or a task now: the page handles it.
      if (!handledByPage(err)) console.error('Failed to add comment:', err);
      return false;
    }
  };

  return { load, notice, retry: reload, add, edit, remove };
}
