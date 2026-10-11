'use client';

import { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { CheckSquare, CornerDownRight, FileText, Plus } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { ChecklistItem } from '@/services/api';

/**
 * The grid's pinned bottom bar: the new-milestone composer (FS-FSFEC R29–R41).
 *
 * Three states. Resting is a quiet "+ New milestone" button. Naming takes a
 * title and creates a top-level task. Filling then adds task steps to the
 * milestone just created, which the grid traces because it may sit anywhere
 * on the page, far from this input (R33). With the type set to note, naming
 * makes top-level notes instead and never fills, since notes take no steps
 * (R39).
 *
 * Tab means the opposite of the list's add bar here: it takes you *out* of
 * filling to name the next milestone, never *in* under an existing one (D4,
 * D5). The composer only ever traces what it created itself.
 *
 * The mode lives here and nowhere else, so nothing persists it and every
 * load starts resting (R42). The typed text is the caller's, shared with the
 * list's add bar, so it outlives a switch of view (R41). The caller learns the
 * traced id through `onTraceChange`, which is all the grid needs to draw the
 * trace.
 *
 * Only the traced milestone's id is held. Whether it can still take steps,
 * and what it is called, are asked of `resolveMilestone` on every render, so
 * a rename shows at once and a milestone archived, deleted, converted, nested
 * or filtered out drops the composer to naming (R38).
 *
 * An empty plan has no grid for a lone bar to sit under, so resting moves up
 * into the grid as a "Start your first milestone" invitation (R46, D5). It is
 * the same resting state drawn somewhere else, not a second way in: it opens
 * the same naming, and Esc hands focus back to it.
 */

type ComposerState =
  | { mode: 'resting' }
  | { mode: 'naming' }
  | { mode: 'filling'; milestoneId: string };

export interface MilestoneComposerProps {
  /** What naming creates: a milestone (task) or a top-level note (R39). */
  type: 'task' | 'note';
  /** Flip `type`; the icon is offered in naming only. */
  onToggleType: () => void;
  /** The typed text, kept by the caller so a view switch doesn't lose it. */
  text: string;
  onTextChange: (text: string) => void;
  /**
   * Create a top-level item of `type` and bring its page into view. Resolves
   * to the created item, or null when the create failed and the caller has
   * already said so.
   */
  onCreate: (
    text: string,
    type: 'task' | 'note'
  ) => Promise<ChecklistItem | null>;
  /** Create a task step under `parentId`; null on failure, as above. */
  onAddStep: (parentId: string, text: string) => Promise<ChecklistItem | null>;
  /**
   * The milestone with this id while it can still take steps, else null.
   * Asked on every render, never cached.
   */
  resolveMilestone: (id: string) => ChecklistItem | null;
  /** The traced milestone's id whenever it changes; null when none is. */
  onTraceChange: (id: string | null) => void;
  /**
   * Where the grid invites the first milestone, while the plan is empty;
   * resting is drawn there instead of as the bar (R46). Null otherwise.
   */
  invitation?: HTMLElement | null;
}

export default function MilestoneComposer({
  type,
  onToggleType,
  text,
  onTextChange: setText,
  onCreate,
  onAddStep,
  resolveMilestone,
  onTraceChange,
  invitation = null,
}: MilestoneComposerProps) {
  // Back from the list with text still typed, the composer opens on it (R41).
  const [state, setState] = useState<ComposerState>(() =>
    text ? { mode: 'naming' } : { mode: 'resting' }
  );
  const [isSubmitting, setIsSubmitting] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  // The resting bar, or the invitation standing in for it.
  const restingRef = useRef<HTMLButtonElement>(null);
  // Set by a move someone made, so focus follows it. Never set on a page
  // load, a remount, or the milestone being lost from under the composer:
  // none of those should pull the cursor down to the bar.
  const focusNext = useRef(false);

  const milestone =
    state.mode === 'filling' ? resolveMilestone(state.milestoneId) : null;
  // A traced milestone that can no longer take steps reads as naming at once,
  // in this very render, so no step can be aimed at it (R38)...
  const lost = state.mode === 'filling' && !milestone;
  const mode = lost ? 'naming' : state.mode;
  // ...and is forgotten, not held, so it can't re-trace itself when, say, the
  // type filter is switched back.
  useEffect(() => {
    if (lost) setState({ mode: 'naming' });
  }, [lost]);

  const tracedId = milestone?.id ?? null;
  useEffect(() => {
    onTraceChange(tracedId);
  }, [tracedId, onTraceChange]);
  // Gone (the view switched), so nothing is traced any more.
  useEffect(() => () => onTraceChange(null), [onTraceChange]);

  // Each move puts the cursor where the next keystroke belongs: in the input
  // while open, back on the resting button after Esc.
  useEffect(() => {
    if (!focusNext.current) return;
    focusNext.current = false;
    if (mode === 'resting') restingRef.current?.focus();
    else inputRef.current?.focus();
  }, [mode]);

  const go = (next: ComposerState) => {
    focusNext.current = true;
    setState(next);
  };

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    const value = text.trim();
    // Read-only while a create is in flight, and Enter is ignored rather than
    // queued (R43).
    if (!value || isSubmitting || mode === 'resting') return;

    setIsSubmitting(true);
    try {
      if (milestone) {
        // A failed step keeps its text to retry (§Edge States).
        if (await onAddStep(milestone.id, value)) setText('');
      } else {
        const created = await onCreate(value, type);
        // A failed create stays in naming with the text kept (R31). A note
        // takes no steps, so naming carries on for the next one (R39). A
        // milestone the type filter hides is traced and at once dropped
        // again, so the composer stays naming then too (R38).
        if (created) {
          setText('');
          if (type === 'task') go({ mode: 'filling', milestoneId: created.id });
        }
      }
    } finally {
      setIsSubmitting(false);
      inputRef.current?.focus();
    }
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Tab') {
      // Swallowed in both states, so focus never leaves mid-entry. From
      // filling, either direction steps out to name the next milestone, text
      // kept and trace cleared (R35). Held still while a create is in flight,
      // so its result can't land in a state it wasn't typed for.
      e.preventDefault();
      if (mode === 'filling' && !isSubmitting) go({ mode: 'naming' });
    } else if (e.key === 'Escape' && !isSubmitting) {
      // Rest, as the card's own step composer does: text and trace go (R36).
      e.preventDefault();
      setText('');
      go({ mode: 'resting' });
    }
  };

  // The invitation stands for as long as the plan is empty, naming included,
  // so the grid never goes blank under the cursor; activating it again only
  // returns the cursor to the input, text kept.
  const invite =
    invitation &&
    createPortal(
      <button
        ref={restingRef}
        type="button"
        onClick={() =>
          mode === 'resting' ? go({ mode: 'naming' }) : inputRef.current?.focus()
        }
        data-milestone-invitation
        className="group/invite flex w-full flex-col items-center gap-2 rounded-xl bg-foreground/[0.04] px-4 py-10 text-base text-foreground/50 outline-none backdrop-blur-sm transition-[background-color,box-shadow,color] duration-300 hover:bg-foreground/[0.06] hover:text-primary hover:shadow-[0_2px_18px_-12px_rgba(247,111,83,0.55)] focus-visible:text-primary focus-visible:ring-2 focus-visible:ring-primary/40"
      >
        <span className="grid h-9 w-9 place-items-center rounded-full bg-primary/10 text-primary">
          <Plus aria-hidden strokeWidth={2} className="h-[18px] w-[18px]" />
        </span>
        Start your first milestone
      </button>,
      invitation
    );

  if (mode === 'resting') {
    return (
      <div data-milestone-composer="resting">
        {/* A real button, so click, Enter and Space all open it (R30). With
            the invitation up there is no bar: it would sit under nothing. */}
        {invite || (
          <button
            ref={restingRef}
            type="button"
            onClick={() => go({ mode: 'naming' })}
            className="group/rest flex w-full items-center gap-2 border-b border-foreground/10 py-3 text-left text-base text-foreground/40 outline-none transition-colors duration-300 hover:border-foreground/20 hover:text-primary focus-visible:border-primary/50 focus-visible:text-primary"
          >
            <span className="-ml-2 grid h-8 w-8 shrink-0 place-items-center">
              <Plus aria-hidden strokeWidth={2} className="h-[18px] w-[18px]" />
            </span>
            {type === 'note' ? 'New note' : 'New milestone'}
          </button>
        )}
      </div>
    );
  }

  const title = milestone?.description;
  const noting = !milestone && type === 'note';

  return (
    <div data-milestone-composer={mode}>
      {invite}
      <form
        onSubmit={submit}
        className="group/add relative flex items-center border-b border-foreground/15 py-3 transition-[border-color] duration-300 focus-within:border-transparent"
      >
        {/* The same ember underline the list's add bar draws on focus. */}
        <span
          aria-hidden
          className="pointer-events-none absolute inset-x-0 -bottom-px h-px origin-left scale-x-0 bg-gradient-to-r from-primary via-primary/60 to-primary/0 shadow-[0_0_10px_rgba(247,111,83,0.45)] transition-transform duration-500 [transition-timing-function:cubic-bezier(0.22,1,0.36,1)] group-focus-within/add:scale-x-100"
        />
        {milestone ? (
          // No type control while filling: steps made here are always tasks
          // (R32). The glyph only says a step is being typed.
          <span
            aria-hidden
            className="-ml-2 grid h-8 w-8 shrink-0 place-items-center text-foreground/40 transition-colors duration-300 group-focus-within/add:text-primary"
          >
            <CornerDownRight strokeWidth={1.75} className="h-[18px] w-[18px]" />
          </span>
        ) : (
          // Naming's type icon, the list bar's own toggle: note makes the
          // composer a note maker with no filling (R39). Focus goes straight
          // back to the input, so the toggle never costs the typing.
          <button
            type="button"
            onClick={() => {
              onToggleType();
              inputRef.current?.focus();
            }}
            title={
              noting
                ? 'Creating a note — click to switch to milestone'
                : 'Creating a milestone — click to switch to note'
            }
            aria-label={noting ? 'Switch to adding a task' : 'Switch to adding a note'}
            className="-ml-2 grid h-8 w-8 shrink-0 place-items-center rounded-full text-foreground/40 transition-colors duration-300 hover:bg-primary/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 group-focus-within/add:text-primary"
          >
            {/* Both glyphs share one grid cell and cross-fade on swap. */}
            <CheckSquare
              strokeWidth={1.75}
              className={cn(
                'col-start-1 row-start-1 h-[18px] w-[18px] transition-all duration-300',
                noting
                  ? '-rotate-90 scale-75 opacity-0'
                  : 'rotate-0 scale-100 opacity-100'
              )}
            />
            <FileText
              strokeWidth={1.75}
              className={cn(
                'col-start-1 row-start-1 h-[18px] w-[18px] transition-all duration-300',
                noting
                  ? 'rotate-0 scale-100 opacity-100'
                  : 'rotate-90 scale-75 opacity-0'
              )}
            />
          </button>
        )}
        {/* Names the traced milestone where the typing happens, since the
            card itself may be far off (R32, R33). The input's own label
            carries the same name for assistive tech. */}
        {milestone && (
          <span
            aria-hidden
            data-composer-target
            className="mr-2 max-w-[40%] shrink-0 truncate text-sm text-primary/80"
          >
            ↳ steps for {title}
          </span>
        )}
        {/* readOnly, not disabled, while saving: disabled drops focus. */}
        <input
          ref={inputRef}
          type="text"
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={onKeyDown}
          readOnly={isSubmitting}
          placeholder={
            milestone
              ? 'Add a step…'
              : noting
              ? 'Jot down a note…'
              : 'Name a new milestone…'
          }
          aria-label={
            milestone
              ? `Add a step to ${title}`
              : noting
              ? 'New note'
              : 'Name a new milestone'
          }
          className="h-8 min-w-0 flex-1 bg-transparent text-base leading-8 text-foreground caret-primary placeholder:italic placeholder:text-foreground/35 focus:outline-none read-only:opacity-60"
        />
        <button
          type="submit"
          disabled={isSubmitting}
          className={cn(
            'ml-3 inline-flex h-8 shrink-0 items-center gap-1.5 rounded-full px-3.5 text-sm font-medium transition-all duration-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:pointer-events-none disabled:opacity-60',
            text.trim()
              ? 'bg-primary text-primary-foreground shadow-[0_2px_14px_-3px_rgba(247,111,83,0.6)] hover:bg-primary/90'
              : 'text-foreground/40 hover:bg-primary/10 hover:text-primary'
          )}
        >
          <Plus strokeWidth={2.25} className="h-3.5 w-3.5" />
          {isSubmitting ? 'Adding…' : 'Add'}
        </button>
      </form>
    </div>
  );
}
